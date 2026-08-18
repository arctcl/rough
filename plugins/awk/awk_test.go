package awk

import (
	"testing"

	"github.com/arctcl/rough/engine"
)

// Фильтр по регулярке: остаются только строки с ERROR.
func TestAwkFilter(t *testing.T) {
	out, err := engine.RunSteps([]string{"awk --filter=ERROR"}, []string{"x ok", "y ERROR 1", "z ERROR 2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] != "y ERROR 1" || out[1] != "z ERROR 2" {
		t.Fatalf("awk фильтр = %v, ждали строки с ERROR", out)
	}
}

// Фильтр + поле: 3-е поле только из строк ERROR.
func TestAwkFilterField(t *testing.T) {
	out, err := engine.RunSteps([]string{"awk --filter=ERROR --fields=3"}, []string{"x ok 0", "y ERROR 42", "z ERROR 7"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] != "42" || out[1] != "7" {
		t.Fatalf("awk поле = %v, ждали [42 7]", out)
	}
}
