package engine

import (
	"github.com/gdamore/tcell/v2"
)

// handleMouse обрабатывает мышь (источник — терминал/tcell, десктоп):
// колесо над меню/статусом/тайлом, клик по хит-тесту, перетаскивание
// (drag-скролл уровня меню). Вызывается из главного цикла.
func handleMouse(e *tcell.EventMouse, pages Pages, route *string, w, h int) {
	// Позиция мыши — для подсветки квадратика под курсором.
	mouseX, mouseY = e.Position()
	// Колесо: над меню select — прокрутка его уровня; над статус-блоком —
	// прокрутка статуса; иначе — скролл тайла.
	if e.Buttons()&(tcell.WheelUp|tcell.WheelDown) != 0 {
		if selectMode {
			if li := menuLevelAt(mouseX, mouseY); li >= 0 {
				lv := &selectStack[li]
				if e.Buttons()&tcell.WheelUp != 0 {
					lv.scroll--
				} else {
					lv.scroll++
				}
				clampScroll(lv)
				return
			}
		}
		if statusRectH > 0 && mouseX >= statusRectX && mouseX < statusRectX+statusRectW &&
			mouseY >= statusRectY && mouseY < statusRectY+statusRectH {
			if e.Buttons()&tcell.WheelUp != 0 {
				statusScroll--
			}
			if e.Buttons()&tcell.WheelDown != 0 {
				statusScroll++
			}
			return
		}
		if e.Buttons()&tcell.WheelUp != 0 {
			scrollTile(pages, *route, mouseX, mouseY, w, h, -1)
			return
		}
		if e.Buttons()&tcell.WheelDown != 0 {
			scrollTile(pages, *route, mouseX, mouseY, w, h, 1)
			return
		}
	}
	// Левая кнопка: нажатие — клик; удержание с движением — drag-скролл
	// уровня меню под курсором (прокрутка «мышкой», как просили).
	if e.Buttons()&tcell.Button1 != 0 {
		if mouseBtn1 {
			// Кнопка уже была нажата — это перетаскивание.
			if selectMode && (mouseX != mouseLastX || mouseY != mouseLastY) {
				if li := menuLevelAt(mouseX, mouseY); li >= 0 {
					lv := &selectStack[li]
					lv.scroll += mouseLastY - mouseY
					clampScroll(lv)
				}
			}
			mouseLastX, mouseLastY = mouseX, mouseY
			return
		}
		// Нажатие (кнопка только что нажата) — обычный клик.
		mouseBtn1 = true
		mouseLastX, mouseLastY = mouseX, mouseY
		x, y := e.Position()
		// Меню select открыто: карта экрана ПОЛНОСТЬЮ отключена,
		// работает ТОЛЬКО список. Клик по варианту — выбор/подменю,
		// клик в любом другом месте — закрыть всё меню.
		if selectMode {
			level, idx, ok := HitSelect(x, y)
			if ok {
				selectOption(level, idx)
			} else {
				selectMode = false
				selectStack = nil
			}
			return
		}
		kind, act, href, label, output, options := HitTest(x, y)
		// Кнопка «Закрыть» — закрыть строку вывода (не выход из приложения).
		if kind == "quit" {
			statusMsg = ""
			debugLines = nil
			return
		}
		// Клик не по активному полю — завершаем ввод.
		if inputMode && !(kind == "input" && act == inputAction && label == inputLabel) {
			inputMode = false
		}
		if href != "" {
			if _, ok := pages[href]; ok {
				*route = href
			}
		}
		if act == "" {
			return
		}
		// Поле ввода: активируем, ввод идёт прямо в поле.
		if kind == "input" {
			inputMode = true
			inputAction = act
			inputLabel = label
			inputOutput = output
			inputBuf = ""
			statusMsg = ""
			debugLines = nil
			return
		}
		// Селект: открываем выпадающее меню СПРАВА от элемента.
		// Якорь — кнопка select (её позиция и ширина), а не точка клика.
		if kind == "select" {
			var hz *Hotzone
			for i := range hotzones {
				if hotzones[i].Kind == "select" && x >= hotzones[i].X && x < hotzones[i].X+hotzones[i].W && y >= hotzones[i].Y && y < hotzones[i].Y+hotzones[i].H {
					hz = &hotzones[i]
					break
				}
			}
			if hz == nil {
				hz = &Hotzone{X: x, Y: y, W: 1}
			}
			openSelect(act, label, output, options, hz.X, hz.Y, hz.W)
			return
		}
		execAction(act, output)
		return
	}
	mouseBtn1 = false
	mouseLastX, mouseLastY = mouseX, mouseY
}
