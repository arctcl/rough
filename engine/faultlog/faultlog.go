// Пакет faultlog — фрейм отлова и диагностики движка rough.
//
// Зачем: раньше паника в фоновой горутине (чтение событий терминала, мышь)
// молча убивала весь процесс — главный recover в Run её не видел, и crash.log
// не писался. Это и были «немые» вылеты на Windows-терминале.
//
// Что даёт пакет:
//   - GoSafe        — запуск горутины с защитой от паники (пишет дамп + колбэк);
//   - LogPanic      — фатальное событие в crash.log + rough.log;
//   - StackTrace    — стек всех горутин;
//   - MemStats      — однострочный снимок памяти и числа горутин;
//   - WriteCrash    — дамп краша (crash.log);
//   - AppendLog     — история событий (rough.log, кольцевой);
//   - InitSignals   — корректный выход по Ctrl+C / SIGTERM.
//
// Пакет изолирован, без зависимостей движка, покрыт тестами.
package faultlog

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	crashFile   = "crash.log" // фатальные дампы (перезаписывается)
	historyFile = "rough.log" // история событий (кольцевая)
	maxHistory  = 256 * 1024  // rough.log не растёт бесконечно (256 КБ)
)

// GoSafe запускает fn в отдельной горутине с защитой от паники.
//
// Если fn паникует:
//   - пишется полный дамп (crash.log + rough.log + стек всех горутин);
//   - вызывается onPanic(r, stack), чтобы основной цикл мог корректно
//     завершиться с ошибкой, а не схлопнуться молча.
//
// Без GoSafe паника в фоновой горутине уронила бы весь процесс, не оставив
// следов. onPanic может быть nil — тогда только логируем.
func GoSafe(name string, fn func(), onPanic func(r any, stack []byte)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := StackTrace()
				LogPanic(name, r, stack)
				if onPanic != nil {
					onPanic(r, stack)
				}
			}
		}()
		fn()
	}()
}

// LogPanic пишет фатальное событие в оба журнала: где упало, значение паники
// и стек. Горутина-жертва уже паникует, поэтому стек всех горутин (через
// StackTrace) даёт полную картину.
func LogPanic(where string, r any, stack []byte) {
	WriteCrash(where, r, stack)
	AppendLog("panic in %s: %v", where, r)
}

// StackTrace возвращает стек ВСЕХ горутин (а не только текущей), усечённый
// до 1 МБ. По нему видно, что делали остальные горутины в момент сбоя.
func StackTrace() []byte {
	buf := make([]byte, 1<<20) // 1 МБ
	n := runtime.Stack(buf, true)
	return buf[:n]
}

// MemStats возвращает однострочный снимок памяти и числа горутин:
//
//	mem HeapAlloc=... HeapSys=... NumGC=... NumGoroutine=...
//
// Полезно сопоставлять с «дневником» перед крашем — видно, росла ли память
// или что-то утекло.
func MemStats() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf(
		"mem HeapAlloc=%d HeapSys=%d NumGC=%d NumGoroutine=%d",
		m.HeapAlloc, m.HeapSys, m.NumGC, runtime.NumGoroutine())
}

// WriteCrash записывает фатальный дамп в crash.log (перезаписывает).
// Внутри: время, где упало, значение паники, стек всех горутин и снимок
// памяти — всё, что нужно, чтобы понять причину без отладчика.
func WriteCrash(where string, r any, stack []byte) {
	body := fmt.Sprintf(
		"=== CRASH %s ===\nwhere: %s\npanic: %v\n%s\n\n%s\n",
		time.Now().Format(time.RFC3339), where, r, stack, MemStats())
	_ = os.WriteFile(crashFile, []byte(body), 0o644)
}

// AppendLog дописывает строку события в rough.log (история «что было до
// краха»). Файл ограничен: при превышении maxHistory переименовывается
// в rough.log.old и начинается заново — история не разрастается бесконечно.
func AppendLog(format string, a ...any) {
	line := fmt.Sprintf("%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, a...))
	f, err := os.OpenFile(historyFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
	trimHistory()
}

// trimHistory следит, чтобы rough.log не превышал maxHistory.
func trimHistory() {
	fi, err := os.Stat(historyFile)
	if err != nil || fi.Size() < maxHistory {
		return
	}
	_ = os.Rename(historyFile, historyFile+".old")
}

// InitSignals регистрирует обработку сигналов ОС (Ctrl+C / SIGTERM).
// При сигнале вызывается onExit — движок корректно закрывает экран
// (tcell.Fini уже зарегистрирован в Run) и выходит, а не «схлопывается»
// посреди кадра. На Windows реально приходит только os.Interrupt (Ctrl+C).
func InitSignals(onExit func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-ch
		AppendLog("signal: %v", s)
		if onExit != nil {
			onExit()
		}
	}()
}
