package engine

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// selNode — вариант меню выбора; children — вложенное подменю (если есть).
type selNode struct {
	label    string     // отображаемый текст варианта
	children []*selNode // вложенное подменю (nil — лист без подменю)
}

// selLevel — один уровень меню выбора (список вариантов) в стеке selectStack.
// x,y,w,h — рамка уровня на экране (пересчитывается при каждой отрисовке).
type selLevel struct {
	nodes  []*selNode // варианты уровня
	idx    int        // курсор (активный вариант)
	scroll int        // смещение прокрутки (если варианты не влезают)
	x, y   int        // верхний левый угол рамки уровня
	w, h   int        // размеры рамки уровня
}

// Состояние меню выбора (select): пока selectMode включён, работают только
// варианты меню — карта экрана отключена. selectStack — стек уровней:
// [0] — корневой список, дальше — вложенные подменю (каждое справа от родителя).
// Это ВИДЖЕТ интерфейса (инпут внутри интерфейса = фронт).
var (
	selectMode   bool
	selectAction string
	selectOutput string
	// Позиция и ширина кнопки select — якорь для корневого списка:
	// левый верхний угол меню ставится строго наискосок от кнопки,
	// вне зависимости от того, как вызвали меню (клик или Enter).
	selectX, selectY, selectW int
	selectStack               []selLevel
)

// widgetSelectKey обрабатывает клавиши в выпадающем меню (инпут внутри
// интерфейса): стрелки — вариант, → — подменю, ← — назад, Enter — применить,
// Esc — отмена. Прокрутка стрелками подтягивает список за курсором.
func widgetSelectKey(e *tcell.EventKey) {
	if len(selectStack) == 0 {
		selectMode = false
		return
	}
	lv := &selectStack[len(selectStack)-1]
	switch e.Key() {
	case tcell.KeyUp:
		if lv.idx > 0 {
			lv.idx--
			keepVisible(lv)
		}
	case tcell.KeyDown:
		if lv.idx < len(lv.nodes)-1 {
			lv.idx++
			keepVisible(lv)
		}
	case tcell.KeyRight:
		enterSubmenu()
	case tcell.KeyLeft:
		if len(selectStack) > 1 {
			selectStack = selectStack[:len(selectStack)-1]
		}
	case tcell.KeyEnter:
		selectOption(len(selectStack)-1, lv.idx)
	case tcell.KeyEscape:
		selectMode = false
		selectStack = nil
		statusMsg = "выбор отменён"
	}
}

// openSelect открывает выпадающее меню select: корневой список якорится к кнопке
// (строго справа от неё), подменю раскрываются дальше вправо. x, y, w — позиция
// и ширина кнопки select (якорь меню, а не точка клика).
func openSelect(act, label, output, options string, x, y, w int) {
	selectMode = true
	selectAction = act
	selectOutput = output
	selectX, selectY, selectW = x, y, w
	selectStack = nil
	selectStack = append(selectStack, selLevel{nodes: parseSelTree(options)})
	statusMsg = ""
	debugLines = nil
}

// drawSelectMenu рисует меню выбора. Корневой список ставится СТРОГО СПРАВА
// от кнопки select (левый верхний угол — наискосок от кнопки, по y — сразу
// под ней), вложенные подменю — справа от родительского меню, верхняя граница
// вровень с родителем. Высота уровня ограничена 50% высоты экрана; если
// варианты не влезают — прокрутка (колесо, мышью-перетаскиванием, стрелками).
// Каждый уровень непрозрачен и изолирован: под ним ничего не видно/не кликается.
// Каждый вариант — хотзона "selopt" на всю ширину строки.
func drawSelectMenu(b *Buffer, out *[]Hotzone, w, h int) {
	if len(selectStack) == 0 {
		return
	}
	// 1) Геометрия уровней: от корня к листьям (дочернему нужны размеры родителя).
	for li := range selectStack {
		lv := &selectStack[li]
		selLevelSize(lv, w, h)
		if li == 0 {
			// Корень: строго справа и ниже кнопки select (наискосок от неё),
			// вне зависимости от того, как вызвали меню (клик или Enter).
			lv.x = selectX + selectW + 1
			lv.y = selectY + 1
		} else {
			// Подменю: справа от родительского меню, верх вровень с ним.
			p := &selectStack[li-1]
			lv.x = p.x + p.w + 1
			lv.y = p.y
		}
		// Не влезает вправо — разворачиваем влево от родителя/кнопки.
		if lv.x+lv.w > w {
			if li == 0 {
				lv.x = selectX - lv.w - 1
			} else {
				lv.x = selectStack[li-1].x - lv.w - 1
			}
		}
		if lv.x < 0 {
			lv.x = 0
		}
		if lv.y+lv.h > h {
			lv.y = h - lv.h
		}
		if lv.y < 0 {
			lv.y = 0
		}
		clampScroll(lv)
	}
	// 2) Рисуем все уровни.
	for li := range selectStack {
		drawSelLevel(b, out, &selectStack[li], li)
	}
}

