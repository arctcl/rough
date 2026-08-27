// Плагин chch — инжектор «секретных» страниц. Сам по себе ничего не рисует:
// читает конфиг chch.json из вшитой папки /rough, регистрирует страницы
// (AddPage) и для каждого секретного кода — переход на страницу (AddCheatRoute):
// набрал код → движок открывает страницу.
//
// Секретные страницы задаются ЗДЕСЬ, в chch.json (а не в tiles.json): движок
// не трогаем «скрытием» страниц — инжектор сам добавляет их в общий список
// при старте. Тайлы — обычные, со своим html в tiles/.
//
// Плагин полностью опционален: удалишь импорт chch — движок работает как
// обычно, просто секретных страниц нет.
//
// Формат chch.json (стиль как у тем: сверху заголовок-описание, внутри плоские
// списки страниц и кодов, без вложенных заголовков):
//
//	{
//	  "title": "chch — секретные страницы",
//	  "description": "Введи код на клавиатуре — откроется скрытая страница.",
//	  "pages": {
//	    "/ps": [
//	      { "id": "ps", "x": "0%", "y": "0%", "w": "100%", "h": "100%", "file": "tiles/ps_page.html" }
//	    ]
//	  },
//	  "cheats": {
//	    "ps+": "/ps",
//	    "UUDDLRLRba": "/konami"
//	  }
//	}
package chch

import (
	"encoding/json"
	"io/fs"

	"github.com/arctcl/rough"
)

// chchConf — конфиг инжектора: заголовок-описание, секретные страницы и коды.
type chchConf struct {
	Title       string                  `json:"title"`       // заголовок (справочно)
	Description string                  `json:"description"` // описание (справочно)
	Pages       map[string][]rough.Tile `json:"pages"`       // секретные страницы (роут → тайлы)
	Cheats      map[string]string       `json:"cheats"`      // секретный код → роут страницы
}

func init() {
	// Конфиг читаем после старта движка (в init() вшитой папки ещё нет).
	// OnReady — общий хук движка для плагинов-инжекторов.
	rough.OnReady(load)
}

// load читает chch.json из вшитой папки /rough, регистрирует секретные
// страницы (AddPage) и переходы на них (AddCheatRoute). Нет файла или ошибка —
// плагин молча бездействует (движок работает как обычно).
func load(fsys fs.FS) {
	b, err := fs.ReadFile(fsys, "chch.json")
	if err != nil {
		return
	}
	var c chchConf
	if err := json.Unmarshal(b, &c); err != nil {
		return
	}
	// Секретные страницы — обычные тайлы, но из конфига инжектора, не из tiles.json.
	for route, tiles := range c.Pages {
		rough.AddPage(route, tiles)
	}
	// Для каждого кода — переход на секретную страницу (навигация, как по вкладке).
	for seq, route := range c.Cheats {
		rough.AddCheatRoute(seq, route)
	}
}
