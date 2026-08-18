package engine

import (
	"strings"
	"testing"
)

// Параметры для теста: как у гипотетического плагина demo.
var testParams = []Param{
	{Name: "place", Required: true}, // обязательный, позиция 1
	{Name: "time", Default: "10"},   // с дефолтом, позиция 2
	{Name: "speed", Default: "4"},   // с дефолтом, позиция 3
	{Name: "mode", Default: "on"},   // с дефолтом, позиция 4
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
	v, err := ParseArgs([]string{"back", "5", "9", "off"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"place": "back", "time": "5", "speed": "9", "mode": "off",
	})
}

// Именованные флаги: порядок не важен, дефолты не трогаем.
func TestParseArgsFlags(t *testing.T) {
	v, err := ParseArgs([]string{"--mode=off", "--place=front", "--time=3"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"place": "front", "time": "3", "speed": "4", "mode": "off",
	})
}

// Несколько флагов в одном аргументе через пробел.
func TestParseArgsFlagsOneArg(t *testing.T) {
	v, err := ParseArgs([]string{"--place=back --time=7 --mode=off"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"place": "back", "time": "7", "speed": "4", "mode": "off",
	})
}

// Микс: пустой слот ":" + флаг, остальное позиционно.
func TestParseArgsMix(t *testing.T) {
	// demo::10:4:off --place=back → место из флага, остальное позиционно.
	v, err := ParseArgs([]string{"", "10", "4", "off", "--place=back"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"place": "back", "time": "10", "speed": "4", "mode": "off",
	})
}

// Дефолты подставляются, когда параметр не задан.
func TestParseArgsDefaults(t *testing.T) {
	v, err := ParseArgs([]string{"back"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"place": "back", "time": "10", "speed": "4", "mode": "on",
	})
}

// Частичный ввод: demo:back — только первый параметр, остальные уходят в
// дефолты (отсутствие разраб обрабатывает сам).
func TestParseArgsPartial(t *testing.T) {
	v, err := ParseArgs([]string{"back"}, testParams)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{
		"place": "back", "time": "10", "speed": "4", "mode": "on",
	})
	// Частичный + дефолт «пустое»: необязательный без дефолта → пустая строка.
	params := []Param{{Name: "place"}, {Name: "detail"}}
	v2, err := ParseArgs([]string{"back"}, params)
	if err != nil {
		t.Fatal(err)
	}
	if v2["place"] != "back" || v2["detail"] != "" {
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
	_, err := ParseArgs([]string{"--place=back", "--junk=1"}, testParams)
	if err == nil {
		t.Fatal("нужна ошибка: неизвестный флаг")
	}
}

// Флаг без "=значение" — ошибка.
func TestParseArgsFlagNoValue(t *testing.T) {
	_, err := ParseArgs([]string{"--place"}, testParams)
	if err == nil {
		t.Fatal("нужна ошибка: флаг без значения")
	}
}

// Последний параметр глотает остаток двоеточий (как ssh:host:команда).
func TestParseArgsGreedyLast(t *testing.T) {
	params := []Param{
		{Name: "host", Required: true},
		{Name: "cmd"}, // необязательный, глотает остаток
	}
	v, err := ParseArgs([]string{"user@host", "cat /a:b"}, params)
	if err != nil {
		t.Fatal(err)
	}
	checkVals(t, v, map[string]string{"host": "user@host", "cmd": "cat /a:b"})
}

// Позиционное значение с пробелом не режется (в отличие от флаг-региона).
func TestParseArgsPosSpaces(t *testing.T) {
	params := []Param{{Name: "cmd"}}
	v, err := ParseArgs([]string{"cat /etc/hostname"}, params)
	if err != nil {
		t.Fatal(err)
	}
	if v["cmd"] != "cat /etc/hostname" {
		t.Fatalf("команда = %q", v["cmd"])
	}
}

// ParamsUsage строит обе формы ввода для man.
func TestParamsUsage(t *testing.T) {
	u := ParamsUsage("demo", testParams)
	// Обязательный — без скобок, опциональные — вложенными [:...].
	// В флагах дефолты видны: --time=10, --speed=4, --mode=on.
	for _, want := range []string{
		"demo:PLACE[:TIME[:SPEED[:MODE]]]",
		"--place=VAL", "--time=10", "--speed=4", "--mode=on",
	} {
		if !strings.Contains(u, want) {
			t.Fatalf("в ParamsUsage нет %q:\n%s", want, u)
		}
	}
}

