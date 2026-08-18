// Плагин tr — замена или удаление символов в строках (как tr).
// Умеет диапазоны (a-z, 0-9) и регистр (lower/upper).
// Юникс-лайк: `tr 'a-z' 'A-Z'`, `tr -d '0-9'` — здесь единые quick-параметры:
// ... | tr:FROM:TO, ... | tr:FROM (удаление), ... | tr:lower / tr:upper.
// Работает с текстом в пайпе — обработка перед bars/chart (см. принцип
// «в одном плагине — одно дело»).
package tr

import (
	"errors"
	"strings"

	"github.com/arctcl/rough"
	"github.com/arctcl/rough/engine"
)

// man_tr — справка по плагину (для man).
const man_tr = `tr — замена или удаление символов в строках (как tr).

Использование (единые quick-параметры):
  часть пайпа: ... | tr:FROM:TO   — заменить символы FROM на TO
               ... | tr:FROM      — удалить символы FROM (как tr -d)
               ... | tr:lower     — весь текст в нижний регистр
               ... | tr:upper     — весь текст в верхний регистр

Аргументы:
  from — что менять: символы и диапазоны (a-z, 0-9), или слово lower/upper.
  to   — на что менять (символы/диапазоны). Пусто:
           from = lower/upper → регистр; иначе → удаление символов from.

Примеры:
  action="cat:log | tr:a-z:A-Z"      — лог в верхний регистр
  action="cat:log | tr:upper"        — то же самое, проще
  action="cat:file | tr:0-9:"        — удалить цифры
  action="cat:file | tr:a-z:"        — удалить строчные буквы`

// trParams — параметры tr. Порядок = позиции: from, to.
var trParams = []engine.Param{
	{Name: "from", Required: true},
	{Name: "to"},
}

func init() {
	rough.AddMan("tr", man_tr)
	rough.AddPlugin("tr", func(in []string, args []string) ([]string, error) {
		vals, err := engine.ParseArgs(args, trParams)
		if err != nil {
			return nil, err
		}
		from, to := vals["from"], vals["to"]

		// Регистр целиком: tr:lower / tr:upper (TO не нужен).
		switch from {
		case "lower":
			return mapLines(in, strings.ToLower), nil
		case "upper":
			return mapLines(in, strings.ToUpper), nil
		}

		fromRunes, err := expandSet(from)
		if err != nil {
			return nil, err
		}
		if len(fromRunes) == 0 {
			return nil, errors.New("tr: пустой набор from")
		}

		// Удаление (как tr -d): TO пуст — выкидываем символы FROM.
		if to == "" {
			del := make(map[rune]bool, len(fromRunes))
			for _, r := range fromRunes {
				del[r] = true
			}
			return mapLines(in, func(s string) string {
				return strings.Map(func(r rune) rune {
					if del[r] {
						return -1
					}
					return r
				}, s)
			}), nil
		}

		// Замена: пары FROM→TO. Лишним FROM повторяется последний TO (как tr).
		toRunes, err := expandSet(to)
		if err != nil {
			return nil, err
		}
		if len(toRunes) == 0 {
			return nil, errors.New("tr: пустой набор to")
		}
		sub := make(map[rune]rune, len(fromRunes))
		for i, r := range fromRunes {
			t := toRunes[len(toRunes)-1]
			if i < len(toRunes) {
				t = toRunes[i]
			}
			sub[r] = t
		}
		return mapLines(in, func(s string) string {
			return strings.Map(func(r rune) rune {
				if t, ok := sub[r]; ok {
					return t
				}
				return r
			}, s)
		}), nil
	})
}

// mapLines применяет f к каждой строке входа и собирает результат.
func mapLines(in []string, f func(string) string) []string {
	out := make([]string, 0, len(in))
	for _, ln := range in {
		out = append(out, f(ln))
	}
	return out
}

// expandSet разворачивает набор символов с диапазонами: "a-z" → a..z,
// "0-9a-f" → 0..9 + a..f; одиночные символы — как есть. Символ '-' между
// двумя рунами (prev < next) задаёт диапазон, иначе — обычный символ.
func expandSet(s string) ([]rune, error) {
	rs := []rune(s)
	var out []rune
	for i := 0; i < len(rs); i++ {
		// Диапазон X-Y: X < Y (например a-z, 0-9).
		if i+2 < len(rs) && rs[i+1] == '-' && rs[i] < rs[i+2] {
			for r := rs[i]; r <= rs[i+2]; r++ {
				out = append(out, r)
			}
			i += 2
			continue
		}
		out = append(out, rs[i])
	}
	return out, nil
}
