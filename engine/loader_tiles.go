package engine

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// Tile — один тайл на странице (позиция и размеры в % или px).
type Tile struct {
	ID, X, Y, W, H, File string
}

// Pages — карта роутов в список тайлов.
type Pages map[string][]Tile

// defaultPattern — порядок полей в массиве тайла, если паттерн не задан.
var defaultPattern = []string{"id", "x", "y", "w", "h", "file"}

// LoadPages читает tiles.json из вшитой папки интерфейса (fs.FS).
// Пути внутри fs.FS — относительные: "tiles.json", "tiles/time.html".
// Формат: сначала паттерн (один раз), потом строки данных — без повторов ключей.
func LoadPages(fsys fs.FS) (Pages, error) {
	b, err := fs.ReadFile(fsys, "tiles.json")
	if err != nil {
		return nil, fmt.Errorf("tiles.json: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}

	pattern := defaultPattern
	pages := Pages{}
	for key, val := range raw {
		// Ключ паттерна объявляет схему строк данных один раз.
		if key == "pattern" {
			if arr, ok := val.([]any); ok {
				p := make([]string, 0, len(arr))
				for _, v := range arr {
					if s, ok := v.(string); ok {
						p = append(p, s)
					}
				}
				if len(p) > 0 {
					pattern = p
				}
			}
			continue
		}
		// Ключ "theme" — имя темы, это не страница.
		if key == "theme" {
			continue
		}
		// Ключ "menu" — вкладки (пары [имя, роут]), это не страница.
		if key == "menu" {
			continue
		}

		items, ok := val.([]any)
		if !ok {
			continue
		}
		var tiles []Tile
		for _, it := range items {
			arr, ok := it.([]any)
			if !ok {
				continue
			}
			t := Tile{}
			for i, v := range arr {
				if i >= len(pattern) {
					break
				}
				s := fmt.Sprintf("%v", v)
				switch pattern[i] {
				case "id":
					t.ID = s
				case "x":
					t.X = s
				case "y":
					t.Y = s
				case "w":
					t.W = s
				case "h":
					t.H = s
				case "file":
					t.File = s
				}
			}
			tiles = append(tiles, t)
		}
		pages[key] = tiles
	}
	return pages, nil
}

// FirstRoute возвращает первый роут (для старта, если /main нет).
func (p Pages) FirstRoute() string {
	for r := range p {
		return r
	}
	return ""
}

// LoadMenu читает вкладки "menu" из tiles.json: список пар [имя, роут].
// Вкладки рисуются внизу интерфейса, переключение — клик/Tab/цифры.
func LoadMenu(fsys fs.FS) [][]string {
	b, err := fs.ReadFile(fsys, "tiles.json")
	if err != nil {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil
	}
	arr, ok := raw["menu"].([]any)
	if !ok {
		return nil
	}
	var menu [][]string
	for _, it := range arr {
		pair, ok := it.([]any)
		if !ok || len(pair) < 2 {
			continue
		}
		name, _ := pair[0].(string)
		route, _ := pair[1].(string)
		if name != "" && route != "" {
			menu = append(menu, []string{name, route})
		}
	}
	return menu
}
