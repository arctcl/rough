package uniq

import (
	"testing"

	"github.com/arctcl/rough/engine"
)

// Убирает только соседние дубли (как uniq).
func TestUniqAdjacent(t *testing.T) {
	out, err := engine.RunSteps([]string{"uniq"}, []string{"a", "a", "b", "a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != "a" || out[1] != "b" || out[2] != "a" {
		t.Fatalf("uniq = %v, ждали [a b a]", out)
	}
}

// Счётчик повторов (как uniq -c).
func TestUniqCount(t *testing.T) {
	out, err := engine.RunSteps([]string{"uniq --count=1"}, []string{"a", "a", "b", "b", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != "2 a" || out[1] != "3 b" || out[2] != "1 c" {
		t.Fatalf("uniq --count=1 = %v, ждали [2 a 3 b 1 c]", out)
	}
}
