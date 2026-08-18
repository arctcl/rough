package engine

import (
	"strings"
	"testing"
)

// TestRenderBoldText — текст внутри <b> не должен пропадать.
func TestRenderBoldText(t *testing.T) {
	root, err := ParseHTML(strings.NewReader(`<p><b>rough</b> — тайл</p>`))
	if err != nil {
		t.Fatal(err)
	}
	dumpNode(t, root, 0)
	b := NewBuffer(40, 4)
	var hz []Hotzone
	RenderHTML(root, b, 0, 0, &hz)
	if !strings.Contains(dumpBuffer(b), "rough") {
		t.Fatalf("<b> текст потерялся:\n%s", dumpBuffer(b))
	}
}

// dumpNode печатает дерево узлов (отладка).
func dumpNode(t *testing.T, n *Node, depth int) {
	ind := strings.Repeat("  ", depth)
	t.Logf("%stag=%q text=%q", ind, n.Tag, n.Text)
	for _, c := range n.Children {
		dumpNode(t, c, depth+1)
	}
}
