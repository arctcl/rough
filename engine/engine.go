package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime/debug"
	"strings"
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
		showError(err)
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
// Верхнеуровневый перехват паники: пишет crash.log (Windows не затирает
// панику из консоли), возвращает ошибку вместо мгновенной смерти терминала.
func Run(fsys fs.FS) (err error) {
	defer func() {
		if r := recover(); r != nil {
			crash := fmt.Sprintf("panic: %v\n%s", r, debug.Stack())
			_ = os.WriteFile("crash.log", []byte(crash), 0o644)
			err = fmt.Errorf("вылет: %v (детали в crash.log)", r)
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
			case *tcell.EventKey:
				// Сырой ввод от человека → раздаём виджетам/глобальным клавишам.
				if handleKey(e, pages, menu, &route) {
					return nil
				}
			case *tcell.EventMouse:
				handleMouse(e, pages, &route, w, h)
			}
		case <-tick.C:
			// Таймер: перерисовка (обновление плагинов с interval).
		}
	}
}
