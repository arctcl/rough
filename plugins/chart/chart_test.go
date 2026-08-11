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
