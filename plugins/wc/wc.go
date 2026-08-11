// Плагин wc — подсчёт строк входа (как wc -l).
// Вызов: ... | wc
package wc

import (
	"strconv"

	"rough"
)

// man_wc — справка по плагину (для man).
const man_wc = `wc — подсчёт строк входа (как wc -l).

Использование:
  часть пайпа: ... | wc

Примеры:
  action="cat:/var/log/x.log | wc"          — сколько строк в логе
  action="curl:https://example.com | wc"    — сколько строк в ответе`

func init() {
	rough.AddMan("wc", man_wc)
	rough.AddPlugin("wc", func(in []string, args []string) ([]string, error) {
		return []string{strconv.Itoa(len(in))}, nil
	})
}
