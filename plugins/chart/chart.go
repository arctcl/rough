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
	"strings"

	"github.com/rivo/uniseg"

	"rough"
	"rough/engine"
)

// man_chart — справка по плагину (для man).
const man_chart = `chart — живой график: обычные столбики (bars) или японские свечи (japanse).

Использование:
  часть пайпа: ... | chart:МИН:МАКС[:ВИД[:ШИРИНА[:СЕКУНД[:ЗАГОЛОВОК]]]]

Аргументы:
  МИН, МАКС  — диапазон значений (например 0:100 для CPU в процентах).
  ВИД        — bars (столбики, по умолчанию) или japanse (японские свечи).
  ШИРИНА     — ширина столбика/свечи в клетках (по умолчанию 1).
  СЕКУНД     — секунд на столбик (для подписи времени, по умолчанию 2).
  ЗАГОЛОВОК  — название графика (рисуется в разрыве верхней линии).

Новый столбик/свеча появляется справа (у оси), старые уходят влево. Ось Y —
справа: сверху — максимум, в перекрестии снизу — минимум; на нижней линии
в разрыве — сколько столбиков и сколько это времени. Сверху — линия с
названием графика. Высота зоны задаётся в HTML (height на <plugin>),
обновление — через interval.

Для japanse вход должен давать OHLC: open high low close (4 числа строкой).
Для непрерывности свечей open должен быть равен close предыдущей (гепы
исчезнут). Тело — от open до close (полублоки ▄/▀ для границ), фитиль │ —
от high до low. Бычья (close>=open) — зелёная, медвежья — красная.

Примеры:
  <plugin pipe="emu_cpu | chart:0:100:bars:1:2:CPU" height="14" interval="2s"/>
  <plugin pipe="emu_candle | chart:0:100:japanse:1:2:ETH" height="14" interval="2s"/>`

// series — история точек по сигнатуре плагина (серии): точка = []float64
// (1 число для bars, 4 числа OHLC для japanse).
var series = map[string][][]float64{}

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
		// Режим: bars (по умолчанию) или japanse.
		kind := "bars"
		ai := 2
		if len(args) > 2 && (args[2] == "bars" || args[2] == "japanse") {
			kind = args[2]
			ai = 3
		}
		colW := 1
		if len(args) > ai {
			if v, err := strconv.Atoi(args[ai]); err == nil && v > 0 {
				colW = v
			}
		}
		secPer := 2
		if len(args) > ai+1 {
			if v, err := strconv.Atoi(args[ai+1]); err == nil && v > 0 {
				secPer = v
			}
		}
		// Название графика — в разрыве верхней линии (остаток аргументов).
		title := ""
		if len(args) > ai+2 {
			title = strings.Join(args[ai+2:], ":")
		}

		// Точка данных: одно число (bars) или OHLC (japanse).
		var pt []float64
		if kind == "japanse" {
			nums := lastLineNumbers(in)
			if len(nums) >= 4 {
				pt = nums[:4] // open high low close
			} else if len(nums) > 0 {
				c := nums[len(nums)-1]
				pt = []float64{c, c, c, c}
			} else {
				pt = []float64{lo, lo, lo, lo}
			}
		} else {
			nums := chartNumbers(in)
			if len(nums) == 0 {
				nums = []float64{lo}
			}
			pt = []float64{nums[len(nums)-1]}
		}

		// История: новый столбик/свеча добавляется, старые уходят влево.
		key := engine.PluginKey()
		series[key] = append(series[key], pt)
		s := series[key]

		w, h := engine.Window()
		// Область графика — между верхней и нижней линиями.
		plotH := h - 2
		if plotH < 2 {
			plotH = 2
		}

		// Подписи диапазона справа от оси Y (по самой широкой).
		hiLabel := fmt.Sprintf("%g", hi)
		loLabel := fmt.Sprintf("%g", lo)
		labelW := uniseg.StringWidth(hiLabel)
		if lw := uniseg.StringWidth(loLabel); lw > labelW {
			labelW = lw
		}
		gx := w - labelW - 1 // колонка оси Y (справа)
		gw := gx             // ширина области графика (слева от оси)
		if gw < 1 {
			gw = 1
		}

		cols := gw / colW
		if cols < 1 {
			cols = 1
		}
		if len(s) > cols {
			s = s[len(s)-cols:]
		}
		series[key] = s

		// Цвета свечей по колонкам (бычья G / медвежья R).
		var colColor []byte
		if kind == "japanse" {
			colColor = make([]byte, w)
			for i, p := range s {
				if len(p) < 4 {
					continue // не OHLC (например мусор из другой серии)
				}
				col := i * colW
				if col >= gw {
					break
				}
				c := byte('G')
				if p[3] < p[0] {
					c = 'R'
				}
				for ccx := 0; ccx < colW && col+ccx < w; ccx++ {
					colColor[col+ccx] = c
				}
			}
		}

		// Собираем строки от верха к низу.
		out := make([]string, 0, h)
		for y := 0; y < h; y++ {
			row := make([]rune, w)
			for i := range row {
				row[i] = ' '
			}

			if y == 0 {
				// Верхняя линия с названием графика в разрыве.
				for x := 0; x < gw; x++ {
					row[x] = '─'
				}
				if title != "" {
					tw := uniseg.StringWidth(title)
					tx := (gw - tw) / 2
					if tx < 0 {
						tx = 0
					}
					if tx+tw <= gw {
						copy(row[tx:], []rune(title))
					}
				}
				row[gx] = '┐'
				if gx+1+labelW <= w {
					copy(row[gx+1:], []rune(hiLabel))
				}
				out = append(out, renderRow(row, colColor))
				continue
			}

			if y == h-1 {
				// Нижняя ось: линия, угол, подпись минимума.
				for x := 0; x <= gx; x++ {
					row[x] = '─'
				}
				row[gx] = '┘'
				if gx+1+labelW <= w {
					copy(row[gx+1:], []rune(loLabel))
				}
				// Подпись «N шт · Tс» в разрыве нижней линии.
				bot := fmt.Sprintf("%d шт · %ds", len(s), len(s)*secPer)
				bw := uniseg.StringWidth(bot)
				bx := (gw - bw) / 2
				if bx < 0 {
					bx = 0
				}
				if bx+bw <= gw {
					copy(row[bx:], []rune(bot))
				}
				out = append(out, renderRow(row, colColor))
				continue
			}

			// Область: ось Y справа, столбики/свечи (без фона).
			row[gx] = '│'
			rowY := y - 1
			if kind == "japanse" {
				for i, p := range s {
					if len(p) < 4 {
						continue
					}
					col := i * colW
					if col >= gw {
						break
					}
					drawCandle(row, col, colW, rowY, plotH, lo, hi, p)
				}
			} else {
				for i, p := range s {
					if len(p) < 1 {
						continue
					}
					col := i * colW
					if col >= gw {
						break
					}
					val := p[0]
					norm := (val - lo) / (hi - lo)
					if norm < 0 {
						norm = 0
					}
					if norm > 1 {
						norm = 1
					}
					// Высота в полупикселях: █ = 2, ▄ = 1.
					hPix := int(norm * float64(plotH) * 2)
					if hPix <= 0 {
						continue // столбик не закрашен (значение = минимум)
					}
					full := hPix / 2
					half := hPix%2 == 1
					for c := 0; c < colW && col+c < gw; c++ {
						if rowY >= plotH-full && rowY <= plotH-1 {
							row[col+c] = '█'
						} else if half && rowY == plotH-full-1 {
							row[col+c] = '▄'
						}
					}
				}
			}
			out = append(out, renderRow(row, colColor))
		}
		return out, nil
	})
}

