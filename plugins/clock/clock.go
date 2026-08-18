// Плагин clock — текущие дата и время (живые часы для тайла по таймеру).
// Вызов: <plugin name="clock" interval="1s"/>
package clock

import (
	"time"

	"github.com/arctcl/rough"
)

// man_clock — справка по плагину (для man).
const man_clock = `clock — текущие дата и время (живые часы).

Использование:
  <plugin name="clock" interval="1s"/>

Примеры:
  <plugin name="clock" interval="1s"/>          — часы в тайле
  <div width="50%"><plugin name="clock" interval="1s"/></div>  — часы в колонке`

func init() {
	rough.AddMan("clock", man_clock)
	rough.AddPlugin("clock", func(in []string, args []string) ([]string, error) {
		now := time.Now()
		return []string{now.Format("02.01.2006"), now.Format("15:04:05")}, nil
	})
}
