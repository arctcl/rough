package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arctcl/rough/engine/faultlog"
	"github.com/gdamore/tcell/v2"
)

// crashNow — флаг тестового краша (кнопка «Краш» в демо). Ставится плагином
// crash через Crash(), проверяется главным циклом ВНЕ изоляции пайпов — чтобы
// паника дошла до верхнеуровневого перехвата в Run и записалась в crash.log.
var crashNow bool

// onReady — колбэки плагинов, вызываемые в Run сразу после загрузки вшитой
// папки. Нужны, потому что в init() вшитой папки ещё нет (curFS ставится
// позже), поэтому свои конфиги плагины читают именно здесь.
var onReady []func(fs.FS)

// AddOnReady регистрирует колбэк, вызываемый в Run сразу после того, как
// вшитая папка готова (curFS = fsys). Колбэк получает саму вшитую папку.
func AddOnReady(fn func(fs.FS)) {
	onReady = append(onReady, fn)
}

// Crash поднимает флаг тестового краша. Намеренный сбой для проверки crash.log:
// интерфейс вылетит на следующем кадре, а не мгновенно в пайпе.
func Crash() { crashNow = true }

// debugMode проверяет флаг --tui-debug: подробный лог в rough.log + пауза при
// краше (не закрываемся сразу, чтобы можно было отладить/снять состояние).
func debugMode() bool {
	for _, a := range os.Args[1:] {
		if a == "--tui-debug" {
			return true
		}
	}
	return false
}

// execAction выполняет action из HTML (кнопка/поле/пайп) без внешнего ввода.
func execAction(raw, target string) { execActionIn(raw, target, nil) }

// execActionIn выполняет action с возможным ВХОДОМ пайпа (ввод снаружи, инпут).
// Если в action есть $in — введённое уже подставлено аргументом в нужное место
// (пластичность); иначе введённое (in) уходит ВХОДОМ первому плагину пайпа
// (linux-стиль, как "echo введённое | плагин | ...").
// target — id блока для вывода (output="..."), пусто = статус-строка.
// Если в action есть "| confirm" — открывает окно подтверждения и ждёт.
func execActionIn(raw, target string, in []string) {
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
		pendingIn = in
		statusMsg = ""
		return
	}
	// Несколько пайпов ("a && b") — выполняем каждый последовательно.
	runAllPipes(steps, target, in)
}

