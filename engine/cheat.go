package engine

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// Секретные последовательности клавиш (пасхалки вроде конами-кода).
// Плагин регистрирует последовательность и действие через AddCheat; движок
// отслеживает ввод и при совпадении выполняет действие.

// cheatEntry — зарегистрированная последовательность клавиш и её эффект:
// либо выполнить action, либо перейти на страницу (роут). Ровно одно из
// полей action/route заполнено.
type cheatEntry struct {
	seq    string
	action string // если не пусто — выполнить действие
	route  string // если не пусто — перейти на страницу (роут)
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

// AddCheatRoute регистрирует секретную последовательность клавиш, при вводе
// которой движок ПЕРЕХОДИТ на страницу (роут) — навигация, как по вкладке.
// Используется инжекторами (например chch) для «секретных» страниц: страница
// есть в tiles.json обычным тайлом, но не в menu — вкладок не видно, попасть
// можно только секретным кодом.
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
// последовательность, выполняет её эффект (action или переход на страницу).
// route — указатель на текущий роут: при совпадении навигации движок меняет
// роут и сбрасывает фокус (страница новая — хотзоны другие). Вызывается из
// handleKey.
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
			if ch.route != "" {
				// Секретный переход на страницу: навигация — дело движка.
				if route != nil {
					*route = ch.route
					focusIdx = -1 // новая страница — фокус сбрасывается
				}
			} else {
				execAction(ch.action, "")
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
