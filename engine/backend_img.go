package engine

import (
	"errors"
	"os"
	"strconv"

	"github.com/gdamore/tcell/v2"
)

// ppmImage — растровое изображение PPM (P6), RGB-пиксели.
type ppmImage struct {
	w, h int
	data []byte // w*h*3 байт RGB
}

// pixel возвращает цвет пикселя (px, py).
func (img *ppmImage) pixel(px, py int) tcell.Color {
	i := (py*img.w + px) * 3
	if i+2 >= len(img.data) {
		return tcell.ColorDefault
	}
	return tcell.NewRGBColor(int32(img.data[i]), int32(img.data[i+1]), int32(img.data[i+2]))
}

// renderImg рисует PPM-картинку половинчатыми блоками ▀▄█ (2 пикселя в блок).
// Масштаб по ширине тайла, цвета — как в картинке.
func renderImg(n *Node, b *Buffer, f *flowState) {
	path := n.Attrs["src"]
	if path == "" {
		f.put(b, "img: нет src")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		f.put(b, "img: "+err.Error())
		return
	}
	img, err := decodePPM(data)
	if err != nil {
		f.put(b, "img: "+err.Error())
		return
	}
	// Масштаб: сколько пикселей по горизонтали в одном блоке (минимум 1).
	sx := img.w / b.W
	if sx < 1 {
		sx = 1
	}
	for row := 0; row < b.H-1; row++ {
		top := row * 2
		if top >= img.h {
			break
		}
		f.x = 0
		for x := 0; x < b.W; x++ {
			px := x * sx
			if px >= img.w {
				break
			}
			tc := img.pixel(px, top)
			bc := tc
			if top+1 < img.h {
				bc = img.pixel(px, top+1)
			}
			tb := colorLum(tc) >= 128
			bb := colorLum(bc) >= 128
			var r rune
			var col tcell.Color
			switch {
			case tb && bb:
				r = '█'
				col = midColor(tc, bc)
			case tb:
				r = '▀'
				col = tc
			case bb:
				r = '▄'
				col = bc
			default:
				r = ' '
				col = midColor(tc, bc)
			}
			b.Set(f.x, f.y, r, Style{Fg: col, Bg: col})
			f.x++
		}
		f.nl(b)
	}
}

// decodePPM разбирает PPM P6: "P6 W H MAX" + RGB-байты.
func decodePPM(b []byte) (*ppmImage, error) {
	pos := 0
	next := func() (string, error) {
		for pos < len(b) {
			if b[pos] == '#' {
				for pos < len(b) && b[pos] != '\n' {
					pos++
				}
				continue
			}
			if b[pos] == ' ' || b[pos] == '\t' || b[pos] == '\n' || b[pos] == '\r' {
				pos++
				continue
			}
			break
		}
		start := pos
		for pos < len(b) && b[pos] != ' ' && b[pos] != '\t' && b[pos] != '\n' && b[pos] != '\r' {
			pos++
		}
		return string(b[start:pos]), nil
	}
	magic, _ := next()
	if magic != "P6" {
		return nil, errors.New("нужен PPM P6")
	}
	ws, _ := next()
	hs, _ := next()
	next() // max (обычно 255)
	w, _ := strconv.Atoi(ws)
	h, _ := strconv.Atoi(hs)
	if w <= 0 || h <= 0 {
		return nil, errors.New("PPM: плохой размер")
	}
	if pos < len(b) && (b[pos] == ' ' || b[pos] == '\n' || b[pos] == '\t' || b[pos] == '\r') {
		pos++
	}
	need := w * h * 3
	if pos+need > len(b) {
		return nil, errors.New("PPM: не хватает данных")
	}
	return &ppmImage{w: w, h: h, data: b[pos : pos+need]}, nil
}

// colorLum — яркость цвета (0..255).
func colorLum(c tcell.Color) int {
	r, g, b := c.RGB()
	return (int(r) + int(g) + int(b)) / 3
}

// midColor — средний цвет двух.
func midColor(a, b tcell.Color) tcell.Color {
	r1, g1, b1 := a.RGB()
	r2, g2, b2 := b.RGB()
	return tcell.NewRGBColor((r1+r2)/2, (g1+g2)/2, (b1+b2)/2)
}
