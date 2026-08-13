// Плагин export — сохранить вывод в переменную движка (для $подстановки в action).
// Юникс-лайк: `export ИМЯ=значение` — здесь: ... | export:имя.
// Значение хранит движок (engine.SetVar), дальше подставляется через $имя.
// export пропускает строки дальше по пайпу (как tee), чтобы удобно цеплять.
package export

import (
	"errors"

	"rough"
	"rough/engine"
)

// man_export — справка по плагину (для man).
const man_export = `export — сохранить вывод в переменную движка.

Использование:
  часть пайпа: ... | export:ИМЯ

Аргументы:
  ИМЯ — имя переменной. Дальше в любом action её можно подставить через $ИМЯ
        (или ${ИМЯ}, если нужно отделить от соседних символов).

Примеры:
  action="ssh:root:srv1::hostname | export:host"    — запомнить хост
  action="ssh:root:$host::uptime"                   — подставить переменную
  action="cat:app.conf | cut::2 | export:val | bars" — и сохранить, и показать`

func init() {
	rough.AddMan("export", man_export)
	rough.AddPlugin("export", func(in []string, args []string) ([]string, error) {
		if len(args) == 0 || args[0] == "" {
			return nil, errors.New("export: нужно имя переменной")
		}
		engine.SetVar(args[0], in)
		return in, nil // пропускаем дальше (как tee в bash)
	})
}
