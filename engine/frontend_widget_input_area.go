package engine

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// Состояние поля ввода: пока inputMode включён, клавиши идут в буфер,
// а Enter дописывает значение к действию и выполняет его.
// Это ВИДЖЕТ интерфейса (инпут внутри интерфейса = фронт).
var (
	inputMode   bool   // открыто ли окно ввода
	inputAction string // действие, которое выполним (без значения)
	inputLabel  string // подпись (что редактируем, например MAX_USERS)
	inputBuf    string // набранное значение
	inputOutput string // id блока, куда направить результат (output="...")
)

// widgetInputKey обрабатывает клавиши в поле ввода (инпут внутри интерфейса).
func widgetInputKey(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyEnter:
		// Дописываем значение как последний аргумент действия.
		act := inputAction
		if !strings.HasSuffix(act, ":") {
			act += ":"
		}
		act += inputBuf
		inputMode = false
		execAction(act, inputOutput)
	case tcell.KeyEscape:
		inputMode = false
		statusMsg = "ввод отменён"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(inputBuf) > 0 {
			inputBuf = inputBuf[:len(inputBuf)-1]
		}
	default:
		if e.Rune() != 0 {
			inputBuf += string(e.Rune())
		}
	}
}

// drawInputModal рисует модальное окно ввода по центру экрана.
func drawInputModal(b *Buffer, w, h int) {
	title := "✎ " + inputLabel
	line := inputLabel + " = " + inputBuf
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

	frameFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorYellow)
	titleBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	titleFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	inFg := curTheme.ResolveColor(themeColor("input_fg"), tcell.ColorGreen)

	drawFrame(b, x0-1, y0-1, width+2, 4)
	b.SetString(x0, y0-1, " "+title+" ", Style{Bg: titleBg, Fg: titleFg})
	b.SetString(x0, y0, line, Style{Fg: inFg})
	b.Set(x0+uniseg.StringWidth(line), y0, curTheme.Sym("cursor", "█"), Style{Fg: inFg})
	b.SetString(x0, y0+1, " Enter — применить, Esc — отмена", Style{Fg: frameFg})
}
