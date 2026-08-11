// Плагин chart — живой столбчатый график (rolling): берёт число из входа,
// добавляет столбик справа, старые сдвигаются влево. Столбики привязаны
// к низу и закрашены квадратиками. Рисует на всей доступной зоне: ширину
// берёт из окна (engine.Window), высоту — из атрибута height на <plugin>.
// Обновление — через interval (дефолт 2 с): каждый тик новый столбик.
// Вызов: ... | chart:МИН:МАКС[:ШИРИНА]
package chart

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/rivo/uniseg"

	"rough"
	"rough/engine"
)

// man_chart — справка по плагину (для man).
const man_chart = `chart — живой столбчатый график с осями.

Использование:
  часть пайпа: ... | chart:МИН:МАКС[:ШИРИНА[:СЕКУНД]]

Аргументы:
  МИН, МАКС — диапазон значений (например 0:100 для CPU в процентах).
  ШИРИНА    — ширина столбика в клетках (по умолчанию 1).
  СЕКУНД    — секунд на столбик (для подписи времени, по умолчанию 2).

Столбики привязаны к низу и левому краю, закрашены █ с полублоками ▄
для детализации, могут быть пустыми (значение = минимум). Снизу и слева —
тонкие оси с подписями: в перекрестии — минимум, сверху — максимум,
на нижней линии в разрыве — сколько столбиков и сколько это времени.
Фон области — ░. Высота зоны задаётся в HTML (height на <plugin>),
обновление — через interval (дефолт 2 с).

Примеры:
  <plugin pipe="emu_cpu | chart:0:100:1:2" height="14" interval="2s"/>
  <plugin pipe="emu_mem | chart:0:100:2:2" height="10" interval="2s"/>`

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
		secPer := 2
		if len(args) > 3 {
			if v, err := strconv.Atoi(args[3]); err == nil && v > 0 {
				secPer = v
			}
		}

		// Данные: последнее число из входа (текущее значение).
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
		plotH := h - 1 // строки 0..h-2, нижняя — ось X
		if plotH < 2 {
			plotH = 2
		}

		// Подписи диапазона слева от оси Y (по самой широкой).
		hiLabel := fmt.Sprintf("%g", hi)
		loLabel := fmt.Sprintf("%g", lo)
		labelW := uniseg.StringWidth(hiLabel)
		if lw := uniseg.StringWidth(loLabel); lw > labelW {
			labelW = lw
		}
		gx := labelW + 1 // колонка оси Y
		gw := w - gx - 1 // ширина области графика
		if gw < 1 {
			gw = 1
		}

		cols := gw / colW
		if cols < 1 {
			cols = 1
		}
		if len(s) > cols {
			s = s[len(s)-cols:] // сдвигаем старые столбики влево
		}
		series[key] = s

		// Собираем строки от верха к низу.
		out := make([]string, 0, h)
		for y := 0; y < h; y++ {
			row := make([]rune, w)
			for i := range row {
				row[i] = ' '
			}

			if y == h-1 {
				// Нижняя ось: подпись минимума, угол, линия.
				copy(row, []rune(loLabel))
				row[gx] = '└'
				for x := gx + 1; x < w; x++ {
					row[x] = '─'
				}
				// Подпись «N шт · Tс» прямо в разрыве нижней линии.
				bot := fmt.Sprintf("%d шт · %ds", len(s), len(s)*secPer)
				bw := uniseg.StringWidth(bot)
				bx := (w - bw) / 2
				if bx < gx+1 {
					bx = gx + 1
				}
				if bx+bw <= w {
					copy(row[bx:], []rune(bot))
				}
				out = append(out, string(row))
				continue
			}

			// Ось Y, подпись максимума наверху.
			row[gx] = '│'
			if y == 0 {
				copy(row[:labelW], []rune(hiLabel))
			}
			// Фон области из символов ░.
			for x := gx + 1; x < w; x++ {
				row[x] = '░'
			}
			// Столбики: привязаны к низу, могут быть пустыми.
			// Самый свежий — слева, старые уходят вправо.
			for i, val := range s {
				x := len(s) - 1 - i
				col := gx + 1 + x*colW
				if col >= w {
					break
				}
				norm := (val - lo) / (hi - lo)
				if norm < 0 {
					norm = 0
				}
				if norm > 1 {
					norm = 1
				}
				// Высота в полупикселях: полный блок █ = 2, нижний полублок ▄ = 1.
				hPix := int(norm * float64(plotH) * 2)
				if hPix <= 0 {
					continue // столбик не закрашен (значение = минимум)
				}
				full := hPix / 2
				half := hPix%2 == 1
				for c := 0; c < colW && col+c < w; c++ {
					if y >= plotH-full && y <= plotH-1 {
						row[col+c] = '█'
					} else if half && y == plotH-full-1 {
						row[col+c] = '▄'
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
