// Плагин toggle — переключить флаг в конфиге приложения (как перещелкнуть
// выключатель). Инвертирует значение: 0↔1, on↔off, true↔false. Ключа нет —
// добавляет включённым. Формат конфига (ключ=значение или ключ значение) —
// флаг --разделитель (по умолчанию "=").
// Вызов: action="toggle:ФАЙЛ:КЛЮЧ [--разделитель=СИМВОЛ]"
package toggle

import (
	"errors"
	"os"
	"strings"

	"rough"
	"rough/engine"
)

// man_toggle — справка по плагину (для man).
const man_toggle = `toggle — переключить флаг в конфиге (как перещелкнуть выключатель).

Использование:
  action="toggle:ФАЙЛ:КЛЮЧ"
  action="toggle:ФАЙЛ:КЛЮЧ --разделитель=пробел"  — конфиг через пробел

Аргументы:
  ФАЙЛ — конфиг приложения (по строке на ключ).
  КЛЮЧ — что переключить. Инвертирует: 0↔1, on↔off, true↔false.
  --разделитель=СИМВОЛ — что отделяет ключ от значения. По умолчанию "="
                (конфиг вида ключ=значение). Для конфига вида "ключ значение"
                укажи --разделитель=пробел.

Примеры:
  action="toggle:/etc/app.conf:logging"
  <checkbox action="toggle:app.conf:debug">Отладка</checkbox>
  <checkbox action="toggle:/etc/mailcow:debug --разделитель=пробел">Отладка</checkbox>`

func init() {
	rough.AddMan("toggle", man_toggle)
	rough.AddPlugin("toggle", func(in []string, args []string) ([]string, error) {
		sep, args := engine.FlagValue(args, "разделитель")
		sep = normSep(sep)
		if len(args) < 2 {
			return nil, errors.New("toggle: нужен файл и ключ")
		}
		// Режим чтения: toggle:файл:ключ:get — вернуть текущее значение (для checkbox).
		if len(args) >= 3 && args[2] == "get" {
			return readValue(args[0], args[1], sep)
		}
		return toggleKey(args[0], args[1], sep)
	})
}

// normSep нормализует разделитель ключ-значение: пусто/"=" → "=";
// слово "пробел"/"space" → пробел.
func normSep(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "=":
		return "="
	case "пробел", "space":
		return " "
	}
	return s
}

// readValue возвращает текущее значение ключа (для checkbox). Ключа нет — "0".
func readValue(file, key, sep string) ([]string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	for _, ln := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(ln)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		i := strings.Index(trim, sep)
		if i < 0 || strings.TrimSpace(trim[:i]) != key {
			continue
		}
		return []string{strings.TrimSpace(trim[i+len(sep):])}, nil
	}
	return []string{"0"}, nil
}

// toggleKey инвертирует значение ключа в файле и пишет обратно.
func toggleKey(file, key, sep string) ([]string, error) {
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
		idx := strings.Index(trim, sep)
		if idx < 0 || strings.TrimSpace(trim[:idx]) != key {
			continue
		}
		lines[i] = trim[:idx] + sep + invert(strings.TrimSpace(trim[idx+len(sep):]))
		found = true
	}
	if !found {
		lines = append(lines, key+sep+"1")
	}
	if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return nil, err
	}
	return []string{key + " = " + currentValue(lines, key, sep)}, nil
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
func currentValue(lines []string, key, sep string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		trim := strings.TrimSpace(lines[i])
		if idx := strings.Index(trim, sep); idx >= 0 && strings.TrimSpace(trim[:idx]) == key {
			return strings.TrimSpace(trim[idx+len(sep):])
		}
	}
	return "?"
}
