package engine

import (
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// inVarRe — маркер $in / ${in} в action поля ввода: введённое подставляется
// аргументом именно в это место (пластичность). Если $in нет — введённое
// уходит ВХОДОМ первому плагину пайпа (linux-стиль).
var inVarRe = regexp.MustCompile(`\$\{in\}|\$in\b`)

// hasInVar — есть ли $in в action.
func hasInVar(s string) bool { return inVarRe.MatchString(s) }

// Состояние поля ввода: пока inputMode включён, клавиши идут в буфер,
// а Enter выполняет действие с введённым значением.
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
		// Кавычка в значении конфликтует с обёрткой в $in-режиме — безопасно отклоняем.
		if strings.Contains(inputBuf, "'") {
			inputMode = false
			statusMsg = "input contains quote ' — not allowed"
			return
		}
		// Если в action есть $in — подставляем введённое аргументом в это место.
		// Иначе — введённое уходит ВХОДОМ первому плагину пайпа (linux-стиль).
		if hasInVar(inputAction) {
			act := inVarRe.ReplaceAllString(inputAction, "'"+inputBuf+"'")
			inputMode = false
			execActionIn(act, inputOutput, nil)
		} else {
			var in []string
			if inputBuf != "" {
				in = []string{inputBuf}
			}
			inputMode = false
			execActionIn(inputAction, inputOutput, in)
		}
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
