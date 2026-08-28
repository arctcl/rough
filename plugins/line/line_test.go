package line

import (
	"reflect"
	"testing"
)

// pickLines: "N" — одна строка, "N-M" — диапазон строк.
func TestPickLines(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	cases := []struct {
		spec string
		want []string
	}{
		{"2", []string{"b"}},
		{"2-4", []string{"b", "c", "d"}},
		{"1-1", []string{"a"}},
		{"5", []string{"e"}},
		{"0", nil},
		{"6", nil},
	}
	for _, c := range cases {
		got := pickLines(lines, c.spec)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("pickLines(%q) = %v, ждали %v", c.spec, got, c.want)
		}
	}
}
