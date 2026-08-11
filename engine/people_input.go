package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// statusMsg — последний результат действия/ошибка для нижней строки.
var statusMsg string

// debugLines — отладочный вывод плагина tobotom (рисуется в статус-блоке).
var debugLines []string

// SetDebug — плагин отладки (tobotom) показывает строки в статус-блоке.
func SetDebug(lines []string) { debugLines = lines }

// statusScroll — сдвиг прокрутки внутри статус-блока (если строк больше 3).
var statusScroll int

// statusShownAt — когда статус показан в последний раз (авто-скрытие через 10 с).
var statusShownAt time.Time

// statusRect — прямоугольник статус-блока на экране (для прокрутки колесом).
var statusRectX, statusRectY, statusRectW, statusRectH int

// Позиция мыши — для подсветки квадратика под курсором (-1 — мышь вне экрана).
var mouseX, mouseY = -1, -1

// focusIdx — индекс сфокусированной хотзоны (навигация стрелками, -1 — нет фокуса).
var focusIdx = -1

// scrollOff — вертикальный скролл внутри тайла (id тайла → смещение строк).
var scrollOff = map[string]int{}

// Состояние окна ввода: пока inputMode включён, клавиши идут в буфер,
// а Enter дописывает значение к действию и выполняет его.
var (
	inputMode   bool   // открыто ли окно ввода
	inputAction string // действие, которое выполним (без значения)
	inputLabel  string // подпись (что редактируем, например MAX_USERS)
	inputBuf    string // набранное значение
	inputOutput string // id блока, куда направить результат (output="...")
)

// selNode — вариант меню выбора; children — вложенное подменю (если есть).
type selNode struct {
	label    string     // отображаемый текст варианта
	children []*selNode // вложенное подменю (nil — лист без подменю)
}

// selLevel — один уровень меню выбора (список вариантов) в стеке selectStack.
// x,y,w,h — рамка уровня на экране (пересчитывается при каждой отрисовке).
type selLevel struct {
	nodes  []*selNode // варианты уровня
	idx    int        // курсор (активный вариант)
	scroll int        // смещение прокрутки (если варианты не влезают)
	x, y   int        // верхний левый угол рамки уровня
	w, h   int        // размеры рамки уровня
}

// Состояние меню выбора (select): пока selectMode включён, работают только
// варианты меню — карта экрана отключена. selectStack — стек уровней:
// [0] — корневой список, дальше — вложенные подменю (каждое справа от родителя).
var (
	selectMode   bool
	selectAction string
	selectOutput string
	// Позиция и ширина кнопки select — якорь для корневого списка:
	// левый верхний угол меню ставится строго наискосок от кнопки,
	// вне зависимости от того, где нажали (клик или Enter).
	selectX, selectY, selectW int
	selectStack               []selLevel
)

// Состояние мыши для перетаскивания (drag-скролл меню): зажата ли левая
// кнопка и прошлая позиция — по дельте движения крутим список под курсором.
var (
	mouseBtn1              bool
	mouseLastX, mouseLastY int
)

// Состояние окна подтверждения (шаг "| confirm" в action).
var (
	confirmMode   bool     // открыто ли окно подтверждения
	confirmMsg    string   // текст вопроса
	pendingSteps  []string // шаги, которые выполним после подтверждения
	pendingOutput string   // куда направить вывод после подтверждения
)

// execAction выполняет action из HTML (кнопка/поле/пайп).
// target — id блока для вывода (output="..."), пусто = статус-строка.
// Если в action есть "| confirm" — открывает окно подтверждения и ждёт.
func execAction(raw, target string) {
	// Новое действие — убираем отладочный вывод, скролл статуса и сбрасываем таймер.
	debugLines = nil
	statusScroll = 0
	statusShownAt = time.Now()
	steps, need := PrepareAction(raw)
	if need {
		confirmMode = true
		confirmMsg = "Выполнить?"
		pendingSteps = steps
		pendingOutput = target
		statusMsg = ""
		return
	}
	runStepsAndShow(steps, target)
}

// runStepsAndShow выполняет пайп и показывает результат (в тайл или статус).
// Ошибка ErrStop — пайп остановлен плагином отладки tobotom, вывод уже показан.
func runStepsAndShow(steps []string, target string) {
	out, err := RunSteps(steps, nil)
	if errors.Is(err, ErrStop) {
		return
	}
	if err != nil {
		statusMsg = "ошибка: " + err.Error()
		return
	}
	putOutput(out, target)
}

// putOutput направляет результат действия: в блок вывода (по id) или в статус-строку.
// Маркеры цветов вычищаем — тут раскраска не рисуется, остался бы мусор.
func putOutput(out []string, target string) {
	for i := range out {
		out[i] = StripMarkers(out[i])
	}
	if target != "" {
		outputCache[target] = out
		statusMsg = "выполнено → " + target
		return
	}
	statusMsg = strings.Join(out, " | ")
}

