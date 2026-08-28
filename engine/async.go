package engine

import "time"

// Async-модель: явный флаг `async` на элементе запускает ВЕСЬ его пайплайн
// в отдельной фоновой горутине (плагин-служба). Ядро НЕ опрашивает async-задачу
// и НЕ кеширует её — задача сама шлёт готовый вывод по каналу. Экран рисует
// только ядро (в главном цикле), поэтому рейсов нет: горутина не трогает
// состояние ядра, только свой канал.
//
// Ключ задачи — РАСКРЫТАЯ команда (после $подстановки). Две одинаковые
// async-кнопки = одна задача (защита от двойного запуска).

// asyncJob — фоновая задача async-элемента.
type asyncJob struct {
	key    string           // раскрытая команда (ключ задачи)
	target string           // куда вывести (output="id"); пусто = статус-строка
	done   chan asyncResult // канал с готовым выводом (буфер 1)
}

// asyncResult — итог фоновой задачи.
type asyncResult struct {
	lines []string
	err   error
}

// asyncJobs — реестр активных async-задач по ключу. Доступ — из главного цикла
// (однопоточно); горутина пишет только в свой done. Рейсов нет.
var asyncJobs = map[string]*asyncJob{}

// asyncBusy — идёт ли задача с таким ключом (кнопка занята).
func asyncBusy(key string) bool {
	_, ok := asyncJobs[key]
	return ok
}

// startAsyncJob запускает фоновую задачу: выполняет пайп в горутине, результат
// шлёт по каналу done. Если задача с ключом уже идёт — игнорируем (повторный
// клик не перезапускает). Буферизованный канал: горутина не блокируется, даже
// если ядро ещё не опросило.
func startAsyncJob(key, target string, steps []string) {
	if asyncBusy(key) {
		return
	}
	job := &asyncJob{key: key, target: target, done: make(chan asyncResult, 1)}
	asyncJobs[key] = job
	// Показываем «идёт…» в приёмнике, чтобы было видно, что кнопка занята.
	if target != "" {
		outputCache[target] = []string{"running…"}
	}
	go func() {
		out, err := RunSteps(steps, nil)
		job.done <- asyncResult{lines: out, err: err}
	}()
}

// pollAsyncJobs доставляет готовые async-задачи. Вызывается в главном цикле
// каждый тик: готовый результат показываем и убираем задачу из реестра.
func pollAsyncJobs() {
	for key, job := range asyncJobs {
		select {
		case res := <-job.done:
			if res.err != nil {
				showError(res.err)
			} else {
				putOutput(res.lines, job.target)
			}
			delete(asyncJobs, key)
		default:
			// ещё выполняется — ждём дальше
		}
	}
}

// --- async-живые <plugin> (плагин-служба) ---

// liveFrame — кадр живого async-плагина по ключу.
type liveFrame struct {
	key   string
	lines []string
}

// asyncLive — последний кадр каждого живого async-плагина (обновляется в
// главном цикле). Горутина шлёт кадры по asyncLiveCh — общего состояния нет,
// поэтому рейсов нет.
var (
	asyncLive        = map[string][]string{}
	asyncLiveCh      = make(chan liveFrame, 64)
	asyncLiveStarted = map[string]bool{}
	// asyncLiveStop / asyncLiveDone — каналы управления живой службой по ключу:
	// stop закрывает вызывающий (остановить), done закрывает горутина при выходе
	// (дождаться). Без остановки утёкшая горутина гонялась бы с записью в plugins
	// при cleanup тестов → -race падает.
	asyncLiveStop = map[string]chan struct{}{}
	asyncLiveDone = map[string]chan struct{}{}
)

// startAsyncLive запускает службу async-<plugin>: в своей горутине по interval
// выполняет пайп и шлёт готовый кадр в asyncLiveCh. Запускается один раз на ключ.
func startAsyncLive(key string, steps []string, iv time.Duration) {
	asyncLiveStarted[key] = true
	stop := make(chan struct{})
	done := make(chan struct{})
	asyncLiveStop[key] = stop
	asyncLiveDone[key] = done
	go func() {
		defer close(done)
		t := time.NewTicker(iv)
		defer t.Stop()
		for {
			// Остановка снаружи — выходим, не дожидаясь следующего тика.
			select {
			case <-stop:
				return
			default:
			}
			out, err := RunSteps(steps, nil)
			if err != nil {
				out = []string{"error: " + err.Error()}
			}
			// Отправка кадра и ожидание тика — тоже с проверкой остановки: если
			// asyncLiveCh полон, после отмены не блокируемся навсегда.
			select {
			case asyncLiveCh <- liveFrame{key: key, lines: out}:
			case <-stop:
				return
			}
			select {
			case <-t.C:
			case <-stop:
				return
			}
		}
	}()
}

// stopAsyncLive останавливает живую async-службу по ключу и ДОЖИДАЕТСЯ выхода
// горутины (по done). После возврата горутина больше не читает plugins и не шлёт
// в asyncLiveCh — тест может безопасно удалять плагины. Иначе утёкшая горутина
// гонялась бы с записью в plugins (AddPlugin/delete) → -race падает.
func stopAsyncLive(key string) {
	if ch, ok := asyncLiveStop[key]; ok {
		close(ch)
		delete(asyncLiveStop, key)
	}
	if d, ok := asyncLiveDone[key]; ok {
		delete(asyncLiveDone, key)
		<-d // дождаться фактического выхода горутины
	}
	delete(asyncLiveStarted, key)
}

// drainAsyncLive забирает готовые кадры живых async-плагинов в asyncLive.
// Вызывается в главном цикле каждый тик (вместе с pollAsyncJobs).
func drainAsyncLive() {
	for {
		select {
		case fr := <-asyncLiveCh:
			asyncLive[fr.key] = fr.lines
		default:
			return
		}
	}
}
