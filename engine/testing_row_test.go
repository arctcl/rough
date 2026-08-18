package engine

import (
	"strings"
	"testing"
)

// TestRenderRowColumns — колонки <row> должны быть РЯДОМ (на одной строке).
func TestRenderRowColumns(t *testing.T) {
	root, err := ParseHTML(strings.NewReader(
		`<row><div width="50%"><p>LEFT</p></div><div width="50%"><p>RIGHT</p></div></row>`))
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuffer(40, 10)
	var hz []Hotzone
	RenderHTML(root, b, 0, 0, &hz)
	rows := dumpBuffer(b)
	onSameLine := false
	for _, line := range strings.Split(rows, "\n") {
		if strings.Contains(line, "LEFT") && strings.Contains(line, "RIGHT") {
			onSameLine = true
			break
		}
	}
	if !onSameLine {
		t.Fatalf("колонки не рядом (LEFT и RIGHT должны быть на одной строке):\n%s", rows)
	}
}

// TestRenderRowStack — два <row> должны идти СВЕРХУ ВНИЗ (не затирать друг друга).
func TestRenderRowStack(t *testing.T) {
	root, err := ParseHTML(strings.NewReader(
		`<row><div width="50%"><p>A</p></div><div width="50%"><p>B</p></div></row>
		 <row><div width="50%"><p>C</p></div><div width="50%"><p>D</p></div></row>`))
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuffer(40, 12)
	var hz []Hotzone
	RenderHTML(root, b, 0, 0, &hz)
	rows := dumpBuffer(b)
	// Найдём строки, где есть A и где есть C — они должны быть РАЗНЫМИ.
	lineA, lineC := -1, -1
	for i, line := range strings.Split(rows, "\n") {
		if strings.Contains(line, "A") && lineA < 0 {
			lineA = i
		}
		if strings.Contains(line, "C") && lineC < 0 {
			lineC = i
		}
	}
	if lineA < 0 || lineC < 0 {
		t.Fatalf("не нашли ряды:\n%s", rows)
	}
	if lineA == lineC {
		t.Fatalf("второй ряд затёр первый (A и C на одной строке):\n%s", rows)
	}
	t.Logf("A на строке %d, C на строке %d — ок", lineA, lineC)
}
