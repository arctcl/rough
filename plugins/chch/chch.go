// Плагин chch — инжектор «секретных» кодов. Сам по себе ничего не рисует:
// только читает конфиг chch.json из вшитой папки /rough и для каждого секретного
// кода регистрирует в движке действие (AddCheat): набрал код → выполнится пайп.
//
// Плагин полностью опционален: удалишь импорт chch — движок работает как
// обычно, просто секретных кодов нет.
//
// Формат chch.json (стиль как у тем: сверху заголовок-описание, внутри плоский
// список кодов, без вложенных заголовков):
//
//	{
//	  "title": "chch — секретные коды",
//	  "description": "Введи код на клавиатуре — выполнится действие.",
//	  "cheats": {
//	    "UUDDLRLRba": "cat GLHF mate!",
//	    "ps+": "ps --track=1"
//	  }
//	}
package chch

import (
	"encoding/json"
	"io/fs"

	"github.com/arctcl/rough"
)

// chchConf — конфиг инжектора: заголовок-описание и плоский список кодов.
type chchConf struct {
	Title       string            `json:"title"`       // заголовок (справочно)
	Description string            `json:"description"` // описание (справочно)
	Cheats      map[string]string `json:"cheats"`      // секретный код → действие (пайп)
}

func init() {
	// Конфиг читаем после старта движка (в init() вшитой папки ещё нет).
	// OnReady — общий хук движка для плагинов-инжекторов.
	rough.OnReady(load)
}

// load читает chch.json из вшитой папки /rough и регистрирует секретные коды
// как действия. Нет файла или ошибка — плагин молча бездействует (движок
// работает как обычно).
func load(fsys fs.FS) {
	b, err := fs.ReadFile(fsys, "chch.json")
	if err != nil {
		return
	}
	var c chchConf
	if err := json.Unmarshal(b, &c); err != nil {
		return
	}
	// Для каждого кода — действие (пайп), как обычная кнопка action.
	for seq, action := range c.Cheats {
		rough.AddCheat(seq, action)
	}
}
