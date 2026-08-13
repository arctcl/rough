package engine

import (
	"github.com/gdamore/tcell/v2"
)

// handleMouse — десктопный ИСТОЧНИК мыши (терминал/tcell): переводит
// tcell.EventMouse в единый MouseEvent и отдаёт в общий обработчик
// handleMouseEvent. Вся логика (клик/колесо/drag) живёт в одном месте.
func handleMouse(e *tcell.EventMouse, pages Pages, route *string, w, h int) {
	x, y := e.Position()
	me := MouseEvent{X: x, Y: y}
	// Колесо: tcell.WheelUp — вверх (-1), WheelDown — вниз (+1).
	switch {
	case e.Buttons()&tcell.WheelUp != 0:
		me.Wheel = -1
	case e.Buttons()&tcell.WheelDown != 0:
		me.Wheel = 1
	}
	me.Left = e.Buttons()&tcell.Button1 != 0
	handleMouseEvent(me, pages, route, w, h)
}
