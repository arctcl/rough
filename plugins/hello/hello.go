// Плагин hello — приветствие. Просто пример: строки → строки.
package hello

import (
	"strings"

	"rough"
)

func init() {
	rough.AddPlugin("hello", func(in []string, args []string) ([]string, error) {
		who := "мир"
		if len(args) > 0 && args[0] != "" {
			who = strings.Join(args, ":")
		}
		return []string{"привет, " + who + "!"}, nil
	})
}
