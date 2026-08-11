package engine

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
	"golang.org/x/net/html"
)

// Node — узел HTML-дерева (мини-вёрстка).
type Node struct {
	Tag      string            // имя тега
	Attrs    map[string]string // атрибуты
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

// convertNode переводит дерево x/net/html в наше.
func convertNode(n *html.Node) *Node {
	nn := &Node{}
	switch n.Type {
	case html.ElementNode:
		nn.Tag = n.Data
		nn.Attrs = map[string]string{}
		for _, a := range n.Attr {
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

// Hotzone — кликабельная зона (кнопка/ссылка/поле ввода) в абсолютных координатах экрана.
type Hotzone struct {
	X, Y, W, H int
	Action     string // что выполнить по клику
	Href       string // куда перейти (роут)
	Kind       string // вид зоны: "" (действие), "nav" (ссылка), "input" (поле ввода), "select"
	Label      string // подпись для окна ввода
	Output     string // id блока, куда направить вывод (пусто = статус-строка)
	Options    string // варианты для select (через ":")
}

// hotzones — хотзоны последнего кадра, по ним делаем hit-test мыши.
var hotzones []Hotzone

// outputCache — результаты команд, направленных в блоки с id (id блока → строки).
// Блок <div id="..."> или <col id="..."> при рендере рисует эти строки.
var outputCache = map[string][]string{}

// HitTest ищет хотзону по координатам клика.
func HitTest(x, y int) (kind, action, href, label, output, options string) {
	for _, hz := range hotzones {
		if x >= hz.X && x < hz.X+hz.W && y >= hz.Y && y < hz.Y+hz.H {
			return hz.Kind, hz.Action, hz.Href, hz.Label, hz.Output, hz.Options
		}
	}
	return "", "", "", "", "", ""
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
		f.put(b, n.Text)
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
		old := *f
		f.bold = true
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		*f = old
	case "i":
		old := *f
		f.italic = true
		for _, c := range n.Children {
			renderNode(c, b, f, ox, oy, out)
		}
		*f = old
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
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Action: act})
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
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Action: act, Output: n.Attrs["output"]})
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
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Action: act, Kind: "input", Label: label, Output: n.Attrs["output"]})
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
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Action: act, Kind: "select", Label: label, Output: n.Attrs["output"], Options: n.Attrs["options"]})
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

// renderPlugin обрабатывает тег <plugin>: собирает пайп из атрибутов,
// выполняет его и выводит строки результата в тайл.
// pluginCache — кэш результатов <plugin>: не перезапускается чаще interval.
var pluginCache = map[string]pluginEntry{}

// pluginEntry — кэшированный результат плагина и время последнего запуска.
type pluginEntry struct {
	at    time.Time
	lines []string
}

func renderPlugin(n *Node, b *Buffer, f *flowState) {
	// Размер окна тайла — чтобы рисовалки (bars и т.п.) адаптировались.
	// height на <plugin> — высота зоны графика (chart рисует на ней).
	curW, curH = b.W, b.H
	if hv := n.Attrs["height"]; hv != "" {
		if hh := parseLen(hv, b.H); hh > 0 {
			curH = hh
		}
	}

	steps := pluginSteps(n)
	if len(steps) == 0 {
		f.put(b, "ошибка: пустой плагин")
		return
	}
	// Интервал обновления: не задан → дефолт 2 секунды (не дёргаем чаще).
	iv := parseDur(n.Attrs["interval"])
	if iv <= 0 {
		iv = 2 * time.Second
	}
	key := strings.Join(steps, "|")
	curPluginKey = key // сигнатура для stateful-плагинов (chart)
	var lines []string
	if c, ok := pluginCache[key]; ok && time.Since(c.at) < iv {
		lines = c.lines // ещё не время — показываем прошлый результат
	} else {
		out, err := RunSteps(steps, nil)
		if err != nil {
			lines = []string{"ошибка: " + err.Error()}
		} else {
			lines = out
		}
		pluginCache[key] = pluginEntry{at: time.Now(), lines: lines}
	}
	for _, ln := range lines {
		f.put(b, ln)
		f.nl(b)
	}
}

// pluginSteps собирает пайп из атрибутов <plugin>.
// Явный pipe — приоритет; иначе sugar: [source|file:path] + name[:mask].
func pluginSteps(n *Node) []string {
	if p := n.Attrs["pipe"]; p != "" {
		return SplitSteps(p)
	}
	var steps []string
	src := n.Attrs["source"]
	if src == "" {
		if path := n.Attrs["path"]; path != "" {
			src = "file:" + path
		}
	}
	if src != "" {
		steps = append(steps, src)
	}
	name := n.Attrs["name"]
	if name == "" {
		name = "text"
	}
	if m := n.Attrs["mask"]; m != "" {
		name += ":" + m
	}
	steps = append(steps, name)
	return steps
}