// Run запускает движок: читает всё из вшитой папки (fs.FS), рисует интерфейс,
// обрабатывает ввод. Работает, пока пользователь не нажмёт q / Esc / Ctrl+C.
func Run(fsys fs.FS) error {
	// Единый загрузчик: страницы (tiles.json) + тема — всё из папки /rough.
	ui, err := LoadUI(fsys)
	if err != nil {
		return err
	}
	pages := ui.Pages
	menu := ui.Menu

	// Проверяльщик синтаксиса: пока есть ошибки — интерфейс не стартует.
	if errs := CheckSyntax(fsys, pages); len(errs) > 0 {
		return fmt.Errorf("проверка синтаксиса:\n  %s", syntaxErrorsOneLine(errs))
	}

	curTheme = ui.Theme
	// Вшитая папка — нужна плагину theme (переключение тем на лету).
	curFS = fsys

	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}
	defer s.Fini()
	s.EnableMouse()
	s.Clear()
	s.Show()

	route := pages.FirstRoute()
	w, h := s.Size()

	// Пул событий: PollEvent блокирует, поэтому крутим его в горутине.
	evCh := make(chan tcell.Event, 16)
	go func() {
		for {
			evCh <- s.PollEvent()
		}
	}()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		renderFrame(s, pages, route, menu, w, h, fsys)
		select {
		case ev := <-evCh:
			switch e := ev.(type) {
			case *tcell.EventResize:
				w, h = e.Size()
			case *tcell.EventKey:
				// Окно ввода: клавиши идут в буфер.
				if inputMode {
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
					case tcell.KeyCtrlC:
						return nil
					default:
						if e.Rune() != 0 {
							inputBuf += string(e.Rune())
						}
					}
					break
				}
				// Меню выбора: стрелки — вариант, → — открыть подменю, ← — назад,
				// Enter — применить/открыть подменю, Esc — отмена. Прокрутка
				// стрелками работает и при переполнении: курсор подтягивает список.
				if selectMode {
					if len(selectStack) == 0 {
						selectMode = false
						break
					}
					lv := &selectStack[len(selectStack)-1]
					switch e.Key() {
					case tcell.KeyUp:
						if lv.idx > 0 {
							lv.idx--
							keepVisible(lv)
						}
					case tcell.KeyDown:
						if lv.idx < len(lv.nodes)-1 {
							lv.idx++
							keepVisible(lv)
						}
					case tcell.KeyRight:
						enterSubmenu()
					case tcell.KeyLeft:
						if len(selectStack) > 1 {
							selectStack = selectStack[:len(selectStack)-1]
						}
					case tcell.KeyEnter:
						selectOption(len(selectStack)-1, lv.idx)
					case tcell.KeyEscape:
						selectMode = false
						selectStack = nil
						statusMsg = "выбор отменён"
					case tcell.KeyCtrlC:
						return nil
					}
					break
				}
				// Окно подтверждения: Enter — да, Esc — нет.
				if confirmMode {
					switch e.Key() {
					case tcell.KeyEnter:
						confirmMode = false
						debugLines = nil
						statusShownAt = time.Now()
						runStepsAndShow(pendingSteps, pendingOutput)
					case tcell.KeyEscape:
						confirmMode = false
						statusMsg = "отменено"
					case tcell.KeyCtrlC:
						return nil
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
					break
				}
				// Поле ввода активируется только кликом или фокусом+Enter.
				// Никакого перехвата печати в неактивное поле: q закрывает статус,
				// а не пишется в инпут.
				// Ctrl+C — всегда выход. Esc — закрыть строку вывода, если она открыта,
				// иначе выход (стандартный способ закрыть терминальное приложение).
				if e.Key() == tcell.KeyCtrlC {
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					if statusMsg != "" || len(debugLines) > 0 {
						statusMsg = ""
						debugLines = nil
					} else {
						return nil
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
						if activateFocus(&pages, &route) {
							return nil
						}
					}
				}
				// Вкладки: как в хроме — Ctrl+Tab, Ctrl+Shift+Tab, Ctrl+цифры;
				// плюс Tab / Shift+Tab без Ctrl.
				switch {
				case e.Key() == tcell.KeyTab && !ctrl:
					route = nextRoute(menu, route, 1)
				case e.Key() == tcell.KeyBacktab && !ctrl:
					route = nextRoute(menu, route, -1)
				case e.Key() == tcell.KeyTab && ctrl:
					route = nextRoute(menu, route, 1)
				case e.Key() == tcell.KeyBacktab && ctrl:
					route = nextRoute(menu, route, -1)
				case ctrl && e.Rune() >= '1' && e.Rune() <= '9':
					if i := int(e.Rune() - '1'); i < len(menu) {
						route = menu[i][1]
					}
				}
			case *tcell.EventMouse:
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
							break
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
						break
					}
					if e.Buttons()&tcell.WheelUp != 0 {
						scrollTile(pages, route, mouseX, mouseY, w, h, -1)
						break
					}
					if e.Buttons()&tcell.WheelDown != 0 {
						scrollTile(pages, route, mouseX, mouseY, w, h, 1)
						break
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
						break
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
						break
					}
					kind, act, href, label, output, options := HitTest(x, y)
					// Кнопка «Закрыть» — закрыть строку вывода (не выход из приложения).
					if kind == "quit" {
						statusMsg = ""
						debugLines = nil
						break
					}
					// Клик не по активному полю — завершаем ввод.
					if inputMode && !(kind == "input" && act == inputAction && label == inputLabel) {
						inputMode = false
					}
					if href != "" {
						if _, ok := pages[href]; ok {
							route = href
						}
					}
					if act == "" {
						break
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
						break
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
						break
					}
					execAction(act, output)
					break
				}
				mouseBtn1 = false
				mouseLastX, mouseLastY = mouseX, mouseY
			}
		case <-tick.C:
			// Таймер: перерисовка (обновление плагинов с interval).
		}
	}
}

