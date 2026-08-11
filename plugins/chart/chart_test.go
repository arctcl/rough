package chart

import (
	"strings"
	"testing"

	"rough/engine"
)

// TestChartAxisRight — подпись максимума («100») должна быть СПРАВА (у правой оси).
func TestChartAxisRight(t *testing.T) {
	engine.SetWindowSize(40, 14)
	defer engine.SetWindowSize(0, 0)
	delete(series, "")
	delete(lastAdd, "")

	out, err := engine.RunSteps([]string{"chart:0:100:1:2"}, []string{"50"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 14 {
		t.Fatalf("ожидали 14 строк, получили %d", len(out))
	}
	// Верхняя строка: линия, угол ┐, подпись «100» справа.
	top := strings.TrimRight(out[0], " ")
	if !strings.HasSuffix(top, "┐100") {
		t.Fatalf("подпись 100 не справа (ожидали ...┐100):\n%s", dump(out))
	}
	// Нижняя строка: угол ┘ справа, подпись минимума «0» после него.
	bot := strings.TrimRight(out[13], " ")
	if !strings.HasSuffix(bot, "┘0") {
		t.Fatalf("нижняя ось неверна:\n%s", dump(out))
	}
}

// TestChartFlags — гибридный ввод: те же параметры через --флаги (и микс).
func TestChartFlags(t *testing.T) {
	engine.SetWindowSize(40, 14)
	defer engine.SetWindowSize(0, 0)
	delete(series, "CPU")
	delete(lastAdd, "CPU")

	// Только флаги: мин/макс обязательные, заголовок по имени.
	out, err := engine.RunSteps([]string{"chart --мин=0 --макс=100 --заголовок=CPU"}, []string{"50"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 14 {
		t.Fatalf("ожидали 14 строк, получили %d", len(out))
	}
	if !strings.Contains(out[0], "CPU") {
		t.Fatalf("заголовок CPU не на верхней линии:\n%s", dump(out))
	}
	// Микс: позиционные + флаг, заголовок по имени.
	delete(series, "MEM")
	delete(lastAdd, "MEM")
	out2, err := engine.RunSteps([]string{"chart:0:100:1:2 --заголовок=MEM"}, []string{"50"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2[0], "MEM") {
		t.Fatalf("микс: заголовок MEM не найден:\n%s", dump(out2))
	}
	// Ошибка: обязательный мин не задан.
	if _, err := engine.RunSteps([]string{"chart --макс=100"}, []string{"50"}); err == nil {
		t.Fatal("нужна ошибка: обязательный параметр мин не задан")
	}
}

// TestChartSeriesGrows — серия должна расти: каждый вызов добавляет столбик.
func TestChartSeriesGrows(t *testing.T) {
	engine.SetWindowSize(40, 14)
	defer engine.SetWindowSize(0, 0)
	delete(series, "")
	delete(lastAdd, "")

	for i := 0; i < 3; i++ {
		if _, err := engine.RunSteps([]string{"chart:0:100:1:2"}, []string{"50"}); err != nil {
			t.Fatal(err)
		}
		delete(lastAdd, "") // каждый вызов — как новый тик
	}
	if n := len(series[""]); n != 3 {
		t.Fatalf("серия должна расти: len=%d, ждали 3\n%v", n, series[""])
	}
}

// TestChartLeftToRight — столбики заполняются слева направо: первый слева,
// новые справа (в колонках 0, 1, 2...).
func TestChartLeftToRight(t *testing.T) {
	engine.SetWindowSize(40, 14)
	defer engine.SetWindowSize(0, 0)
	delete(series, "")
	delete(lastAdd, "")

	for i := 0; i < 3; i++ {
		if _, err := engine.RunSteps([]string{"chart:0:100:1:2"}, []string{"50"}); err != nil {
			t.Fatal(err)
		}
		delete(lastAdd, "")
	}
	out, err := engine.RunSteps([]string{"chart:0:100:1:2"}, []string{"50"})
	if err != nil {
		t.Fatal(err)
	}
	// В области столбики начинаются с нижней части: строка y=7 (первая из 4).
	// 4 столбика по ширине 1 — префикс "████" (слева направо).
	row := out[7]
	if !strings.HasPrefix(row, "████") {
		t.Fatalf("столбики не слева направо (ожидали ████ в начале строки):\n%s", dump(out))
	}
}

func dump(rows []string) string {
	var sb strings.Builder
	for _, r := range rows {
		sb.WriteString(r)
		sb.WriteByte('\n')
	}
	return sb.String()
}
