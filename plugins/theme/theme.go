// Плагин theme — смена темы интерфейса на лету. Темы — файлы themes/*.json
// в проекте. Переключение сразу перерисовывает весь экран (движок читает
// активную тему при каждом кадре).
package theme

import (
	"strings"

	"rough"
	"rough/engine"
)

// man_theme — справка по плагину (для man).
const man_theme = `theme — сменить тему интерфейса на лету.

Использование:
  theme            — показать текущую тему
  theme:list       — список доступных тем (файлы themes/*.json)
  theme:ИМЯ        — переключиться на тему ИМЯ

Темы лежат в папке themes/ проекта (themes/<имя>.json): символы рамок/кнопок
и цвета (bg, fg, frame, title_fg, status_*…). Плагинам доступны color_0..color_15
(палитра терминала: 0 — чёрный, 1 — красный, 2 — зелёный, … 15 — белый) и любые
ключи темы — движковый engine.ThemeColor.

Примеры:
  action="theme:list"      — список тем
  action="theme:terminal"  — переключиться на тему terminal (цвета терминала)`

func init() {
	rough.AddMan("theme", man_theme)
	rough.AddPlugin("theme", func(in []string, args []string) ([]string, error) {
		// Без аргументов — просто показать текущую тему.
		if len(args) == 0 || args[0] == "" {
			return []string{"тема: " + engine.CurrentThemeName()}, nil
		}
		switch args[0] {
		case "list":
			return engine.ListThemes(), nil
		default:
			// Может прийти "имя с пробелом" — берём только первый токен.
			name := strings.Fields(args[0])[0]
			if err := engine.SwitchTheme(name); err != nil {
				return nil, err
			}
			return []string{"тема → " + engine.CurrentThemeName()}, nil
		}
	})
}
