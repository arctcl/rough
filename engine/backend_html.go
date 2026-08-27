package engine

import (
	"io"
	"io/fs"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
	"golang.org/x/net/html"
)

// Node — узел HTML-дерева (мини-вёрстка).
type Node struct {
	Tag      string            // имя тега
	Attrs    map[string]string // атрибуты
	Actions  []string          // все action="..." по порядку (кнопка выполняет их последовательно)
	Text     string            // текст (для текстовых узлов)
	Children []*Node           // вложенные узлы
}

// ParseHTML разбирает HTML в наше дерево.
func ParseHTML(r io.Reader) (*Node, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	return convertNode(doc), nil
}

// tileCache — кэш распарсенного HTML тайлов (файл → дерево). HTML тайла
// статичен: парсим ОДИН раз, дальше рендерим из кэша — не открываем и не
// парсим файл на каждую отрисовку (сверхлёгкая отрисовка).
var tileCache = map[string]*Node{}

// loadTile возвращает распарсенное дерево HTML тайла, кэшируя его при первом
// чтении. Дерево не мутируется при рендере, поэтому его можно переиспользовать.
func loadTile(fsys fs.FS, file string) (*Node, error) {
	if n, ok := tileCache[file]; ok {
		return n, nil
	}
	f, err := fsys.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	n, err := ParseHTML(f)
	if err != nil {
		return nil, err
	}
	tileCache[file] = n
	return n, nil
}

// convertNode переводит дерево x/net/html в наше.
func convertNode(n *html.Node) *Node {
	nn := &Node{}
	switch n.Type {
	case html.ElementNode:
		nn.Tag = n.Data
		nn.Attrs = map[string]string{}
		for _, a := range n.Attr {
			// Несколько action="..." — все сохраняем по порядку (кнопка
			// выполняет их последовательно); в Attrs["action"] — первый.
			if a.Key == "action" {
				nn.Actions = append(nn.Actions, a.Val)
				if _, ok := nn.Attrs["action"]; !ok {
					nn.Attrs["action"] = a.Val
				}
				continue
			}
			nn.Attrs[a.Key] = a.Val
		}
	case html.TextNode:
		nn.Text = n.Data
	default:
		nn.Tag = n.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		cc := convertNode(c)
		if cc.Tag != "" || cc.Text != "" || len(cc.Children) > 0 {
			nn.Children = append(nn.Children, cc)
		}
	}
	return nn
}

// RenderHTML рендерит HTML-дерево в буфер тайла, собирая хотзоны.
// ox, oy — смещение тайла на экране (чтобы хотзоны были абсолютными).
// Цвета текста/фона по умолчанию берутся из темы (ключи fg/bg).
func RenderHTML(n *Node, b *Buffer, ox, oy int, out *[]Hotzone) {
	f := &flowState{
		fg: curTheme.ResolveColor(themeColor("fg"), tcell.ColorDefault),
		bg: curTheme.ResolveColor(themeColor("bg"), tcell.ColorDefault),
	}
	renderNode(n, b, f, ox, oy, out)
}

// flowState — курсор простого потокового рендера.
type flowState struct {
	x, y         int
	center       bool
	bold, italic bool
	underline    bool
	fg, bg       tcell.Color
}

// nl — переход на новую строку.
func (f *flowState) nl(b *Buffer) {
	f.x = 0
	f.y++
}

// style собирает текущий стиль в Style.
func (f *flowState) style() Style {
	return Style{Fg: f.fg, Bg: f.bg, Bold: f.bold, Italic: f.italic, Underline: f.underline}
}

// drawX — x-позиция начала строки с учётом центрирования (f.center).
// Нужна для хотзон: кнопка/поле рисуются по центру — и зона должна совпадать,
// иначе клики/подсветка уезжают влево.
func (f *flowState) drawX(b *Buffer, s string) int {
	if !f.center {
		return f.x
	}
	x := (b.W - uniseg.StringWidth(s)) / 2
	if x < 0 {
		x = 0
	}
	return x
}

