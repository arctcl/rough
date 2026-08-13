package sort

import (
	"testing"

	"rough/engine"
)

// По алфавиту по возрастанию.
func TestSortAsc(t *testing.T) {
	out, err := engine.RunSteps([]string{"sort"}, []string{"b", "a", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != "a" || out[1] != "b" || out[2] != "c" {
		t.Fatalf("sort = %v, ждали [a b c]", out)
	}
}

// По убыванию.
func TestSortReverse(t *testing.T) {
	out, err := engine.RunSteps([]string{"sort --reverse=1"}, []string{"a", "c", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != "c" || out[1] != "b" || out[2] != "a" {
		t.Fatalf("sort --reverse=1 = %v, ждали [c b a]", out)
	}
}

// Числовая сортировка с суффиксами K/M/G (как sort -h).
func TestSortNumeric(t *testing.T) {
	out, err := engine.RunSteps([]string{"sort --numeric=1"}, []string{"10M", "2K", "1G"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != "2K" || out[1] != "10M" || out[2] != "1G" {
		t.Fatalf("sort --numeric=1 = %v, ждали [2K 10M 1G]", out)
	}
}

// Позиционная форма: sort:0:1 — reverse=0, numeric=1 (число в начале строки).
func TestSortPositional(t *testing.T) {
	out, err := engine.RunSteps([]string{"sort:0:1"}, []string{"10M", "2K", "1G"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != "2K" || out[1] != "10M" || out[2] != "1G" {
		t.Fatalf("sort:0:1 = %v, ждали [2K 10M 1G]", out)
	}
}
