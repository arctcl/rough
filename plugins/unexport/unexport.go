// Плагин unexport — удалить переменную движка (антипод export).
// Юникс-лайк: export ИМЯ=... / unset ИМЯ — здесь: ... | unexport:имя.
// Значение хранит движок (engine.DelVar), после этого $имя больше не
// подставляется (пустая строка).
// unexport пропускает строки дальше по пайпу (как export/tee), чтобы удобно
// цеплять в середине конвейера.
package unexport

import (
	"errors"

	"github.com/arctcl/rough"
	"github.com/arctcl/rough/engine"
)

// man_unexport — справка по плагину (для man).
const man_unexport = `unexport — удалить переменную движка (антипод export).

Использование:
  часть пайпа: ... | unexport:ИМЯ

Аргументы:
  ИМЯ — имя переменной. После удаления $ИМЯ больше нигде не подставляется.

Примеры:
  ... | unexport:tmp     — выкинуть временную переменную
  action="export:count | ... | unexport:count" — записать, использовать, удалить`

func init() {
	rough.AddMan("unexport", man_unexport)
	rough.AddPlugin("unexport", func(in []string, args []string) ([]string, error) {
		if len(args) == 0 || args[0] == "" {
			return nil, errors.New("unexport: нужно имя переменной")
		}
		engine.DelVar(args[0])
		return in, nil // пропускаем дальше (как tee в bash)
	})
}
