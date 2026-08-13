package engine

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// Состояние окна подтверждения (шаг "| confirm" в action).
// Это ВИДЖЕТ интерфейса (инпут внутри интерфейса = фронт).
var (
	confirmMode   bool     // открыто ли окно подтверждения
	confirmMsg    string   // текст вопроса
	pendingSteps  []string // шаги, которые выполним после подтверждения
	pendingOutput string   // куда направить вывод после подтверждения
)

// widgetConfirmKey обрабатывает клавиши в окне подтверждения (инпут внутри
// интерфейса): Enter/y — да, Esc/n — нет.
func widgetConfirmKey(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyEnter:
		confirmMode = false
		debugLines = nil
		statusShownAt = time.Now()
		runStepsAndShow(pendingSteps, pendingOutput)
	case tcell.KeyEscape:
		confirmMode = false
		statusMsg = "отменено"
	default:
		switch e.Rune() {
		case 'y', 'Y':
			confirmMode = false
			debugLines = nil
			statusShownAt = time.Now()
			runStepsAndShow(pendingSteps, pendingOutput)
		case 'n', 'N':
			confirmMode = false
			statusMsg = "отменено"
		}
	}
}

// drawConfirmModal рисует модальное окно подтверждения по центру экрана.
func drawConfirmModal(b *Buffer, w, h int) {
	title := "Подтверждение"
	line := confirmMsg + "  (Enter — да, Esc — нет)"
	width := uniseg.StringWidth(line) + 4
	if tw := uniseg.StringWidth(title) + 4; tw > width {
		width = tw
	}
	if width > w-4 {
		width = w - 4
	}
	x0 := (w - width) / 2
	y0 := h/2 - 1
	if y0 < 1 {
		y0 = 1
	}
	titleBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	titleFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	inFg := curTheme.ResolveColor(themeColor("input_fg"), tcell.ColorGreen)

	drawFrame(b, x0-1, y0-1, width+2, 3)
	b.SetString(x0, y0-1, " "+title+" ", Style{Bg: titleBg, Fg: titleFg})
	b.SetString(x0, y0, line, Style{Fg: inFg})
}
