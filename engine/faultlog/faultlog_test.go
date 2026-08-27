package faultlog

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestGoSafeCatchesPanic — паника в защищённой горутине не убивает процесс,
// пишет дамп и вызывает onPanic с нужным значением.
func TestGoSafeCatchesPanic(t *testing.T) {
	var mu sync.Mutex
	var got string
	called := make(chan struct{})

	GoSafe("test", func() {
		panic("boom")
	}, func(r any, stack []byte) {
		mu.Lock()
		got, _ = r.(string)
		mu.Unlock()
		close(called)
	})

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("onPanic не вызван — горутина не защищена")
	}
	if got != "boom" {
		t.Fatalf("хотим boom, получили %q", got)
	}
}

// TestGoSafeNoPanic — если паники нет, onPanic не вызывается и fn доработала.
func TestGoSafeNoPanic(t *testing.T) {
	ran := make(chan struct{})
	GoSafe("ok", func() {
		close(ran)
	}, func(r any, stack []byte) {
		t.Fatal("onPanic не должен вызываться без паники")
	})
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("fn не выполнилась")
	}
}

// TestStackTraceNonEmpty — стек содержит текущую функцию (маркер полезности).
func TestStackTraceNonEmpty(t *testing.T) {
	s := StackTrace()
	if len(s) == 0 {
		t.Fatal("стек пуст")
	}
	if !strings.Contains(string(s), "TestStackTraceNonEmpty") {
		t.Fatal("в стеке нет текущей функции")
	}
}

// TestMemStatsFormat — снимок памяти в ожидаемом формате.
func TestMemStatsFormat(t *testing.T) {
	s := MemStats()
	if !strings.HasPrefix(s, "mem HeapAlloc=") {
		t.Fatalf("неверный формат снимка: %q", s)
	}
}

// TestLogPanicWritesFiles — LogPanic создаёт crash.log и rough.log.
// Работаем во временной папке, чтобы не засорять репозиторий.
func TestLogPanicWritesFiles(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	LogPanic("test", "x", []byte("stack"))

	for _, f := range []string{crashFile, historyFile} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("файл %s не создан: %v", f, err)
		}
	}
}

// TestAppendLogGrows — AppendLog дописывает (не перезаписывает) историю.
func TestAppendLogGrows(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	AppendLog("first")
	AppendLog("second")

	b, err := os.ReadFile(historyFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "\n") != 2 {
		t.Fatalf("хотим 2 строки, получили:\n%s", b)
	}
}
