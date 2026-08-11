package engine

import (
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// statusMsg — последний результат действия/ошибка для нижней строки.
var statusMsg string

// Состояние окна ввода: пока inputMode включён, клавиши идут в буфер,
// а Enter дописывает значение к действию и выполняет его.
var (
	inputMode   bool   // открыто ли окно ввода
	inputAction string // действие, которое выполним (без значения)
	inputLabel  string // подпись (что редактируем, например MAX_USERS)
	inputBuf    string // набранное значение
	inputOutput string // id блока, куда направить результат (output="...")
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
	steps, need := PrepareAction(raw)
	if need {
		confirmMode = true
		confirmMsg = "Выполнить?"
		pendingSteps = steps
		pendingOutput = target
		statusMsg = ""
		return
	}
	out, err := RunSteps(steps, nil)
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
				// Окно подтверждения: Enter — да, Esc — нет.
				if confirmMode {
					switch e.Key() {
					case tcell.KeyEnter:
						confirmMode = false
						out, err := RunSteps(pendingSteps, nil)
						if err != nil {
							statusMsg = "ошибка: " + err.Error()
						} else {
							putOutput(out, pendingOutput)
						}
					case tcell.KeyEscape:
						confirmMode = false
						statusMsg = "отменено"
					case tcell.KeyCtrlC:
						return nil
					default:
						switch e.Rune() {
						case 'y', 'Y':
							confirmMode = false
							out, err := RunSteps(pendingSteps, nil)
							if err != nil {
								statusMsg = "ошибка: " + err.Error()
							} else {
								putOutput(out, pendingOutput)
							}
						case 'n', 'N':
							confirmMode = false
							statusMsg = "отменено"
						}
					}
					break
				}
				if e.Key() == tcell.KeyCtrlC || e.Key() == tcell.KeyEscape {
					return nil
				}
				if e.Rune() == 'q' || e.Rune() == 'Q' {
					return nil
				}
				// Вкладки: Tab — вперёд, Shift+Tab — назад, цифры 1-9 — по номеру.
				switch {
				case e.Key() == tcell.KeyTab:
					route = nextRoute(menu, route, 1)
				case e.Key() == tcell.KeyBacktab:
					route = nextRoute(menu, route, -1)
				case e.Rune() >= '1' && e.Rune() <= '9':
					if i := int(e.Rune() - '1'); i < len(menu) {
						route = menu[i][1]
					}
				}
			case *tcell.EventMouse:
				// Левый клик — hit-test по хотзонам.
				if e.Buttons()&tcell.Button1 != 0 {
					x, y := e.Position()
					kind, act, href, label, output := HitTest(x, y)
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
					RenderHTML(root, inner, x+1, y+1, &hotzones)
				}
				f.Close()
			}
		}
		bg.Copy(inner, x+1, y+1)
	}

	// Вкладки — под всеми тайлами (нижняя строка), статус — над ними.
	statusY := h - 1
	if len(menu) > 0 {
		statusY = h - 2
		drawTabs(bg, menu, route, &hotzones)
	}
	if statusMsg != "" {
		statusFg := curTheme.ResolveColor(themeColor("status_fg"), tcell.ColorYellow)
		bg.SetString(0, statusY, statusMsg, Style{Bold: true, Fg: statusFg})
	}

	// Окно подтверждения рисуется поверх всего (ввод идёт прямо в поле, без модалки).
	if confirmMode {
		drawConfirmModal(bg, w, h)
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
