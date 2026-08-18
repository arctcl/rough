package engine

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// statusMsg — последний результат действия/ошибка для нижней строки.
var statusMsg string

// debugLines — отладочный вывод плагина tobotom (рисуется в статус-блоке).
var debugLines []string

// SetDebug — плагин отладки (tobotom) показывает строки в статус-блоке.
func SetDebug(lines []string) { debugLines = lines }

// statusScroll — сдвиг прокрутки внутри статус-блока (если строк больше 3).
var statusScroll int

// statusShownAt — когда статус показан в последний раз (авто-скрытие через 10 с).
var statusShownAt time.Time

// statusRect — прямоугольник статус-блока на экране (для прокрутки колесом).
var statusRectX, statusRectY, statusRectW, statusRectH int

// showError показывает ошибку в всплывающем окошке. Многострочная трасса
// (паника плагина) рисуется построчно со скроллом — как отладочный вывод,
// чтобы стек был читаем, а интерфейс оставался рабочим.
func showError(err error) {
	msg := err.Error()
	if strings.Contains(msg, "\n") {
		debugLines = strings.Split(msg, "\n")
		statusMsg = ""
		return
	}
	statusMsg = "error: " + msg
}

// drawStatus рисует статус-сообщение в правом нижнем углу экрана.
// Блок: 30% ширины, прижат к правому краю с отступом 2 и к низу с отступом 2
// (над вкладками, если они есть). Высота — по содержимому: от 1 до 3 строк
// текста плюс рамка. Если строк больше 3 — прокрутка колесом мыши над блоком.
// Фон непрозрачный (status_bg), чтобы сквозь статус ничего не просвечивало.
func drawStatus(b *Buffer, w, h int) {
	if statusMsg == "" && len(debugLines) == 0 {
		statusRectH = 0
		return
	}
	const textRows = 3

	// Ширина блока: 30% экрана (минимум 20, чтобы не превращался в щель).
	bw := w * 30 / 100
	if bw < 20 {
		bw = 20
	}
	if bw > w-4 {
		bw = w - 4
	}
	innerW := bw - 2

	// Все строки содержимого: отладочный вывод или перенос статуса по ширине.
	var all []string
	if len(debugLines) > 0 {
		all = debugLines
	} else {
		all = wrapStatus(statusMsg, innerW)
	}
	// Окно до textRows строк со скроллом (если контента больше).
	maxOff := len(all) - textRows
	lines := all
	if maxOff > 0 {
		if statusScroll > maxOff {
			statusScroll = maxOff
		}
		if statusScroll < 0 {
			statusScroll = 0
		}
		lines = all[statusScroll : statusScroll+textRows]
	} else {
		statusScroll = 0
	}
	if len(lines) == 0 {
		lines = []string{""}
	}

	// Высота блока: рамка + строки (от 1 до 3).
	blockH := len(lines) + 2

	// Позиция: правый нижний угол, отступы 2 от края (и от вкладок, если они есть).
	x0 := w - 2 - bw
	if x0 < 0 {
		x0 = 0
	}
	bottom := h - 3 // вкладки на последней строке — блок выше них
	top := bottom - blockH + 1
	if top < 0 {
		top = 0
	}

	frameFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
	statusFg := curTheme.ResolveColor(themeColor("status_fg"), tcell.ColorYellow)
	// Фон статуса: ключ status_bg, фоллбэк — bg темы, затем чёрный.
	statusBg := curTheme.ResolveColor(themeColor("status_bg"),
		curTheme.ResolveColor(themeColor("bg"), tcell.ColorBlack))
	frame := Style{Fg: frameFg, Bg: statusBg}
	text := Style{Bold: true, Fg: statusFg, Bg: statusBg}

	// Заливаем весь блок фоном — чтобы сквозь статус не просвечивали тайлы.
	fill := Style{Bg: statusBg}
	for y := top; y <= bottom; y++ {
		for x := x0; x < x0+bw; x++ {
			b.Set(x, y, ' ', fill)
		}
	}

	// Символы рамки: status_* или запасные tile_* (углы/линии тайла).
	tl := curTheme.Sym("status_tl", string(curTheme.Sym("tile_tl", "┌")))
	tr := curTheme.Sym("status_tr", string(curTheme.Sym("tile_tr", "┐")))
	bl := curTheme.Sym("status_bl", string(curTheme.Sym("tile_bl", "└")))
	br := curTheme.Sym("status_br", string(curTheme.Sym("tile_br", "┘")))
	hh := curTheme.Sym("status_h", string(curTheme.Sym("tile_h", "─")))
	vv := curTheme.Sym("status_v", string(curTheme.Sym("tile_v", "│")))

	b.Set(x0, top, tl, frame)
	b.Set(x0+bw-1, top, tr, frame)
	b.Set(x0, bottom, bl, frame)
	b.Set(x0+bw-1, bottom, br, frame)
	for x := x0 + 1; x < x0+bw-1; x++ {
		b.Set(x, top, hh, frame)
		b.Set(x, bottom, hh, frame)
	}
	for y := top + 1; y < bottom; y++ {
		b.Set(x0, y, vv, frame)
		b.Set(x0+bw-1, y, vv, frame)
	}

	// Текст: строки, обрезанные по ширине блока.
	for i, ln := range lines {
		b.SetString(x0+1, top+1+i, truncateWidth(ln, innerW), text)
	}

	// Полоса прокрутки внутри статуса (если контента больше 3 строк).
	if maxOff > 0 {
		scFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
		sc := Style{Fg: scFg}
		sx := x0 + bw - 2 // перед правой рамкой
		thumbH := textRows * textRows / len(all)
		if thumbH < 1 {
			thumbH = 1
		}
		thumbY := statusScroll * (textRows - thumbH) / maxOff
		for i := 0; i < textRows; i++ {
			if i >= thumbY && i < thumbY+thumbH {
				b.Set(sx, top+1+i, '█', sc)
			} else {
				b.Set(sx, top+1+i, '│', sc)
			}
		}
	}

	// Прямоугольник блока — для прокрутки колесом мыши.
	statusRectX, statusRectY, statusRectW, statusRectH = x0, top, bw, blockH
}

// truncateWidth обрезает строку до ширины в клетках (по uniseg).
func truncateWidth(s string, width int) string {
	if width < 1 {
		return ""
	}
	var out []rune
	w := 0
	for _, r := range s {
		rw := uniseg.StringWidth(string(r))
		if w+rw > width {
			break
		}
		out = append(out, r)
		w += rw
	}
	return string(out)
}

// wrapStatus разбивает текст на строки по ширине (перенос по рунам,
// ширина считается через uniseg). Возвращает все строки; вызывающий берёт
// столько, сколько влезает в блок статуса.
func wrapStatus(text string, width int) []string {
	if width < 1 {
		return nil
	}
	var lines []string
	var cur []rune
	w := 0
	for _, r := range text {
		rw := uniseg.StringWidth(string(r))
		if w+rw > width {
			lines = append(lines, string(cur))
			cur = nil
			w = 0
		}
		cur = append(cur, r)
		w += rw
	}
	if len(cur) > 0 {
		lines = append(lines, string(cur))
	}
	return lines
}
