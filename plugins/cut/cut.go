// Плагин cut — вырезать поля из строки (как cut -d -f).
// Юникс-лайк: `cut -d'Д' -fN` — здесь единые quick-параметры:
// ... | cut --разделитель=Д --поля=N   или   ... | cut::N (пробел по умолчанию).
// Работает с текстом в пайпе: числа/поля для bars и chart вытаскиваются здесь,
// а не внутри рисовалок (см. принцип «в одном плагине — одно дело»).
package cut

import (
	"errors"
	"strconv"
	"strings"

	"rough"
	"rough/engine"
)

// man_cut — справка по плагину (для man).
const man_cut = `cut — вырезать поля из строки (как cut -d -f).

Использование (единые quick-параметры):
  часть пайпа: ... | cut --разделитель=Д --поля=N
               ... | cut::N        — разделитель по умолчанию (пробел)

Аргументы:
  разделитель — чем делить строку на поля (по умолчанию пробел).
  поля        — номер поля (1-е = 1), диапазон N-M или список через запятую (1,3).

Примеры:
  action="cat:app.conf | cut --разделитель== --поля=2"    — значения ключей (key=value)
  action="cat:load.log | cut::2 | bars"                     — 2-е поле по пробелу → график
  action="cat:x.log | cut --поля=2-4"                       — поля со 2-го по 4-е`

// cutParams — параметры cut. Порядок = позиции: разделитель, поля.
var cutParams = []engine.Param{
	{Name: "разделитель", Default: " "},
	{Name: "поля"},
}

func init() {
	rough.AddMan("cut", man_cut)
	rough.AddPlugin("cut", func(in []string, args []string) ([]string, error) {
		vals, err := engine.ParseArgs(args, cutParams)
		if err != nil {
			return nil, err
		}
		sep := vals["разделитель"]
		fields := vals["поля"]
		if fields == "" {
			return nil, errors.New("cut: нужны поля (--поля=N или N-M)")
		}
		var out []string
		for _, ln := range in {
			// Разделитель по умолчанию (пробел) — схлопываем повторные пробелы,
			// как в awk (strings.Fields). Иначе — честный Split по разделителю.
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

// pickFields собирает нужные поля по спецификации "N", "N-M" или "N,M".
// Поля нумеруются с 1 (как в cut); несуществующие — пропускаются.
func pickFields(parts []string, spec string) string {
	var keep []string
	for _, s := range strings.Split(spec, ",") {
		s = strings.TrimSpace(s)
		if i := strings.Index(s, "-"); i >= 0 {
			lo, _ := strconv.Atoi(s[:i])
			hi, _ := strconv.Atoi(s[i+1:])
			for n := lo; n <= hi; n++ {
				if n >= 1 && n <= len(parts) {
					keep = append(keep, parts[n-1])
				}
			}
			continue
		}
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(parts) {
			keep = append(keep, parts[n-1])
		}
	}
	return strings.Join(keep, " ")
}