// selLevelSize считает размер рамки уровня меню по контенту.
// Высота ограничена 50% высоты экрана — дальше только прокрутка.
func selLevelSize(lv *selLevel, w, h int) {
	maxOpt := 0
	for _, n := range lv.nodes {
		lw := uniseg.StringWidth(n.label) + 2 // " label "
		if len(n.children) > 0 {
			lw += 2 // стрелка " ▶"
		}
		if lw > maxOpt {
			maxOpt = lw
		}
	}
	textW := maxOpt
	if textW < 1 {
		textW = 1
	}
	lv.w = textW + 2 // рамка
	if lv.w > w {
		lv.w = w
	}
	rows := len(lv.nodes)
	if maxRows := (h * 50) / 100; rows > maxRows {
		rows = maxRows
	}
	if rows < 1 {
		rows = 1
	}
	lv.h = rows + 2 // рамка
	if lv.h > h {
		lv.h = h
	}
}

// drawSelLevel рисует один уровень меню: непрозрачная заливка, рамка, варианты
// с учётом прокрутки, полоса прокрутки и хотзоны вариантов.
func drawSelLevel(b *Buffer, out *[]Hotzone, lv *selLevel, li int) {
	titleBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	inFg := curTheme.ResolveColor(themeColor("input_fg"), tcell.ColorGreen)
	frameFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)

	// Изоляция: непрозрачный фон всей области уровня.
	fill := Style{Bg: titleBg}
	for yy := lv.y; yy < lv.y+lv.h && yy < b.H; yy++ {
		for xx := lv.x; xx < lv.x+lv.w && xx < b.W; xx++ {
			b.Set(xx, yy, ' ', fill)
		}
	}
	// Рамка с фоном (поверх заливки).
	drawFrameStyled(b, lv.x, lv.y, lv.w, lv.h, Style{Fg: frameFg, Bg: titleBg})

	// Видимые варианты (с учётом прокрутки уровня).
	rows := lv.h - 2
	if rows > len(lv.nodes) {
		rows = len(lv.nodes)
	}
	for i := 0; i < rows; i++ {
		oi := lv.scroll + i
		if oi < 0 || oi >= len(lv.nodes) {
			break
		}
		node := lv.nodes[oi]
		oy := lv.y + 1 + i
		text := " " + node.label
		if len(node.children) > 0 {
			text += " ▶"
		}
		text += " "
		if oi == lv.idx {
			// Активный вариант — закрашиваем всю строку, текст чёрным.
			for xx := 0; xx < lv.w-2; xx++ {
				b.Set(lv.x+1+xx, oy, ' ', Style{Bg: inFg})
			}
			b.SetString(lv.x+1, oy, text, Style{Fg: tcell.ColorBlack, Bg: inFg, Bold: true})
		} else {
			b.SetString(lv.x+1, oy, text, Style{Fg: inFg, Bg: titleBg})
		}
		// Хотзона на всю строку — клик в любом месте строки.
		*out = append(*out, Hotzone{X: lv.x + 1, Y: oy, W: lv.w - 2, H: 1, Kind: "selopt", SelLevel: li, SelIdx: oi, Action: selectAction, Output: selectOutput})
	}

	// Полоса прокрутки уровня (если вариантов больше, чем влезает).
	maxOff := len(lv.nodes) - (lv.h - 2)
	if maxOff > 0 {
		scFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
		sc := Style{Fg: scFg}
		sx := lv.x + lv.w - 2 // перед правой рамкой
		thumbH := rows * rows / len(lv.nodes)
		if thumbH < 1 {
			thumbH = 1
		}
		thumbY := lv.scroll * (rows - thumbH) / maxOff
		for i := 0; i < rows; i++ {
			if i >= thumbY && i < thumbY+thumbH {
				b.Set(sx, lv.y+1+i, '█', sc)
			} else {
				b.Set(sx, lv.y+1+i, '│', sc)
			}
		}
	}
}

