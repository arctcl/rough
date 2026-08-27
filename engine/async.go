package engine

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
	key    string       // раскрытая команда (ключ задачи)
	target string       // куда вывести (output="id"); пусто = статус-строка
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
