// Плагин clear — маркер склейки: в пайпе "clear && a && b" движок очищает
// приёмник, и последующие пайпы ДОБАВЛЯЮТ свой вывод (а не перезаписывают).
// Сам возвращает пустой вывод.
package clear

import "github.com/arctcl/rough"

func init() {
	rough.AddMan("clear", `clear — начать склейку с чистого листа.

Синтаксис: clear (без аргументов).

В связке "clear && пайп1 && пайп2" очищает блок вывода, после чего каждый
пайп ДОБАВЛЯЕТ свой результат к блоку (как "cat a b"), а не перезаписывает.

Пример:
  action="clear && man:ssh && cat:/etc/hosts"
  → сначала блок очищается, потом в него подряд складываются справка и файл.`)
	rough.AddPlugin("clear", func(in []string, args []string) ([]string, error) {
		return nil, nil
	})
}
