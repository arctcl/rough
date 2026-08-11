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
		w, h := engine.Window()
		if len(vals) == 0 {
			return []string{"нет чисел"}, nil
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
		rows := len(vals)
		if rows > h-1 {
			rows = h - 1
		}
		out := make([]string, 0, rows)
		for i := 0; i < rows; i++ {
			barW := int(vals[i] / max * float64(w-1))
			if barW > w-1 {
				barW = w - 1
			}
			out = append(out, strings.Repeat("█", barW)+" "+strconv.FormatFloat(vals[i], 'f', 1, 64))
		}
		return out, nil
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
