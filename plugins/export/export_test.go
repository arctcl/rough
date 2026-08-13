package export

import (
	"testing"

	"rough/engine"
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
	v, ok := engine.GetVar("host")
	if !ok || len(v) != 1 || v[0] != "srv1" {
		t.Fatalf("переменная host = %v (ok=%v)", v, ok)
	}
}