// runStepsAndShow выполняет пайп и показывает результат (в тайл или статус).
// Ошибка ErrStop — пайп остановлен плагином отладки tobotom, вывод уже показан.
func runStepsAndShow(steps []string, target string, in []string) {
	out, err := RunSteps(steps, in)
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
func runAllPipes(pipes [][]string, target string, in []string) {
	if len(pipes) == 0 {
		return
	}
	if len(pipes) == 1 {
		runStepsAndShow(pipes[0], target, in)
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
			runStepsAndShow(p, target, in)
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
		runStepsAppend(p, target, in)
	}
}

// runStepsAppend выполняет пайп и ДОБАВЛЯЕТ вывод к приёмнику (склейка),
// а не перезаписывает его.
func runStepsAppend(steps []string, target string, in []string) {
	out, err := RunSteps(steps, in)
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
// Верхнеуровневый перехват паники (в горутине main): пишет полный дамп
// (crash.log + rough.log), возвращает ошибку вместо мгновенной смерти
// терминала. С флагом --tui-debug — пауза: не закрываемся сразу, ждём Enter,
// чтобы можно было подцепить отладчик к живому процессу.
func Run(fsys fs.FS) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := faultlog.StackTrace()
			faultlog.LogPanic("main", r, stack)
			err = fmt.Errorf("crash: %v (details in crash.log)", r)
			if debugMode() {
				fmt.Fprintln(os.Stderr, "\n[rough --tui-debug] паника:", r)
				fmt.Fprintln(os.Stderr, "дамп записан в crash.log. Жми Enter, чтобы выйти.")
				var b [1]byte
				_, _ = os.Stdin.Read(b[:])
			}
		}
	}()
	// Канал завершения: по сигналу ОС (graceful) или после паники в фоновой
	// горутине главный цикл должен завершиться корректно — с ошибкой или без.
	done := make(chan error, 1)
	var once sync.Once
	fail := func(err error) {
		once.Do(func() { done <- err })
	}
	// Единый загрузчик: страницы (tiles.json) + тема — всё из папки /rough.
	ui, err := LoadUI(fsys)
	if err != nil {
		return err
	}
	pages := ui.Pages
	menu := ui.Menu

	curTheme = ui.Theme
	// Вшитая папка — нужна плагину theme (переключение тем на лету).
	curFS = fsys
	// Плагины читают свои конфиги после готовности папки.
	for _, fn := range onReady {
		fn(fsys)
	}
	// Страницы, зарегистрированные программно, добавляем в общий список
	// (не трогаем tiles.json — они живут в конфиге плагина).
	for r, ts := range extraPages {
		pages[r] = ts
	}

	// Проверяльщик синтаксиса: пока есть ошибки — интерфейс не стартует.
	// Проверяем ПОСЛЕ добавления программных страниц, чтобы ловить и их.
	if errs := CheckSyntax(fsys, pages); len(errs) > 0 {
		return fmt.Errorf("проверка синтаксиса:\n  %s", syntaxErrorsOneLine(errs))
	}

	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}
	// Закрываем ПОСЛЕДНИЙ экран (при перезапуске старые закрываем вручную).
	defer func() { s.Fini() }()
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
		// Защищённая горутина: паника здесь раньше убивала процесс молча
		// (recover в Run её не видел). Теперь — дамп в crash.log и выход с ошибкой.
		faultlog.GoSafe("teletype", func() {
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
		}, func(r any, _ []byte) {
			fail(fmt.Errorf("телеграфная мышь: %v", r))
		})
	}

	// Пул событий: PollEvent блокирует, поэтому крутим его в защищённой
	// горутине. Паника здесь (tcell на Windows-консоли: ресайз, быстрый ввод,
	// спецклавиши) раньше роняла весь процесс без следа. Теперь при панике
	// ПЕРЕЗАПУСКАЕМ экран и продолжаем (а не выходим) — см. restartCh в select.
	restartCh := make(chan struct{}, 1)
	evCh := make(chan tcell.Event, 16)

	// startPoll запускает горутину чтения событий для текущего экрана s.
	// s и evCh захватываются по ссылке — после перезапуска новая горутина
	// читает уже новый экран.
	startPoll := func() {
		faultlog.GoSafe("poll", func() {
			for {
				evCh <- s.PollEvent()
			}
		}, func(r any, _ []byte) {
			faultlog.AppendLog("чтение событий: паника, перезапуск экрана: %v", r)
			select {
			case restartCh <- struct{}{}:
			default: // перезапуск уже запрошен — не дублируем
			}
		})
	}
	startPoll()

	// restartScreen закрывает старый экран и создаёт новый (после паники в
	// чтении событий). Возвращает ошибку — тогда главный цикл выйдет запасным
	// путём (с логом), а не зависнет.
	restartScreen := func() error {
		s.Fini()
		ns, err := tcell.NewScreen()
		if err != nil {
			return err
		}
		if err := ns.Init(); err != nil {
			return err
		}
		ns.EnableMouse()
		ns.Clear()
		ns.Show()
		s = ns
		return nil
	}

	// Сигналы ОС: Ctrl+C / SIGTERM → корректный выход (tcell.Fini уже
	// зарегистрирован в defer) вместо «схлопывания» посреди кадра.
	faultlog.InitSignals(func() { fail(nil) })

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
		case <-restartCh:
			// Паника в чтении событий: перезапускаем экран и продолжаем.
			if err := restartScreen(); err != nil {
				return err
			}
			startPoll()
		case <-tick.C:
			// Таймер: перерисовка активной страницы.
			// async: доставляем готовые фоновые задачи и кадры живых плагинов.
			pollAsyncJobs()
			drainAsyncLive()
		case err := <-done:
			// Завершение: сигнал ОС (nil) или паника в фоновой горутине (err).
			return err
		}
	}
}
