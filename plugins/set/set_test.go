package set

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Дефолт: конфиг вида ключ=значение.
func TestSetDefaultEqual(t *testing.T) {
	f := filepath.Join(t.TempDir(), "c.conf")
	os.WriteFile(f, []byte("a=1\n"), 0644)
	if _, err := setKey(f, "a", "2", "="); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(f)
	if !strings.Contains(string(b), "a=2") {
		t.Fatalf("не записано с '=':\n%s", b)
	}
}

// Конфиг через пробел: ключ значение.
func TestSetSpaceSeparator(t *testing.T) {
	f := filepath.Join(t.TempDir(), "c.conf")
	os.WriteFile(f, []byte("foo 3\nbar 5\n"), 0644)
	if _, err := setKey(f, "bar", "10", " "); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(f)
	if !strings.Contains(string(b), "bar 10") {
		t.Fatalf("не записано через пробел:\n%s", b)
	}
}

// Чтение значения (для select) из пробельного конфига.
func TestSetReadKeySpace(t *testing.T) {
	f := filepath.Join(t.TempDir(), "c.conf")
	os.WriteFile(f, []byte("foo 3\n"), 0644)
	out, err := readKey(f, "foo", " ")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "3" {
		t.Fatalf("чтение = %v, ждали [3]", out)
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
