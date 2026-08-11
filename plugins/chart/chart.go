// Плагин chart — живой столбчатый график (rolling): берёт число из входа,
// добавляет столбик справа, старые сдвигаются влево. Столбики привязаны
// к низу и закрашены квадратиками. Рисует на всей доступной зоне: ширину
// берёт из окна (engine.Window), высоту — из атрибута height на <plugin>.
// Обновление — через interval (дефолт 2 с): каждый тик новый столбик.
// Вызов: ... | chart:МИН:МАКС[:ШИРИНА]
package chart

import (
	"errors"
	"regexp"
	"strconv"

	"rough"
	"rough/engine"
)

// man_chart — справка по плагину (для man).
const man_chart = `chart — живой столбчатый график (столбики привязаны к низу).

Использование:
  часть пайпа: ... | chart:МИН:МАКС[:ШИРИНА]

Аргументы:
  МИН, МАКС — диапазон значений (например 0:100 для CPU в процентах).
  ШИРИНА    — ширина столбика в клетках (по умолчанию 1).

Высота зоны графика задаётся в HTML атрибутом height на <plugin>,
обновление — через interval (дефолт 2 с). Каждый тик добавляется новый
столбик справа, старые сдвигаются влево.

Примеры:
  <plugin pipe="emu_cpu | chart:0:100" height="12" interval="2s"/>
  <plugin pipe="emu_mem | chart:0:100:2" height="10" interval="2s"/>`

// series — история значений по сигнатуре плагина (серии): карта → срез чисел.
var series = map[string][]float64{}

func init() {
	rough.AddMan("chart", man_chart)
	rough.AddPlugin("chart", func(in []string, args []string) ([]string, error) {
		if len(args) < 2 {
			return nil, errors.New("chart: нужны мин и макс")
		}
		lo, err1 := strconv.ParseFloat(args[0], 64)
		hi, err2 := strconv.ParseFloat(args[1], 64)
		if err1 != nil || err2 != nil || hi <= lo {
			return nil, errors.New("chart: нужен диапазон мин<макс")
		}
		colW := 1
		if len(args) > 2 {
			if v, err := strconv.Atoi(args[2]); err == nil && v > 0 {
				colW = v
			}
		}

		// Данные: берём последнее число из входа (текущее значение).
		nums := chartNumbers(in)
		if len(nums) == 0 {
			nums = []float64{lo}
		}
		v := nums[len(nums)-1]

		// История по сигнатуре плагина: новый столбик справа.
		key := engine.PluginKey()
		s := series[key]
		s = append(s, v)

		w, h := engine.Window()
		cols := w / colW
		if cols < 1 {
			cols = 1
		}
		// Держим историю не длиннее числа колонок — сдвигаем влево.
		if len(s) > cols {
			s = s[len(s)-cols:]
		}
		series[key] = s

		// Рисуем h строк от верха к низу. База — нижняя строка (y=h-1).
		base := h - 1
		if base < 1 {
			base = 1
		}
		out := make([]string, 0, h)
		for y := 0; y < h; y++ {
			row := make([]rune, w)
			for i := range row {
				row[i] = ' '
			}
			for x, val := range s {
				col := x * colW
				norm := (val - lo) / (hi - lo)
				if norm < 0 {
					norm = 0
				}
				if norm > 1 {
					norm = 1
				}
				colH := int(norm * float64(base))
				for c := 0; c < colW && col+c < w; c++ {
					if y >= base-colH {
						row[col+c] = '█'
					}
				}
			}
			out = append(out, string(row))
		}
		return out, nil
	})
}

// chartNumbers извлекает числа из строк — первое число в каждой строке.
func chartNumbers(lines []string) []float64 {
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
