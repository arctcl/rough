package engine

import (
	"encoding/json"
	"errors"
	"io/fs"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Theme — тема интерфейса: символы рамок/элементов и цвета.
// Лежит в папке themes/ рядом с плагинами. Формат:
//
//	{
//	  "name": "default",
//	  "symbols": { "tile_tl": "┌", "tile_tr": "┐", ... },
//	  "colors":  { "frame": "3", "title_fg": "#00ff00", ... }
//	}
//
// Цвет — это ССЫЛКА на палитру терминала (номер 0-255, который юзер сам
// перекрасил в настройках терминала) либо жёсткий hex (#rrggbb).
type Theme struct {
	Name    string            `json:"name"`
	Symbols map[string]string `json:"symbols"`
	Colors  map[string]string `json:"colors"`
}

// curTheme — активная тема движка (устанавливается при старте в Run).
var curTheme *Theme

// Sym возвращает символ по имени; если в теме нет — запасной.
func (t *Theme) Sym(name, fallback string) rune {
	if t != nil {
		if s, ok := t.Symbols[name]; ok && s != "" {
			for _, r := range s {
				return r
			}
		}
	}
	for _, r := range fallback {
		return r
	}
	return ' '
}

// themeColor возвращает цвет из темы по имени (или пустую строку).
func themeColor(name string) string {
	if curTheme == nil {
		return ""
	}
	return curTheme.Colors[name]
}

// ConfigTheme возвращает имя темы из tiles.json (ключ "theme", по умолчанию "default").
func ConfigTheme(fsys fs.FS) string {
	b, err := fs.ReadFile(fsys, "tiles.json")
	if err != nil {
		return "default"
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return "default"
	}
	if s, ok := raw["theme"].(string); ok && s != "" {
		return s
	}
	return "default"
}

// LoadTheme читает тему themes/<name>.json из вшитой папки.
// Если файла нет — пустая тема (запасные символы/цвета).
func LoadTheme(fsys fs.FS, name string) *Theme {
	b, err := fs.ReadFile(fsys, "themes/"+name+".json")
	if err != nil {
		return &Theme{Name: name}
	}
	t := &Theme{Name: name}
	if err := json.Unmarshal(b, t); err != nil {
		return &Theme{Name: name}
	}
	return t
}

// ResolveColor превращает цвет из HTML/темы в tcell.Color.
// Понимает: имя из темы ("frame"), hex ("#00ff00"),
// номер палитры терминала ("3" — терминал покажет свою перекрашенную третью),
// имя палитры tcell ("red").
func (t *Theme) ResolveColor(s string, def tcell.Color) tcell.Color {
	return t.resolveColor(s, def, 0)
}

func (t *Theme) resolveColor(s string, def tcell.Color, depth int) tcell.Color {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	// Имя из темы (с защитой от зацикливания).
	if t != nil && depth < 8 {
		if c, ok := t.Colors[s]; ok {
			return t.resolveColor(c, def, depth+1)
		}
	}
	// Hex: #rrggbb — жёсткий цвет, одинаковый у всех.
	if strings.HasPrefix(s, "#") {
		if c, err := parseHexColor(s); err == nil {
			return c
		}
	}
	// Число — ссылка на палитру терминала (юзер сам её перекрасил).
	if n, err := strconv.Atoi(s); err == nil {
		return tcell.PaletteColor(n)
	}
	// Имя из палитры tcell (red, green, ...).
	if c := tcell.GetColor(s); c != tcell.ColorDefault {
		return c
	}
	return def
}

// parseHexColor разбирает #rrggbb в truecolor.
func parseHexColor(s string) (tcell.Color, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return tcell.ColorDefault, errors.New("hex: нужен формат #rrggbb")
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return tcell.ColorDefault, err
	}
	return tcell.NewHexColor(int32(n)), nil
}