// pIn — полупиксель p внутри диапазона [lo, hi].
func pIn(p, lo, hi int) bool { return p >= lo && p <= hi }

// posOf — позиция значения в полупикселях ОТ ВЕРХА (0..plotH*2-1).
func posOf(v, lo, hi float64, plotH int) int {
	norm := (v - lo) / (hi - lo)
	if norm < 0 {
		norm = 0
	}
	if norm > 1 {
		norm = 1
	}
	ph := int(norm * float64(plotH) * 2)
	p := plotH*2 - 1 - ph
	if p < 0 {
		p = 0
	}
	if p > plotH*2-1 {
		p = plotH*2 - 1
	}
	return p
}

// drawCandle рисует японскую свечу (OHLC) в колонке col шириной colW.
// Строка y (0..plotH-1), полупиксели: верх клетки = y*2, низ = y*2+1.
// Тело — от open до close (полублоки ▄/▀ для границ), фитиль │ — от high до low.
// Цвет задаётся отдельно через colColor (бычья G / медвежья R).
func drawCandle(row []rune, col, colW, y, plotH int, lo, hi float64, p []float64) {
	o, hh, ll, cc := p[0], p[1], p[2], p[3]
	posHigh := posOf(hh, lo, hi, plotH)
	posLow := posOf(ll, lo, hi, plotH)
	posOpen := posOf(o, lo, hi, plotH)
	posClose := posOf(cc, lo, hi, plotH)
	bodyLo := min(posOpen, posClose)
	bodyHi := max(posOpen, posClose)
	for ccx := 0; ccx < colW && col+ccx < len(row); ccx++ {
		up := y * 2
		dn := y*2 + 1
		switch {
		case pIn(up, bodyLo, bodyHi) && pIn(dn, bodyLo, bodyHi):
			row[col+ccx] = '█'
		case pIn(dn, bodyLo, bodyHi):
			row[col+ccx] = '▄'
		case pIn(up, bodyLo, bodyHi):
			row[col+ccx] = '▀'
		case pIn(up, posHigh, posLow) || pIn(dn, posHigh, posLow):
			row[col+ccx] = '│'
		}
	}
}

// renderRow собирает строку, вставляя цветовые маркеры свечей (colColor).
// Маркер \x01G/\x01R ставится перед каждым символом свечи — движок красит.
func renderRow(row []rune, colColor []byte) string {
	if colColor == nil {
		return string(row)
	}
	var sb strings.Builder
	for x, r := range row {
		if x < len(colColor) && colColor[x] != 0 {
			sb.WriteByte('\x01')
			sb.WriteByte(colColor[x])
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// lastLineNumbers — все числа из последней строки входа (для OHLC свечей).
func lastLineNumbers(lines []string) []float64 {
	if len(lines) == 0 {
		return nil
	}
	return chartNumbers([]string{lines[len(lines)-1]})
}

// chartNumbers извлекает ВСЕ числа из строк (для bars — первое, для OHLC — все).
func chartNumbers(lines []string) []float64 {
	re := regexp.MustCompile(`\d+(?:\.\d+)?`)
	var out []float64
	for _, ln := range lines {
		for _, m := range re.FindAllString(ln, -1) {
			if f, err := strconv.ParseFloat(m, 64); err == nil {
				out = append(out, f)
			}
		}
	}
	return out
}
