// Плагин line — показать строку по номеру (как sed -n 'Np').
// Вызов: ... | line:N  (строка из пайпа)  или  line:ФАЙЛ:N
package line

import (
	"os"
	"strconv"
	"strings"

	"github.com/arctcl/rough"
)

// man_line — справка по плагину (для man).
const man_line = `line — показать строку по номеру (как sed -n 'Np').

Использование:
  из пайпа:  ... | line:N
  из файла:  action="line:ФАЙЛ:N"

Аргументы:
  ФАЙЛ — путь к файлу (когда строка не из пайпа).
  N    — номер строки (с 1).

Примеры:
  action="line:/etc/x.conf:5"        — 5-я строка файла
  action="cat:/etc/x.conf | line:5"  — 5-я строка из пайпа`

func init() {
	rough.AddMan("line", man_line)
	rough.AddPlugin("line", func(in []string, args []string) ([]string, error) {
		lines := in
		idx := 1
		if len(args) > 1 {
			// line:файл:N — читаем файл и берём строку N.
			b, err := os.ReadFile(args[0])
			if err != nil {
				return nil, err
			}
			lines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
			if n, err := strconv.Atoi(args[1]); err == nil {
				idx = n
			}
		} else if len(args) > 0 {
			if n, err := strconv.Atoi(args[0]); err == nil {
				idx = n
			}
		}
		if idx < 1 || idx > len(lines) {
			return nil, nil
		}
		return []string{lines[idx-1]}, nil
	})
}
