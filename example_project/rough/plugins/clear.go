// Демо-плагин clear — маркер склейки: в пайпе "clear && a && b" движок
// очищает приёмник, и последующие пайпы добавляют вывод (а не перезаписывают).
// Сам возвращает пустой вывод.
package plugins

import "github.com/arctcl/rough"

func init() {
	rough.AddPlugin("clear", func(in []string, args []string) ([]string, error) {
		return nil, nil
	})
}
