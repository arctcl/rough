// Плагин chart — живой столбчатый график (rolling): берёт число из входа,
// добавляет столбик справа, старые уходят влево. Ось Y — справа, верхняя
// линия с названием графика, нижняя — с подписью «N шт · время». Высота
// зоны задаётся в HTML (height на <plugin>), обновление — через interval.
// Вызов: ... | chart:МИН:МАКС[:ШИРИНА[:СЕКУНД[:ЗАГОЛОВОК]]]
package chart

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/uniseg"

	"rough"
	"rough/engine"
)

// chartParams — параметры плагина (гибрид: позиционные двоеточиями + --флаги).
// Порядок = позиции; мин/макс обязательные, остальные с дефолтами.
var chartParams = []engine.Param{
	{Name: "мин", Required: true},  // диапазон снизу
	{Name: "макс", Required: true}, // диапазон сверху
	{Name: "ширина", Default: "1"}, // ширина столбика в клетках
	{Name: "секунд", Default: "2"}, // секунд на столбик (подпись времени)
	{Name: "заголовок"},            // название графика (необязательный)
}

// man_chart — справка по плагину (для man). Строки использования генерируются
// из chartParams через engine.ParamsUsage — обе формы входа (двоеточия и
// --флаги) всегда в синхроне с кодом.
var man_chart = "chart — живой столбчатый график (rolling).\n\n" +
	"Использование (гибрид: двоеточия по порядку или --флаги, можно микс):\n" +
	"  ... | " + engine.ParamsUsage("chart", chartParams) + "\n\n" +
	"Параметры:\n" +
	"  МИН, МАКС — диапазон значений, обязательные (например 0:100 для CPU).\n" +
	"  ШИРИНА    — ширина столбика в клетках (по умолчанию 1).\n" +
	"  СЕКУНД    — секунд на столбик (для подписи времени, по умолчанию 2).\n" +
	"  ЗАГОЛОВОК — название графика (в разрыве верхней линии).\n\n" +
	"Пустой слот \":\" — параметр берётся из флага или дефолта. Частичный ввод\n" +
	"работает: chart:0:100 — остальное уходит в дефолты. Последний параметр\n" +
	"глотает остаток двоеточий. Микс: chart::1:2:CPU --мин=0 --макс=100.\n\n" +
	"Новый столбик появляется справа (у оси), старые уходят влево. Ось Y — справа:\n" +
	"сверху — максимум, в перекрестии снизу — минимум; на нижней линии в разрыве —\n" +
	"сколько столбиков и сколько это времени. Сверху — линия с названием графика.\n" +
	"Высота зоны задаётся в HTML (height на <plugin>), обновление — через interval.\n\n" +
	"Примеры:\n" +
	"  <plugin pipe=\"emu_cpu | chart:0:100:1:2:CPU\" height=\"14\" interval=\"2s\"/>\n" +
	"  <plugin pipe=\"emu_mem | chart:0:100:1:2:MEM\" height=\"14\" interval=\"2s\"/>\n" +
	"  <plugin pipe=\"emu_cpu | chart --мин=0 --макс=100 --заголовок=CPU\" height=\"14\" interval=\"2s\"/>"

// series — история значений по названию графика (серии).
var series = map[string][]float64{}

// lastAdd — время последнего добавления столбика по серии (не чаще СЕКУНД).
var lastAdd = map[string]time.Time{}

func init() {
	rough.AddMan("chart", man_chart)
	rough.AddPlugin("chart", func(in []string, args []string) ([]string, error) {
		// Гибридный разбор параметров: двоеточия по порядку или --флаги (и микс).
		vals, err := engine.ParseArgs(args, chartParams)
		if err != nil {
			return nil, err
		}
		lo, err1 := strconv.ParseFloat(vals["мин"], 64)
		hi, err2 := strconv.ParseFloat(vals["макс"], 64)
		if err1 != nil || err2 != nil || hi <= lo {
			return nil, errors.New("chart: нужен диапазон мин<макс")
		}
		colW := 1
		if v, err := strconv.Atoi(vals["ширина"]); err == nil && v > 0 {
			colW = v
		}
		secPer := 2
		if v, err := strconv.Atoi(vals["секунд"]); err == nil && v > 0 {
			secPer = v
		}
		// Название графика — в разрыве верхней линии (остаток двоеточий глотает).
		title := vals["заголовок"]

		// Значение: последнее число из входа (текущая точка).
		nums := chartNumbers(in)
		if len(nums) == 0 {
			nums = []float64{lo}
		}
		v := nums[len(nums)-1]

		// Серия по названию графика (стабильный ключ), иначе — сигнатура движка.
		skey := engine.PluginKey()
		if title != "" {
			skey = title
		}
		if t, ok := lastAdd[skey]; !ok || time.Since(t) >= time.Duration(secPer)*time.Second {
			series[skey] = append(series[skey], v)
			lastAdd[skey] = time.Now()
		}
		s := series[skey]

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
		gw := gx             // ширина области графика
		if gw < 1 {
			gw = 1
		}

		cols := gw / colW
		if cols < 1 {
			cols = 1
		}
		if len(s) > cols {
			s = s[len(s)-cols:] // старые уходят влево
		}
		series[skey] = s

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
				out = append(out, string(row))
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
				out = append(out, string(row))
				continue
			}

			// Область: ось Y справа, столбики (без фона).
			row[gx] = '│'
			rowY := y - 1
			for i, val := range s {
				col := i * colW
				if col >= gw {
					break
				}
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
			// Столбики красим цветом из темы (color_2), рамку/ось — нет.
			out = append(out, paintBars(string(row)))
		}
		return out, nil
	})
}

// paintBars обрамляет столбики (█/▄) маркерами цвета из темы: \x01{color_2}.
// Красятся ТОЛЬКО сами столбики — рамка, ось и подписи остаются в своём цвете
// (движковый putColored разбирает маркеры, они не влияют на ширину).
func paintBars(row string) string {
	runes := []rune(row)
	var sb strings.Builder
	i := 0
	for i < len(runes) {
		if runes[i] == '█' || runes[i] == '▄' {
			sb.WriteString("\x01{color_2}")
			for i < len(runes) && (runes[i] == '█' || runes[i] == '▄') {
				sb.WriteRune(runes[i])
				i++
			}
			sb.WriteString("\x01{}")
			continue
		}
		sb.WriteRune(runes[i])
		i++
	}
	return sb.String()
}

// chartNumbers — все числа из строк (первое в строке — значение).
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
