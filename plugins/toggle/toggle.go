// Плагин toggle — переключить флаг в конфиге key=value (как перещелкнуть выключатель).
// Инвертирует значение: 0↔1, on↔off, true↔false. Ключа нет — добавляет включённым.
// Вызов: action="toggle:ФАЙЛ:КЛЮЧ"
package toggle

import (
	"errors"
	"os"
	"strings"

	"rough"
)

// man_toggle — справка по плагину (для man).
const man_toggle = `toggle — переключить флаг в конфиге (как перещелкнуть выключатель).

Использование:
  action="toggle:ФАЙЛ:КЛЮЧ"

Аргументы:
  ФАЙЛ — конфиг вида ключ=значение (по строке на ключ).
  КЛЮЧ — что переключить. Инвертирует: 0↔1, on↔off, true↔false.

Примеры:
  action="toggle:/etc/app.conf:logging"
  <checkbox action="toggle:app.conf:debug">Отладка</checkbox>`

func init() {
	rough.AddMan("toggle", man_toggle)
	rough.AddPlugin("toggle", func(in []string, args []string) ([]string, error) {
		if len(args) < 2 {
			return nil, errors.New("toggle: нужен файл и ключ")
		}
		// Режим чтения: toggle:файл:ключ:get — вернуть текущее значение (для checkbox).
		if len(args) >= 3 && args[2] == "get" {
			return readValue(args[0], args[1])
		}
		return toggleKey(args[0], args[1])
	})
}

// readValue возвращает текущее значение ключа (для checkbox). Ключа нет — "0".
func readValue(file, key string) ([]string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	for _, ln := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(ln)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		eq := strings.Index(trim, "=")
		if eq < 0 || strings.TrimSpace(trim[:eq]) != key {
			continue
		}
		return []string{strings.TrimSpace(trim[eq+1:])}, nil
	}
	return []string{"0"}, nil
}

// toggleKey инвертирует значение ключа в файле и пишет обратно.
func toggleKey(file, key string) ([]string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	found := false
	for i, ln := range lines {
		trim := strings.TrimSpace(ln)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		eq := strings.Index(trim, "=")
		if eq < 0 || strings.TrimSpace(trim[:eq]) != key {
			continue
		}
		lines[i] = trim[:eq] + "=" + invert(strings.TrimSpace(trim[eq+1:]))
		found = true
	}
	if !found {
		lines = append(lines, key+"=1")
	}
	if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return nil, err
	}
	return []string{key + " = " + currentValue(lines, key)}, nil
}

// invert инвертирует значение: 0↔1, on↔off, true↔false, иначе 1.
func invert(v string) string {
	switch strings.ToLower(v) {
	case "1", "on", "true", "yes":
		return "0"
	default:
		return "1"
	}
}

// currentValue возвращает последнее значение ключа (для сообщения в статусе).
func currentValue(lines []string, key string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		trim := strings.TrimSpace(lines[i])
		if eq := strings.Index(trim, "="); eq >= 0 && strings.TrimSpace(trim[:eq]) == key {
			return strings.TrimSpace(trim[eq+1:])
		}
	}
	return "?"
}
