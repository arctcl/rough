package export

import (
	"testing"

	"github.com/arctcl/rough/engine"
)

// export сохраняет вход в переменную и пропускает строки дальше.
func TestExportSetVar(t *testing.T) {
	out, err := engine.RunSteps([]string{"export:host"}, []string{"srv1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "srv1" {
		t.Fatalf("export должен пропустить вход дальше: %v", out)
	}
	if v := engine.VarLine("host"); v != "srv1" {
		t.Fatalf("переменная host = %q", v)
	}
}