// renderFrame рисует текущий кадр: фон, заголовок, тайлы, вкладки и статус.
func renderFrame(s tcell.Screen, pages Pages, route string, menu [][]string, w, h int, fsys fs.FS) {
	hotzones = hotzones[:0]

	// Авто-скрытие статус-блока через 10 секунд после последнего действия.
	if (statusMsg != "" || len(debugLines) > 0) && time.Since(statusShownAt) > 10*time.Second {
		statusMsg = ""
		debugLines = nil
		statusScroll = 0
	}

	bg := NewBuffer(w, h)
	// Фон экрана и цвет текста по умолчанию — из темы (ключи bg/fg).
	bg.Fill(' ', Style{
		Fg: curTheme.ResolveColor(themeColor("fg"), tcell.ColorDefault),
		Bg: curTheme.ResolveColor(themeColor("bg"), tcell.ColorDefault),
	})

	// Шапка с текущим роутом и подсказкой выхода (цвета из темы).
	hdrFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	hdrBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	bg.SetString(1, 0, " rough: "+route+"    [q] выход ", Style{Bg: hdrBg, Fg: hdrFg})

	for _, t := range pages[route] {
		x, y, tw, th := t.Rect(w, h)
		if tw <= 0 || th <= 0 {
			continue
		}
		drawFrame(bg, x, y, tw, th)

		inner := NewBuffer(tw-2, th-2)

		if t.File != "" {
			if f, err := fsys.Open(t.File); err == nil {
				if root, perr := ParseHTML(f); perr == nil {
					renderTile(root, inner, t.ID, x+1, y+1, th-2, &hotzones)
				}
				f.Close()
			}
		}
		bg.Copy(inner, x+1, y+1)

		// Название тайла — НА верхней рамке (поверх линии), как в модалках.
		titleFg := curTheme.ResolveColor(themeColor("title_fg"), tcell.ColorGreen)
		hdrBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
		title := " " + t.ID + " "
		if x+1+uniseg.StringWidth(title) <= x+tw-1 {
			bg.SetString(x+1, y, title, Style{Bold: true, Fg: titleFg, Bg: hdrBg})
		}
	}

	// Статус — внизу, обведён рамкой из темы (поверх тайлов). Вкладки — под ним.
	drawStatus(bg, w, h, len(menu) > 0)
	if len(menu) > 0 {
		drawTabs(bg, menu, route, &hotzones)
	}
	// Кнопка «Закрыть» — ТОЛЬКО при открытой строке вывода/отладки, на правом краю нижней строки.
	if statusMsg != "" || len(debugLines) > 0 {
		drawQuit(bg, &hotzones)
	}

	// Окно подтверждения рисуется поверх всего (ввод идёт прямо в поле, без модалки).
	if confirmMode {
		drawConfirmModal(bg, w, h)
	}
	// Выпадающее меню select — поверх всего, под самим элементом.
	if selectMode {
		drawSelectMenu(bg, &hotzones, w, h)
	}

	// Квадратик под курсором мыши и подсветка сфокусированной хотзоны.
	if mouseX >= 0 && mouseY >= 0 && mouseX < w && mouseY < h {
		bg.Highlight(mouseX, mouseY)
	}
	if focusIdx >= 0 && focusIdx < len(hotzones) {
		hz := hotzones[focusIdx]
		for yy := hz.Y; yy < hz.Y+hz.H; yy++ {
			for xx := hz.X; xx < hz.X+hz.W; xx++ {
				bg.Highlight(xx, yy)
			}
		}
	}

	bg.Blit(s, 0, 0)
	s.Show()
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

// drawSelectMenu рисует меню выбора. Корневой список ставится СТРОГО СПРАВА
// от кнопки select (левый верхний угол — наискосок от кнопки, по y — сразу
// под ней), вложенные подменю — справа от родительского меню, верхняя граница
// вровень с родителем. Высота уровня ограничена 50% высоты экрана; если
// варианты не влезают — прокрутка (колесо, мышью-перетаскиванием, стрелками).
// Каждый уровень непрозрачен и изолирован: под ним ничего не видно/не кликается.
// Каждый вариант — хотзона "selopt" на всю ширину строки.
func drawSelectMenu(b *Buffer, out *[]Hotzone, w, h int) {
	if len(selectStack) == 0 {
		return
	}
	// 1) Геометрия уровней: от корня к листьям (дочернему нужны размеры родителя).
	for li := range selectStack {
		lv := &selectStack[li]
		selLevelSize(lv, w, h)
		if li == 0 {
			// Корень: строго справа и ниже кнопки select (наискосок от неё),
			// вне зависимости от того, как вызвали меню (клик или Enter).
			lv.x = selectX + selectW + 1
			lv.y = selectY + 1
		} else {
			// Подменю: справа от родительского меню, верх вровень с ним.
			p := &selectStack[li-1]
			lv.x = p.x + p.w + 1
			lv.y = p.y
		}
		// Не влезает вправо — разворачиваем влево от родителя/кнопки.
		if lv.x+lv.w > w {
			if li == 0 {
				lv.x = selectX - lv.w - 1
			} else {
				lv.x = selectStack[li-1].x - lv.w - 1
			}
		}
		if lv.x < 0 {
			lv.x = 0
		}
		if lv.y+lv.h > h {
			lv.y = h - lv.h
		}
		if lv.y < 0 {
			lv.y = 0
		}
		clampScroll(lv)
	}
	// 2) Рисуем все уровни.
	for li := range selectStack {
		drawSelLevel(b, out, &selectStack[li], li)
	}
}

// selLevelSize считает размер рамки уровня меню по контенту.
// Высота ограничена 50% высоты экрана — дальше только прокрутка.
func selLevelSize(lv *selLevel, w, h int) {
	maxOpt := 0
	for _, n := range lv.nodes {
		lw := uniseg.StringWidth(n.label) + 2 // " label "
		if len(n.children) > 0 {
			lw += 2 // стрелка " ▶"
		}
		if lw > maxOpt {
			maxOpt = lw
		}
	}
	textW := maxOpt
	if textW < 1 {
		textW = 1
	}
	lv.w = textW + 2 // рамка
	if lv.w > w {
		lv.w = w
	}
	rows := len(lv.nodes)
	if maxRows := (h * 50) / 100; rows > maxRows {
		rows = maxRows
	}
	if rows < 1 {
		rows = 1
	}
	lv.h = rows + 2 // рамка
	if lv.h > h {
		lv.h = h
	}
}

// drawSelLevel рисует один уровень меню: непрозрачная заливка, рамка, варианты
// с учётом прокрутки, полоса прокрутки и хотзоны вариантов.
func drawSelLevel(b *Buffer, out *[]Hotzone, lv *selLevel, li int) {
	titleBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	inFg := curTheme.ResolveColor(themeColor("input_fg"), tcell.ColorGreen)
	frameFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)

	// Изоляция: непрозрачный фон всей области уровня.
	fill := Style{Bg: titleBg}
	for yy := lv.y; yy < lv.y+lv.h && yy < b.H; yy++ {
		for xx := lv.x; xx < lv.x+lv.w && xx < b.W; xx++ {
			b.Set(xx, yy, ' ', fill)
		}
	}
	// Рамка с фоном (поверх заливки).
	drawFrameStyled(b, lv.x, lv.y, lv.w, lv.h, Style{Fg: frameFg, Bg: titleBg})

	// Видимые варианты (с учётом прокрутки уровня).
	rows := lv.h - 2
	if rows > len(lv.nodes) {
		rows = len(lv.nodes)
	}
	for i := 0; i < rows; i++ {
		oi := lv.scroll + i
		if oi < 0 || oi >= len(lv.nodes) {
			break
		}
		node := lv.nodes[oi]
		oy := lv.y + 1 + i
		text := " " + node.label
		if len(node.children) > 0 {
			text += " ▶"
		}
		text += " "
		if oi == lv.idx {
			// Активный вариант — закрашиваем всю строку, текст чёрным.
			for xx := 0; xx < lv.w-2; xx++ {
				b.Set(lv.x+1+xx, oy, ' ', Style{Bg: inFg})
			}
			b.SetString(lv.x+1, oy, text, Style{Fg: tcell.ColorBlack, Bg: inFg, Bold: true})
		} else {
			b.SetString(lv.x+1, oy, text, Style{Fg: inFg, Bg: titleBg})
		}
		// Хотзона на всю строку — клик в любом месте строки.
		*out = append(*out, Hotzone{X: lv.x + 1, Y: oy, W: lv.w - 2, H: 1, Kind: "selopt", SelLevel: li, SelIdx: oi, Action: selectAction, Output: selectOutput})
	}

	// Полоса прокрутки уровня (если вариантов больше, чем влезает).
	maxOff := len(lv.nodes) - (lv.h - 2)
	if maxOff > 0 {
		scFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
		sc := Style{Fg: scFg}
		sx := lv.x + lv.w - 2 // перед правой рамкой
		thumbH := rows * rows / len(lv.nodes)
		if thumbH < 1 {
			thumbH = 1
		}
		thumbY := lv.scroll * (rows - thumbH) / maxOff
		for i := 0; i < rows; i++ {
			if i >= thumbY && i < thumbY+thumbH {
				b.Set(sx, lv.y+1+i, '█', sc)
			} else {
				b.Set(sx, lv.y+1+i, '│', sc)
			}
		}
	}
}

