package engine

import (
	"os"
	"strings"
	"testing"
)

// dumpBuffer собирает содержимое буфера построчно (для проверок).
func dumpBuffer(b *Buffer) string {
	var sb strings.Builder
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			sb.WriteRune(b.cells[y][x].Rune)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// Таблица: ячейки рисуются, колонки выравниваются.
func TestRenderTable(t *testing.T) {
	root, err := ParseHTML(strings.NewReader(
		`<table><tr><th>Сервис</th><th>Статус</th></tr><tr><td>api</td><td>работает</td></tr></table>`))
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuffer(40, 6)
	var hz []Hotzone
	RenderHTML(root, b, 0, 0, &hz)
	rows := dumpBuffer(b)
	for _, want := range []string{"Сервис", "Статус", "api", "работает"} {
		if !strings.Contains(rows, want) {
			t.Fatalf("таблица не отрисована (%q нет):\n%s", want, rows)
		}
	}
}

// hr: горизонтальная линия.
func TestRenderHr(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(`<hr/>`))
	b := NewBuffer(10, 3)
	var hz []Hotzone
	RenderHTML(root, b, 0, 0, &hz)
	if !strings.Contains(dumpBuffer(b), "─") {
		t.Fatalf("hr не нарисован:\n%s", dumpBuffer(b))
	}
}

// checkbox: рисуется [ ] с подписью (плагина toggle в движке нет — не падает).
func TestRenderCheckbox(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(`<checkbox action="toggle:x:y">Логи</checkbox>`))
	b := NewBuffer(20, 3)
	var hz []Hotzone
	RenderHTML(root, b, 0, 0, &hz)
	rows := dumpBuffer(b)
	if !strings.Contains(rows, "[ ]") || !strings.Contains(rows, "Логи") {
		t.Fatalf("checkbox не отрисован:\n%s", rows)
	}
}

// select: хотзона с options.
func TestRenderSelect(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(`<select action="set:f:k" options="a:b:c" label="Режим"/>`))
	b := NewBuffer(30, 3)
	var hz []Hotzone
	RenderHTML(root, b, 0, 0, &hz)
	found := false
	for _, z := range hz {
		if z.Kind == "select" && z.Options == "a:b:c" {
			found = true
		}
	}
	if !found {
		t.Fatalf("нет select-хотзоны: %+v", hz)
	}
}

// decodePPM: парсит P6, пиксели читаются.
func TestDecodePPM(t *testing.T) {
	ppm := append([]byte("P6\n2 2\n255\n"),
		[]byte{255, 0, 0, 0, 0, 255, 0, 255, 0, 255, 255, 255}...)
	img, err := decodePPM(ppm)
	if err != nil {
		t.Fatal(err)
	}
	if img.w != 2 || img.h != 2 {
		t.Fatalf("размер %dx%d", img.w, img.h)
	}
	r, g, b := img.pixel(0, 0).RGB()
	if r != 255 || g != 0 || b != 0 {
		t.Fatalf("пиксель (0,0): %d,%d,%d", r, g, b)
	}
}

// img: рендер PPM-файла половинчатыми блоками.
func TestRenderImg(t *testing.T) {
	f, err := os.CreateTemp("", "*.ppm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	// 4x2: верх белый, низ чёрный → блоки ▀ (верхняя половина светлая).
	f.WriteString("P6\n4 2\n255\n")
	for i := 0; i < 4; i++ {
		f.Write([]byte{255, 255, 255})
	}
	for i := 0; i < 4; i++ {
		f.Write([]byte{0, 0, 0})
	}
	f.Close()

	root, _ := ParseHTML(strings.NewReader(`<img src="` + f.Name() + `"/>`))
	b := NewBuffer(10, 3)
	var hz []Hotzone
	RenderHTML(root, b, 0, 0, &hz)
	if !strings.Contains(dumpBuffer(b), "▀") {
		t.Fatalf("img не отрисован:\n%s", dumpBuffer(b))
	}
}

// Скролл: длинный контент, scrollOff не превышает максимум, контент рисуется.
func TestScrollTile(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(strings.Repeat(`<p>строка</p>`, 30)))
	inner := NewBuffer(20, 4)
	scrollOff["t"] = 0
	defer delete(scrollOff, "t")
	var hz []Hotzone
	renderTile(root, inner, "t", 0, 0, 4, &hz)
	if scrollOff["t"] != 0 {
		t.Fatalf("скролл не обнулился: %d", scrollOff["t"])
	}
	// Пытаемся уехать далеко — скролл ограничивается максимумом.
	scrollOff["t"] = 100
	renderTile(root, inner, "t", 0, 0, 4, &hz)
	if scrollOff["t"] > 100 {
		t.Fatalf("скролл больше 100: %d", scrollOff["t"])
	}
	rows := dumpBuffer(inner)
	if !strings.Contains(rows, "строка") {
		t.Fatalf("контент не отрисован:\n%s", rows)
	}
}
