package engine

import "testing"

// Ввод снаружи в пайп: если $in нет — введённое уходит ВХОДОМ первому плагину
// (linux-стиль "echo введённое | плагин"), а не аргументом в конец.
func TestExecActionInStdin(t *testing.T) {
	AddPlugin("__echo_in", func(in, args []string) ([]string, error) {
		return in, nil
	})
	defer delete(plugins, "__echo_in")

	statusMsg = ""
	execActionIn("__echo_in", "", []string{"hello"})
	if statusMsg != "hello" {
		t.Fatalf("ввод не ушёл входом первому плагину: statusMsg=%q", statusMsg)
	}
}

// Маркер $in: введённое подставляется аргументом в указанное место пайпа.
func TestInVarRegexp(t *testing.T) {
	// $in — заменяется.
	if got := inVarRe.ReplaceAllString("grep:$in | sort", "'x'"); got != "grep:'x' | sort" {
		t.Fatalf("$in подстановка: %q", got)
	}
	// ${in} — тоже заменяется.
	if got := inVarRe.ReplaceAllString("ssh:${in}:cmd", "'x'"); got != "ssh:'x':cmd" {
		t.Fatalf("${in} подстановка: %q", got)
	}
	// $input НЕ должен ловиться как $in (граница слова).
	if got := inVarRe.ReplaceAllString("x:$input", "'x'"); got != "x:$input" {
		t.Fatalf("$input не должен матчить $in: %q", got)
	}
	// hasInVar.
	if !hasInVar("a:$in") || hasInVar("a") {
		t.Fatal("hasInVar определяет $in неверно")
	}
}