// SplitAction: флаги через пробел без двоеточий не уходят в имя.
func TestSplitActionFlags(t *testing.T) {
	name, args := SplitAction("demo --place=back --time=7")
	if name != "demo" {
		t.Fatalf("имя = %q, нужно demo", name)
	}
	if len(args) != 1 || !strings.Contains(args[0], "--place=back") || !strings.Contains(args[0], "--time=7") {
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
	params := []Param{{Name: "min", Required: true}, {Name: "max", Required: true}}
	AddPlugin("hybrid", func(in []string, args []string) ([]string, error) {
		v, err := ParseArgs(args, params)
		if err != nil {
			return nil, err
		}
		return []string{v["min"] + "-" + v["max"]}, nil
	})
	defer delete(plugins, "hybrid")

	// Флаги через пробел.
	out, err := RunSteps([]string{"hybrid --min=0 --max=100"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "0-100" {
		t.Fatalf("флаги: %v", out)
	}
	// Микс: позиционные + флаг в одном аргументе.
	out, err = RunSteps([]string{"hybrid:0 --max=50"}, nil)
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
	name, args = SplitAction("cut --sep=':' --fields=1")
	if name != "cut" || len(args) != 1 || !strings.Contains(args[0], "--sep=':'") {
		t.Fatalf("флаг с ':' → имя=%q args=%v", name, args)
	}
}

// Фикс Бага 1: лишние слоты НЕ дублируют последний параметр (был "cat:/a:b" → "cat:cat:/a:b").
func TestParseArgsGreedyLastNoDup(t *testing.T) {
	params := []Param{{Name: "host"}, {Name: "cmd"}}
	v, err := ParseArgs([]string{"user@host", "cat", "/a:b"}, params)
	if err != nil {
		t.Fatal(err)
	}
	if v["cmd"] != "cat:/a:b" {
		t.Fatalf("команда = %q, ждали cat:/a:b (без дубля)", v["cmd"])
	}
}

// SplitSteps: "|" внутри кавычек не разделяет шаги (экранирование ввода).
func TestSplitStepsQuoted(t *testing.T) {
	steps := SplitSteps("a | b | 'c | d'")
	if len(steps) != 3 || steps[0] != "a" || steps[1] != "b" || steps[2] != "'c | d'" {
		t.Fatalf("SplitSteps с кавычками = %v", steps)
	}
}

// FlagValue: вытаскивает флаг --name=value из args (для плагинов, которые
// не используют ParseArgs — например set/toggle с --sep).
func TestFlagValue(t *testing.T) {
	// Флаг «приклеен» к последнему позиционному аргументу через пробел.
	val, rest := FlagValue([]string{"file", "key", "val --sep=space"}, "sep")
	if val != "space" || len(rest) != 3 || rest[0] != "file" || rest[1] != "key" || rest[2] != "val" {
		t.Fatalf("FlagValue = %q %v, ждали space [file key val]", val, rest)
	}
	// Флага нет — возвращаем аргументы как есть, значение пустое.
	val, rest = FlagValue([]string{"a", "b"}, "sep")
	if val != "" || len(rest) != 2 {
		t.Fatalf("без флага = %q %v", val, rest)
	}
}

// Инъекция через поле ввода: ввод оборачивается в кавычки, "|" внутри не
// создаёт новый шаг — произвольная команда не выполняется.
func TestInputInjectionBlocked(t *testing.T) {
	injected := "cat | ssh:root:evil::reboot"
	act := "man:'" + injected + "'"
	pipes, _ := PrepareAction(act)
	if len(pipes) != 1 || len(pipes[0]) != 1 {
		t.Fatalf("инъекция: %d пайпов, ждали 1 пайп из 1 шага: %v", len(pipes), pipes)
	}
	steps := pipes[0]
	// Единственный шаг — один литеральный аргумент плагина man.
	name, args := SplitAction(steps[0])
	if name != "man" || len(args) != 1 || args[0] != injected {
		t.Fatalf("инъекция не заблокирована: имя=%q args=%v", name, args)
	}
}
