package engine

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// Состояние окна подтверждения (шаг "| confirm" в action).
// Это ВИДЖЕТ интерфейса (инпут внутри интерфейса = фронт).
var (
	confirmMode   bool       // открыто ли окно подтверждения
	confirmMsg    string     // текст вопроса
	pendingPipes  [][]string // пайпы, которые выполним после подтверждения ("a && b")
	pendingOutput string     // куда направить вывод после подтверждения
	pendingIn     []string   // вход пайпа (ввод снаружи), если был
)

// confirmYes — подтверждение (Enter/y или клик по «Да»): выполняет отложенные пайпы.
func confirmYes() {
	confirmMode = false
	debugLines = nil
	statusShownAt = time.Now()
	// Кнопка может нести несколько пайпов ("a && b") — склеиваем вывод.
	runAllPipes(pendingPipes, pendingOutput, pendingIn)
}

// confirmNo — отмена (Esc/n или клик по «Нет»).
func confirmNo() {
	confirmMode = false
	statusMsg = "cancelled"
}

// widgetConfirmKey обрабатывает клавиши в окне подтверждения (инпут внутри
// интерфейса): Enter/y — да, Esc/n — нет.
func widgetConfirmKey(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyEnter:
		confirmYes()
	case tcell.KeyEscape:
		confirmNo()
	default:
		switch e.Rune() {
		case 'y', 'Y':
			confirmYes()
		case 'n', 'N':
			confirmNo()
		}
	}
}

// drawConfirmModal рисует модальное окно подтверждения по центру экрана.
// У окна есть кнопки «Да/Нет» — это хотзоны: подтвердить/отменить можно и
// мышью, и клавишами (Enter — да, Esc — нет).
func drawConfirmModal(b *Buffer, w, h int, out *[]Hotzone) {
	title := "Confirm"
	line := confirmMsg
	bl := curTheme.Sym("button_l", "⟨")
	br := curTheme.Sym("button_r", "⟩")
	yes := " " + string(bl) + " Yes " + string(br) + " "
	no := " " + string(bl) + " No " + string(br) + " "
	pair := yes + " " + no
	width := uniseg.StringWidth(line) + 4
	if tw := uniseg.StringWidth(title) + 4; tw > width {
		width = tw
	}
	if pw := uniseg.StringWidth(pair) + 4; pw > width {
		width = pw
	}
	if width > w-4 {
		width = w - 4
	}
	x0 := (w - width) / 2
	y0 := h/2 - 2
	if y0 < 1 {
		y0 = 1
	}
	titleBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	titleFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	inFg := curTheme.ResolveColor(themeColor("input_fg"), tcell.ColorGreen)

	// Рамка: заголовок на верхней линии, вопрос, кнопки, подсказка на нижней.
	drawFrame(b, x0-1, y0-1, width+2, 4)
	b.SetString(x0, y0-1, " "+title+" ", Style{Bg: titleBg, Fg: titleFg})
	b.SetString(x0, y0, line, Style{Fg: inFg})

	// Кнопки «Да / Нет» — хотзоны: клик мыши тоже подтверждает/отменяет.
	py := y0 + 1
	px := x0 + (width-uniseg.StringWidth(pair))/2
	b.SetString(px, py, yes, Style{Bg: titleBg, Fg: titleFg, Bold: true})
	*out = append(*out, Hotzone{X: px, Y: py, W: uniseg.StringWidth(yes), H: 1, Kind: "confirm_yes"})
	px += uniseg.StringWidth(yes) + 1
	b.SetString(px, py, no, Style{Bg: titleBg, Fg: titleFg, Bold: true})
	*out = append(*out, Hotzone{X: px, Y: py, W: uniseg.StringWidth(no), H: 1, Kind: "confirm_no"})
	b.SetString(x0, y0+2, " Enter — yes, Esc — no", Style{Bg: titleBg, Fg: titleFg})
}
