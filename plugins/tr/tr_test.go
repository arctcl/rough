package tr

import (
	"testing"

	"github.com/arctcl/rough/engine"
)

// Замена одного символа: a → b.
func TestTrReplace(t *testing.T) {
	out, err := engine.RunSteps([]string{"tr:a:b"}, []string{"banana"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "bbnbnb" {
		t.Fatalf("tr:a:b = %v, ждали [bbnbnb]", out)
	}
}

// Диапазоны: a-z → A-Z.
func TestTrRange(t *testing.T) {
	out, err := engine.RunSteps([]string{"tr:a-z:A-Z"}, []string{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "HELLO WORLD" {
		t.Fatalf("tr:a-z:A-Z = %v, ждали [HELLO WORLD]", out)
	}
}

// Регистр целиком: tr:upper.
func TestTrUpper(t *testing.T) {
	out, err := engine.RunSteps([]string{"tr:upper"}, []string{"hi there"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "HI THERE" {
		t.Fatalf("tr:upper = %v, ждали [HI THERE]", out)
	}
}

// Удаление цифр: tr:0-9: (пустой TO — как tr -d).
func TestTrDelete(t *testing.T) {
	out, err := engine.RunSteps([]string{"tr:0-9:"}, []string{"a1b2c3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "abc" {
		t.Fatalf("tr:0-9: = %v, ждали [abc]", out)
	}
}

// Флаги: tr --from=a --to=x.
func TestTrFlags(t *testing.T) {
	out, err := engine.RunSteps([]string{"tr --from=a --to=x"}, []string{"banana"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "bxnxnx" {
		t.Fatalf("tr --from=a --to=x = %v, ждали [bxnxnx]", out)
	}
}
