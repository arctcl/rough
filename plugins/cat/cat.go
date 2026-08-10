// Плагин cat — показать файл. Работает с реальной системой (os.ReadFile),
// потому что конфиги/файлы живут на диске, а не во вшитой папке.
package cat

import (
	"errors"
	"os"
	"strings"

	"rough"
)

func init() {
	rough.AddPlugin("cat", func(in []string, args []string) ([]string, error) {
		if len(args) < 1 {
			return nil, errors.New("cat: нужен файл")
		}
		b, err := os.ReadFile(args[0])
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			return nil, nil
		}
		return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), nil
	})
}
