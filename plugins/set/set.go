// Плагин set — поставить значение ключа в конфиге приложения.
// Конфиги бывают двух форматов: ключ=значение и ключ значение (через пробел).
// Какой символ отделяет ключ от значения — флаг --разделитель (по умолчанию "=").
// Вызов: action="set:ФАЙЛ:КЛЮЧ:ЗНАЧЕНИЕ [--разделитель=СИМВОЛ]"
package set

import (
	"errors"
	"os"
	"strings"

	"rough"
	"rough/engine"
)

// man_set — справка по плагину (для man).
const man_set = `set — поставить значение ключа в конфиге приложения.

Использование:
  action="set:ФАЙЛ:КЛЮЧ:ЗНАЧЕНИЕ"
  action="set:ФАЙЛ:КЛЮЧ:ЗНАЧЕНИЕ --разделитель=пробел"  — конфиг через пробел

Аргументы:
  ФАЙЛ     — конфиг приложения.
  КЛЮЧ     — какой ключ менять.
  ЗНАЧЕНИЕ — новое значение (остаток, склеивается через «:»).
  --разделитель=СИМВОЛ — что отделяет ключ от значения. По умолчанию "="
                (конфиг вида ключ=значение). Для конфига вида "ключ значение"
                укажи --разделитель=пробел.

Примеры:
  action="set:app.conf:loglevel:debug"                    — ключ=значение
  action="set:/etc/mailcow:limit:100 --разделитель=пробел" — ключ значение`

func init() {
	rough.AddMan("set", man_set)
	rough.AddPlugin("set", func(in []string, args []string) ([]string, error) {
		sep, args := engine.FlagValue(args, "разделитель")
		sep = normSep(sep)
		if len(args) < 3 {
			return nil, errors.New("set: нужен файл, ключ и значение")
		}
		// Режим чтения: set:файл:ключ:get — вернуть текущее значение (для select).
		if args[2] == "get" {
			return readKey(args[0], args[1], sep)
		}
		return setKey(args[0], args[1], strings.Join(args[2:], ":"), sep)
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

// readKey возвращает текущее значение ключа (для select). Ключа нет — "?".
func readKey(file, key, sep string) ([]string, error) {
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
	return []string{"?"}, nil
}

// setKey ставит значение ключа в файле (ключ отсутствует — добавляет).
func setKey(file, key, val, sep string) ([]string, error) {
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
		lines[i] = trim[:idx] + sep + val
		found = true
	}
	if !found {
		lines = append(lines, key+sep+val)
	}
	if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return nil, err
	}
	return []string{key + " = " + val}, nil
}
