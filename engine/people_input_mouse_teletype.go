package engine

// Телеграфная мышь: читаем устройство мыши НАПРЯМУЮ (без X/Wayland/терминала).
// Нужна для голого Linux VT, где tcell не даёт мышь. Мышь ищем сами на любой
// системе, парсим протокол PS/2 (пакеты по 3 или 4 байта) и отдаём те же
// MouseEvent, что и десктопный источник. Нужны права: root или группа input.

import (
	"errors"
	"os"
	"strconv"
)

// findMouseDevice ищет мышь на любом Linux: сначала агрегат /dev/input/mice,
// затем отдельные /dev/input/mouseN, затем /dev/psaux. Возвращает путь
// или понятную ошибку (мыши нет / нет прав).
func findMouseDevice() (string, error) {
	cands := []string{"/dev/input/mice", "/dev/psaux"}
	for i := 0; i < 16; i++ {
		cands = append(cands, "/dev/input/mouse"+strconv.Itoa(i))
	}
	for _, p := range cands {
		if f, err := os.Open(p); err == nil {
			f.Close()
			return p, nil
		}
	}
	return "", errors.New("мышь не найдена: нужен /dev/input/mice (проверь права: root или группа input)")
}

// parsePS2Packet разбирает один пакет PS/2 (3 или 4 байта).
//
//	byte0: [Y-ovf][X-ovf][Y-sign][X-sign][1][M][R][L]   — бит 3 всегда 1
//	byte1: X-дельта (знак в byte0 бит 4)
//	byte2: Y-дельта (знак в byte0 бит 5)
//	byte3: колесо (IntelliMouse, знаковое) — только при mode=4
//
// Возвращает дельты, кнопки и ok=false, если пакет невалиден (сдвиг не выровнен).
func parsePS2Packet(p []byte, mode int) (dx, dy, wheel int, left, right, mid bool, ok bool) {
	if len(p) < 3 {
		return 0, 0, 0, false, false, false, false
	}
	b0 := p[0]
	// Бит 3 у валидного пакета PS/2 всегда 1 — по нему ловим невыровненность.
	if b0&0x08 == 0 {
		return 0, 0, 0, false, false, false, false
	}
	dx = int(p[1])
	if b0&0x10 != 0 {
		dx -= 256
	}
	dy = int(p[2])
	if b0&0x20 != 0 {
		dy -= 256
	}
	left = b0&0x01 != 0
	right = b0&0x02 != 0
	mid = b0&0x04 != 0
	if mode >= 4 && len(p) >= 4 {
		wheel = int(int8(p[3]))
	}
	return dx, dy, wheel, left, right, mid, true
}

// teletypeMouse — состояние чтения телеграфной мыши: файл устройства, размер
// пакета (3 или 4), накопленная абсолютная позиция и остаток байт.
type teletypeMouse struct {
	file   *os.File
	mode   int // размер пакета PS/2: 3 или 4
	pos    mousePos
	carry  []byte // невыровненные байты между чтениями
	badSeq int    // подряд невалидных пакетов (для автосмены размера)
}

// mousePos — абсолютная позиция телеграфной мыши (накапливается из дельт).
type mousePos struct{ X, Y int }

// openTeletypeMouse находит и открывает мышь. Источники взаимоисключающие:
// на X/Wayland (DISPLAY/WAYLAND_DISPLAY) мышь даёт терминал (tcell) — сырое
// устройство не открываем. На голом Linux VT без графики — читаем /dev/input/mice.
// Нет устройства/прав — возвращаем nil: движок продолжит с десктопной мышью.
func openTeletypeMouse() *teletypeMouse {
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return nil
	}
	p, err := findMouseDevice()
	if err != nil {
		return nil
	}
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	return &teletypeMouse{file: f, mode: 4}
}

// Close закрывает устройство.
func (t *teletypeMouse) Close() {
	if t != nil && t.file != nil {
		t.file.Close()
	}
}

// read читает доступные байты из устройства и превращает их в MouseEvent.
// w,h — размер экрана (для ограничения позиции; 0 — не ограничивать).
// Ошибка — устройство закрылось/отвалилось.
func (t *teletypeMouse) read(w, h int) ([]MouseEvent, error) {
	buf := make([]byte, 32)
	n, err := t.file.Read(buf)
	if err != nil {
		return nil, err
	}
	t.carry = append(t.carry, buf[:n]...)

	var evs []MouseEvent
	for len(t.carry) >= t.mode {
		pkt := t.carry[:t.mode]
		t.carry = t.carry[t.mode:]
		dx, dy, wheel, left, _, _, ok := parsePS2Packet(pkt, t.mode)
		if !ok {
			// Невыровненный байт — сдвигаемся на 1 и считаем сбой.
			t.carry = append([]byte{pkt[0]}, t.carry...)
			t.badSeq++
			// Много сбоев подряд — похоже, размер пакета не тот: меняем.
			if t.badSeq >= 4 {
				if t.mode == 4 {
					t.mode = 3
				} else {
					t.mode = 4
				}
				t.badSeq = 0
			}
			continue
		}
		t.badSeq = 0
		// Позиция накапливается из дельт (как курсор в сыром режиме).
		t.pos.X += dx
		t.pos.Y += dy
		if t.pos.X < 0 {
			t.pos.X = 0
		}
		if t.pos.Y < 0 {
			t.pos.Y = 0
		}
		if w > 0 && t.pos.X >= w {
			t.pos.X = w - 1
		}
		if h > 0 && t.pos.Y >= h {
			t.pos.Y = h - 1
		}
		evs = append(evs, MouseEvent{X: t.pos.X, Y: t.pos.Y, Left: left, Wheel: -wheel})
	}
	return evs, nil
}
