// Плагин sort — отсортировать строки (как sort).
// Юникс-лайк: `sort`, `sort -r`, `sort -h` — здесь единые quick-параметры:
// ... | sort [--reverse=1] [--numeric=1].
// Работает с текстом в пайпе — сортировка перед bars/chart/head/tail.
package sort

import (
	"sort"
	"strconv"
	"strings"

	"github.com/arctcl/rough"
	"github.com/arctcl/rough/engine"
)

// man_sort — справка по плагину (для man).
const man_sort = `sort — отсортировать строки (как sort).

Использование (единые quick-параметры):
  часть пайпа: ... | sort [--reverse=1] [--numeric=1]

Аргументы:
  reverse — 1 — по убыванию (по умолчанию 0 — по возрастанию).
  numeric — 1 — по числовому значению с суффиксами K/M/G (как sort -h).

Примеры:
  action="cat:hosts | sort"                       — по алфавиту
  action="cat:hosts | sort --reverse=1"           — по убыванию
  action="cat:du.log | sort --numeric=1 | tail:5" — топ-5 по размеру`

// sortParams — параметры sort. Порядок = позиции: reverse, numeric.
var sortParams = []engine.Param{
	{Name: "reverse", Default: "0"},
	{Name: "numeric", Default: "0"},
}

func init() {
	rough.AddMan("sort", man_sort)
	rough.AddPlugin("sort", func(in []string, args []string) ([]string, error) {
		vals, err := engine.ParseArgs(args, sortParams)
		if err != nil {
			return nil, err
		}
		rev := vals["reverse"] == "1"
		num := vals["numeric"] == "1"

		out := make([]string, len(in))
		copy(out, in)

		// Стабильная сортировка: равные строки не переставляются.
		sort.SliceStable(out, func(i, j int) bool {
			if num {
				ni, nj := humanNumber(out[i]), humanNumber(out[j])
				if ni == nj {
					return out[i] < out[j]
				}
				return ni < nj
			}
			return out[i] < out[j]
		})
		if rev {
			for l, r := 0, len(out)-1; l < r; l, r = l+1, r-1 {
				out[l], out[r] = out[r], out[l]
			}
		}
		return out, nil
	})
}

// humanNumber достаёт число из строки: префикс-число + суффикс K/M/G/T/P
// (как sort -h). Не распарсилось число — 0.
func humanNumber(s string) float64 {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && (s[i] == '+' || s[i] == '-' || s[i] == '.' ||
		(s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	num, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0
	}
	mult := map[byte]float64{'k': 1e3, 'm': 1e6, 'g': 1e9, 't': 1e12, 'p': 1e15}
	if i < len(s) {
		if m, ok := mult[strings.ToLower(string(s[i]))[0]]; ok {
			return num * m
		}
	}
	return num
}
