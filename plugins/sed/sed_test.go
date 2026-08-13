package sed

import (
	"testing"

	"rough/engine"
)

// Простая замена: error → ERROR в каждой строке.
func TestSedReplace(t *testing.T) {
	out, err := engine.RunSteps([]string{"sed:error:ERROR"}, []string{"no error", "one error here"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] != "no ERROR" || out[1] != "one ERROR here" {
		t.Fatalf("sed:error:ERROR = %v, ждали [no ERROR one ERROR here]", out)
	}
}

// Значение с ":" в кавычках: sed:':':1 — заменить ":" на "1".
func TestSedQuotedColon(t *testing.T) {
	out, err := engine.RunSteps([]string{"sed:':':1"}, []string{"a:b:c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "a1b1c" {
		t.Fatalf("sed:':':1 = %v, ждали [a1b1c]", out)
	}
}
