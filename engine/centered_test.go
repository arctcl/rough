package engine

import (
	"strings"
	"testing"
)

// TestCenteredInViewport — <div align="center"> центрируется по видимой высоте
// тайла (curViewH), а не по запасному буферу скролла. Раньше контент уезжал
// вниз за экран — тайлы с align="center" были пустыми.
func TestCenteredInViewport(t *testing.T) {
	viewH := 20
	big := NewBuffer(40, viewH*4) // как в renderTile: запас x4
	curViewH = viewH
	defer func() { curViewH = 0 }()

	root, err := ParseHTML(strings.NewReader(`<div align="center">Привет</div>`))
	if err != nil {
		t.Fatal(err)
	}
	var hz []Hotzone
	RenderHTML(root, big, 0, 0, &hz)

	// Контент должен оказаться в первых viewH строках (видимая область).
	found := false
	for y := 0; y < viewH; y++ {
		for x := 0; x < big.W; x++ {
			if big.cells[y][x].Rune != 0 {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("контент align=center уехал ниже видимой области (viewH=%d):\n%s", viewH, dumpBuffer(big))
	}
}
