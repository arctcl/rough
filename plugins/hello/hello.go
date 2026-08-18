// Плагин hello — приветствие. Просто пример: строки → строки.
package hello

import (
	"strings"

	"github.com/arctcl/rough"
)

// man_hello — справка по плагину (для man).
const man_hello = `hello — поздороваться.

Использование:
  action="hello[:ИМЯ]"

Аргументы:
  ИМЯ — к кому обратиться (необязательно).

Примеры:
  action="hello"                      — привет, мир!
  action="hello:админ"                — привет, админ!
  action="cat:/etc/hostname | hello"  — привет, имя_хоста!
  action="man:hello | head:3"         — первые 3 строки этой справки`

func init() {
	rough.AddMan("hello", man_hello)

	rough.AddPlugin("hello", func(in []string, args []string) ([]string, error) {
		who := "мир"
		if len(args) > 0 && args[0] != "" {
			who = strings.Join(args, ":")
		}
		return []string{"привет, " + who + "!"}, nil
	})
}