// parseDur разбирает длительность вида "1s", "500ms", "1m".
func parseDur(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
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
	x := 0
	for i, c := range cols {
		if widths[i] <= 0 {
			continue
		}
		cb := NewBuffer(widths[i], b.H)
		cf := &flowState{fg: f.fg, bg: f.bg}
		renderNode(c, cb, cf, ox+x, oy, out)
		b.Copy(cb, x, 0)
		x += widths[i]
	}
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

// checkboxOn читает состояние чекбокса: вызывает action с ":get" — плагин
// (например toggle) возвращает текущее значение; true — если включено.
func checkboxOn(act string) bool {
	steps := SplitSteps(act)
	if len(steps) == 0 {
		return false
	}
	out, err := RunSteps([]string{steps[len(steps)-1] + ":get"}, nil)
	if err != nil || len(out) == 0 {
		return false
	}
	v := strings.TrimSpace(out[len(out)-1])
	return v == "1" || strings.EqualFold(v, "on") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// selectValue — выбранные значения select по action (для показа в подписи).
var selectValue = map[string]string{}

// currentSelect возвращает текущее значение select: сначала — выбранное в сессии,
// затем пробует прочитать через плагин (последний шаг action + ":get", как у
// toggle/set). Если не знаем — "?".
func currentSelect(act string) string {
	if v, ok := selectValue[act]; ok && v != "" {
		return v
	}
	steps := SplitSteps(act)
	if len(steps) > 0 {
		if out, err := RunSteps([]string{steps[len(steps)-1] + ":get"}, nil); err == nil && len(out) > 0 {
			return strings.TrimSpace(out[len(out)-1])
		}
	}
	return "?"
}

// ppmImage — растровое изображение PPM (P6), RGB-пиксели.
type ppmImage struct {
	w, h int
	data []byte // w*h*3 байт RGB
}

// pixel возвращает цвет пикселя (px, py).
func (img *ppmImage) pixel(px, py int) tcell.Color {
	i := (py*img.w + px) * 3
	if i+2 >= len(img.data) {
		return tcell.ColorDefault
	}
	return tcell.NewRGBColor(int32(img.data[i]), int32(img.data[i+1]), int32(img.data[i+2]))
}

// renderImg рисует PPM-картинку половинчатыми блоками ▀▄█ (2 пикселя в блок).
// Масштаб по ширине тайла, цвета — как в картинке.
func renderImg(n *Node, b *Buffer, f *flowState) {
	path := n.Attrs["src"]
	if path == "" {
		f.put(b, "img: нет src")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		f.put(b, "img: "+err.Error())
		return
	}
	img, err := decodePPM(data)
	if err != nil {
		f.put(b, "img: "+err.Error())
		return
	}
	// Масштаб: сколько пикселей по горизонтали в одном блоке (минимум 1).
	sx := img.w / b.W
	if sx < 1 {
		sx = 1
	}
	for row := 0; row < b.H-1; row++ {
		top := row * 2
		if top >= img.h {
			break
		}
		f.x = 0
		for x := 0; x < b.W; x++ {
			px := x * sx
			if px >= img.w {
				break
			}
			tc := img.pixel(px, top)
			bc := tc
			if top+1 < img.h {
				bc = img.pixel(px, top+1)
			}
			tb := colorLum(tc) >= 128
			bb := colorLum(bc) >= 128
			var r rune
			var col tcell.Color
			switch {
			case tb && bb:
				r = '█'
				col = midColor(tc, bc)
			case tb:
				r = '▀'
				col = tc
			case bb:
				r = '▄'
				col = bc
			default:
				r = ' '
				col = midColor(tc, bc)
			}
			b.Set(f.x, f.y, r, Style{Fg: col, Bg: col})
			f.x++
		}
		f.nl(b)
	}
}

// decodePPM разбирает PPM P6: "P6 W H MAX" + RGB-байты.
func decodePPM(b []byte) (*ppmImage, error) {
	pos := 0
	next := func() (string, error) {
		for pos < len(b) {
			if b[pos] == '#' {
				for pos < len(b) && b[pos] != '\n' {
					pos++
				}
				continue
			}
			if b[pos] == ' ' || b[pos] == '\t' || b[pos] == '\n' || b[pos] == '\r' {
				pos++
				continue
			}
			break
		}
		start := pos
		for pos < len(b) && b[pos] != ' ' && b[pos] != '\t' && b[pos] != '\n' && b[pos] != '\r' {
			pos++
		}
		return string(b[start:pos]), nil
	}
	magic, _ := next()
	if magic != "P6" {
		return nil, errors.New("нужен PPM P6")
	}
	ws, _ := next()
	hs, _ := next()
	next() // max (обычно 255)
	w, _ := strconv.Atoi(ws)
	h, _ := strconv.Atoi(hs)
	if w <= 0 || h <= 0 {
		return nil, errors.New("PPM: плохой размер")
	}
	if pos < len(b) && (b[pos] == ' ' || b[pos] == '\n' || b[pos] == '\t' || b[pos] == '\r') {
		pos++
	}
	need := w * h * 3
	if pos+need > len(b) {
		return nil, errors.New("PPM: не хватает данных")
	}
	return &ppmImage{w: w, h: h, data: b[pos : pos+need]}, nil
}

// colorLum — яркость цвета (0..255).
func colorLum(c tcell.Color) int {
	r, g, b := c.RGB()
	return (int(r) + int(g) + int(b)) / 3
}

// midColor — средний цвет двух.
func midColor(a, b tcell.Color) tcell.Color {
	r1, g1, b1 := a.RGB()
	r2, g2, b2 := b.RGB()
	return tcell.NewRGBColor((r1+r2)/2, (g1+g2)/2, (b1+b2)/2)
}
