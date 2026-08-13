package engine

import (
	"os"
	"testing"
)

// parsePS2Packet: дельты, знаки, кнопки, колесо, невалидные пакеты.
func TestParsePS2Packet(t *testing.T) {
	// Движение вправо на 1: bit3 всегда 1, без кнопок (0x08 = только флаг валидности).
	dx, dy, wheel, left, _, _, ok := parsePS2Packet([]byte{0x08, 0x01, 0x00}, 4)
	if !ok || dx != 1 || dy != 0 || wheel != 0 || left {
		t.Fatalf("вправо: ok=%v dx=%d dy=%d wheel=%d left=%v", ok, dx, dy, wheel, left)
	}
	// Влево на 1: X-sign (bit4) + byte1=0xFF.
	dx, dy, _, _, _, _, ok = parsePS2Packet([]byte{0x19, 0xFF, 0x00}, 3)
	if !ok || dx != -1 {
		t.Fatalf("влево: dx=%d", dx)
	}
	// Вверх на 1: Y-sign (bit5) + byte2=0xFF.
	dx, dy, _, _, _, _, ok = parsePS2Packet([]byte{0x29, 0x00, 0xFF}, 3)
	if !ok || dy != -1 {
		t.Fatalf("вверх: dy=%d", dy)
	}
	// Левая кнопка: bit0.
	_, _, _, left, _, _, ok = parsePS2Packet([]byte{0x09, 0x00, 0x00}, 3)
	if !ok || !left {
		t.Fatalf("левая кнопка: left=%v", left)
	}
	// Колесо вверх: byte3=1 (4-байтовый пакет).
	_, _, wheel, _, _, _, ok = parsePS2Packet([]byte{0x08, 0x00, 0x00, 0x01}, 4)
	if !ok || wheel != 1 {
		t.Fatalf("колесо: wheel=%d", wheel)
	}
	// В 3-байтовом режиме колесо игнорируется.
	_, _, wheel, _, _, _, ok = parsePS2Packet([]byte{0x08, 0x00, 0x00, 0x01}, 3)
	if !ok || wheel != 0 {
		t.Fatalf("3-байтовый: wheel=%d", wheel)
	}
	// Невалидный (bit3=0).
	if _, _, _, _, _, _, ok = parsePS2Packet([]byte{0x00, 0x00, 0x00}, 3); ok {
		t.Fatal("невалидный пакет принят")
	}
}

// read: накапливает абсолютную позицию из дельт и ограничивает экраном.
func TestTeletypeRead(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	tm := &teletypeMouse{file: pr, mode: 4}
	// Два пакета: вправо на 2, потом вниз на 1.
	pw.Write([]byte{0x08, 0x02, 0x00, 0x00, 0x08, 0x00, 0x01, 0x00})
	evs, err := tm.read(40, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("событий %d, ждали 2", len(evs))
	}
	if evs[0].X != 2 || evs[0].Y != 0 {
		t.Fatalf("первое событие: %+v", evs[0])
	}
	if evs[1].X != 2 || evs[1].Y != 1 {
		t.Fatalf("второе событие: %+v", evs[1])
	}
}

// findMouseDevice на Windows вернёт понятную ошибку (устройств нет) — не паника.
func TestFindMouseDevice(t *testing.T) {
	p, err := findMouseDevice()
	_ = p
	if err == nil {
		t.Logf("мышь найдена: %s", p) // на Linux с правами — путь; на Windows — ошибка
		return
	}
	if err.Error() == "" {
		t.Fatal("пустая ошибка")
	}
}
