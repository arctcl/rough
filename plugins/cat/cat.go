// Плагин cat — показать файл. Работает с реальной системой (os.ReadFile),
// потому что конфиги/файлы живут на диске, а не во вшитой папке.
package cat

import (
	"errors"
	"os"
	"strings"

	"rough"
)

// man_cat — справка по плагину (для man). Лежит внутри плагина, как и требует контракт.
const man_cat = `cat — показать содержимое файла.

Использование:
  action="cat:ФАЙЛ"

Аргументы:
  ФАЙЛ — путь к файлу на реальной файловой системе.

Примеры:
  action="cat:/etc/hostname"
  action="cat:/var/log/app.log | tail:20"     — последние 20 строк лога
  action="cat:/etc/x.conf | grep:^server"     — строки, начинающиеся с server
  action="cat:/etc/os-release | head:3"       — первые 3 строки`

func init() {
	rough.AddMan("cat", man_cat)

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
