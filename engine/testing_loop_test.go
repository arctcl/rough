package engine

import (
	"testing"
	"testing/fstest"
)

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

// Аккумулятор export:ИМЯ += — движок ПРИБАВЛЯЕТ число к текущему значению
// переменной (а не перезаписывает, как обычный export:ИМЯ). Так по кускам
// файла собирается общая сумма: "cat | line | wc | export:count +=".
func TestEngineAccumulator(t *testing.T) {
	// __n:N выдаёт одну строку-число (как wc в реальном пайпе).
	AddPlugin("__n", func(in, args []string) ([]string, error) {
		return []string{args[0]}, nil
	})
	defer delete(plugins, "__n")

	// export:ИМЯ +=  — обычный export ПО-ПРЕЖНЕМУ перезаписывает.
	if _, err := RunSteps([]string{"__n:100", "export:acc"}, nil); err != nil {
		t.Fatal(err)
	}
	if got := VarLine("acc"); got != "100" {
		t.Fatalf("обычный export: acc = %q, ждали 100", got)
	}

	// export:acc +=  — прибавляет число из вывода пайпа (cur): 100 + 7 = 107.
	if _, err := RunSteps([]string{"__n:7", "export:acc +="}, nil); err != nil {
		t.Fatal(err)
	}
	if got := VarLine("acc"); got != "107" {
		t.Fatalf("аккумулятор 1: acc = %q, ждали 107", got)
	}

	// Ещё кусок: 107 + 5 = 112 (накапливается, а не перезаписывает).
	if _, err := RunSteps([]string{"__n:5", "export:acc +="}, nil); err != nil {
		t.Fatal(err)
	}
	if got := VarLine("acc"); got != "112" {
		t.Fatalf("аккумулятор 2: acc = %q, ждали 112", got)
	}

	// export:ИМЯ += ЧИСЛО — литерал прибавляется тоже: 112 + 3 = 115.
	if _, err := RunSteps([]string{"export:acc += 3"}, nil); err != nil {
		t.Fatal(err)
	}
	if got := VarLine("acc"); got != "115" {
		t.Fatalf("литерал: acc = %q, ждали 115", got)
	}

	// export:ИМЯ + ЧИСЛО — та же форма без '=': 115 + 10 = 125.
	if _, err := RunSteps([]string{"export:acc + 10"}, nil); err != nil {
		t.Fatal(err)
	}
	if got := VarLine("acc"); got != "125" {
		t.Fatalf("форма +: acc = %q, ждали 125", got)
	}

	// Аккумулятор с пустым выводом (нет строк в cur) — ошибка.
	if _, err := RunSteps([]string{"export:acc +="}, nil); err == nil {
		t.Fatal("аккумулятор с пустым выводом не дал ошибку")
	}

	// Аккумулятор с нечисловым выводом — ошибка.
	AddPlugin("__txt", func(in, args []string) ([]string, error) {
		return []string{"not-a-number"}, nil
	})
	defer delete(plugins, "__txt")
	if _, err := RunSteps([]string{"__txt", "export:acc +="}, nil); err == nil {
		t.Fatal("аккумулятор с нечисловым выводом не дал ошибку")
	}
}

// Резервные слова движка (export/unexport/loop/confirm) — НЕ плагины: синтакс-
// чекер не должен ругаться на них в action и <plugin pipe>. Иначе примеры с
// export не стартовали бы (баг: пропускался только confirm).
func TestSyntaxReservedWords(t *testing.T) {
	// __probe — реальный плагин, чтобы проверить, что обычные шаги валидны.
	AddPlugin("__probe", func(in, args []string) ([]string, error) { return in, nil })
	defer delete(plugins, "__probe")

	fsys := fstest.MapFS{
		"tiles.json": &fstest.MapFile{Data: []byte(
			`{"pattern":["id","x","y","w","h","file"],` +
				`"main":[["a","0","0","1","1","t.html"]]}`)},
		"t.html": &fstest.MapFile{Data: []byte(`
			<button action="export:host"></button>
			<button action="__probe | export:n"></button>
			<button action="loop:3 | __probe"></button>
			<button action="unexport:tmp"></button>
			<button action="__probe | confirm"></button>
			<button action="__probe | export:n +="></button>
			<plugin pipe="__probe | export:n"></plugin>
		`)},
	}
	pages, err := LoadPages(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if errs := CheckSyntax(fsys, pages); len(errs) != 0 {
		t.Fatalf("CheckSyntax ошибочно ругнулся на резервные слова: %v", errs)
	}

	// А вот неизвестный плагин чекер обязан поймать.
	fsys2 := fstest.MapFS{
		"tiles.json": &fstest.MapFile{Data: []byte(
			`{"pattern":["id","x","y","w","h","file"],` +
				`"main":[["a","0","0","1","1","t2.html"]]}`)},
		"t2.html": &fstest.MapFile{Data: []byte(`<button action="__nope"></button>`)},
	}
	pages2, err := LoadPages(fsys2)
	if err != nil {
		t.Fatal(err)
	}
	if errs := CheckSyntax(fsys2, pages2); len(errs) == 0 {
		t.Fatal("CheckSyntax не поймал неизвестный плагин")
	}
}