// put рисует строку с переносом по ширине буфера.
// Сдвиг — по реальной ширине руны (uniseg): кириллица/иероглифы занимают 2 клетки,
// иначе буквы наезжают друг на друга.
func (f *flowState) put(b *Buffer, s string) {
	sw := uniseg.StringWidth(s)
	if f.center {
		f.x = (b.W - sw) / 2
		if f.x < 0 {
			f.x = 0
		}
	}
	for _, r := range s {
		// Перевод строки внутри текста (например в <pre>).
		if r == '\n' {
			f.nl(b)
			continue
		}
		if f.x >= b.W {
			f.nl(b)
			if f.center {
				f.x = (b.W - sw) / 2
				if f.x < 0 {
					f.x = 0
				}
			}
		}
		b.Set(f.x, f.y, r, f.style())
		f.x += uniseg.StringWidth(string(r))
	}
}

// putColored рисует строку плагина с поддержкой цветовых маркеров \x01{имя}.
// Маркер \x01{имя} ставит цвет текста из темы (ключ в colors: color_2, frame…),
// \x01{} — сброс к цвету по умолчанию. Маркеры невидимы и не влияют на ширину —
// рамка/ось графика не съезжают, красится только то, что обёрнуто.
func (f *flowState) putColored(b *Buffer, s string) {
	// Разбиваем строку на сегменты: каждый несёт свой цвет переднего плана.
	type seg struct {
		fg  tcell.Color
		val string
	}
	var segs []seg
	cur := f.fg
	rest := s
	for len(rest) > 0 {
		i := strings.Index(rest, "\x01")
		if i < 0 {
			segs = append(segs, seg{cur, rest})
			break
		}
		if i > 0 {
			segs = append(segs, seg{cur, rest[:i]})
		}
		rest = rest[i+1:]
		// После \x01 ждём {имя} — иначе рисуем маркер как обычный символ.
		if strings.HasPrefix(rest, "{") {
			if j := strings.Index(rest, "}"); j >= 0 {
				name := rest[1:j]
				rest = rest[j+1:]
				if name == "" {
					cur = f.fg // сброс
				} else {
					cur = curTheme.ResolveColor(themeColor(name), f.fg)
				}
				continue
			}
		}
		segs = append(segs, seg{cur, "\x01"})
	}
	// Центрируем ВСЮ строку один раз (по ширине без маркеров), а не каждый
	// сегмент: иначе при f.center каждый сегмент ляжет по своему центру и
	// наложится на соседний (графики внутри <div align="center"> ломались).
	center := f.center
	if center {
		visW := 0
		for _, g := range segs {
			visW += uniseg.StringWidth(g.val)
		}
		f.x = (b.W - visW) / 2
		if f.x < 0 {
			f.x = 0
		}
	}
	// Рисуем сегменты подряд (центр уже учтён — повторно не центрируем).
	f.center = false
	for _, g := range segs {
		old := f.fg
		f.fg = g.fg
		f.put(b, g.val)
		f.fg = old
	}
	f.center = center
}

// StripMarkers убирает цветовые маркеры \x01{имя} из строки. Нужно там, где
// вывод плагина показывается БЕЗ раскраски (статус-строка, блоки output):
// иначе в текст попал бы видимый мусор {color_2}.
func StripMarkers(s string) string {
	var sb strings.Builder
	rest := s
	for len(rest) > 0 {
		i := strings.Index(rest, "\x01")
		if i < 0 {
			sb.WriteString(rest)
			break
		}
		sb.WriteString(rest[:i])
		rest = rest[i+1:]
		if strings.HasPrefix(rest, "{") {
			if j := strings.Index(rest, "}"); j >= 0 {
				rest = rest[j+1:]
				continue
			}
		}
		sb.WriteString("\x01")
	}
	return sb.String()
}

