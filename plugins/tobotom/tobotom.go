// Плагин tobotom — отладка пайпа: вывести текущий вывод в строку состояния (вниз).
// Можно вставить в любое место пайпа, чтобы увидеть, что реально пришло на вход.
//   ... | tobotom:pass   — показать вывод в статус-блоке и продолжить пайп
//   ... | tobotom:stop   — показать вывод в статус-блоке и остановить пайп
package tobotom

import (
	"errors"

	"rough"
	"rough/engine"
)

// man_tobotom — справка по плагину (для man).
const man_tobotom = `tobotom — отладка: вывести вывод в строку состояния (вниз).
Вставь в любое место пайпа — увидишь, что реально приходит на вход.

Использование:
  часть пайпа: ... | tobotom:pass   — вывести в статус и продолжить пайп
                ... | tobotom:stop   — вывести в статус и остановить пайп

Аргументы:
  pass — показать текущий вывод в статус-блоке, передать данные дальше.
  stop — показать текущий вывод в статус-блоке, дальше пайп не работает.

Примеры:
  action="cat:x | tobotom:pass | grep:y"      — что выдал cat, перед grep
  action="emu_log | grep:ERROR | tobotom:pass | tail:3"  — проверить grep
  action="cat:x | tobotom:stop"               — что выдал cat, и всё`

func init() {
	rough.AddMan("tobotom", man_tobotom)
	rough.AddPlugin("tobotom", func(in []string, args []string) ([]string, error) {
		if len(args) == 0 {
			return nil, errors.New("tobotom: нужен pass или stop")
		}
		// Показываем текущий вывод в статус-блоке (последние 3 строки — как tail).
		shown := in
		if len(shown) == 0 {
			shown = []string{"(пусто)"}
		}
		engine.SetDebug(shown)

		switch args[0] {
		case "pass":
			// Продолжаем пайп с теми же данными.
			return in, nil
		case "stop":
			// Останавливаем пайп — движок видит ErrStop и не идёт дальше.
			return nil, engine.ErrStop
		}
		return nil, errors.New("tobotom: нужен pass или stop")
	})
}