// clampScroll ограничивает прокрутку уровня меню видимым диапазоном.
func clampScroll(lv *selLevel) {
	rows := lv.h - 2
	if rows < 1 {
		rows = 1
	}
	maxOff := len(lv.nodes) - rows
	if maxOff < 0 {
		maxOff = 0
	}
	if lv.scroll > maxOff {
		lv.scroll = maxOff
	}
	if lv.scroll < 0 {
		lv.scroll = 0
	}
}

// keepVisible подтягивает прокрутку уровня меню, чтобы курсор был виден
// (нужно для прокрутки стрелками: вверх/вниз за краем списка).
func keepVisible(lv *selLevel) {
	rows := lv.h - 2
	if rows < 1 {
		rows = 1
	}
	if lv.idx < lv.scroll {
		lv.scroll = lv.idx
	}
	if lv.idx >= lv.scroll+rows {
		lv.scroll = lv.idx - rows + 1
	}
	clampScroll(lv)
}

// menuLevelAt возвращает индекс уровня меню (от глубочайшего к корню),
// в прямоугольник которого попадает точка, или -1. Нужен для прокрутки
// колесом и перетаскиванием: крутим именно уровень под курсором.
func menuLevelAt(x, y int) int {
	for i := len(selectStack) - 1; i >= 0; i-- {
		lv := &selectStack[i]
		if x >= lv.x && x < lv.x+lv.w && y >= lv.y && y < lv.y+lv.h {
			return i
		}
	}
	return -1
}

// selectOption обрабатывает выбор варианта (level, idx): у родителя с подменю —
// открывает подменю справа, у листа — выбирает значение и выполняет действие.
func selectOption(level, idx int) {
	if level < 0 || level >= len(selectStack) {
		return
	}
	lv := &selectStack[level]
	if idx < 0 || idx >= len(lv.nodes) {
		return
	}
	node := lv.nodes[idx]
	if len(node.children) > 0 {
		// Родитель с подменю — открываем его.
		lv.idx = idx
		pushLevel(level, idx)
		return
	}
	// Лист — выбор значения и выполнение действия.
	selectMode = false
	selectStack = nil
	selectValue[selectAction] = node.label
	execAction(selectAction+":"+node.label, selectOutput)
}

// pushLevel открывает подменю родителя (level, idx). Уровни глубже родителя
// отсекаются; позицию подменю посчитает drawSelectMenu (справа от родителя).
func pushLevel(level, idx int) {
	lv := &selectStack[level]
	node := lv.nodes[idx]
	if len(node.children) == 0 {
		return
	}
	selectStack = selectStack[:level+1]
	selectStack = append(selectStack, selLevel{nodes: node.children})
}

// enterSubmenu открывает подменю активного варианта текущего уровня (клавиша →).
func enterSubmenu() {
	if len(selectStack) == 0 {
		return
	}
	lv := &selectStack[len(selectStack)-1]
	if lv.idx >= 0 && lv.idx < len(lv.nodes) && len(lv.nodes[lv.idx].children) > 0 {
		pushLevel(len(selectStack)-1, lv.idx)
	}
}

// parseSelTree разбирает options-строку в дерево вариантов.
// Синтаксис: "a:b:c" — плоский список; "Родитель [д1:д2]:Просто" — вариант
// с подменю (дети в квадратных скобках, разделитель ":"), вложенность любая.
func parseSelTree(s string) []*selNode {
	return parseSelLevel(s, 0, len(s))
}

// parseSelLevel разбирает один уровень дерева вариантов в диапазоне [start,end).
func parseSelLevel(s string, start, end int) []*selNode {
	var nodes []*selNode
	i := start
	for i < end {
		// Пропускаем пробелы и разделители уровней.
		for i < end && (s[i] == ' ' || s[i] == '\t' || s[i] == ':') {
			i++
		}
		if i >= end {
			break
		}
		// Метка до '[', ':' или ']'.
		labelStart := i
		for i < end && s[i] != '[' && s[i] != ':' && s[i] != ']' {
			i++
		}
		label := strings.TrimSpace(s[labelStart:i])
		node := &selNode{label: label}
		if i < end && s[i] == '[' {
			// Подменю: ищем парную закрывающую скобку.
			depth := 0
			j := i
			for j < end {
				if s[j] == '[' {
					depth++
				}
				if s[j] == ']' {
					depth--
					if depth == 0 {
						break
					}
				}
				j++
			}
			node.children = parseSelLevel(s, i+1, j)
			i = j + 1 // после ']'
		}
		if node.label != "" || len(node.children) > 0 {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
