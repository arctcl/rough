package engine

import (
	"testing"
	"testing/fstest"

	"github.com/gdamore/tcell/v2"
)

// StripMarkers: маркеры \x01{имя} и \x01{} убираются, текст остаётся.
func TestStripMarkers(t *testing.T) {
	got := StripMarkers("A\x01{color_2}████\x01{}│")
	if got != "A████│" {
		t.Fatalf("StripMarkers = %q, ждали %q", got, "A████│")
	}
	// Без маркеров — строка не меняется.
	if s := StripMarkers("просто текст"); s != "просто текст" {
		t.Fatalf("без маркеров: %q", s)
	}
}

// putColored: маркер красит текст цветом из темы, сброс возвращает дефолт,
// маркеры в текст не попадают (ширина не ломается).
func TestPutColored(t *testing.T) {
	curTheme = &Theme{Name: "t", Colors: map[string]string{"color_2": "#00ff00"}}
	defer func() { curTheme = nil }()

	b := NewBuffer(10, 1)
	f := &flowState{}
	f.putColored(b, "A\x01{color_2}B\x01{}C")

	green := tcell.NewHexColor(0x00ff00)
	cells := []struct {
		x  int
		r  rune
		fg tcell.Color
	}{
		{0, 'A', tcell.ColorDefault},
		{1, 'B', green},
		{2, 'C', tcell.ColorDefault},
	}
	for _, c := range cells {
		cell := b.cells[0][c.x]
		if cell.Rune != c.r || cell.Style.Fg != c.fg {
			t.Fatalf("клетка %d: руна=%q fg=%v, ждали %q/%v",
				c.x, cell.Rune, cell.Style.Fg, c.r, c.fg)
		}
	}
	// Маркеры не заняли клеток — ширина строки 3.
	if b.cells[0][3].Rune != 0 {
		t.Fatalf("после строки остался мусор: %q", b.cells[0][3].Rune)
	}
}

// putColored в центрированном блоке: строка центрируется ОДИН раз целиком,
// сегменты не накладываются (иначе графики в align="center" разваливались).
func TestPutColoredCentered(t *testing.T) {
	curTheme = &Theme{Name: "t", Colors: map[string]string{"color_2": "#00ff00"}}
	defer func() { curTheme = nil }()

	b := NewBuffer(12, 1)
	f := &flowState{center: true}
	// "█x" + "..." — маркеры не входят в ширину.
	f.putColored(b, "\x01{color_2}██\x01{}....")

	// Видимая ширина 6 → центр: x=(12-6)/2=3. Сегмент "██" на x=3,4, "...." на 5..8.
	for x := 0; x < 12; x++ {
		want := rune(0)
		if x >= 3 && x <= 4 {
			want = '█'
		}
		if x >= 5 && x <= 8 {
			want = '.'
		}
		if b.cells[0][x].Rune != want {
			t.Fatalf("клетка %d: руна=%q, ждали %q (строка разъехалась)",
				x, b.cells[0][x].Rune, want)
		}
	}
}

// SwitchTheme/ListThemes/CurrentThemeName/ThemeColor — переключение на лету.
func TestSwitchTheme(t *testing.T) {
	curFS = fstest.MapFS{
		"themes/a.json": &fstest.MapFile{Data: []byte(`{"name":"a","colors":{"fg":"#111111"}}`)},
		"themes/b.json": &fstest.MapFile{Data: []byte(`{"name":"b","symbols":{}}`)},
	}
	defer func() { curFS = nil; curTheme = nil }()

	names := ListThemes()
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("ListThemes = %v", names)
	}
	if err := SwitchTheme("a"); err != nil {
		t.Fatal(err)
	}
	if CurrentThemeName() != "a" {
		t.Fatalf("CurrentThemeName = %q", CurrentThemeName())
	}
	if err := SwitchTheme("zz"); err == nil {
		t.Fatal("нужна ошибка: темы zz нет")
	}
	// ThemeColor из активной темы.
	if c := ThemeColor("fg", tcell.ColorDefault); c != tcell.NewHexColor(0x111111) {
		t.Fatalf("ThemeColor(fg) = %v", c)
	}
	// Неизвестный ключ — запасной.
	if c := ThemeColor("нет_такого", tcell.ColorRed); c != tcell.ColorRed {
		t.Fatalf("ThemeColor(нет_такого) = %v", c)
	}
}
