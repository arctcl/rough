// Плагин awk — обработка текста как упрощённый awk.
// Юникс-лайк: `awk -F'Д' '/пат/{print $N}'` — здесь единые quick-параметры:
// ... | awk --sep=Д --fields=N --filter=REGEX.
// Умеет: фильтровать строки по регулярке и вырезать поля. Для чисел/полей
// перед bars и chart — отдельный плагин (не рисует сам, только обрабатывает).
package awk

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/arctcl/rough"
	"github.com/arctcl/rough/engine"
)

// man_awk — справка по плагину (для man).
const man_awk = `awk — обработка текста (упрощённый awk).

Использование (единые quick-параметры):
  часть пайпа: ... | awk --sep=Д --fields=N [--filter=REGEX]

Аргументы:
  sep    — чем делить строку на поля (по умолчанию пробел).
  fields — номер поля (1-е = 1), диапазон N-M или список (1,3). Пусто — вся строка.
  filter — регулярка: оставить только строки, где она встречается (как awk '/пат/').

Примеры:
  action="cat:log | awk --filter=ERROR"                  — только строки с ERROR
  action="cat:log | awk --filter=ERROR --fields=3"       — 3-е поле строк ERROR
  action="cat:cpu.log | awk --fields=2 | bars"           — 2-е поле → график
  action="cat:app.conf | awk --sep== --fields=2"           — значения ключей конфига`

// awkParams — параметры awk. Порядок = позиции: sep, fields, filter.
var awkParams = []engine.Param{
	{Name: "sep", Default: " "},
	{Name: "fields"},
	{Name: "filter"},
}

func init() {
	rough.AddMan("awk", man_awk)
	rough.AddPlugin("awk", func(in []string, args []string) ([]string, error) {
		vals, err := engine.ParseArgs(args, awkParams)
		if err != nil {
			return nil, err
		}
		sep := vals["sep"]
		fields := vals["fields"]
		filter := vals["filter"]
		var re *regexp.Regexp
		if filter != "" {
			re, err = regexp.Compile(filter)
			if err != nil {
				return nil, err
			}
		}
		var out []string
		for _, ln := range in {
			// Фильтр: строка, где нет регулярки, пропускается.
			if re != nil && !re.MatchString(ln) {
				continue
			}
			// Поля не заданы — оставляем строку как есть.
			if fields == "" {
				out = append(out, ln)
				continue
			}
			var parts []string
			if sep == " " {
				parts = strings.Fields(ln)
			} else {
				parts = strings.Split(ln, sep)
			}
			out = append(out, pickFields(parts, fields))
		}
		return out, nil
	})
}

// pickFields собирает нужные поля по спецификации "N", "N-M" или "N,M"
// (поля нумеруются с 1, несуществующие пропускаются).
func pickFields(parts []string, spec string) string {
	var keep []string
	for _, s := range strings.Split(spec, ",") {
		s = strings.TrimSpace(s)
		// Диапазон N-M.
		if lo, hi, ok := rangeSpec(s); ok {
			for n := lo; n <= hi; n++ {
				if n >= 1 && n <= len(parts) {
					keep = append(keep, parts[n-1])
				}
			}
			continue
		}
		// Один номер.
		if n, ok := fieldNum(s); ok && n >= 1 && n <= len(parts) {
			keep = append(keep, parts[n-1])
		}
	}
	return strings.Join(keep, " ")
}

// rangeSpec разбирает "N-M" на числа (ok=false, если не диапазон).
func rangeSpec(s string) (lo, hi int, ok bool) {
	i := strings.Index(s, "-")
	if i < 0 {
		return 0, 0, false
	}
	a, err1 := strconv.Atoi(s[:i])
	b, err2 := strconv.Atoi(s[i+1:])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return a, b, true
}

// fieldNum разбирает одно число поля.
func fieldNum(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil
}
