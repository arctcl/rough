package engine

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
	eng.mouseX, eng.mouseY = me.X, me.Y
	// Колесо: над меню select — его прокрутка; над статус-блоком — статус; иначе — тайл.
	if me.Wheel != 0 {
		if selectMode {
			if li := menuLevelAt(eng.mouseX, eng.mouseY); li >= 0 {
				lv := &selectStack[li]
				lv.scroll += me.Wheel
				clampScroll(lv)
				return
			}
		}
		if statusRectH > 0 && eng.mouseX >= statusRectX && eng.mouseX < statusRectX+statusRectW &&
			eng.mouseY >= statusRectY && eng.mouseY < statusRectY+statusRectH {
			statusScroll += me.Wheel
			return
		}
		scrollTile(pages, *route, eng.mouseX, eng.mouseY, w, h, me.Wheel)
		return
	}
	// Левая кнопка: нажатие — клик; удержание с движением — drag-скролл меню.
	if me.Left {
		if eng.mouseBtn1 {
			// Кнопка уже была нажата — это перетаскивание.
			if selectMode && (eng.mouseX != eng.mouseLastX || eng.mouseY != eng.mouseLastY) {
				if li := menuLevelAt(eng.mouseX, eng.mouseY); li >= 0 {
					lv := &selectStack[li]
					lv.scroll += eng.mouseLastY - eng.mouseY
					clampScroll(lv)
				}
			}
			eng.mouseLastX, eng.mouseLastY = eng.mouseX, eng.mouseY
			return
		}
		// Нажатие (кнопка только что нажата) — обычный клик.
		eng.mouseBtn1 = true
		eng.mouseLastX, eng.mouseLastY = eng.mouseX, eng.mouseY
		x, y := me.X, me.Y
		// Модалка подтверждения открыта: клики работают ТОЛЬКО по её кнопкам
		// «Да/Нет», всё остальное игнорируем — нельзя выполнить действие без
		// подтверждения.
		if confirmMode {
			for i := range hotzones {
				hz := &hotzones[i]
				if (hz.Kind != "confirm_yes" && hz.Kind != "confirm_no") ||
					x < hz.X || x >= hz.X+hz.W || y < hz.Y || y >= hz.Y+hz.H {
					continue
				}
				if hz.Kind == "confirm_yes" {
					confirmYes()
				} else {
					confirmNo()
				}
				return
			}
			return // клик мимо кнопок — игнорируем
		}
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
		hz := HitTest(x, y)
		kind, act, href, label, output, options := "", "", "", "", "", ""
		if hz != nil {
			kind, act, href, label, output, options = hz.Kind, hz.Action, hz.Href, hz.Label, hz.Output, hz.Options
		}
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
				focusIdx = -1 // новая страница — фокус сбрасывается
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
		// Одна кнопка может нести несколько action — выполняем их все
		// последовательно (runHotzone); иначе — одиночное действие как раньше.
		if hz != nil && len(hz.Actions) > 0 {
			runHotzone(hz)
			return
		}
		execAction(act, output)
		return
	}
	eng.mouseBtn1 = false
	eng.mouseLastX, eng.mouseLastY = eng.mouseX, eng.mouseY
}
