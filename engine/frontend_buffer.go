package engine

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// Style — стиль клетки (цвета и атрибуты).
type Style struct {
	Fg, Bg                  tcell.Color
	Bold, Italic, Underline bool
}

// Cell — одна клетка терминала (руна + стиль).
type Cell struct {
	Rune  rune
	Style Style
}

// Buffer — двумерный буфер клеток (внутренний холст).
type Buffer struct {
	W, H  int
	cells [][]Cell
}

// NewBuffer создаёт буфер w×h. Клетки пустые (Rune=0) — прозрачные для копирования.
func NewBuffer(w, h int) *Buffer {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	cells := make([][]Cell, h)
	for y := range cells {
		cells[y] = make([]Cell, w)
	}
	return &Buffer{W: w, H: h, cells: cells}
}

// In проверяет, что координата внутри буфера.
func (b *Buffer) In(x, y int) bool { return x >= 0 && y >= 0 && x < b.W && y < b.H }

// Set кладёт руну в клетку (вне границ — молча пропускает).
func (b *Buffer) Set(x, y int, r rune, st Style) {
	if !b.In(x, y) {
		return
	}
	b.cells[y][x] = Cell{Rune: r, Style: st}
}

// SetString рисует строку слева направо без переноса.
// Сдвиг — по реальной ширине руны (uniseg), чтобы кириллица не наезжала.
func (b *Buffer) SetString(x, y int, s string, st Style) {
	for _, r := range s {
		b.Set(x, y, r, st)
		x += uniseg.StringWidth(string(r))
	}
}

// Highlight подсвечивает клетку (квадратик под курсором мыши / фокусом).
// Непустую клетку инвертирует по цветам, пустую — закрашивает белым,
// чтобы «квадратик» был виден даже на пустом месте экрана.
func (b *Buffer) Highlight(x, y int) {
	if !b.In(x, y) {
		return
	}
	c := b.cells[y][x]
	if c.Style.Fg == tcell.ColorDefault && c.Style.Bg == tcell.ColorDefault {
		c.Style.Bg = tcell.ColorWhite
		c.Style.Fg = tcell.ColorBlack
	} else {
		c.Style.Fg, c.Style.Bg = c.Style.Bg, c.Style.Fg
	}
	if c.Rune == 0 {
		c.Rune = ' '
	}
	b.cells[y][x] = c
}

// Fill заливает буфер руной и стилем.
func (b *Buffer) Fill(r rune, st Style) {
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			b.cells[y][x] = Cell{Rune: r, Style: st}
		}
	}
}

// Copy копирует содержимое src в этот буфер со смещением (ox, oy).
// Пустые клетки (Rune=0) не копируются — так внутренние тайлы прозрачны.
func (b *Buffer) Copy(src *Buffer, ox, oy int) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			c := src.cells[y][x]
			if c.Rune != 0 {
				b.Set(ox+x, oy+y, c.Rune, c.Style)
			}
		}
	}
}

// prevFrame — предыдущий отрисованный кадр (для дифф-рендеринга):
// на экран отправляются только ИЗМЕНИВШИЕСЯ клетки — сверхлёгкая отрисовка
// (статику не перерисовываем, живое меняется точечно).
var prevFrame *Buffer

// Blit выводит буфер на экран tcell, отправляя ТОЛЬКО изменившиеся клетки
// относительно предыдущего кадра. Первый кадр и ресайз рисуют всё.
func (b *Buffer) Blit(s tcell.Screen, ox, oy int) {
	// Первый кадр или изменился размер — рисуем всё, запоминаем кадр.
	if prevFrame == nil || prevFrame.W != b.W || prevFrame.H != b.H {
		prevFrame = NewBuffer(b.W, b.H)
		b.blitAll(s, ox, oy)
		copyBuffer(prevFrame, b)
		s.Show()
		return
	}
	// Дальше — только изменённые клетки (дифф с предыдущим кадром).
	changed := false
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.cells[y][x]
			if c == prevFrame.cells[y][x] {
				continue // клетка не менялась — не отправляем на экран
			}
			prevFrame.cells[y][x] = c
			s.SetContent(ox+x, oy+y, c.Rune, nil, cellStyle(c.Style))
			changed = true
		}
	}
	if changed {
		s.Show()
	}
}

// blitAll выводит весь буфер на экран (первый кадр / ресайз).
func (b *Buffer) blitAll(s tcell.Screen, ox, oy int) {
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.cells[y][x]
			s.SetContent(ox+x, oy+y, c.Rune, nil, cellStyle(c.Style))
		}
	}
}

// copyBuffer копирует src в dst (для хранения предыдущего кадра).
func copyBuffer(dst, src *Buffer) {
	for y := 0; y < src.H; y++ {
		copy(dst.cells[y], src.cells[y])
	}
}

// cellStyle собирает tcell-стиль из нашего Style.
func cellStyle(st Style) tcell.Style {
	s := tcell.StyleDefault
	if st.Bold {
		s = s.Bold(true)
	}
	if st.Italic {
		s = s.Italic(true)
	}
	if st.Underline {
		s = s.Underline(true)
	}
	if st.Fg != tcell.ColorDefault {
		s = s.Foreground(st.Fg)
	}
	if st.Bg != tcell.ColorDefault {
		s = s.Background(st.Bg)
	}
	return s
}
