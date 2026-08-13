package engine

// mouseX, mouseY — позиция мыши (для подсветки квадратика под курсором, -1 — вне экрана).
var mouseX, mouseY = -1, -1

// mouseBtn1 — зажата ли левая кнопка; mouseLastX/Y — прошлая позиция (для drag).
var (
	mouseBtn1               bool
	mouseLastX, mouseLastY int
)

// MouseEvent — высокоуровневое событие мыши, единое для всех источников:
// десктоп (терминал/tcell) и телеграфный (сырой /dev/input/mice). Пока
// десктопный источник работает напрямую с tcell.EventMouse; тип заведён,
// чтобы телеграфный источник отдавал то же самое.
type MouseEvent struct {
	X, Y  int
	Left  bool
	Wheel int // -1 вверх, +1 вниз, 0 нет
	Held  bool // кнопка уже была зажата (перетаскивание)
}
