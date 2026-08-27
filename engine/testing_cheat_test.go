package engine

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// Конами-код: ↑ ↑ ↓ ↓ ← → ← → B A запускает зарегистрированное действие.
func TestCheckCheat(t *testing.T) {
	ran := false
	AddPlugin("__cheat_probe", func(in []string, args []string) ([]string, error) {
		ran = true
		return []string{"glhf"}, nil
	})
	defer func() { delete(plugins, "__cheat_probe") }()

	// Сброс реестра и буфера.
	cheats = nil
	keyBuffer = ""
	cheatMaxLen = 0
	AddCheat("UUDDLRLRba", "__cheat_probe")

	seq := []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyUp, 0, 0), tcell.NewEventKey(tcell.KeyUp, 0, 0),
		tcell.NewEventKey(tcell.KeyDown, 0, 0), tcell.NewEventKey(tcell.KeyDown, 0, 0),
		tcell.NewEventKey(tcell.KeyLeft, 0, 0), tcell.NewEventKey(tcell.KeyRight, 0, 0),
		tcell.NewEventKey(tcell.KeyLeft, 0, 0), tcell.NewEventKey(tcell.KeyRight, 0, 0),
		tcell.NewEventKey(tcell.KeyRune, 'b', 0), tcell.NewEventKey(tcell.KeyRune, 'A', 0),
	}
	route := ""
	for _, ev := range seq {
		checkCheat(ev, &route)
	}
	if !ran {
		t.Fatal("конами-код не запустил действие")
	}
	// Сброс реестра для других тестов.
	cheats = nil
	keyBuffer = ""
	cheatMaxLen = 0
}

// Секретный код с переходом (AddCheatRoute) меняет роут — навигация.
func TestCheckCheatRoute(t *testing.T) {
	// Сброс реестра и буфера.
	cheats = nil
	keyBuffer = ""
	cheatMaxLen = 0
	AddCheatRoute("ps+", "/ps")

	route := "/charts"
	for _, r := range []rune("ps+") {
		checkCheat(tcell.NewEventKey(tcell.KeyRune, r, 0), &route)
	}
	if route != "/ps" {
		t.Fatalf("секретный код не открыл страницу: роут = %q", route)
	}
	// Сброс реестра для других тестов.
	cheats = nil
	keyBuffer = ""
	cheatMaxLen = 0
}
