package engine

import "github.com/gdamore/tcell/v2"

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
func (b *Buffer) SetString(x, y int, s string, st Style) {
	for i, r := range s {
		b.Set(x+i, y, r, st)
	}
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

// Blit выводит буфер на экран tcell со смещением (ox, oy).
func (b *Buffer) Blit(s tcell.Screen, ox, oy int) {
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.cells[y][x]
			st := tcell.StyleDefault
			if c.Style.Bold {
				st = st.Bold(true)
			}
			if c.Style.Italic {
				st = st.Italic(true)
			}
			if c.Style.Underline {
				st = st.Underline(true)
			}
			if c.Style.Fg != tcell.ColorDefault {
				st = st.Foreground(c.Style.Fg)
			}
			if c.Style.Bg != tcell.ColorDefault {
				st = st.Background(c.Style.Bg)
			}
			s.SetContent(ox+x, oy+y, c.Rune, nil, st)
		}
	}
}
