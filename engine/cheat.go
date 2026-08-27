package engine

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// Последовательности клавиш: при наборе движок переходит на страницу.
// Плагин регистрирует последовательность через AddCheatRoute; движок
// отслеживает ввод и при совпадении открывает страницу.

// cheatEntry — зарегистрированная последовательность клавиш и роут, на
// который движок переходит при её вводе.
type cheatEntry struct {
	seq   string
	route string
}

// cheats — реестр последовательностей клавиш.
var cheats []cheatEntry

// keyBuffer — последние нажатия в кодировке AddCheatRoute (отслеживаемый хвост).
var keyBuffer string

// cheatMaxLen — длина самой длинной последовательности (потолок буфера).
var cheatMaxLen int

// AddCheatRoute регистрирует последовательность клавиш, при вводе которой
// движок ПЕРЕХОДИТ на страницу (роут) — навигация, как по вкладке. Кодировка:
//
//	'U','D','L','R' — стрелки; буква/цифра — соответствующая клавиша;
//	'+' — клавиша плюса.
func AddCheatRoute(seq, route string) {
	if seq == "" || route == "" {
		return
	}
	cheats = append(cheats, cheatEntry{seq: seq, route: route})
	if len(seq) > cheatMaxLen {
		cheatMaxLen = len(seq)
	}
}

// checkCheat отслеживает нажатую клавишу и, если набрана зарегистрированная
// последовательность, открывает страницу (роут). route — указатель на текущий
// роут: при совпадении движок меняет роут и сбрасывает фокус (страница новая —
// хотзоны другие). Вызывается из handleKey.
func checkCheat(e *tcell.EventKey, route *string) {
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
			// Переход на страницу: навигация — дело движка.
			if route != nil {
				*route = ch.route
				focusIdx = -1 // новая страница — фокус сбрасывается
			}
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
		// Буквы/цифры — как есть; '+' — маркер «плюса» (в кодах вроде ps+).
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' {
			return byte(r)
		}
	}
	return 0
}
