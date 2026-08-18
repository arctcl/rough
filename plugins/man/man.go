// Плагин man — справка по плагинам (юникс-like help/man).
// action="man" — список всех плагинов с коротким описанием;
// action="man:ИМЯ" — полная справка по конкретному плагину.
// Описания берутся из реестра справок: каждый плагин кладёт туда свою
// переменную man_<имя> через rough.AddMan в init().
package man

import (
	"fmt"
	"strings"

	"github.com/arctcl/rough"
)

// man_man — справка по самому плагину man.
const man_man = `man — справка по плагинам (аналог man в Linux).

Использование:
  action="man"          — список всех плагинов с коротким описанием
  action="man:ИМЯ"      — полная справка по конкретному плагину

Примеры:
  action="man"                      — что вообще есть
  action="man:ssh"                  — справка по ssh
  action="man:curl | head:10"       — справка по curl, первые 10 строк
  action="man:cat | grep:пайп"      — где в cat описаны пайпы`

func init() {
	rough.AddMan("man", man_man)

	rough.AddPlugin("man", func(in []string, args []string) ([]string, error) {
		// Без аргументов — список всех плагинов с первой строкой справки.
		if len(args) == 0 {
			var lines []string
			for _, name := range rough.ManNames() {
				desc, _ := rough.ManText(name)
				first := strings.SplitN(desc, "\n", 2)[0]
				lines = append(lines, first)
			}
			if len(lines) == 0 {
				return []string{"нет справок"}, nil
			}
			return lines, nil
		}

		// С аргументом — полная справка по плагину.
		text, ok := rough.ManText(args[0])
		if !ok {
			return nil, fmt.Errorf("man: нет справки у %s", args[0])
		}
		return strings.Split(strings.TrimRight(text, "\n"), "\n"), nil
	})
}
