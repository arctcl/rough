package engine

import (
	"reflect"
	"testing"
)

// Разворачивание диапазонов [N-M] / [a-b] / [v1,v2] в перебор значений.
func TestExpandRanges(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		// Без диапазонов — как есть, один элемент.
		{"ssh:host:apt", []string{"ssh:host:apt"}},
		// Числовой диапазон — перебор адресов.
		{"ssh:192.168.1.[1-3]:apt",
			[]string{
				"ssh:192.168.1.1:apt",
				"ssh:192.168.1.2:apt",
				"ssh:192.168.1.3:apt",
			}},
		// Обратный диапазон.
		{"ping[3-1]", []string{"ping3", "ping2", "ping1"}},
		// Буквенный диапазон.
		{"[a-c]x", []string{"ax", "bx", "cx"}},
		// Список через запятую.
		{"ssh:10.0.0.[1,4,9]:cmd",
			[]string{"ssh:10.0.0.1:cmd", "ssh:10.0.0.4:cmd", "ssh:10.0.0.9:cmd"}},
		// Несколько диапазонов — декартово произведение.
		{"[1-2][a-b]",
			[]string{"1a", "1b", "2a", "2b"}},
		// Не диапазон — оставить как есть (скобки не трогаем).
		{"set:[foo]", []string{"set:[foo]"}},
	}
	for _, c := range cases {
		got := expandRanges(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("expandRanges(%q) = %v, ждали %v", c.in, got, c.want)
		}
	}
}
