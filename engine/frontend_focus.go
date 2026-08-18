package engine

// focusIdx — индекс сфокусированной хотзоны (навигация стрелками, -1 — нет фокуса).
var focusIdx = -1

// moveFocus двигает фокус по хотзонам (в порядке рендера: сверху вниз, слева направо).
func moveFocus(dir int) {
	if len(hotzones) == 0 {
		focusIdx = -1
		return
	}
	if focusIdx < 0 {
		if dir > 0 {
			focusIdx = 0
		} else {
			focusIdx = len(hotzones) - 1
		}
		return
	}
	focusIdx = (focusIdx + dir + len(hotzones)) % len(hotzones)
}

// activateFocus активирует сфокусированную хотзону (как клик): переход по ссылке,
// активация поля ввода или выполнение действия. Возвращает true, если нужно
// выйти из приложения (кнопка «Закрыть»).
func activateFocus(pages *Pages, route *string) bool {
	hz := hotzones[focusIdx]
	// Кнопка «Закрыть» — закрыть строку вывода (не выход из приложения).
	if hz.Kind == "quit" {
		statusMsg = ""
		debugLines = nil
		return false
	}
	if hz.Href != "" {
		if _, ok := (*pages)[hz.Href]; ok {
			*route = hz.Href
		}
		return false
	}
	if hz.Kind == "input" {
		inputMode = true
		inputAction = hz.Action
		inputLabel = hz.Label
		inputOutput = hz.Output
		inputBuf = ""
		statusMsg = ""
		debugLines = nil
		return false
	}
	// Селект: Enter открывает выпадающее меню (якорь — сам элемент select).
	if hz.Kind == "select" {
		openSelect(hz.Action, hz.Label, hz.Output, hz.Options, hz.X, hz.Y, hz.W)
		return false
	}
	if hz.Action != "" || len(hz.Actions) > 0 {
		runHotzone(&hz)
	}
	return false
}
