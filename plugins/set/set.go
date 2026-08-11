// Плагин set — поставить значение ключа в конфиге key=value.
// Вызов: action="set:ФАЙЛ:КЛЮЧ:ЗНАЧЕНИЕ"
package set

import (
	"errors"
	"os"
	"strings"

	"rough"
)

// man_set — справка по плагину (для man).
const man_set = `set — поставить значение ключа в конфиге.

Использование:
  action="set:ФАЙЛ:КЛЮЧ:ЗНАЧЕНИЕ"

Аргументы:
  ФАЙЛ     — конфиг вида ключ=значение.
  КЛЮЧ     — какой ключ менять.
  ЗНАЧЕНИЕ — новое значение (остаток, склеивается через «:»).

Примеры:
  action="set:app.conf:loglevel:debug"
  action="set:app.conf:max_users:100"`

func init() {
	rough.AddMan("set", man_set)
	rough.AddPlugin("set", func(in []string, args []string) ([]string, error) {
		if len(args) < 3 {
			return nil, errors.New("set: нужен файл, ключ и значение")
		}
		return setKey(args[0], args[1], strings.Join(args[2:], ":"))
	})
}

// setKey ставит значение ключа в файле (ключ отсутствует — добавляет).
func setKey(file, key, val string) ([]string, error) {
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
		lines[i] = trim[:eq] + "=" + val
		found = true
	}
	if !found {
		lines = append(lines, key+"="+val)
	}
	if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return nil, err
	}
	return []string{key + " = " + val}, nil
}
