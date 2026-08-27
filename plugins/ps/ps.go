// Плагин ps — «просмотр процессов»: список всех запущенных горутин движка
// (id, статус, верхняя функция) + сводка по памяти. total — ВСЯ память,
// занятая процессом (рантайм Go взял у ОС); heap/stack — её части. Память
// терминала не входит — это отдельный процесс. С флагом --track=1 и запуском
// с interval+async помечает горутины, исчезнувшие с прошлого запуска, как
// "DEAD" — удобно отлаживать утечки горутин.
package ps

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/arctcl/rough"
)

const man_ps = `ps: list all running goroutines (id, state, top func) + memory.
total = all memory used by this process (Go runtime from the OS).
Usage:
  ps               — один снимок
  ps --track=1     — помнить горутины, исчезнувшие с прошлого запуска помечать DEAD
Example:
  <plugin pipe="ps" interval="1s" async/>   — постоянно отслеживать`

// prev — id горутин с прошлого запуска (для трекинга DEAD).
var prev = map[int]bool{}

func init() {
	rough.AddMan("ps", man_ps)
	rough.AddPlugin("ps", func(in []string, args []string) ([]string, error) {
		track := false
		for _, a := range args {
			if strings.Contains(a, "track") || a == "1" {
				track = true
			}
		}

		// Дамп всех горутин.
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		dump := string(buf[:n])

		cur := map[int]bool{}
		var lines []string
		for _, block := range strings.Split(dump, "\n\n") {
			block = strings.TrimSpace(block)
			if !strings.HasPrefix(block, "goroutine ") {
				continue
			}
			id, state, top := parseGoroutine(block)
			cur[id] = true
			lines = append(lines, fmt.Sprintf("%-5d %-12s %s", id, state, top))
		}

		// Трекинг DEAD: горутины из прошлого запуска, которых больше нет.
		if track {
			for id := range prev {
				if !cur[id] {
					lines = append(lines, fmt.Sprintf("%-5d %-12s %s", id, "DEAD", ""))
				}
			}
			prev = cur
		} else {
			prev = map[int]bool{}
		}

		// Сводка по памяти: total = вся память, занятая процессом (сколько рантайм
		// Go взял у ОС). Память терминала не входит — тот отдельный процесс.
		// heap/stack — части этой памяти (для понимания, куда она ушла).
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		lines = append(lines, "",
			fmt.Sprintf("goroutines: %d  total: %s  heap: %s  stack: %s",
				runtime.NumGoroutine(), humanBytes(m.Sys), humanBytes(m.HeapAlloc), humanBytes(m.StackInuse)))
		return lines, nil
	})
}

// parseGoroutine разбирает блок дампа вида:
//
//	goroutine 5 [chan receive]:
//	main.main()
//	    /path/main.go:10 +0x25
func parseGoroutine(block string) (id int, state, top string) {
	first := block
	if i := strings.IndexByte(block, '\n'); i >= 0 {
		first = block[:i]
	}
	// "goroutine 5 [chan receive]:"
	rest := strings.TrimSuffix(strings.TrimPrefix(first, "goroutine "), ":")
	if i := strings.Index(rest, " ["); i >= 0 {
		fmt.Sscanf(rest[:i], "%d", &id)
		state = strings.Trim(rest[i+1:], "[]")
	} else {
		fmt.Sscanf(rest, "%d", &id)
	}
	// Верхняя функция — строка после заголовка.
	if i := strings.IndexByte(block, '\n'); i >= 0 {
		top = strings.TrimSpace(block[i+1:])
		if j := strings.IndexByte(top, '('); j >= 0 {
			top = top[:j]
		}
		if j := strings.IndexByte(top, ' '); j >= 0 {
			top = top[:j]
		}
	}
	return
}

// humanBytes — человекочитаемый размер (B/KB/MB/GB).
func humanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for nnd := n / unit; nnd >= unit; nnd /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
