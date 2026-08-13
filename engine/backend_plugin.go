package engine

import (
	"strings"
	"time"
)

// pluginCache — кэш результатов <plugin>: не перезапускается чаще interval.
var pluginCache = map[string]pluginEntry{}

// pluginEntry — кэшированный результат плагина и время последнего запуска.
type pluginEntry struct {
	at    time.Time
	lines []string
}

// renderPlugin обрабатывает тег <plugin>: собирает пайп из атрибутов,
// выполняет его и выводит строки результата в тайл.
func renderPlugin(n *Node, b *Buffer, f *flowState) {
	// Размер окна тайла — чтобы рисовалки (bars и т.п.) адаптировались.
	// height на <plugin> — высота зоны графика (chart рисует на ней).
	curW, curH = b.W, b.H
	if hv := n.Attrs["height"]; hv != "" {
		if hh := parseLen(hv, b.H); hh > 0 {
			curH = hh
		}
	}

	steps := pluginSteps(n)
	if len(steps) == 0 {
		f.put(b, "ошибка: пустой плагин")
		return
	}
	// Подстановка переменных $имя (движок) — живой контент тоже умеет.
	for i := range steps {
		steps[i] = expandVars(steps[i])
	}
	// Интервал обновления: не задан → дефолт 2 секунды (не дёргаем чаще).
	iv := parseDur(n.Attrs["interval"])
	if iv <= 0 {
		iv = 2 * time.Second
	}
	key := strings.Join(steps, "|")
	curPluginKey = key // сигнатура для stateful-плагинов (chart)
	var lines []string
	if c, ok := pluginCache[key]; ok && time.Since(c.at) < iv {
		lines = c.lines // ещё не время — показываем прошлый результат
	} else {
		out, err := RunSteps(steps, nil)
		if err != nil {
			lines = []string{"ошибка: " + err.Error()}
		} else {
			lines = out
		}
		pluginCache[key] = pluginEntry{at: time.Now(), lines: lines}
	}
	for _, ln := range lines {
		// Вывод плагина может нести цветовые маркеры \x01{имя} из темы.
		f.putColored(b, ln)
		f.nl(b)
	}
}

// pluginSteps собирает пайп из атрибутов <plugin>.
// Явный pipe — приоритет; иначе sugar: [source|file:path] + name[:mask].
func pluginSteps(n *Node) []string {
	if p := n.Attrs["pipe"]; p != "" {
		return SplitSteps(p)
	}
	var steps []string
	src := n.Attrs["source"]
	if src == "" {
		if path := n.Attrs["path"]; path != "" {
			src = "file:" + path
		}
	}
	if src != "" {
		steps = append(steps, src)
	}
	name := n.Attrs["name"]
	if name == "" {
		name = "text"
	}
	if m := n.Attrs["mask"]; m != "" {
		name += ":" + m
	}
	steps = append(steps, name)
	return steps
}

// parseDur разбирает длительность вида "1s", "500ms", "1m".
func parseDur(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}
