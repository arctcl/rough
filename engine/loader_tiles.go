package engine

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// Tile — один тайл на странице (позиция и размеры в % или px).
// json-теги нужны, чтобы страницы можно было задавать в конфиге (json).
type Tile struct {
	ID   string `json:"id"`
	X    string `json:"x"`
	Y    string `json:"y"`
	W    string `json:"w"`
	H    string `json:"h"`
	File string `json:"file"`
}

// Pages — карта роутов в список тайлов.
type Pages map[string][]Tile

// extraPages — страницы, зарегистрированные программно: живут не в tiles.json
// и добавляются в общий список страниц при старте Run.
var extraPages = Pages{}

// AddPage регистрирует страницу (роут → тайлы) программно: она не трогает
// tiles.json и добавляется в общий список страниц при старте Run.
func AddPage(route string, tiles []Tile) {
	if route == "" || len(tiles) == 0 {
		return
	}
	extraPages[route] = tiles
}

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

	// Паттерн читаем ДО цикла по страницам: порядок обхода map в Go случайный.
	// Если бы pattern разбирался внутри цикла, применение к страницам зависело
	// бы от того, попадётся ли ключ раньше них (недетерминизм между запусками).
	pattern := defaultPattern
	if arr, ok := raw["pattern"].([]any); ok {
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

	pages := Pages{}
	for key, val := range raw {
		// Служебные ключи (паттерн/тема/меню) — не страницы.
		if key == "pattern" || key == "theme" || key == "menu" {
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