// clampScroll ограничивает прокрутку уровня меню видимым диапазоном.
func clampScroll(lv *selLevel) {
	rows := lv.h - 2
	if rows < 1 {
		rows = 1
	}
	maxOff := len(lv.nodes) - rows
	if maxOff < 0 {
		maxOff = 0
	}
	if lv.scroll > maxOff {
		lv.scroll = maxOff
	}
	if lv.scroll < 0 {
		lv.scroll = 0
	}
}

// keepVisible подтягивает прокрутку уровня меню, чтобы курсор был виден
// (нужно для прокрутки стрелками: вверх/вниз за краем списка).
func keepVisible(lv *selLevel) {
	rows := lv.h - 2
	if rows < 1 {
		rows = 1
	}
	if lv.idx < lv.scroll {
		lv.scroll = lv.idx
	}
	if lv.idx >= lv.scroll+rows {
		lv.scroll = lv.idx - rows + 1
	}
	clampScroll(lv)
}

// menuLevelAt возвращает индекс уровня меню (от глубочайшего к корню),
// в прямоугольник которого попадает точка, или -1. Нужен для прокрутки
// колесом и перетаскиванием: крутим именно уровень под курсором.
func menuLevelAt(x, y int) int {
	for i := len(selectStack) - 1; i >= 0; i-- {
		lv := &selectStack[i]
		if x >= lv.x && x < lv.x+lv.w && y >= lv.y && y < lv.y+lv.h {
			return i
		}
	}
	return -1
}

