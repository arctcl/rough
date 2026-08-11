// Плагин tail — последние N строк входа (как tail -n N).
// Вызов: ... | tail[:N]  (по умолчанию 10)
package tail

import (
	"strconv"

	"rough"
)

// man_tail — справка по плагину (для man).
const man_tail = `tail — последние строки входа (как tail -n N).

Использование:
  часть пайпа: ... | tail[:N]

Аргументы:
  N — сколько строк оставить (по умолчанию 10).

Примеры:
  action="cat:/var/log/app.log | tail:20"              — последние 20 строк лога
  action="ssh:root@srv:journalctl -u api | tail:50"    — последние 50 строк журнала`

func init() {
	rough.AddMan("tail", man_tail)
	rough.AddPlugin("tail", func(in []string, args []string) ([]string, error) {
		n := 10
		if len(args) > 0 {
			if v, err := strconv.Atoi(args[0]); err == nil {
				n = v
			}
		}
		if n <= 0 {
			return nil, nil
		}
		if len(in) <= n {
			return in, nil
		}
		return in[len(in)-n:], nil
	})
}
