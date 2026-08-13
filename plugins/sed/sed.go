// Плагин sed — замена текста в строках (как sed 's/что/на_что/').
// Юникс-лайк: `sed 's/что/на_что/'` — здесь единые quick-параметры:
// ... | sed:ЧТО:НА_ЧТО. Значение с двоеточием — в кавычках: sed:':':1
// (заменить ":" на "1"). Работает с текстом в пайпе — обработка перед bars/chart.
package sed

import (
	"errors"
	"strings"

	"rough"
	"rough/engine"
)

// man_sed — справка по плагину (для man).
const man_sed = `sed — замена текста в строках (как sed 's/что/на_что/').

Использование (единые quick-параметры):
  часть пайпа: ... | sed:ЧТО:НА_ЧТО

Аргументы:
  ЧТО   — что заменить (пусто — ошибка).
  НА_ЧТО — на что заменить (пусто — удалить).
  Если ЧТО или НА_ЧТО содержит ":" — оберни в кавычки: sed:':':1.

Примеры:
  action="cat:x | sed:error:ERROR"              — заменить error на ERROR
  action="cat:x | sed:':':1"                    — заменить ":" на "1"
  action="cat:x | sed:log:: | wc"               — удалить подстроку log`

// sedParams — параметры sed. Порядок = позиции: что, на что.
var sedParams = []engine.Param{
	{Name: "что"},
	{Name: "на"},
}

func init() {
	rough.AddMan("sed", man_sed)
	rough.AddPlugin("sed", func(in []string, args []string) ([]string, error) {
		vals, err := engine.ParseArgs(args, sedParams)
		if err != nil {
			return nil, err
		}
		what := vals["что"]
		to := vals["на"]
		if what == "" {
			return nil, errors.New("sed: нужно что заменить (--что= или sed:что:на)")
		}
		var out []string
		for _, ln := range in {
			out = append(out, strings.ReplaceAll(ln, what, to))
		}
		return out, nil
	})
}
