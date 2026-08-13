// Плагин bars — полосковый график из чисел (спарклайн).
// Числа берутся из входа через маску (или напрямую из строки).
// Ширина — реальный размер окна через engine.Window(), поэтому bars сам
// сжимается/растягивается при изменении ширины тайла или колонки.
// Вызов: ... | bars[:МАСКА]
package bars

import (
	"regexp"
	"strconv"
	"strings"

	"rough"
	"rough/engine"
)

// man_bars — справка по плагину (для man).
const man_bars = `bars — полосковый график из чисел (спарклайн).

Использование:
  часть пайпа: ... | bars[:ЦВЕТ]

Аргументы:
  ЦВЕТ — имя ключа темы (color_5 и т.п.), опционально, в конце.
         Нет аргумента — цвет из темы по умолчанию (color_2).

Числа bars берёт сам (первое число из каждой строки). Для вытаскивания
чисел из текста — отдельные плагины-обработчики (cut, awk, sed) перед bars.

Ширина подстраивается под размер тайла/колонки (engine.Window()):
поменял 50% на 40% — график сам стал уже.

Примеры:
  <plugin pipe="emu_cpu | bars" interval="1s"/>
  action="cat:/proc/loadavg | bars"
  action="cat:load.log | cut::2 | bars:color_3"`

func init() {
	rough.AddMan("bars", man_bars)
	rough.AddPlugin("bars", func(in []string, args []string) ([]string, error) {
		// Числа: первое число из каждой строки (numbersIn). Обработку текста
		// не делаем — для этого отдельные плагины (cut/awk/sed) перед bars.
		vals := numbersIn(in)
		if len(vals) == 0 {
			return []string{"нет чисел"}, nil
		}
		w, _ := engine.Window()
		// Спарклайн: одна строка из блоков ▁▂▃▄▅▆▇█ по ширине окна.
		cols := w - 1
		if cols < 1 {
			cols = 1
		}
		max := 0.0
		for _, v := range vals {
			if v > max {
				max = v
			}
		}
		if max <= 0 {
			max = 1
		}
		levels := "▁▂▃▄▅▆▇█"
		line := make([]rune, cols)
		for x := 0; x < cols; x++ {
			lo := x * len(vals) / cols
			hi := (x + 1) * len(vals) / cols
			if hi <= lo {
				hi = lo + 1
			}
			if hi > len(vals) {
				hi = len(vals)
			}
			avg := 0.0
			for i := lo; i < hi; i++ {
				avg += vals[i]
			}
			avg /= float64(hi - lo)
			lvl := int(avg / max * 7)
			if lvl < 0 {
				lvl = 0
			}
			if lvl > 7 {
				lvl = 7
			}
			line[x] = rune(levels[lvl])
		}
		// Цвет — опциональный аргумент в конце (имя ключа темы: color_5, ...).
		// Нет аргумента — цвет из темы по умолчанию (color_2).
		col := "color_2"
		if len(args) > 0 {
			if c := strings.TrimSpace(args[len(args)-1]); c != "" {
				col = c
			}
		}
		// Движковый putColored разбирает маркер, маркер не ломает ширину.
		return []string{"\x01{" + col + "}" + string(line) + "\x01{}"}, nil
	})
}

// numbersIn извлекает числа из строк без маски — первое число в каждой строке.
func numbersIn(lines []string) []float64 {
	re := regexp.MustCompile(`\d+(?:\.\d+)?`)
	var out []float64
	for _, ln := range lines {
		if m := re.FindString(ln); m != "" {
			if f, err := strconv.ParseFloat(m, 64); err == nil {
				out = append(out, f)
			}
		}
	}
	return out
}
