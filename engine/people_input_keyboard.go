package engine

import (
	"github.com/gdamore/tcell/v2"
)

// handleKey обрабатывает клавиатуру (сырой ввод от человека): раздаёт событие
// активному виджету интерфейса (поле/select/confirm) или обрабатывает
// глобальные клавиши (Esc/q, стрелки-фокус, Tab-вкладки).
// Возвращает true, если приложение должно выйти (Ctrl+C / Esc без статуса).
func handleKey(e *tcell.EventKey, pages Pages, menu [][]string, route *string) bool {
	// Ctrl+C — всегда выход.
	if e.Key() == tcell.KeyCtrlC {
		return true
	}
	// Сырой ввод отдаём активному виджету (инпут внутри интерфейса = фронт).
	if inputMode {
		widgetInputKey(e)
		return false
	}
	if selectMode {
		widgetSelectKey(e)
		return false
	}
	if confirmMode {
		widgetConfirmKey(e)
		return false
	}
	// Секретные последовательности клавиш (конами-код и т.п.) — см. cheat.go.
	checkCheat(e)
	// Глобальные клавиши вне виджетов.
	if e.Key() == tcell.KeyEscape {
		if statusMsg != "" || len(debugLines) > 0 {
			statusMsg = ""
			debugLines = nil
		} else {
			return true
		}
	}
	// q — закрыть строку вывода (всплывающее окошко), НЕ выход из приложения.
	if e.Rune() == 'q' || e.Rune() == 'Q' {
		statusMsg = ""
		debugLines = nil
	}
	// Стрелки — фокус по элементам, Enter — активировать (как клик).
	ctrl := e.Modifiers()&tcell.ModCtrl != 0
	switch e.Key() {
	case tcell.KeyUp, tcell.KeyLeft:
		moveFocus(-1)
	case tcell.KeyDown, tcell.KeyRight:
		moveFocus(1)
	case tcell.KeyEnter:
		if focusIdx >= 0 && focusIdx < len(hotzones) {
			if activateFocus(&pages, route) {
				return true
			}
		}
	}
	// Вкладки: как в хроме — Ctrl+Tab, Ctrl+Shift+Tab, Ctrl+цифры; плюс Tab/Shift+Tab.
	// Смена вкладки сбрасывает фокус: хотзоны другие, старый индекс устарел.
	switch {
	case e.Key() == tcell.KeyTab && !ctrl:
		*route = nextRoute(menu, *route, 1)
		focusIdx = -1
	case e.Key() == tcell.KeyBacktab && !ctrl:
		*route = nextRoute(menu, *route, -1)
		focusIdx = -1
	case e.Key() == tcell.KeyTab && ctrl:
		*route = nextRoute(menu, *route, 1)
		focusIdx = -1
	case e.Key() == tcell.KeyBacktab && ctrl:
		*route = nextRoute(menu, *route, -1)
		focusIdx = -1
	case ctrl && e.Rune() >= '1' && e.Rune() <= '9':
		if i := int(e.Rune() - '1'); i < len(menu) {
			*route = menu[i][1]
			focusIdx = -1
		}
	}
	return false
}

// nextRoute — следующая/предыдущая вкладка относительно текущего роута (шаг ±1).
func nextRoute(menu [][]string, route string, step int) string {
	if len(menu) == 0 {
		return route
	}
	idx := 0
	for i, m := range menu {
		if m[1] == route {
			idx = i
			break
		}
	}
	idx = (idx + step + len(menu)) % len(menu)
	return menu[idx][1]
}
