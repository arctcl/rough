package engine

import (
	"io/fs"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// scrollOff — вертикальный скролл внутри тайла (id тайла → смещение строк).
var scrollOff = map[string]int{}

// maxTileHeight — жёсткий потолок высоты запасного буфера тайла (F5):
// чтобы длинный вывод был достижим, но память не раздувалась бесконечно.
const maxTileHeight = 4000

// renderFrame рисует текущий кадр: фон, заголовок, тайлы, вкладки и статус.
// Тайлы растягиваются по своим Rect под текущий размер окна (ресайз терминала
// подхватывается каждый кадр — это и есть «растягиватель»).
func renderFrame(s tcell.Screen, pages Pages, route string, menu [][]string, w, h int, fsys fs.FS) {
	hotzones = hotzones[:0]

	// Авто-скрытие статус-блока через 10 секунд после последнего действия.
	if (statusMsg != "" || len(debugLines) > 0) && time.Since(statusShownAt) > 10*time.Second {
		statusMsg = ""
		debugLines = nil
		statusScroll = 0
	}

	bg := NewBuffer(w, h)
	// Фон экрана и цвет текста по умолчанию — из темы (ключи bg/fg).
	bg.Fill(' ', Style{
		Fg: curTheme.ResolveColor(themeColor("fg"), tcell.ColorDefault),
		Bg: curTheme.ResolveColor(themeColor("bg"), tcell.ColorDefault),
	})

	// Шапка с текущим роутом и подсказкой выхода (цвета из темы).
	hdrFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	hdrBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	bg.SetString(1, 0, " rough: "+route+"    [q] quit ", Style{Bg: hdrBg, Fg: hdrFg})

	for _, t := range pages[route] {
		x, y, tw, th := t.Rect(w, h)
		if tw <= 0 || th <= 0 {
			continue
		}
		drawFrame(bg, x, y, tw, th)

		inner := NewBuffer(tw-2, th-2)

		// HTML тайла распарсен один раз (кэш) — здесь только рендер из кэша.
		if t.File != "" {
			if root, perr := loadTile(fsys, t.File); perr == nil {
				renderTile(root, inner, route+"/"+t.ID, x+1, y+1, th-2, &hotzones)
			}
		}
		bg.Copy(inner, x+1, y+1)

		// Название тайла — НА верхней рамке (поверх линии), как в модалках.
		titleFg := curTheme.ResolveColor(themeColor("title_fg"), tcell.ColorGreen)
		hdrBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
		title := " " + t.ID + " "
		if x+1+uniseg.StringWidth(title) <= x+tw-1 {
			bg.SetString(x+1, y, title, Style{Bold: true, Fg: titleFg, Bg: hdrBg})
		}
	}

	// Статус — внизу, обведён рамкой из темы (поверх тайлов). Вкладки — под ним.
	drawStatus(bg, w, h)
	if len(menu) > 0 {
		drawTabs(bg, menu, route, &hotzones)
	}
	// Кнопка «Закрыть» — ТОЛЬКО при открытой строке вывода/отладки, на правом краю нижней строки.
	if statusMsg != "" || len(debugLines) > 0 {
		drawQuit(bg, &hotzones)
	}

	// Окно подтверждения рисуется поверх всего (ввод идёт прямо в поле, без модалки).
	if confirmMode {
		drawConfirmModal(bg, w, h, &hotzones)
	}
	// Выпадающее меню select — поверх всего, под самим элементом.
	if selectMode {
		drawSelectMenu(bg, &hotzones, w, h)
	}

	// Квадратик под курсором мыши и подсветка сфокусированной хотзоны.
	if eng.mouseX >= 0 && eng.mouseY >= 0 && eng.mouseX < w && eng.mouseY < h {
		bg.Highlight(eng.mouseX, eng.mouseY)
	}
	if focusIdx >= 0 && focusIdx < len(hotzones) {
		hz := hotzones[focusIdx]
		for yy := hz.Y; yy < hz.Y+hz.H; yy++ {
			for xx := hz.X; xx < hz.X+hz.W; xx++ {
				bg.Highlight(xx, yy)
			}
		}
	}

	bg.Blit(s, 0, 0)
	s.Show()
}

// renderBackgroundPages обновляет «живые» плагины неактивных страниц в фоне.
// Движок рисует на экран только активную страницу — но плагины с interval
// (графики, часы и т.п.) должны продолжать собирать данные, пока пользователь
// на другой вкладке. Прогоняем их в черновые буферы (не на экран), чтобы
// renderPlugin перезапускался по своему интервалу и данные не застывали.
// Сама отрисовка черновика отбрасывается — важен побочный эффект (обновление
// pluginCache и состояния плагинов вроде серии графика).
func renderBackgroundPages(pages Pages, active string, w, h int, fsys fs.FS) {
	backgroundRender = true
	defer func() {
		backgroundRender = false
		// Сбрасываем глобалы рисовалок — плагины из action не должны читать
		// чужие размеры/сигнатуру, оставшиеся от фоновых страниц.
		curW, curH, curPluginKey, curViewH = 0, 0, "", 0
	}()
	for route, tiles := range pages {
		if route == active {
			continue
		}
		for _, t := range tiles {
			x, y, tw, th := t.Rect(w, h)
			if tw <= 2 || th <= 2 || t.File == "" {
				continue
			}
			root, perr := loadTile(fsys, t.File)
			if perr != nil {
				continue
			}
			inner := NewBuffer(tw-2, th-2)
			var scratch []Hotzone
			renderTile(root, inner, route+"/"+t.ID, x+1, y+1, th-2, &scratch)
		}
	}
}

// renderTile рендерит содержимое тайла в inner с поддержкой скролла:
// контент рисуется в большой буфер (запас по высоте), окно копируется
// со сдвигом scrollOff[id]; хотзоны сдвигаются так же.
func renderTile(root *Node, inner *Buffer, scrollKey string, ox, oy, viewH int, out *[]Hotzone) {
	// F5: запас по высоте для длинного вывода. Раньше 4× — хвост контента
	// (логи, большие страницы) становился недостижимым. Увеличили до 16×
	// (минимум 200) с жёстким потолком. Полный «честный» расчёт высоты
	// невозможен: повторный рендер перезапустил бы <plugin> (побочные эффекты).
	bigH := viewH * 16
	if bigH < 200 {
		bigH = 200
	}
	if bigH > maxTileHeight {
		bigH = maxTileHeight
	}
	big := NewBuffer(inner.W, bigH)
	var hz []Hotzone
	// Видимая высота — чтобы align="center" центрировал по окну, а не по запасу.
	curViewH = viewH
	RenderHTML(root, big, ox, oy, &hz)

	contentH := usedHeight(big)
	if contentH < 1 {
		contentH = 1
	}
	maxOff := contentH - viewH
	if maxOff < 0 {
		maxOff = 0
	}
	off := scrollOff[scrollKey]
	if off > maxOff {
		off = maxOff
	}
	scrollOff[scrollKey] = off

	// Копируем видимое окно (со сдвигом) в inner.
	for yy := 0; yy < viewH && yy+off < big.H; yy++ {
		for xx := 0; xx < big.W; xx++ {
			c := big.cells[yy+off][xx]
			if c.Rune != 0 {
				inner.Set(xx, yy, c.Rune, c.Style)
			}
		}
	}
	// Хотзоны: сдвигаем на -off, отбрасываем те, что вне окна, и обрезаем
	// частично видимые (чтобы зона не вылезала на рамку тайла).
	for _, z := range hz {
		z.Y -= off
		if z.Y+z.H <= oy || z.Y >= oy+viewH {
			continue
		}
		if z.Y < oy {
			z.H -= oy - z.Y
			z.Y = oy
		}
		if z.Y+z.H > oy+viewH {
			z.H = oy + viewH - z.Y
		}
		if z.H > 0 {
			*out = append(*out, z)
		}
	}

	// Полоса прокрутки: если контент длиннее видимой области.
	if maxOff > 0 && inner.W > 1 {
		scFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
		sc := Style{Fg: scFg}
		sx := inner.W - 1 // правый край тайла (внутри рамки)
		// Высота бегунка — пропорционально видимой части контента.
		thumbH := viewH * viewH / contentH
		if thumbH < 1 {
			thumbH = 1
		}
		if thumbH > viewH {
			thumbH = viewH
		}
		thumbY := 0
		if maxOff > 0 {
			thumbY = off * (viewH - thumbH) / maxOff
		}
		for yy := 0; yy < viewH; yy++ {
			if yy >= thumbY && yy < thumbY+thumbH {
				inner.Set(sx, yy, '█', sc)
			} else {
				inner.Set(sx, yy, '│', sc)
			}
		}
	}
}

// scrollTile меняет скролл тайла под курсором (колесо мыши).
func scrollTile(pages Pages, route string, x, y, w, h, dir int) {
	for _, t := range pages[route] {
		tx, ty, tw, th := t.Rect(w, h)
		if x >= tx && x < tx+tw && y >= ty && y < ty+th {
			scrollOff[t.ID] += dir
			if scrollOff[t.ID] < 0 {
				scrollOff[t.ID] = 0
			}
			return
		}
	}
}

// Rect считает координаты тайла в клетках терминала w×h — это и есть растягиватель:
// терминал сменил размер → w/h другие → тайл пересчитывается на новый.
func (t Tile) Rect(w, h int) (x, y, tw, th int) {
	x = parseLen(t.X, w)
	y = parseLen(t.Y, h)
	tw = parseLen(t.W, w)
	th = parseLen(t.H, h)
	return
}

// parseLen переводит "10%" / "20" / "50vw" / "30vh" в клетки.
// % и vw — от ширины (total), vh — от высоты (тоже total, зависит от вызова).
func parseLen(s string, total int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.HasSuffix(s, "%") {
		return int(parseFloat(s[:len(s)-1]) * float64(total) / 100)
	}
	if strings.HasSuffix(s, "vw") || strings.HasSuffix(s, "vh") {
		return int(parseFloat(s[:len(s)-2]) * float64(total) / 100)
	}
	return int(parseFloat(s))
}

// parseFloat — безопасный разбор числа (при ошибке — 0).
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
