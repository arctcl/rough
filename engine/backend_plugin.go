package engine

import (
	"strings"
	"time"
)

// pluginCache — кэш результатов <plugin>: не перезапускается чаще interval.
var pluginCache = map[string]pluginEntry{}

// maxPluginCache — верхний предел записей pluginCache. Ключ = пайп ПОСЛЕ
// подстановки переменных, поэтому при динамических пайпах (с $переменной)
// ключи меняются и без лимита кэш рос бы бесконечно. При превышении — сброс.
const maxPluginCache = 256

// backgroundRender — флаг фонового рендера неактивных страниц
// (renderBackgroundPages): в фоне выполняются ТОЛЬКО плагины с явным interval
// (живые виджеты — графики, часы), разовые плагины с побочными эффектами
// (toggle/set/ssh/export) пропускаются.
var backgroundRender bool

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
	engineMu.Lock()
	curW, curH = b.W, b.H
	if hv := n.Attrs["height"]; hv != "" {
		if hh := parseLen(hv, b.H); hh > 0 {
			curH = hh
		}
	}
	engineMu.Unlock()

	steps := pluginSteps(n)
	if len(steps) == 0 {
		f.put(b, "error: empty plugin")
		return
	}
	// async: живой плагин-служба — в своей горутине, ядро показывает последний кадр.
	if n.Attrs["async"] != "" {
		for i := range steps {
			steps[i] = expandVars(steps[i])
		}
		key := strings.Join(steps, "|")
		if !asyncLiveStarted[key] {
			iv := parseDur(n.Attrs["interval"])
			if iv <= 0 {
				iv = 2 * time.Second
			}
			startAsyncLive(key, steps, iv)
		}
		for _, ln := range asyncLive[key] {
			f.putColored(b, ln)
			f.nl(b)
		}
		return
	}
	// В фоне (неактивные страницы) выполняем плагины с явным interval (живые
	// виджеты — графики, часы) ИЛИ с флагом updateanytime (автор тайла явно
	// требует обновлять плагин даже на неактивном тайле). Разовые плагины с
	// побочными эффектами (toggle/set/ssh/export) в фоне не запускаем.
	if backgroundRender && n.Attrs["interval"] == "" && n.Attrs["updateanytime"] == "" {
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
	engineMu.Lock()
	curPluginKey = key // сигнатура для stateful-плагинов (chart)
	engineMu.Unlock()
	var lines []string
	if c, ok := pluginCache[key]; ok && time.Since(c.at) < iv {
		lines = c.lines // ещё не время — показываем прошлый результат
	} else {
		out, err := RunSteps(steps, nil)
		if err != nil {
			lines = []string{"error: " + err.Error()}
		} else {
			lines = out
		}
		// F4: ограничиваем рост — динамические пайпы дают всё новые ключи.
		if len(pluginCache) >= maxPluginCache {
			pluginCache = map[string]pluginEntry{}
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
