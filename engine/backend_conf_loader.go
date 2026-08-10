package engine

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// Tile — один тайл на странице (позиция и размеры в % или px).
type Tile struct {
	ID, X, Y, W, H, File string
}

// Pages — карта роутов в список тайлов.
type Pages map[string][]Tile

// defaultPattern — порядок полей в массиве тайла, если паттерн не задан.
var defaultPattern = []string{"id", "x", "y", "w", "h", "файл"}

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
		if key == "паттерн" || key == "pattern" {
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
				case "файл", "file":
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

// Rect считает координаты тайла в клетках терминала w×h.
func (t Tile) Rect(w, h int) (x, y, tw, th int) {
	x = parseLen(t.X, w)
	y = parseLen(t.Y, h)
	tw = parseLen(t.W, w)
	th = parseLen(t.H, h)
	return
}

// parseLen переводит "10%" / "20" / "50vw" / "30vh" в клетки.
// % и vw — от ширины (total), vh — от высоты (тоже total, зависит от вызова).
func parseLen(s string, total int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.HasSuffix(s, "%") {
		return int(parseFloat(s[:len(s)-1]) * float64(total) / 100)
	}
	if strings.HasSuffix(s, "vw") || strings.HasSuffix(s, "vh") {
		return int(parseFloat(s[:len(s)-2]) * float64(total) / 100)
	}
	return int(parseFloat(s))
}

// parseFloat — безопасный разбор числа (при ошибке — 0).
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
