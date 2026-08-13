package engine

import (
	"github.com/gdamore/tcell/v2"
)

// drawFrameStyled рисует рамку символами темы с заданным стилем (для изоляции
// меню: фон рамки непрозрачный).
func drawFrameStyled(b *Buffer, x, y, w, h int, st Style) {
	if w < 1 || h < 1 {
		return
	}
	tl := curTheme.Sym("tile_tl", "┌")
	tr := curTheme.Sym("tile_tr", "┐")
	bl := curTheme.Sym("tile_bl", "└")
	br := curTheme.Sym("tile_br", "┘")
	hh := curTheme.Sym("tile_h", "─")
	vv := curTheme.Sym("tile_v", "│")
	b.Set(x, y, tl, st)
	b.Set(x+w-1, y, tr, st)
	b.Set(x, y+h-1, bl, st)
	b.Set(x+w-1, y+h-1, br, st)
	for i := 1; i < w-1; i++ {
		b.Set(x+i, y, hh, st)
		b.Set(x+i, y+h-1, hh, st)
	}
	for j := 1; j < h-1; j++ {
		b.Set(x, y+j, vv, st)
		b.Set(x+w-1, y+j, vv, st)
	}
}

// drawFrame рисует рамку тайла символами из темы.
func drawFrame(b *Buffer, x, y, w, h int) {
	if w < 1 || h < 1 {
		return
	}
	frameFg := curTheme.ResolveColor(themeColor("frame"), tcell.ColorGray)
	st := Style{Fg: frameFg}
	tl := curTheme.Sym("tile_tl", "┌")
	tr := curTheme.Sym("tile_tr", "┐")
	bl := curTheme.Sym("tile_bl", "└")
	br := curTheme.Sym("tile_br", "┘")
	hh := curTheme.Sym("tile_h", "─")
	vv := curTheme.Sym("tile_v", "│")

	b.Set(x, y, tl, st)
	b.Set(x+w-1, y, tr, st)
	b.Set(x, y+h-1, bl, st)
	b.Set(x+w-1, y+h-1, br, st)
	for i := 1; i < w-1; i++ {
		b.Set(x+i, y, hh, st)
		b.Set(x+i, y+h-1, hh, st)
	}
	for j := 1; j < h-1; j++ {
		b.Set(x, y+j, vv, st)
		b.Set(x+w-1, y+j, vv, st)
	}
}
