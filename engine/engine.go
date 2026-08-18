package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
)

// crashNow — флаг тестового краша (кнопка «Краш» в демо). Ставится плагином
// crash через Crash(), проверяется главным циклом ВНЕ изоляции пайпов — чтобы
// паника дошла до верхнеуровневого перехвата в Run и записалась в crash.log.
var crashNow bool

// Crash поднимает флаг тестового краша. Намеренный сбой для проверки crash.log:
// интерфейс вылетит на следующем кадре, а не мгновенно в пайпе.
func Crash() { crashNow = true }

// execAction выполняет action из HTML (кнопка/поле/пайп).
// target — id блока для вывода (output="..."), пусто = статус-строка.
// Если в action есть "| confirm" — открывает окно подтверждения и ждёт.
func execAction(raw, target string) {
	// Новое действие — убираем отладочный вывод, скролл статуса и сбрасываем таймер.
	debugLines = nil
	statusScroll = 0
	statusShownAt = time.Now()
	// Действие могло изменить файл (toggle/set/...) — сбрасываем кэш состояния
	// чекбоксов/селектов, чтобы следующий кадр перечитал актуальное значение.
	clearStateCache()
	steps, need := PrepareAction(raw)
	if need {
		confirmMode = true
		confirmMsg = "Execute?"
		pendingPipes = steps
		pendingOutput = target
		statusMsg = ""
		return
	}
	// Несколько пайпов ("a && b") — выполняем каждый последовательно.
	runAllPipes(steps, target)
}

// runStepsAndShow выполняет пайп и показывает результат (в тайл или статус).
// Ошибка ErrStop — пайп остановлен плагином отладки tobotom, вывод уже показан.
func runStepsAndShow(steps []string, target string) {
	out, err := RunSteps(steps, nil)
	if errors.Is(err, ErrStop) {
		return
	}
	if err != nil {
		showError(err)
		return
	}
	putOutput(out, target)
}

// isClearPipe — пайп из одного шага clear (маркер «начать склейку с чистого»).
func isClearPipe(p []string) bool {
	if len(p) != 1 {
		return false
	}
	name, _ := SplitAction(p[0])
	return name == "clear"
}

// runAllPipes выполняет пайпы действия. Если среди них есть clear — приёмник
// очищается и каждый пайп ДОБАВЛЯЕТ свой вывод (склейка, как "cat a b"); без
// clear пайпы выполняются, последний перезаписывает приёмник (один пайп —
// обычное выполнение). Так "clear && man:ssh && cat:/etc/hosts" соберёт
// справку и файл подряд в один блок.
func runAllPipes(pipes [][]string, target string) {
	if len(pipes) == 0 {
		return
	}
	if len(pipes) == 1 {
		runStepsAndShow(pipes[0], target)
		return
	}
	hasClear := false
	for _, p := range pipes {
		if isClearPipe(p) {
			hasClear = true
			break
		}
	}
	if !hasClear {
		for _, p := range pipes {
			runStepsAndShow(p, target)
		}
		return
	}
	// Склейка: clear очищает приёмник, остальные пайпы добавляют вывод.
	if target != "" {
		outputCache[target] = nil
	}
	for _, p := range pipes {
		if isClearPipe(p) {
			continue
		}
		runStepsAppend(p, target)
	}
}

// runStepsAppend выполняет пайп и ДОБАВЛЯЕТ вывод к приёмнику (склейка),
// а не перезаписывает его.
func runStepsAppend(steps []string, target string) {
	out, err := RunSteps(steps, nil)
	if errors.Is(err, ErrStop) {
		return
	}
	if err != nil {
		showError(err)
		return
	}
	for i := range out {
		out[i] = StripMarkers(out[i])
	}
	if target != "" {
		outputCache[target] = append(outputCache[target], out...)
		statusMsg = ""
		return
	}
	statusMsg = strings.Join(out, " | ")
}

