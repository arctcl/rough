// Плагин sed — замена текста в строках (как sed 's/from/to/').
// Юникс-лайк: `sed 's/from/to/'` — здесь единые quick-параметры:
// ... | sed:FROM:TO. Значение с двоеточием — в кавычках: sed:':':1
// (заменить ":" на "1"). Работает с текстом в пайпе — обработка перед bars/chart.
package sed

import (
	"errors"
	"strings"

	"rough"
	"rough/engine"
)

// man_sed — справка по плагину (для man).
const man_sed = `sed — замена текста в строках (как sed 's/from/to/').

Использование (единые quick-параметры):
  часть пайпа: ... | sed:FROM:TO

Аргументы:
  FROM — что заменить (пусто — ошибка).
  TO   — на что заменить (пусто — удалить).
  Если FROM или TO содержит ":" — оберни в кавычки: sed:':':1.

Примеры:
  action="cat:x | sed:error:ERROR"              — заменить error на ERROR
  action="cat:x | sed:':':1"                    — заменить ":" на "1"
  action="cat:x | sed:log:: | wc"               — удалить подстроку log`

// sedParams — параметры sed. Порядок = позиции: from, to.
var sedParams = []engine.Param{
	{Name: "from"},
	{Name: "to"},
}

func init() {
	rough.AddMan("sed", man_sed)
	rough.AddPlugin("sed", func(in []string, args []string) ([]string, error) {
		vals, err := engine.ParseArgs(args, sedParams)
		if err != nil {
			return nil, err
		}
		what := vals["from"]
		to := vals["to"]
		if what == "" {
			return nil, errors.New("sed: нужно что заменить (--from= или sed:from:to)")
		}
		var out []string
		for _, ln := range in {
			out = append(out, strings.ReplaceAll(ln, what, to))
		}
		return out, nil
	})
}
