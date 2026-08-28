package engine

import "testing"

// loop:N — повторяет следующие шаги пайпа N раз, выводы склеивает.
func TestLoopRepeat(t *testing.T) {
	AddPlugin("__tag", func(in, args []string) ([]string, error) {
		return []string{"hit"}, nil
	})
	defer delete(plugins, "__tag")

	// loop:3 | __tag  → 3 раза.
	out, err := RunSteps([]string{"loop:3", "__tag"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != "hit" || out[2] != "hit" {
		t.Fatalf("loop:3 = %v, ждали 3×hit", out)
	}

	// loop:1 — как без цикла.
	out, err = RunSteps([]string{"loop:1", "__tag"}, nil)
	if err != nil || len(out) != 1 {
		t.Fatalf("loop:1 = %v, %v; ждали 1×hit", out, err)
	}

	// Без loop — обычный пайп.
	out, err = RunSteps([]string{"__tag"}, nil)
	if err != nil || len(out) != 1 {
		t.Fatalf("обычный пайп = %v, %v", out, err)
	}
}

// Переменные подставляются на момент ВЫПОЛНЕНИЯ (в RunSteps), а не при разборе
// (PrepareAction): export из раннего "&&"-пайпа успевает записать значение до
// того, как поздний его прочитает.
func TestRuntimeVarExpansion(t *testing.T) {
	AddPlugin("__echo", func(in, args []string) ([]string, error) {
		return append([]string{}, args...), nil
	})
	defer delete(plugins, "__echo")

	SetVar("n", []string{"7"})
	// PrepareAction НЕ разворачивает $n — значение раскроется в RunSteps.
	pipes, _ := PrepareAction("__echo:$n")
	if len(pipes) != 1 || pipes[0][0] != "__echo:$n" {
		t.Fatalf("PrepareAction развернул переменную раньше времени: %v", pipes[0])
	}
	// RunSteps подставляет $n в момент выполнения шага.
	out, err := RunSteps(pipes[0], nil)
	if err != nil || len(out) != 1 || out[0] != "7" {
		t.Fatalf("RunSteps: %v %v, ждали [7]", out, err)
	}
}

// loop:N видит переменную, раскрытую в RunSteps (loop:$n → loop:3).
func TestLoopVarCount(t *testing.T) {
	AddPlugin("__tag", func(in, args []string) ([]string, error) {
		return []string{"hit"}, nil
	})
	defer delete(plugins, "__tag")
	SetVar("n", []string{"3"})

	out, err := RunSteps([]string{"loop:$n", "__tag"}, nil)
	if err != nil || len(out) != 3 {
		t.Fatalf("loop:$n = %v, %v; ждали 3×hit", out, err)
	}
}

// export/unexport — резервные слова движка, а не плагины: AddPlugin их не
// примет, а RunSteps сам собирает/удаляет переменную из вывода пайпа.
func TestEngineExportUnexport(t *testing.T) {
	AddPlugin("__n", func(in, args []string) ([]string, error) {
		return []string{args[0]}, nil
	})
	defer delete(plugins, "__n")

	// export нельзя зарегистрировать как плагин (резервное слово движка).
	if AddPlugin("export", func(in, args []string) ([]string, error) { return nil, nil }); HasPlugin("export") {
		t.Fatal("export зарегистрировался как плагин — должен быть резервным словом")
	}

	// __n:7 | export:val — движок сохраняет вывод (7) в переменную val и
	// пропускает строки дальше (как tee).
	out, err := RunSteps([]string{"__n:7", "export:val"}, nil)
	if err != nil || len(out) != 1 || out[0] != "7" {
		t.Fatalf("export: %v %v, ждали [7]", out, err)
	}
	if got := VarLine("val"); got != "7" {
		t.Fatalf("после export val = %q, ждали 7", got)
	}

	// $val читается из другого пайпа.
	out, err = RunSteps([]string{"__n:$val"}, nil)
	if err != nil || len(out) != 1 || out[0] != "7" {
		t.Fatalf("$val = %v %v, ждали [7]", out, err)
	}

	// unexport удаляет переменную.
	if _, err := RunSteps([]string{"unexport:val"}, nil); err != nil {
		t.Fatalf("unexport: %v", err)
	}
	if got := VarLine("val"); got != "" {
		t.Fatalf("после unexport val = %q, ждали пустую", got)
	}

	// export без имени — ошибка.
	if _, err := RunSteps([]string{"export:"}, nil); err == nil {
		t.Fatal("export без имени не дал ошибку")
	}
}
