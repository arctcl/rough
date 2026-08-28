package engine

import "testing"

// loop:N — повторяет следующие шаги пайпа N раз, выводы склеивает.
func TestLoopRepeat(t *testing.T) {
	AddPlugin("__tag", func(in, args []string) ([]string, error) {
		return []string{"hit"}, nil
	})
	defer delete(plugins, "__tag")

	// loop:3 | __tag  → 3 раза.
	out, err := RunSteps([]string{"loop:3", "__tag"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != "hit" || out[2] != "hit" {
		t.Fatalf("loop:3 = %v, ждали 3×hit", out)
	}

	// loop:1 — как без цикла.
	out, err = RunSteps([]string{"loop:1", "__tag"}, nil)
	if err != nil || len(out) != 1 {
		t.Fatalf("loop:1 = %v, %v; ждали 1×hit", out, err)
	}

	// Без loop — обычный пайп.
	out, err = RunSteps([]string{"__tag"}, nil)
	if err != nil || len(out) != 1 {
		t.Fatalf("обычный пайп = %v, %v", out, err)
	}
}
