package engine

// mouseX, mouseY — позиция мыши (для подсветки квадратика под курсором, -1 — вне экрана).
var mouseX, mouseY = -1, -1

// mouseBtn1 — зажата ли левая кнопка; mouseLastX/Y — прошлая позиция (для drag).
var (
	mouseBtn1              bool
	mouseLastX, mouseLastY int
)

// MouseEvent — высокоуровневое событие мыши, ЕДИНОЕ для всех источников:
// десктоп (терминал/tcell), телеграфный (сырой /dev/input/mice) и любой
// будущий (нейролинк и т.п.). Каждый источник переводит свои данные в этот
// тип, а движок обрабатывает MouseEvent в одном месте (handleMouseEvent).
type MouseEvent struct {
	X, Y  int
	Left  bool
	Wheel int  // -1 вверх, +1 вниз, 0 нет
	Held  bool // кнопка уже была зажата (перетаскивание)
}

// handleMouseEvent — ЕДИНЫЙ обработчик мыши (источник ему не важен): колесо
// над меню/статусом/тайлом, клик по хит-тесту, перетаскивание (drag-скролл
// меню). Вызывается и десктопным источником (handleMouse), и телеграфным.
func handleMouseEvent(me MouseEvent, pages Pages, route *string, w, h int) {
	// Позиция — для подсветки квадратика под курсором.
	mouseX, mouseY = me.X, me.Y
	// Колесо: над меню select — его прокрутка; над статус-блоком — статус; иначе — тайл.
	if me.Wheel != 0 {
		if selectMode {
			if li := menuLevelAt(mouseX, mouseY); li >= 0 {
				lv := &selectStack[li]
				lv.scroll += me.Wheel
				clampScroll(lv)
				return
			}
		}
		if statusRectH > 0 && mouseX >= statusRectX && mouseX < statusRectX+statusRectW &&
			mouseY >= statusRectY && mouseY < statusRectY+statusRectH {
			statusScroll += me.Wheel
			return
		}
		scrollTile(pages, *route, mouseX, mouseY, w, h, me.Wheel)
		return
	}
	// Левая кнопка: нажатие — клик; удержание с движением — drag-скролл меню.
	if me.Left {
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
		x, y := me.X, me.Y
		// Меню select открыто: карта экрана ПОЛНОСТЬЮ отключена, работает
		// ТОЛЬКО список. Клик по варианту — выбор/подменю, клик мимо — закрыть.
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
		// Селект: открываем выпадающее меню СПРАВА от элемента. Якорь — кнопка
		// select (её позиция и ширина), а не точка клика.
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
