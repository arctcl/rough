package rough

import (
	"testing"

	"github.com/arctcl/rough/engine"
)

// Механика реестра: плагин регистрируется через AddPlugin и находится по имени.
// (Сами плагины в библиотеке НЕ живут — их подключает проект пользователя.)
func TestPluginRegisterAndLookup(t *testing.T) {
	engine.AddPlugin("hello", func(in []string, args []string) ([]string, error) {
		return []string{"привет"}, nil
	})
	if !engine.HasPlugin("hello") {
		t.Error("плагин hello не зарегистрирован")
	}
	out, err := engine.RunSteps([]string{"hello"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "привет" {
		t.Errorf("неожиданный вывод: %v", out)
	}
}

// Проверка: run запрещён движком — даже если кто-то попробует его зарегистрировать.
func TestRunIsForbidden(t *testing.T) {
	engine.AddPlugin("run", func(in []string, args []string) ([]string, error) { return nil, nil })
	if engine.HasPlugin("run") {
		t.Error("run не должен регистрироваться движком")
	}
}

// Пайпы: выход одного шага идёт на вход следующего.
func TestPipeRuns(t *testing.T) {
	engine.AddPlugin("double", func(in []string, args []string) ([]string, error) {
		var out []string
		for _, ln := range in {
			out = append(out, ln+ln)
		}
		return out, nil
	})
	engine.AddPlugin("seed", func(in []string, args []string) ([]string, error) {
		return []string{"a"}, nil
	})
	out, err := engine.RunSteps([]string{"seed", "double", "double"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "aaaa" {
		t.Errorf("пайп не сработал: %v", out)
	}
}
