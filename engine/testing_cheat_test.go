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
	for _, ev := range seq {
		checkCheat(ev)
	}
	if !ran {
		t.Fatal("конами-код не запустил действие")
	}
	// Сброс реестра для других тестов.
	cheats = nil
	keyBuffer = ""
	cheatMaxLen = 0
}
