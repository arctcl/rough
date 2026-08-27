package engine

import (
	"testing"
	"time"
)

// Async-задача выполняется в фоне (не блокирует), а результат доставляется
// по каналу в приёмник.
func TestAsyncJobRunsInBackground(t *testing.T) {
	AddPlugin("__slow", func(in, args []string) ([]string, error) {
		time.Sleep(60 * time.Millisecond)
		return []string{"done"}, nil
	})
	defer func() { delete(plugins, "__slow") }()
	outputCache = map[string][]string{}

	startAsyncJob("__slow", "out", []string{"__slow"})
	if !asyncBusy("__slow") {
		t.Fatal("задача должна быть busy сразу после запуска")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pollAsyncJobs()
		if !asyncBusy("__slow") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if asyncBusy("__slow") {
		t.Fatal("задача не завершилась за 2с")
	}
	lines := outputCache["out"]
	if len(lines) != 1 || lines[0] != "done" {
		t.Fatalf("результат не доставлен: %v", lines)
	}
}

// Повторный запуск с тем же ключом (раскрытая команда) не создаёт дубль —
// это защита от двойного запуска async-кнопки.
func TestAsyncJobDedup(t *testing.T) {
	AddPlugin("__slow", func(in, args []string) ([]string, error) {
		time.Sleep(50 * time.Millisecond)
		return []string{"x"}, nil
	})
	defer func() { delete(plugins, "__slow") }()
	outputCache = map[string][]string{}

	startAsyncJob("k", "out", []string{"__slow"})
	startAsyncJob("k", "out", []string{"__slow"}) // дубль — игнорируется
	if len(asyncJobs) != 1 {
		t.Fatalf("должна быть одна задача, у нас %d", len(asyncJobs))
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		pollAsyncJobs()
		if len(asyncJobs) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	asyncJobs = map[string]*asyncJob{} // дочистка на случай сбоя
}