// selectOption обрабатывает выбор варианта (level, idx): у родителя с подменю —
// открывает подменю справа, у листа — выбирает значение и выполняет действие.
func selectOption(level, idx int) {
	if level < 0 || level >= len(selectStack) {
		return
	}
	lv := &selectStack[level]
	if idx < 0 || idx >= len(lv.nodes) {
		return
	}
	node := lv.nodes[idx]
	if len(node.children) > 0 {
		// Родитель с подменю — открываем его.
		lv.idx = idx
		pushLevel(level, idx)
		return
	}
	// Лист — выбор значения и выполнение действия.
	selectMode = false
	selectStack = nil
	selectValue[selectAction] = node.label
	execAction(selectAction+":"+node.label, selectOutput)
}

// pushLevel открывает подменю родителя (level, idx). Уровни глубже родителя
// отсекаются; позицию подменю посчитает drawSelectMenu (справа от родителя).
func pushLevel(level, idx int) {
	lv := &selectStack[level]
	node := lv.nodes[idx]
	if len(node.children) == 0 {
		return
	}
	selectStack = selectStack[:level+1]
	selectStack = append(selectStack, selLevel{nodes: node.children})
}

// enterSubmenu открывает подменю активного варианта текущего уровня (клавиша →).
func enterSubmenu() {
	if len(selectStack) == 0 {
		return
	}
	lv := &selectStack[len(selectStack)-1]
	if lv.idx >= 0 && lv.idx < len(lv.nodes) && len(lv.nodes[lv.idx].children) > 0 {
		pushLevel(len(selectStack)-1, lv.idx)
	}
}

// parseSelTree разбирает options-строку в дерево вариантов.
// Синтаксис: "a:b:c" — плоский список; "Родитель [д1:д2]:Просто" — вариант
// с подменю (дети в квадратных скобках, разделитель ":"), вложенность любая.
func parseSelTree(s string) []*selNode {
	return parseSelLevel(s, 0, len(s))
}

