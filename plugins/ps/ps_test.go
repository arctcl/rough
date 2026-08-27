package ps

import "testing"

// parseGoroutine разбирает блок дампа горутины.
func TestParseGoroutine(t *testing.T) {
	block := "goroutine 5 [chan receive]:\nmain.main()\n    /x/main.go:10 +0x25"
	id, state, top := parseGoroutine(block)
	if id != 5 || state != "chan receive" || top != "main.main" {
		t.Fatalf("id=%d state=%q top=%q", id, state, top)
	}

	block2 := "goroutine 1 [running]:\nmain.main()\n    /x/main.go:5"
	id2, state2, top2 := parseGoroutine(block2)
	if id2 != 1 || state2 != "running" || top2 != "main.main" {
		t.Fatalf("id2=%d state2=%q top2=%q", id2, state2, top2)
	}
}

// humanBytes форматирует размеры человекочитаемо.
func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{2048, "2.0 KB"},
		{5 << 20, "5.0 MB"},
	}
	for _, c := range cases {
		if got := humanBytes(c.in); got != c.want {
			t.Fatalf("humanBytes(%d) = %q, ждали %q", c.in, got, c.want)
		}
	}
}
