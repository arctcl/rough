package toggle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Дефолт: конфиг вида ключ=значение.
func TestToggleDefaultEqual(t *testing.T) {
	f := filepath.Join(t.TempDir(), "c.conf")
	os.WriteFile(f, []byte("debug=0\n"), 0644)
	if _, err := toggleKey(f, "debug", "="); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(f)
	if !strings.Contains(string(b), "debug=1") {
		t.Fatalf("не переключено с '=':\n%s", b)
	}
}

// Конфиг через пробел: ключ значение.
func TestToggleSpaceSeparator(t *testing.T) {
	f := filepath.Join(t.TempDir(), "c.conf")
	os.WriteFile(f, []byte("debug 0\n"), 0644)
	if _, err := toggleKey(f, "debug", " "); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(f)
	if !strings.Contains(string(b), "debug 1") {
		t.Fatalf("не переключено через пробел:\n%s", b)
	}
}

// Чтение состояния (для checkbox) из пробельного конфига.
func TestToggleGetSpace(t *testing.T) {
	f := filepath.Join(t.TempDir(), "c.conf")
	os.WriteFile(f, []byte("debug 1\n"), 0644)
	out, err := readValue(f, "debug", " ")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "1" {
		t.Fatalf("чтение = %v, ждали [1]", out)
	}
}

// normSep: пусто и "=" → "=", "space" → пробел.
func TestNormSep(t *testing.T) {
	if normSep("") != "=" || normSep("=") != "=" {
		t.Fatal("дефолт не '='")
	}
	if normSep("space") != " " {
		t.Fatal("space не распознан")
	}
}