// parseSelLevel разбирает один уровень дерева вариантов в диапазоне [start,end).
func parseSelLevel(s string, start, end int) []*selNode {
	var nodes []*selNode
	i := start
	for i < end {
		// Пропускаем пробелы и разделители уровней.
		for i < end && (s[i] == ' ' || s[i] == '\t' || s[i] == ':') {
			i++
		}
		if i >= end {
			break
		}
		// Метка до '[', ':' или ']'.
		labelStart := i
		for i < end && s[i] != '[' && s[i] != ':' && s[i] != ']' {
			i++
		}
		label := strings.TrimSpace(s[labelStart:i])
		node := &selNode{label: label}
		if i < end && s[i] == '[' {
			// Подменю: ищем парную закрывающую скобку.
			depth := 0
			j := i
			for j < end {
				if s[j] == '[' {
					depth++
				}
				if s[j] == ']' {
					depth--
					if depth == 0 {
						break
					}
				}
				j++
			}
			node.children = parseSelLevel(s, i+1, j)
			i = j + 1 // после ']'
		}
		if node.label != "" || len(node.children) > 0 {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// drawFrameStyled рисует рамку символами темы с заданным стилем (для изоляции
// меню: фон рамки непрозрачный).
func drawFrameStyled(b *Buffer, x, y, w, h int, st Style) {
	if w < 1 || h < 1 {
		return
	}
	tl := curTheme.Sym("tile_tl", "┌")
	tr := curTheme.Sym("tile_tr", "┐")
	bl := curTheme.Sym("tile_bl", "└")
	br := curTheme.Sym("tile_br", "┘")
	hh := curTheme.Sym("tile_h", "─")
	vv := curTheme.Sym("tile_v", "│")
	b.Set(x, y, tl, st)
	b.Set(x+w-1, y, tr, st)
	b.Set(x, y+h-1, bl, st)
	b.Set(x+w-1, y+h-1, br, st)
	for i := 1; i < w-1; i++ {
		b.Set(x+i, y, hh, st)
		b.Set(x+i, y+h-1, hh, st)
	}
	for j := 1; j < h-1; j++ {
		b.Set(x, y+j, vv, st)
		b.Set(x+w-1, y+j, vv, st)
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

// drawFrame рисует рамку тайла символами из темы.
func drawFrame(b *Buffer, x, y, w, h int) {
	if w < 1 || h < 1 {
		return
	}
	frameFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
	st := Style{Fg: frameFg}
	tl := curTheme.Sym("tile_tl", "┌")
	tr := curTheme.Sym("tile_tr", "┐")
	bl := curTheme.Sym("tile_bl", "└")
	br := curTheme.Sym("tile_br", "┘")
	hh := curTheme.Sym("tile_h", "─")
	vv := curTheme.Sym("tile_v", "│")

	b.Set(x, y, tl, st)
	b.Set(x+w-1, y, tr, st)
	b.Set(x, y+h-1, bl, st)
	b.Set(x+w-1, y+h-1, br, st)
	for i := 1; i < w-1; i++ {
		b.Set(x+i, y, hh, st)
		b.Set(x+i, y+h-1, hh, st)
	}
	for j := 1; j < h-1; j++ {
		b.Set(x, y+j, vv, st)
		b.Set(x+w-1, y+j, vv, st)
	}
}

// drawTabs рисует вкладки внизу (под всеми тайлами). Активная — подсвечена.
// Каждая вкладка — хотзона-ссылка (Kind "nav", Href=роут), клик переключает страницу.
func drawTabs(b *Buffer, menu [][]string, route string, out *[]Hotzone) {
	tabFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	tabBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	actBg := curTheme.ResolveColor(themeColor("title_fg"), tcell.ColorGreen)
	x := 0
	for _, m := range menu {
		label := " " + m[0] + " "
		lw := uniseg.StringWidth(label)
		st := Style{Fg: tabFg, Bg: tabBg}
		if m[1] == route {
			st = Style{Fg: tabFg, Bg: actBg, Bold: true}
		}
		b.SetString(x, b.H-1, label, st)
		*out = append(*out, Hotzone{X: x, Y: b.H - 1, W: lw, H: 1, Href: m[1], Kind: "nav"})
		x += lw + 1 // пробел в 1 клетку между вкладками
	}
}

// drawQuit рисует кнопку «Закрыть» на правом краю нижней строки (рядом с вкладками).
// Показывается ТОЛЬКО когда открыта строка вывода (statusMsg != ""). Это хотзона
// Kind "quit": клик или Enter закрывают строку вывода (как клавиша q).
// Скобки — из темы (button_l/button_r), цвета — как у вкладок.
func drawQuit(b *Buffer, out *[]Hotzone) {
	tabFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	tabBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	bl := curTheme.Sym("button_l", "⟨")
	br := curTheme.Sym("button_r", "⟩")
	text := string(bl) + " Закрыть " + string(br)
	tw := uniseg.StringWidth(text)
	x := b.W - tw
	if x < 0 {
		x = 0
	}
	b.SetString(x, b.H-1, text, Style{Fg: tabFg, Bg: tabBg})
	*out = append(*out, Hotzone{X: x, Y: b.H - 1, W: tw, H: 1, Kind: "quit"})
}

// drawStatus рисует статус-сообщение в правом нижнем углу экрана.
// Блок: 30% ширины, прижат к правому краю с отступом 2 и к низу с отступом 2
// (над вкладками, если они есть). Высота — по содержимому: от 1 до 3 строк
// текста плюс рамка. Если строк больше 3 — прокрутка колесом мыши над блоком.
// Фон непрозрачный (status_bg), чтобы сквозь статус ничего не просвечивало.
func drawStatus(b *Buffer, w, h int, hasTabs bool) {
	if statusMsg == "" && len(debugLines) == 0 {
		statusRectH = 0
		return
	}
	const textRows = 3

	// Ширина блока: 30% экрана (минимум 20, чтобы не превращался в щель).
	bw := w * 30 / 100
	if bw < 20 {
		bw = 20
	}
	if bw > w-4 {
		bw = w - 4
	}
	innerW := bw - 2

	// Все строки содержимого: отладочный вывод или перенос статуса по ширине.
	var all []string
	if len(debugLines) > 0 {
		all = debugLines
	} else {
		all = wrapStatus(statusMsg, innerW)
	}
	// Окно до textRows строк со скроллом (если контента больше).
	maxOff := len(all) - textRows
	lines := all
	if maxOff > 0 {
		if statusScroll > maxOff {
			statusScroll = maxOff
		}
		if statusScroll < 0 {
			statusScroll = 0
		}
		lines = all[statusScroll : statusScroll+textRows]
	} else {
		statusScroll = 0
	}
	if len(lines) == 0 {
		lines = []string{""}
	}

	// Высота блока: рамка + строки (от 1 до 3).
	blockH := len(lines) + 2

	// Позиция: правый нижний угол, отступы 2 от края (и от вкладок, если они есть).
	x0 := w - 2 - bw
	if x0 < 0 {
		x0 = 0
	}
	bottom := h - 3
	if hasTabs {
		bottom = h - 3 // вкладки на последней строке — блок выше них
	}
	top := bottom - blockH + 1
	if top < 0 {
		top = 0
	}

	frameFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
	statusFg := curTheme.ResolveColor(themeColor("status_fg"), tcell.ColorYellow)
	// Фон статуса: ключ status_bg, фоллбэк — bg темы, затем чёрный.
	statusBg := curTheme.ResolveColor(themeColor("status_bg"),
		curTheme.ResolveColor(themeColor("bg"), tcell.ColorBlack))
	frame := Style{Fg: frameFg, Bg: statusBg}
	text := Style{Bold: true, Fg: statusFg, Bg: statusBg}

	// Заливаем весь блок фоном — чтобы сквозь статус не просвечивали тайлы.
	fill := Style{Bg: statusBg}
	for y := top; y <= bottom; y++ {
		for x := x0; x < x0+bw; x++ {
			b.Set(x, y, ' ', fill)
		}
	}

	// Символы рамки: status_* или запасные tile_* (углы/линии тайла).
	tl := curTheme.Sym("status_tl", string(curTheme.Sym("tile_tl", "┌")))
	tr := curTheme.Sym("status_tr", string(curTheme.Sym("tile_tr", "┐")))
	bl := curTheme.Sym("status_bl", string(curTheme.Sym("tile_bl", "└")))
	br := curTheme.Sym("status_br", string(curTheme.Sym("tile_br", "┘")))
	hh := curTheme.Sym("status_h", string(curTheme.Sym("tile_h", "─")))
	vv := curTheme.Sym("status_v", string(curTheme.Sym("tile_v", "│")))

	b.Set(x0, top, tl, frame)
	b.Set(x0+bw-1, top, tr, frame)
	b.Set(x0, bottom, bl, frame)
	b.Set(x0+bw-1, bottom, br, frame)
	for x := x0 + 1; x < x0+bw-1; x++ {
		b.Set(x, top, hh, frame)
		b.Set(x, bottom, hh, frame)
	}
	for y := top + 1; y < bottom; y++ {
		b.Set(x0, y, vv, frame)
		b.Set(x0+bw-1, y, vv, frame)
	}

	// Текст: строки, обрезанные по ширине блока.
	for i, ln := range lines {
		b.SetString(x0+1, top+1+i, truncateWidth(ln, innerW), text)
	}

	// Полоса прокрутки внутри статуса (если контента больше 3 строк).
	if maxOff > 0 {
		scFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
		sc := Style{Fg: scFg}
		sx := x0 + bw - 2 // перед правой рамкой
		thumbH := textRows * textRows / len(all)
		if thumbH < 1 {
			thumbH = 1
		}
		thumbY := statusScroll * (textRows - thumbH) / maxOff
		for i := 0; i < textRows; i++ {
			if i >= thumbY && i < thumbY+thumbH {
				b.Set(sx, top+1+i, '█', sc)
			} else {
				b.Set(sx, top+1+i, '│', sc)
			}
		}
	}

	// Прямоугольник блока — для прокрутки колесом мыши.
	statusRectX, statusRectY, statusRectW, statusRectH = x0, top, bw, blockH
}

// truncateWidth обрезает строку до ширины в клетках (по uniseg).
func truncateWidth(s string, width int) string {
	if width < 1 {
		return ""
	}
	var out []rune
	w := 0
	for _, r := range s {
		rw := uniseg.StringWidth(string(r))
		if w+rw > width {
			break
		}
		out = append(out, r)
		w += rw
	}
	return string(out)
}

// wrapStatus разбивает текст на строки по ширине (перенос по рунам,
// ширина считается через uniseg). Возвращает все строки; вызывающий берёт
// столько, сколько влезает в блок статуса.
func wrapStatus(text string, width int) []string {
	if width < 1 {
		return nil
	}
	var lines []string
	var cur []rune
	w := 0
	for _, r := range text {
		rw := uniseg.StringWidth(string(r))
		if w+rw > width {
			lines = append(lines, string(cur))
			cur = nil
			w = 0
		}
		cur = append(cur, r)
		w += rw
	}
	if len(cur) > 0 {
		lines = append(lines, string(cur))
	}
	return lines
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

// openSelect открывает выпадающее меню select: корневой список якорится к кнопке
// (строго справа от неё), подменю раскрываются дальше вправо. x, y, w — позиция
// и ширина кнопки select (якорь меню, а не точка клика).
func openSelect(act, label, output, options string, x, y, w int) {
	selectMode = true
	selectAction = act
	selectOutput = output
	selectX, selectY, selectW = x, y, w
	selectStack = nil
	selectStack = append(selectStack, selLevel{nodes: parseSelTree(options)})
	statusMsg = ""
	debugLines = nil
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
	if hz.Action != "" {
		execAction(hz.Action, hz.Output)
	}
	return false
}

// renderTile рендерит содержимое тайла в inner с поддержкой скролла:
// контент рисуется в большой буфер (запас по высоте), окно копируется
// со сдвигом scrollOff[id]; хотзоны сдвигаются так же.
func renderTile(root *Node, inner *Buffer, id string, ox, oy, viewH int, out *[]Hotzone) {
	bigH := viewH * 4
	if bigH < 40 {
		bigH = 40
	}
	big := NewBuffer(inner.W, bigH)
	var hz []Hotzone
	// Видимая высота — чтобы align="center" центрировал по окну, а не по запасу.
	curViewH = viewH
	RenderHTML(root, big, ox, oy, &hz)

	contentH := usedHeight(big)
	if contentH < 1 {
		contentH = 1
	}
	maxOff := contentH - viewH
	if maxOff < 0 {
		maxOff = 0
	}
	off := scrollOff[id]
	if off > maxOff {
		off = maxOff
	}
	scrollOff[id] = off

	// Копируем видимое окно (со сдвигом) в inner.
	for yy := 0; yy < viewH && yy+off < big.H; yy++ {
		for xx := 0; xx < big.W; xx++ {
			c := big.cells[yy+off][xx]
			if c.Rune != 0 {
				inner.Set(xx, yy, c.Rune, c.Style)
			}
		}
	}
	// Хотзоны: сдвигаем на -off, отбрасываем те, что вне окна.
	for _, z := range hz {
		z.Y -= off
		if z.Y+z.H <= oy || z.Y >= oy+viewH {
			continue
		}
		*out = append(*out, z)
	}

	// Полоса прокрутки: если контент длиннее видимой области.
	if maxOff > 0 && inner.W > 1 {
		scFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
		sc := Style{Fg: scFg}
		sx := inner.W - 1 // правый край тайла (внутри рамки)
		// Высота бегунка — пропорционально видимой части контента.
		thumbH := viewH * viewH / contentH
		if thumbH < 1 {
			thumbH = 1
		}
		if thumbH > viewH {
			thumbH = viewH
		}
		thumbY := 0
		if maxOff > 0 {
			thumbY = off * (viewH - thumbH) / maxOff
		}
		for yy := 0; yy < viewH; yy++ {
			if yy >= thumbY && yy < thumbY+thumbH {
				inner.Set(sx, yy, '█', sc)
			} else {
				inner.Set(sx, yy, '│', sc)
			}
		}
	}
}

// scrollTile меняет скролл тайла под курсором (колесо мыши).
func scrollTile(pages Pages, route string, x, y, w, h, dir int) {
	for _, t := range pages[route] {
		tx, ty, tw, th := t.Rect(w, h)
		if x >= tx && x < tx+tw && y >= ty && y < ty+th {
			scrollOff[t.ID] += dir
			if scrollOff[t.ID] < 0 {
				scrollOff[t.ID] = 0
			}
			return
		}
	}
}
