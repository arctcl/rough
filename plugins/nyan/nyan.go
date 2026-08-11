// Плагин nyan — анимация нян-кота.
//
// Движок перерисовывает тайлы по таймеру (~500 мс) и каждый раз вызывает плагин
// заново. Плагин считает кадры и двигает кота по ширине окна (engine.Window()),
// оставляя позади радужный хвост. Адаптируется к размеру тайла.
package nyan

import (
	"strings"

	"rough"
	"rough/engine"
)

// frame — счётчик кадров анимации (глобал плагина).
var frame int

// man_nyan — справка по плагину (для man).
const man_nyan = `nyan — анимация нян-кота, едет по тайлу и оставляет радужный хвост.

Использование (живой тайл, перерисовывается по таймеру):
  <plugin name="nyan" interval="0.5s"/>

Атрибуты тега <plugin>:
  name="nyan"          — имя плагина (обязательно)
  interval="0.5s"      — период перерисовки (таймер движка)

Размер: подстраивается под ширину тайла через engine.Window().

Примеры:
  <plugin name="nyan" interval="0.5s"/>          — стандартный бег
  <plugin name="nyan" interval="1s"/>           — медленный кот`

func init() {
	rough.AddMan("nyan", man_nyan)

	rough.AddPlugin("nyan", func(in []string, args []string) ([]string, error) {
		w, h := engine.Window()
		frame++
		return nyanFrame(w, h, frame), nil
	})
}

// catRows — нян-кот, все строки одной ширины (catW), чтобы ехал ровно.
const catW = 10

var catRows = []string{
	",--------,",
	"|  /\\_/\\  ",
	"| ( o.o )  ",
	"|   > ^ <  ",
	"`--------`",
}

// nyanFrame собирает кадр: радужный хвост слева, кот едет вправо.
func nyanFrame(w, h, frame int) []string {
	if w < catW+2 {
		return []string{"нян"}
	}
	// позиция кота: едем слева направо, потом зацикливаемся
	x := frame % (w - catW + 1)

	rows := catRows
	if len(rows) > h {
		rows = rows[:h]
	}

	out := make([]string, 0, len(rows))
	for _, r := range rows {
		var sb strings.Builder
		// хвост позади кота: ~-~-~-~
		for i := 0; i < x; i++ {
			if i%2 == 0 {
				sb.WriteByte('~')
			} else {
				sb.WriteByte('-')
			}
		}
		sb.WriteString(r)
		out = append(out, sb.String())
	}
	return out
}