// renderNode обходит дерево и рисует в буфер.
func renderNode(n *Node, b *Buffer, f *flowState, ox, oy int, out *[]Hotzone) {
	// Атрибуты color/bg: hex, номер палитры терминала или имя из темы.
	oldFg, oldBg := f.fg, f.bg
	if c := n.Attrs["color"]; c != "" {
		f.fg = curTheme.ResolveColor(c, tcell.ColorDefault)
	}
	if c := n.Attrs["bg"]; c != "" {
		f.bg = curTheme.ResolveColor(c, tcell.ColorDefault)
	}
	defer func() {
		f.fg, f.bg = oldFg, oldBg
	}()

	if n.Text != "" {
		// Пробельный текстовый узел с переводом строки ("\n  " между тегами)
		// не рисуем: блочные элементы сами делают перевод строки в начале и
		// конце, иначе между ними накапливалось бы по 2 пустые строки.
		// Обычный пробел внутри строки (без \n) рисуем как есть.
		if strings.Contains(n.Text, "\n") && strings.TrimSpace(n.Text) == "" {
			// пропускаем межблочный разрыв
		} else {
			f.put(b, n.Text)
		}
	}
	switch n.Tag {
	case "br":
		f.nl(b)
	case "h1", "p":
		f.nl(b)
		old := *f
		f.bold = n.Tag == "h1"
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		*f = old
		f.nl(b)
	case "div":
		// Блок: дети, затем — если у блока есть id и в кэше лежит результат — его строки.
		// Так <div id="out"></div> становится приёмником вывода команды (output="out").
		// Внутри <row> div с width="50%" — колонка (см. renderRow).
		// align="center" — содержимое по центру блока (по горизонтали и вертикали).
		if n.Attrs["align"] == "center" {
			renderCenteredDiv(n, b, f, ox, oy, out)
			break
		}
		f.nl(b)
		old := *f
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		if id := n.Attrs["id"]; id != "" {
			if lines, ok := outputCache[id]; ok {
				for _, ln := range lines {
					f.put(b, ln)
					f.nl(b)
				}
			}
		}
		*f = old
		f.nl(b)
	case "b":
		oldBold := f.bold
		f.bold = true
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		// Возвращаем ТОЛЬКО флаг стиля — позицию x/y не трогаем: иначе после
		// <b>текст</b> курсор сбрасывается и следующий текст перезаписывает жирный.
		f.bold = oldBold
	case "i":
		oldItalic := f.italic
		f.italic = true
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		f.italic = oldItalic
	case "center":
		f.nl(b)
		old := *f
		f.center = true
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		*f = old
		f.nl(b)
	case "hr":
		// Горизонтальная линия на всю ширину тайла.
		f.nl(b)
		hh := curTheme.Sym("tile_h", "─")
		for x := f.x; x < b.W; x++ {
			b.Set(x, f.y, hh, f.style())
		}
		f.nl(b)
	case "pre":
		// Моноширинный блок: блок-абзац без стилей, как есть.
		f.nl(b)
		old := *f
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		*f = old
		f.nl(b)
	case "checkbox":
		// Чекбокс [x]/[ ]: состояние читается у плагина (action + ":get").
		f.nl(b)
		label := strings.TrimSpace(textContent(n))
		act := n.Attrs["action"]
		mark := "[ ]"
		if checkboxOn(act) {
			mark = "[x]"
		}
		s := mark + " " + label
		x0, y0 := f.drawX(b, s), f.y
		f.put(b, s)
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Action: act, Actions: n.Actions, Output: n.Attrs["output"]})
		f.nl(b)
	case "table":
		f.nl(b)
		renderTable(n, b, f)
		f.nl(b)
	case "row":
		f.nl(b)
		renderRow(n, b, f, ox, oy, out)
		f.nl(b)
	case "button":
		f.nl(b)
		label := strings.TrimSpace(textContent(n))
		act := n.Attrs["action"]
		bl := curTheme.Sym("button_l", "⟨")
		br := curTheme.Sym("button_r", "⟩")
		s := string(bl) + " " + label + " " + string(br)
		x0, y0 := f.drawX(b, s), f.y
		f.put(b, s)
		// async="1" — кнопка запускает действие в фоне (не блокирует ядро).
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1,
			Action: act, Actions: n.Actions, Output: n.Attrs["output"], Async: n.Attrs["async"] != ""})
		f.nl(b)
	case "a":
		f.nl(b)
		label := strings.TrimSpace(textContent(n))
		href := n.Attrs["href"]
		s := "→ " + label
		x0, y0 := f.drawX(b, s), f.y
		f.put(b, s)
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Href: href, Kind: "nav"})
		f.nl(b)
	case "input":
		// Поле ввода: обведённое рамкой, как HTML-инпут. Клик — активирует поле,
		// ввод идёт ПРЯМО в него (внутри рамки), Enter — выполняет action,
		// результат — в output (если задан). Никакой модалки по центру.
		f.nl(b)
		label := n.Attrs["label"]
		if label == "" {
			label = strings.TrimSpace(textContent(n))
		}
		act := n.Attrs["action"]
		il := curTheme.Sym("input_l", "[")
		ir := curTheme.Sym("input_r", "]")
		ic := curTheme.Sym("input_icon", "✎")
		// Активное поле: лейбл заменяется вводом — карандаш и скобки остаются.
		// Покой: [ ✎ Пакет ]  →  Ввод: [ ✎ ssh█ ]
		s := string(il) + " " + string(ic) + " "
		if inputMode && inputAction == act && inputLabel == label {
			s += inputBuf + string(curTheme.Sym("cursor", "█"))
		} else {
			s += label
		}
		s += " " + string(ir)
		x0, y0 := f.drawX(b, s), f.y
		f.put(b, s)
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Action: act, Actions: n.Actions, Kind: "input", Label: label, Output: n.Attrs["output"]})
		f.nl(b)
	case "select":
		// Выпадающий список: клик открывает меню выбора, выбор выполняет action
		// с выбранным значением (дописывается ":вариант").
		f.nl(b)
		label := n.Attrs["label"]
		if label == "" {
			label = strings.TrimSpace(textContent(n))
		}
		act := n.Attrs["action"]
		sl := curTheme.Sym("select_icon", "▼")
		// Подпись — отдельное слово, рядом — текущее выбранное значение.
		s := label + "  " + string(sl) + " " + currentSelect(act)
		x0, y0 := f.drawX(b, s), f.y
		f.put(b, s)
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Action: act, Actions: n.Actions, Kind: "select", Label: label, Output: n.Attrs["output"], Options: n.Attrs["options"]})
		f.nl(b)
	case "img":
		// Картинка PPM (P6): рисуется половинчатыми блоками ▀▄█.
		f.nl(b)
		renderImg(n, b, f)
		f.nl(b)
	case "plugin":
		f.nl(b)
		renderPlugin(n, b, f)
		f.nl(b)
	default:
		// Неизвестные теги (html, body, span и т.п.) — просто обходим детей.
		// Без этого документ/тайл не рендерится вообще.
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
	}
}

