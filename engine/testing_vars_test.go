package engine

import "testing"

// Подстановка переменных $имя / ${имя} / \$ в action.
func TestExpandVars(t *testing.T) {
	SetVar("host", []string{"srv1"})
	SetVar("path", []string{"/var/log", "app.log"})
	defer func() { vars = map[string][]string{} }()

	cases := []struct{ in, want string }{
		{"ssh:$host::uptime", "ssh:srv1::uptime"},            // $имя
		{"cat:/${path}/x", "cat://var/log app.log/x"},        // ${имя} (несколько строк → пробел)
		{"echo \\$host $host", "echo $host srv1"},            // \$ — литерал, $host — подстановка
		{"cat:$no_such::x", "cat:::x"},                       // неизвестная — пусто
		{"grep:\\d+", "grep:\\d+"},                           // regex не трогаем
	}
	for _, c := range cases {
		if got := expandVars(c.in); got != c.want {
			t.Fatalf("expandVars(%q) = %q, ждали %q", c.in, got, c.want)
		}
	}
}
