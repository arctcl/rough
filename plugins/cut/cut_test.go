package cut

import (
	"testing"

	"rough/engine"
)

// Выбор 2-го поля по пробелу (разделитель по умолчанию).
func TestCutFieldDefaultSep(t *testing.T) {
	out, err := engine.RunSteps([]string{"cut::2"}, []string{"a b c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "b" {
		t.Fatalf("cut::2 = %v, ждали [b]", out)
	}
}

// Разделитель через флаг: поля по ";". Диапазон N-M.
// (Значение флага не должно содержать ":" — это разделитель аргументов в quick.)
func TestCutSepAndRange(t *testing.T) {
	out, err := engine.RunSteps([]string{"cut --sep=; --fields=2-3"}, []string{"u;host;port;x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "host port" {
		t.Fatalf("cut = %v, ждали [host port]", out)
	}
}

// Поле за границей — пропускается, строка не пустая (нет такого поля).
func TestCutMissingField(t *testing.T) {
	out, err := engine.RunSteps([]string{"cut::5"}, []string{"a b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "" {
		t.Fatalf("cut::5 = %v, ждали ['']", out)
	}
}
