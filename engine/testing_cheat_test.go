package engine

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// Конами-код с переходом (AddCheatRoute) меняет роут — навигация.
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
		t.Fatalf("последовательность не открыла страницу: роут = %q", route)
	}
	// Сброс реестра для других тестов.
	cheats = nil
	keyBuffer = ""
	cheatMaxLen = 0
}
