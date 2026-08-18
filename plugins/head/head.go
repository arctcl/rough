// Плагин head — первые N строк входа (как head -n N).
// Вызов: ... | head[:N]  (по умолчанию 10)
package head

import (
	"strconv"

	"github.com/arctcl/rough"
)

// man_head — справка по плагину (для man).
const man_head = `head — первые строки входа (как head -n N).

Использование:
  часть пайпа: ... | head[:N]

Аргументы:
  N — сколько строк оставить (по умолчанию 10).

Примеры:
  action="cat:/etc/os-release | head:3"
  action="curl:https://example.com | head:5"    — первые 5 строк ответа`

func init() {
	rough.AddMan("head", man_head)
	rough.AddPlugin("head", func(in []string, args []string) ([]string, error) {
		n := 10
		if len(args) > 0 {
			if v, err := strconv.Atoi(args[0]); err == nil {
				n = v
			}
		}
		if n <= 0 || n > len(in) {
			return in, nil
		}
		return in[:n], nil
	})
}
