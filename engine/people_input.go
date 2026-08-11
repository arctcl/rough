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

// Состояние меню выбора (select): пока selectMode включён, стрелки выбирают,
// Enter выполняет action с выбранным вариантом.
var (
	selectMode   bool
	selectOpts   []string
	selectIdx    int
	selectAction string
	selectLabel  string
	selectOutput string
	selectX, selectY int // позиция элемента — выпадающее меню рисуется под ним
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
	// Новое действие — убираем отладочный вывод прошлого раза.
	debugLines = nil
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
func putOutput(out []string, target string) {
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
				// Меню выбора: стрелки — вариант, Enter — применить, Esc — отмена.
				if selectMode {
					switch e.Key() {
					case tcell.KeyUp:
						if selectIdx > 0 {
							selectIdx--
						}
					case tcell.KeyDown:
						if selectIdx < len(selectOpts)-1 {
							selectIdx++
						}
					case tcell.KeyEnter:
						selectMode = false
						if selectIdx >= 0 && selectIdx < len(selectOpts) {
							execAction(selectAction+":"+selectOpts[selectIdx], selectOutput)
						}
					case tcell.KeyEscape:
						selectMode = false
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
				// Колесо — скролл тайла под курсором.
				if e.Buttons()&tcell.WheelUp != 0 {
					scrollTile(pages, route, mouseX, mouseY, w, h, -1)
					break
				}
				if e.Buttons()&tcell.WheelDown != 0 {
					scrollTile(pages, route, mouseX, mouseY, w, h, 1)
					break
				}
				// Левый клик — hit-test по хотзонам.
				if e.Buttons()&tcell.Button1 != 0 {
					x, y := e.Position()
					// Клик закрывает выпадающее меню select, если оно открыто.
					if selectMode {
						selectMode = false
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
					// Селект: открываем выпадающее меню ПОД элементом (как в HTML).
					if kind == "select" {
						selectMode = true
						selectAction = act
						selectLabel = label
						selectOutput = output
						selectIdx = 0
						selectOpts = nil
						for _, o := range strings.Split(options, ":") {
							if o != "" {
								selectOpts = append(selectOpts, o)
							}
						}
						// Запоминаем позицию элемента — меню рисуется под ним.
						selectX, selectY = x, y
						for _, hz := range hotzones {
							if hz.Kind == "select" && x >= hz.X && x < hz.X+hz.W && y >= hz.Y && y < hz.Y+hz.H {
								selectX, selectY = hz.X, hz.Y
								break
							}
						}
						statusMsg = ""
						debugLines = nil
						break
					}
					// Вариант выпадающего меню — клик выбирает.
					if kind == "selopt" {
						execAction(act, output)
						break
					}
					execAction(act, output)
				}
			}
		case <-tick.C:
			// Таймер: перерисовка (обновление плагинов с interval).
		}
	}
}

// renderFrame рисует текущий кадр: фон, заголовок, тайлы, вкладки и статус.
func renderFrame(s tcell.Screen, pages Pages, route string, menu [][]string, w, h int, fsys fs.FS) {
	hotzones = hotzones[:0]

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
		titleFg := curTheme.ResolveColor(themeColor("title_fg"), tcell.ColorGreen)
		inner.SetString(1, 0, t.ID, Style{Bold: true, Fg: titleFg})

		if t.File != "" {
			if f, err := fsys.Open(t.File); err == nil {
				if root, perr := ParseHTML(f); perr == nil {
					renderTile(root, inner, t.ID, x+1, y+1, th-2, &hotzones)
				}
				f.Close()
			}
		}
		bg.Copy(inner, x+1, y+1)
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

// drawSelectMenu рисует выпадающее меню select ПОД самим элементом (как в HTML),
// а не по центру экрана. Не влезло вниз — рисует над элементом. Варианты —
// хотзоны "selopt": клик выбирает.
func drawSelectMenu(b *Buffer, out *[]Hotzone, w, h int) {
	title := "▼ " + selectLabel
	// Ширина меню: под самый длинный вариант + рамка.
	mw := uniseg.StringWidth(title)
	for _, o := range selectOpts {
		if lw := uniseg.StringWidth(o); lw > mw {
			mw = lw
		}
	}
	mw += 2
	// Высота: заголовок + варианты.
	height := 1 + len(selectOpts)
	if height > h {
		height = h
	}
	// Позиция: под элементом; не влезло вниз — над ним; прижимаем к краям экрана.
	x0 := selectX
	if x0+mw > w {
		x0 = w - mw
	}
	if x0 < 0 {
		x0 = 0
	}
	y0 := selectY + 1
	if y0+height > h {
		y0 = selectY - height
	}
	if y0 < 0 {
		y0 = 0
	}

	titleBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	titleFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	inFg := curTheme.ResolveColor(themeColor("input_fg"), tcell.ColorGreen)

	// Рамка + заголовок.
	drawFrame(b, x0, y0, mw, height)
	b.SetString(x0+1, y0, " "+title+" ", Style{Bg: titleBg, Fg: titleFg})
	for i := 0; i < height-1 && i < len(selectOpts); i++ {
		st := Style{Fg: inFg}
		if i == selectIdx {
			st = Style{Fg: tcell.ColorBlack, Bg: inFg, Bold: true}
		}
		opt := " " + selectOpts[i] + " "
		ox := x0 + 1
		b.SetString(ox, y0+1+i, opt, st)
		*out = append(*out, Hotzone{X: ox, Y: y0 + 1 + i, W: uniseg.StringWidth(opt), H: 1, Kind: "selopt", Action: selectAction + ":" + selectOpts[i], Output: selectOutput})
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

// drawStatus рисует статус-сообщение внизу в рамке из темы.
// Блок: верх рамки + 3 строки текста (перенос по ширине) + низ рамки.
// Рисуется поверх тайлов, над вкладками, если они есть. Символы рамки —
// status_* из темы (фоллбэк на tile_*), цвет — frame/status_fg.
func drawStatus(b *Buffer, w, h int, hasTabs bool) {
	if statusMsg == "" && len(debugLines) == 0 {
		return
	}
	// Высота блока: рамка + 3 строки текста = 5 строк.
	const textRows = 3
	total := textRows + 2
	if h < total+1 {
		return
	}
	bottom := h - 1
	if hasTabs {
		bottom = h - 2 // освобождаем последнюю строку под вкладки
	}
	top := bottom - (total - 1) // = bottom-4

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
		for x := 0; x < w; x++ {
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

	b.Set(0, top, tl, frame)
	b.Set(w-1, top, tr, frame)
	b.Set(0, bottom, bl, frame)
	b.Set(w-1, bottom, br, frame)
	for x := 1; x < w-1; x++ {
		b.Set(x, top, hh, frame)
		b.Set(x, bottom, hh, frame)
	}
	for y := top + 1; y < bottom; y++ {
		b.Set(0, y, vv, frame)
		b.Set(w-1, y, vv, frame)
	}

	// Текст: если есть отладочный вывод (tobotom) — показываем его последние textRows
	// строк (как хвост); иначе — статус-сообщение с переносом по ширине.
	if len(debugLines) > 0 {
		lines := debugLines
		if len(lines) > textRows {
			lines = lines[len(lines)-textRows:]
		}
		for i, ln := range lines {
			b.SetString(1, top+1+i, ln, text)
		}
		return
	}
	lines := wrapStatus(statusMsg, w-2)
	if w-2 > 0 {
		for i := 0; i < textRows && i < len(lines); i++ {
			b.SetString(1, top+1+i, lines[i], text)
		}
	}
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
