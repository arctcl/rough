// Плагин set — поставить значение ключа в конфиге приложения.
// Конфиги бывают двух форматов: ключ=значение и ключ значение (через пробел).
// Какой символ отделяет ключ от значения — флаг --sep (по умолчанию "=").
// Вызов: action="set:FILE:KEY:VALUE [--sep=CHAR]"
package set

import (
	"errors"
	"os"
	"strings"

	"github.com/arctcl/rough"
	"github.com/arctcl/rough/engine"
)

// man_set — справка по плагину (для man).
const man_set = `set — поставить значение ключа в конфиге приложения.

Использование:
  action="set:FILE:KEY:VALUE"
  action="set:FILE:KEY:VALUE --sep=space"  — конфиг через пробел

Аргументы:
  FILE  — конфиг приложения.
  KEY   — какой ключ менять.
  VALUE — новое значение (остаток, склеивается через «:»).
  --sep=CHAR — что отделяет ключ от значения. По умолчанию "="
                (конфиг вида ключ=значение). Для конфига вида "ключ значение"
                укажи --sep=space.

Примеры:
  action="set:app.conf:loglevel:debug"                    — ключ=значение
  action="set:/etc/mailcow:limit:100 --sep=space" — ключ значение`

func init() {
	rough.AddMan("set", man_set)
	rough.AddPlugin("set", func(in []string, args []string) ([]string, error) {
		sep, args := engine.FlagValue(args, "sep")
		sep = normSep(sep)
		if len(args) < 3 {
			return nil, errors.New("set: нужен файл, ключ и значение")
		}
		// Режим чтения: set:file:key:get — вернуть текущее значение (для select).
		if args[2] == "get" {
			return readKey(args[0], args[1], sep)
		}
		return setKey(args[0], args[1], strings.Join(args[2:], ":"), sep)
	})
}

// normSep нормализует разделитель ключ-значение: пусто/"=" → "=";
// слово "space" → пробел.
func normSep(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "=":
		return "="
	case "space":
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
