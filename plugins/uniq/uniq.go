// Плагин uniq — убрать повторяющиеся подряд строки (как uniq).
// Юникс-лайк: `uniq`, `uniq -c` — здесь единые quick-параметры:
// ... | uniq [--count=1]. Строки сравниваются по соседству (как в uniq):
// сортировку uniq НЕ делает — для «всех дублей» сначала sort.
package uniq

import (
	"fmt"

	"rough"
	"rough/engine"
)

// man_uniq — справка по плагину (для man).
const man_uniq = `uniq — убрать повторяющиеся подряд строки (как uniq).

Использование (единые quick-параметры):
  часть пайпа: ... | uniq [--count=1]

Аргументы:
  count — 1 — перед строкой число повторов (как uniq -c).

Примеры:
  action="cat:log | uniq"                  — убрать соседние дубли
  action="cat:log | sort | uniq"           — все дубли подряд → убрать
  action="cat:log | sort | uniq --count=1" — уникальные + сколько раз`

// uniqParams — параметры uniq. Порядок = позиции: count.
var uniqParams = []engine.Param{
	{Name: "count", Default: "0"},
}

func init() {
	rough.AddMan("uniq", man_uniq)
	rough.AddPlugin("uniq", func(in []string, args []string) ([]string, error) {
		vals, err := engine.ParseArgs(args, uniqParams)
		if err != nil {
			return nil, err
		}
		withCount := vals["count"] == "1"

		var out []string
		for i, ln := range in {
			// Подряд идущий дубль — пропускаем.
			if i > 0 && ln == in[i-1] {
				continue
			}
			if !withCount {
				out = append(out, ln)
				continue
			}
			// Сколько раз строка идёт подряд (как uniq -c).
			n := 1
			for j := i + 1; j < len(in) && in[j] == ln; j++ {
				n++
			}
			out = append(out, fmt.Sprintf("%d %s", n, ln))
		}
		return out, nil
	})
}
