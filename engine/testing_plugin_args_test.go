package engine

import (
	"strings"
	"testing"
)

// Параметры для теста: как у гипотетического плагина fuck.
var testParams = []Param{
	{Name: "место", Required: true},  // обязательный, позиция 1
	{Name: "время", Default: "10"},   // с дефолтом, позиция 2
	{Name: "скорость", Default: "4"}, // с дефолтом, позиция 3
	{Name: "смазка", Default: "да"},  // с дефолтом, позиция 4
}

// want проверяет карту значений по ожиданию.
func checkVals(t *testing.T, got map[string]string, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("значений %d, нужно %d: %v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s = %q, нужно %q (все: %v)", k, got[k], v, got)
		}
	}
}

// Позиционная форма: все параметры по порядку двоеточиями.
func TestParseArgsPositional(t *testing.T) {
	v, err := ParseArgs([]string{"зад", "5", "9", "нет"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"место": "зад", "время": "5", "скорость": "9", "смазка": "нет",
	})
}

// Именованные флаги: порядок не важен, дефолты не трогаем.
func TestParseArgsFlags(t *testing.T) {
	v, err := ParseArgs([]string{"--смазка=нет", "--место=рот", "--время=3"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"место": "рот", "время": "3", "скорость": "4", "смазка": "нет",
	})
}

// Несколько флагов в одном аргументе через пробел.
func TestParseArgsFlagsOneArg(t *testing.T) {
	v, err := ParseArgs([]string{"--место=зад --время=7 --смазка=нет"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"место": "зад", "время": "7", "скорость": "4", "смазка": "нет",
	})
}

// Микс: пустой слот ":" + флаг, остальное позиционно.
func TestParseArgsMix(t *testing.T) {
	// fuck::10:4:no --место=зад → место из флага, остальное позиционно.
	v, err := ParseArgs([]string{"", "10", "4", "no", "--место=зад"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"место": "зад", "время": "10", "скорость": "4", "смазка": "no",
	})
}

// Дефолты подставляются, когда параметр не задан.
func TestParseArgsDefaults(t *testing.T) {
	v, err := ParseArgs([]string{"зад"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"место": "зад", "время": "10", "скорость": "4", "смазка": "да",
	})
}

// Частичный ввод: fuck:ass — только первый параметр, остальные уходят в
// дефолты (отсутствие разраб обрабатывает сам).
func TestParseArgsPartial(t *testing.T) {
	v, err := ParseArgs([]string{"зад"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"место": "зад", "время": "10", "скорость": "4", "смазка": "да",
	})
	// Частичный + дефолт «пустое»: необязательный без дефолта → пустая строка.
	params := []Param{{Name: "место"}, {Name: "деталь"}}
	v2, err := ParseArgs([]string{"зад"}, params)
	if err != nil {
		t.Fatal(err)
	}
	if v2["место"] != "зад" || v2["деталь"] != "" {
		t.Fatalf("пустой дефолт: %v", v2)
	}
}

// Обязательный параметр без значения — ошибка.
func TestParseArgsRequiredMissing(t *testing.T) {
	_, err := ParseArgs(nil, testParams)
	if err == nil {
		t.Fatal("нужна ошибка: обязательный параметр не задан")
	}
}

// Неизвестный флаг — ошибка (опечатка не проходит молча).
func TestParseArgsUnknownFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--место=зад", "--хуйня=1"}, testParams)
	if err == nil {
		t.Fatal("нужна ошибка: неизвестный флаг")
	}
}

// Флаг без "=значение" — ошибка.
func TestParseArgsFlagNoValue(t *testing.T) {
	_, err := ParseArgs([]string{"--место"}, testParams)
	if err == nil {
		t.Fatal("нужна ошибка: флаг без значения")
	}
}

// Последний параметр глотает остаток двоеточий (как ssh:host:команда).
func TestParseArgsGreedyLast(t *testing.T) {
	params := []Param{
		{Name: "хост", Required: true},
		{Name: "команда"}, // необязательный, глотает остаток
	}
	v, err := ParseArgs([]string{"user@host", "cat /a:b"}, params)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{"хост": "user@host", "команда": "cat /a:b"})
}

// Позиционное значение с пробелом не режется (в отличие от флаг-региона).
func TestParseArgsPosSpaces(t *testing.T) {
	params := []Param{{Name: "команда"}}
	v, err := ParseArgs([]string{"cat /etc/hostname"}, params)
	if err != nil {
		t.Fatal(err)
	}
	if v["команда"] != "cat /etc/hostname" {
		t.Fatalf("команда = %q", v["команда"])
	}
}

// ParamsUsage строит обе формы ввода для man.
func TestParamsUsage(t *testing.T) {
	u := ParamsUsage("fuck", testParams)
	// Обязательный — без скобок, опциональные — вложенными [:...].
	// В флагах дефолты видны: --время=10, --скорость=4, --смазка=да.
	for _, want := range []string{
		"fuck:МЕСТО[:ВРЕМЯ[:СКОРОСТЬ[:СМАЗКА]]]",
		"--место=ЗНАЧ", "--время=10", "--скорость=4", "--смазка=да",
	} {
		if !strings.Contains(u, want) {
			t.Fatalf("в ParamsUsage нет %q:\n%s", want, u)
		}
	}
}

