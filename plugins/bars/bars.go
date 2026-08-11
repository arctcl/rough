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
  часть пайпа: ... | bars[:МАСКА]

Аргументы:
  МАСКА — регулярка с группой-числом (берётся первая группа).
          Без маски — число из всей строки.

Ширина подстраивается под размер тайла/колонки (engine.Window()):
поменял 50% на 40% — график сам стал уже.

Примеры:
  <plugin pipe="file:/tmp/cpu.log | bars:cpu=(\d+)" interval="1s"/>
  action="ssh:root@srv:cat /proc/loadavg | bars"`

func init() {
	rough.AddMan("bars", man_bars)
	rough.AddPlugin("bars", func(in []string, args []string) ([]string, error) {
		// Числа: с маской — через регулярку (группа), без маски — первое число в строке.
		var vals []float64
		if mask := strings.Join(args, ":"); mask != "" {
			vals = engine.ApplyMask(in, mask)
		} else {
			vals = numbersIn(in)
		}
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
		return []string{string(line)}, nil
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
