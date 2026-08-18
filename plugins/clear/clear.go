// Плагин clear — маркер склейки: в пайпе "clear && a && b" движок очищает
// приёмник, и последующие пайпы ДОБАВЛЯЮТ свой вывод (а не перезаписывают).
// Сам возвращает пустой вывод.
package clear

import "github.com/arctcl/rough"

func init() {
	rough.AddMan("clear", `clear — start gluing from a clean slate.

Usage: clear (no arguments).

In a chain "clear && pipe1 && pipe2" it clears the output block, then every
pipe ADDS its result to the block (like "cat a b") instead of overwriting.

Example:
  action="clear && man:ssh && cat:/etc/hosts"
  → first the block is cleared, then the help and the file are stacked in it.`)
	rough.AddPlugin("clear", func(in []string, args []string) ([]string, error) {
		return nil, nil
	})
}
