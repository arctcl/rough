package engine

import (
	"io"
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
	Kind       string // вид зоны: "" (действие), "nav" (ссылка), "input" (поле ввода)
	Label      string // подпись для окна ввода
	Output     string // id блока, куда направить вывод (пусто = статус-строка)
}

// hotzones — хотзоны последнего кадра, по ним делаем hit-test мыши.
var hotzones []Hotzone

// outputCache — результаты команд, направленных в блоки с id (id блока → строки).
// Блок <div id="..."> или <col id="..."> при рендере рисует эти строки.
var outputCache = map[string][]string{}

// HitTest ищет хотзону по координатам клика.
func HitTest(x, y int) (kind, action, href, label, output string) {
	for _, hz := range hotzones {
		if x >= hz.X && x < hz.X+hz.W && y >= hz.Y && y < hz.Y+hz.H {
			return hz.Kind, hz.Action, hz.Href, hz.Label, hz.Output
		}
	}
	return "", "", "", "", ""
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
		x0, y0 := f.x, f.y
		f.put(b, string(bl)+" "+label+" "+string(br))
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(label) + 4, H: 1, Action: act, Output: n.Attrs["output"]})
		f.nl(b)
	case "a":
		f.nl(b)
		label := strings.TrimSpace(textContent(n))
		href := n.Attrs["href"]
		x0, y0 := f.x, f.y
		f.put(b, "→ "+label)
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(label) + 2, H: 1, Href: href, Kind: "nav"})
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
		x0, y0 := f.x, f.y
		s := string(il) + " " + string(ic) + " "
		if inputMode && inputAction == act && inputLabel == label {
			s += inputBuf + string(curTheme.Sym("cursor", "█"))
		} else {
			s += label
		}
		s += " " + string(ir)
		f.put(b, s)
		*out = append(*out, Hotzone{X: ox + x0, Y: oy + y0, W: uniseg.StringWidth(s), H: 1, Action: act, Kind: "input", Label: label, Output: n.Attrs["output"]})
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
func renderPlugin(n *Node, b *Buffer, f *flowState) {
	// Размер окна тайла — чтобы рисовалки (bars и т.п.) адаптировались.
	curW, curH = b.W, b.H

	steps := pluginSteps(n)
	if len(steps) == 0 {
		f.put(b, "ошибка: пустой плагин")
		return
	}
	out, err := RunSteps(steps, nil)
	if err != nil {
		f.put(b, "ошибка: "+err.Error())
		return
	}
	for _, ln := range out {
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
	yOff := (b.H - h) / 2
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
