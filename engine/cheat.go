package engine

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// Секретные последовательности клавиш (пасхалки вроде конами-кода).
// Плагин регистрирует последовательность и действие через AddCheat; движок
// отслеживает ввод и при совпадении выполняет действие.

// cheatEntry — зарегистрированная последовательность клавиш и её действие.
type cheatEntry struct {
	seq    string
	action string
}

// cheats — реестр секретных последовательностей.
var cheats []cheatEntry

// keyBuffer — последние нажатия в кодировке AddCheat (отслеживаемый хвост).
var keyBuffer string

// cheatMaxLen — длина самой длинной последовательности (потолок буфера).
var cheatMaxLen int

// AddCheat регистрирует секретную последовательность клавиш: при вводе
// выполняется action. Кодировка последовательности:
//
//	'U','D','L','R' — стрелки; буква/цифра — соответствующая клавиша.
func AddCheat(seq, action string) {
	if seq == "" || action == "" {
		return
	}
	cheats = append(cheats, cheatEntry{seq: seq, action: action})
	if len(seq) > cheatMaxLen {
		cheatMaxLen = len(seq)
	}
}

// checkCheat отслеживает нажатую клавишу и, если набрана зарегистрированная
// последовательность, выполняет её действие. Вызывается из handleKey.
func checkCheat(e *tcell.EventKey) {
	c := encodeCheatKey(e)
	if c == 0 {
		return
	}
	keyBuffer += string(c)
	// Ограничиваем буфер самой длинной последовательностью.
	if cheatMaxLen > 0 && len(keyBuffer) > cheatMaxLen {
		keyBuffer = keyBuffer[len(keyBuffer)-cheatMaxLen:]
	}
	for _, ch := range cheats {
		if strings.HasSuffix(keyBuffer, ch.seq) {
			keyBuffer = "" // сброс после срабатывания
			execAction(ch.action, "")
			return
		}
	}
}

// encodeCheatKey кодирует клавишу в символ последовательности (0 — не учитываем).
func encodeCheatKey(e *tcell.EventKey) byte {
	switch e.Key() {
	case tcell.KeyUp:
		return 'U'
	case tcell.KeyDown:
		return 'D'
	case tcell.KeyLeft:
		return 'L'
	case tcell.KeyRight:
		return 'R'
	case tcell.KeyRune:
		r := unicode.ToLower(e.Rune())
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return byte(r)
		}
	}
	return 0
}
