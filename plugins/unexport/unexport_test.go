// Тест плагина unexport: удаляет переменную движка (антипод export).
// init() пакета уже зарегистрировал плагин через rough.AddPlugin.
package unexport

import (
	"testing"

	"github.com/arctcl/rough/engine"
)

// unexport удаляет переменную: после вызова $имя больше не подставляется.
func TestUnexportDelVar(t *testing.T) {
	engine.SetVar("tmp", []string{"value"})
	if got := engine.VarLine("tmp"); got != "value" {
		t.Fatalf("до unexport tmp = %q, ждали value", got)
	}

	// Запускаем зарегистрированный плагин через движок.
	_, err := engine.RunSteps([]string{"unexport:tmp"}, nil)
	if err != nil {
		t.Fatalf("RunSteps: %v", err)
	}
	if got := engine.VarLine("tmp"); got != "" {
		t.Fatalf("после unexport tmp = %q, ждали пустую", got)
	}
}

// unexport без имени — ошибка.
func TestUnexportNoName(t *testing.T) {
	if _, err := engine.RunSteps([]string{"unexport:"}, nil); err == nil {
		t.Fatal("unexport без имени не дал ошибку")
	}
}
