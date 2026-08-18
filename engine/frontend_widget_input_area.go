package engine

import (
	"strings"

	"github.com/gdamore/tcell/v2"
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
		// Ввод из поля — ЛИТЕРАЛ: оборачиваем в кавычки, чтобы спецсимволы
		// (пайп |, двоеточие :, подстановка $) не трактовались как синтаксис
		// action. Так через поле ввода нельзя протащить произвольную команду
		// (инъекция: "cat | ssh:..." останется ОДНИМ значением, а не пайпом).
		if strings.Contains(inputBuf, "'") {
			// Кавычка в значении конфликтует с обёрткой — безопасно отклоняем.
			inputMode = false
			statusMsg = "ввод содержит кавычку ' — недопустимо"
			return
		}
		act += "'" + inputBuf + "'"
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