// SplitAction: флаги через пробел без двоеточий не уходят в имя.
func TestSplitActionFlags(t *testing.T) {
	name, args := SplitAction("fuck --место=зад --время=7")
	if name != "fuck" {
		t.Fatalf("имя = %q, нужно fuck", name)
	}
	if len(args) != 1 || !strings.Contains(args[0], "--место=зад") || !strings.Contains(args[0], "--время=7") {
		t.Fatalf("аргументы = %v", args)
	}
	// Старые формы не сломались.
	if name, _ := SplitAction("cat:/a:b"); name != "cat" {
		t.Fatalf("cat: имя = %q", name)
	}
	if name, args := SplitAction("man"); name != "man" || len(args) != 0 {
		t.Fatalf("man: %q %v", name, args)
	}
}

// Паникующий плагин не валит движок: пайп останавливается, ошибка с трассой.
func TestRunStepsPanicRecover(t *testing.T) {
	AddPlugin("boom", func(in []string, args []string) ([]string, error) {
		panic("кривой параметр")
	})
	defer delete(plugins, "boom")

	out, err := RunSteps([]string{"boom:x"}, nil)
	if err == nil {
		t.Fatal("нужна ошибка от паникующего плагина")
	}
	if out != nil {
		t.Fatalf("выход не должен быть: %v", out)
	}
	if !strings.Contains(err.Error(), "паника") {
		t.Fatalf("ошибка без слова «паника»: %v", err)
	}
	// Пайп не дошёл до следующего шага — сам не упал.
	if _, err := RunSteps([]string{"boom:x | boom:y"}, nil); err == nil {
		t.Fatal("пайп из двух паникующих шагов должен дать ошибку")
	}
}

// Полный путь: SplitAction (пайп) → ParseArgs → значения.
func TestRunStepsFlags(t *testing.T) {
	// Плагин-заглушка, читающий гибридные параметры.
	params := []Param{{Name: "мин", Required: true}, {Name: "макс", Required: true}}
	AddPlugin("hybrid", func(in []string, args []string) ([]string, error) {
		v, err := ParseArgs(args, params)
		if err != nil {
			return nil, err
		}
		return []string{v["мин"] + "-" + v["макс"]}, nil
	})
	defer delete(plugins, "hybrid")

	// Флаги через пробел.
	out, err := RunSteps([]string{"hybrid --мин=0 --макс=100"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "0-100" {
		t.Fatalf("флаги: %v", out)
	}
	// Микс: позиционные + флаг в одном аргументе.
	out, err = RunSteps([]string{"hybrid:0 --макс=50"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != "0-50" {
		t.Fatalf("микс: %v", out)
	}
}

// Экранирование: ":" внутри кавычек не разделяет аргументы.
// sed:':':1 → args = [":", "1"] (разделитель ":" как обычное значение).
func TestSplitActionQuoted(t *testing.T) {
	name, args := SplitAction("sed:':':1")
	if name != "sed" || len(args) != 2 || args[0] != ":" || args[1] != "1" {
		t.Fatalf("sed:':':1 → имя=%q args=%v", name, args)
	}
	name, args = SplitAction("sed:\":\":1")
	if name != "sed" || len(args) != 2 || args[0] != ":" || args[1] != "1" {
		t.Fatalf("sed:\":\":1 → имя=%q args=%v", name, args)
	}
	// Флаг со значением в кавычках: ":" внутри не режет имя действия.
	name, args = SplitAction("cut --разделитель=':' --поля=1")
	if name != "cut" || len(args) != 1 || !strings.Contains(args[0], "--разделитель=':'") {
		t.Fatalf("флаг с ':' → имя=%q args=%v", name, args)
	}
}

// Фикс Бага 1: лишние слоты НЕ дублируют последний параметр (был "cat:/a:b" → "cat:cat:/a:b").
func TestParseArgsGreedyLastNoDup(t *testing.T) {
	params := []Param{{Name: "хост"}, {Name: "команда"}}
	v, err := ParseArgs([]string{"user@host", "cat", "/a:b"}, params)
	if err != nil {
		t.Fatal(err)
	}
	if v["команда"] != "cat:/a:b" {
		t.Fatalf("команда = %q, ждали cat:/a:b (без дубля)", v["команда"])
	}
}

// SplitSteps: "|" внутри кавычек не разделяет шаги (экранирование ввода).
func TestSplitStepsQuoted(t *testing.T) {
	steps := SplitSteps("a | b | 'c | d'")
	if len(steps) != 3 || steps[0] != "a" || steps[1] != "b" || steps[2] != "'c | d'" {
		t.Fatalf("SplitSteps с кавычками = %v", steps)
	}
}

// Инъекция через поле ввода: ввод оборачивается в кавычки, "|" внутри не
// создаёт новый шаг — произвольная команда не выполняется.
func TestInputInjectionBlocked(t *testing.T) {
	injected := "cat | ssh:root:evil::reboot"
	act := "man:'" + injected + "'"
	steps, _ := PrepareAction(act)
	if len(steps) != 1 {
		t.Fatalf("инъекция: пайп из %d шагов, ждали 1: %v", len(steps), steps)
	}
	// Единственный шаг — один литеральный аргумент плагина man.
	name, args := SplitAction(steps[0])
	if name != "man" || len(args) != 1 || args[0] != injected {
		t.Fatalf("инъекция не заблокирована: имя=%q args=%v", name, args)
	}
}
