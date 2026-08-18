// Плагин append — дописать строку в конец файла (как echo "..." >> file).
// Вызов: action="append:ФАЙЛ:строка"
package append

import (
	"errors"
	"os"
	"strings"

	"github.com/arctcl/rough"
)

// man_append — справка по плагину (для man).
const man_append = `append — дописать строку в конец файла (как echo >> file).

Использование:
  action="append:ФАЙЛ:строка"

Аргументы:
  ФАЙЛ   — путь к файлу.
  строка — что дописать (остаток, склеивается через «:»).

Примеры:
  action="append:/etc/app.conf:debug=1"
  action="append:/var/log/notes.txt:важная заметка"`

func init() {
	rough.AddMan("append", man_append)
	rough.AddPlugin("append", func(in []string, args []string) ([]string, error) {
		if len(args) < 2 {
			return nil, errors.New("append: нужен файл и строка")
		}
		text := strings.Join(args[1:], ":")
		f, err := os.OpenFile(args[0], os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if _, err := f.WriteString(text + "\n"); err != nil {
			return nil, err
		}
		return []string{"добавлено: " + text}, nil
	})
}
