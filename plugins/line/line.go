// Плагин line — показать строку или диапазон строк (как sed -n 'Np' / 'N,Mp').
// Вызов: ... | line:N      — одна строка
//
//	... | line:N-M    — строки с N по M (включительно)
//	line:ФАЙЛ:N-M     — то же, но читая файл
package line

import (
	"os"
	"strconv"
	"strings"

	"github.com/arctcl/rough"
)

// man_line — справка по плагину (для man).
const man_line = `line — показать строку или диапазон строк.

Использование:
  из пайпа:  ... | line:N  или  ... | line:N-M
  из файла:  action="line:ФАЙЛ:N-M"

Аргументы:
  ФАЙЛ — путь к файлу (когда строки не из пайпа).
  N    — номер строки (с 1), или диапазон N-M (с N по M включительно).

Примеры:
  action="line:/etc/x.conf:5"      — 5-я строка файла
  action="line:/etc/x.conf:5-10"   — строки 5..10
  action="cat:/etc/x.conf | line:5"  — 5-я строка из пайпа`

func init() {
	rough.AddMan("line", man_line)
	rough.AddPlugin("line", func(in []string, args []string) ([]string, error) {
		lines := in
		spec := ""
		if len(args) > 1 {
			// line:файл:N-M — читаем файл и берём строки.
			b, err := os.ReadFile(args[0])
			if err != nil {
				return nil, err
			}
			lines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
			spec = args[1]
		} else if len(args) > 0 {
			spec = args[0]
		}
		if spec == "" {
			return nil, nil
		}
		return pickLines(lines, spec), nil
	})
}

// pickLines выбирает строки по спецификации "N" или "N-M" (с 1).
func pickLines(lines []string, spec string) []string {
	if i := strings.IndexByte(spec, '-'); i > 0 {
		lo, err1 := strconv.Atoi(spec[:i])
		hi, err2 := strconv.Atoi(spec[i+1:])
		if err1 == nil && err2 == nil {
			var out []string
			for n := lo; n <= hi && n <= len(lines); n++ {
				if n >= 1 {
					out = append(out, lines[n-1])
				}
			}
			return out
		}
	}
	if n, err := strconv.Atoi(spec); err == nil {
		if n < 1 || n > len(lines) {
			return nil
		}
		return []string{lines[n-1]}
	}
	return nil
}