// putOutput направляет результат действия: в блок вывода (по id) или в статус-строку.
// Маркеры цветов вычищаем — тут раскраска не рисуется, остался бы мусор.
func putOutput(out []string, target string) {
	for i := range out {
		out[i] = StripMarkers(out[i])
	}
	if target != "" {
		outputCache[target] = out
		// Результат уже виден в тайле-приёмнике — не дублируем его в статус,
		// иначе «done → target» мелькал бы при каждом действии (поля, select,
		// checkbox). Статус остаётся для ошибок и действий без тайла-приёмника.
		statusMsg = ""
		return
	}
	statusMsg = strings.Join(out, " | ")
}

// Run запускает движок: читает всё из вшитой папки (fs.FS), рисует интерфейс,
// обрабатывает ввод. Работает, пока пользователь не нажмёт q / Esc / Ctrl+C.
// Верхнеуровневый перехват паники: пишет crash.log (Windows не затирает
// панику из консоли), возвращает ошибку вместо мгновенной смерти терминала.
func Run(fsys fs.FS) (err error) {
	defer func() {
		if r := recover(); r != nil {
			crash := fmt.Sprintf("panic: %v\n%s", r, debug.Stack())
			_ = os.WriteFile("crash.log", []byte(crash), 0o644)
			err = fmt.Errorf("crash: %v (details in crash.log)", r)
		}
	}()
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
	// Стартовая страница — первая вкладка меню (если есть), иначе первый роут:
	// FirstRoute итерирует map в случайном порядке, а стартовать с первой
	// вкладки меню — предсказуемо для пользователя.
	if len(menu) > 0 {
		route = menu[0][1]
	}
	w, h := s.Size()

	// Телеграфная мышь читает размеры экрана в своей горутине — храним их
	// атомарно, чтобы не было гонки с главным циклом (ресайз перезаписывает w/h).
	var sizeW, sizeH atomic.Int32
	sizeW.Store(int32(w))
	sizeH.Store(int32(h))

	// Телеграфная мышь (голый Linux VT без X): сырое /dev/input/mice.
	// На X/Wayland openTeletypeMouse вернёт nil — мышь идёт через tcell
	// (источники взаимоисключающие). События шлются в свой канал и
	// обрабатываются единым обработчиком handleMouseEvent.
	var telCh chan MouseEvent
	if tm := openTeletypeMouse(); tm != nil {
		defer tm.Close()
		telCh = make(chan MouseEvent, 16)
		go func() {
			for {
				evs, err := tm.read(int(sizeW.Load()), int(sizeH.Load()))
				if err != nil {
					close(telCh)
					return
				}
				for _, ev := range evs {
					telCh <- ev
				}
			}
		}()
	}

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
		// Тестовый краш (кнопка «Краш» в демо): паника ВНЕ изоляции пайпов —
		// верхнеуровневый перехват в Run пишет crash.log и возвращает ошибку.
		if crashNow {
			panic("crash: тестовый краш (см. crash.log)")
		}
		renderFrame(s, pages, route, menu, w, h, fsys)
		select {
		case ev := <-evCh:
			switch e := ev.(type) {
			case *tcell.EventResize:
				// Растягиватель: терминал сменил размер — тайлы пересчитаются
				// на следующем кадре (renderFrame по новым w/h).
				w, h = e.Size()
				sizeW.Store(int32(w))
				sizeH.Store(int32(h))
			case *tcell.EventKey:
				// Сырой ввод от человека → раздаём виджетам/глобальным клавишам.
				if handleKey(e, pages, menu, &route) {
					return nil
				}
			case *tcell.EventMouse:
				handleMouse(e, pages, &route, w, h)
			}
		case me, ok := <-telCh:
			// Телеграфная мышь: единый обработчик (источник ему не важен).
			if ok {
				handleMouseEvent(me, pages, &route, w, h)
			}
		case <-tick.C:
			// Таймер: перерисовка активной страницы + фоновый прогон «живых»
			// плагинов неактивных страниц (графики продолжают собирать данные,
			// пока пользователь на другой вкладке).
			renderBackgroundPages(pages, route, w, h, fsys)
		}
	}
}