// textContent собирает весь текст внутри узла.
func textContent(n *Node) string {
	var sb strings.Builder
	var walk func(*Node)
	walk = func(x *Node) {
		if x.Text != "" {
			sb.WriteString(x.Text)
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// renderRow рисует <row>: дети-<div width="..."> — колонки, делят ширину тайла
// (width в %/px, пусто — поровну). Каждая колонка рендерится в свой мини-буфер
// своей ширины, поэтому плагины внутри видят реальный размер через Window()
// и сжимаются/растягиваются сами. Поменял 50% на 40% — колонка просто стала уже.
func renderRow(n *Node, b *Buffer, f *flowState, ox, oy int, out *[]Hotzone) {
	cols := childrenTags(n, "div")
	if len(cols) == 0 {
		// Нет колонок — дети идут подряд, как обычная строка.
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		return
	}
	// Считаем ширины колонок: явная width, остаток — поровну авто-колонкам.
	total := b.W
	widths := make([]int, len(cols))
	used, auto := 0, 0
	for i, c := range cols {
		if c.Attrs["width"] != "" {
			widths[i] = parseLen(c.Attrs["width"], total)
			used += widths[i]
		} else {
			auto++
		}
	}
	if auto > 0 {
		share := 0
		if rest := total - used; rest > 0 {
			share = rest / auto
		}
		for i, c := range cols {
			if c.Attrs["width"] == "" {
				widths[i] = share
			}
		}
	}
	// Каждую колонку рендерим в свой буфер и вставляем рядом по x-смещению.
	// Ряд кладём на ТЕКУЩУЮ строку потока f.y (а не в 0 — иначе ряды
	// накладываются друг на друга и затирают контент выше). После ряда
	// поток сдвигаем на высоту самого высокого столбца.
	rowY := f.y
	maxH := 0
	x := 0
	for i, c := range cols {
		if widths[i] <= 0 {
			continue
		}
		cb := NewBuffer(widths[i], b.H)
		cf := &flowState{fg: f.fg, bg: f.bg}
		renderNode(c, cb, cf, ox+x, oy+rowY, out)
		b.Copy(cb, x, rowY)
		if h := usedHeight(cb); h > maxH {
			maxH = h
		}
		x += widths[i]
	}
	f.y = rowY + maxH
}

// childrenTags возвращает детей-элементы с заданным тегом.
func childrenTags(n *Node, tag string) []*Node {
	var out []*Node
	for _, c := range n.Children {
		if c.Tag == tag {
			out = append(out, c)
		}
	}
	return out
}

// renderCenteredDiv — блок <div align="center">: содержимое по центру блока
// и по горизонтали, и по вертикали (например, часы по центру тайла).
// Рендерим в мини-буфер, узнаём реальную высоту содержимого и вставляем
// с вертикальным смещением; хотзоны сдвигаем так же.
func renderCenteredDiv(n *Node, b *Buffer, f *flowState, ox, oy int, out *[]Hotzone) {
	cb := NewBuffer(b.W, b.H)
	cf := &flowState{fg: f.fg, bg: f.bg, center: true}
	var hz []Hotzone
	for _, c := range n.Children {
		renderNode(c, cb, cf, ox, oy, &hz)
	}
	if id := n.Attrs["id"]; id != "" {
		if lines, ok := outputCache[id]; ok {
			for _, ln := range lines {
				cf.put(cb, ln)
				cf.nl(cb)
			}
		}
	}
	h := usedHeight(cb)
	// Центрируем по видимой высоте тайла, а не по запасному буферу скролла
	// (иначе контент уезжает вниз за экран). curViewH ставит renderTile.
	vh := b.H
	if curViewH > 0 && curViewH < vh {
		vh = curViewH
	}
	yOff := (vh - h) / 2
	if yOff < 0 {
		yOff = 0
	}
	b.Copy(cb, 0, yOff)
	for _, z := range hz {
		z.Y += yOff
		*out = append(*out, z)
	}
}

// usedHeight — высота занятого содержимым (нижняя непустая строка + 1).
func usedHeight(b *Buffer) int {
	for y := b.H - 1; y >= 0; y-- {
		for x := 0; x < b.W; x++ {
			if b.cells[y][x].Rune != 0 {
				return y + 1
			}
		}
	}
	return 0
}

// renderTable рисует <table>: строки <tr>, ячейки <td>/<th> (th — жирный).
// Колонки выравниваются по максимальной ширине, разделяются "│".
func renderTable(n *Node, b *Buffer, f *flowState) {
	type cell struct {
		text string
		bold bool
	}
	rows := tableRows(n)
	if len(rows) == 0 {
		return
	}
	grid := make([][]cell, 0, len(rows))
	cols := 0
	for _, tr := range rows {
		var row []cell
		for _, td := range tr.Children {
			if td.Tag == "td" || td.Tag == "th" {
				row = append(row, cell{text: strings.TrimSpace(textContent(td)), bold: td.Tag == "th"})
			}
		}
		if len(row) > cols {
			cols = len(row)
		}
		grid = append(grid, row)
	}
	// Ширины колонок — по самой широкой ячейке.
	widths := make([]int, cols)
	for _, row := range grid {
		for i, c := range row {
			if w := uniseg.StringWidth(c.text); w > widths[i] {
				widths[i] = w
			}
		}
	}
	// Рисуем: текст + отступ до ширины колонки + разделитель.
	old := *f
	// Таблица — фиксированная сетка: отключаем центрирование ячеек,
	// иначе каждая ячейка рисуется с центра и обрезается.
	f.center = false
	for _, row := range grid {
		for i := 0; i < cols; i++ {
			var c cell
			if i < len(row) {
				c = row[i]
			}
			wasBold := f.bold
			f.bold = c.bold
			f.put(b, c.text)
			f.bold = wasBold
			for p := uniseg.StringWidth(c.text); p < widths[i]; p++ {
				f.put(b, " ")
			}
			if i < cols-1 {
				f.put(b, " │ ")
			}
		}
		f.nl(b)
	}
	*f = old
}

// tableRows собирает все <tr> внутри таблицы (парсер HTML может обернуть их в tbody).
func tableRows(n *Node) []*Node {
	var rows []*Node
	var walk func(x *Node)
	walk = func(x *Node) {
		if x.Tag == "tr" {
			rows = append(rows, x)
			return
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return rows
}
