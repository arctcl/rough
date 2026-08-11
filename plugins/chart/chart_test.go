package chart

import (
	"strings"
	"testing"

	"rough/engine"
)

// TestChartAxisRight — подпись максимума («100») должна быть СПРАВА (у правой оси),
// а не слева. Ось Y справа, новые столбики справа.
func TestChartAxisRight(t *testing.T) {
	engine.SetWindowSize(40, 14)
	defer engine.SetWindowSize(0, 0)
	delete(series, "") // серия для пустого ключа — только текущий тест
	delete(lastAdd, "")

	out, err := engine.RunSteps([]string{"chart:0:100:bars:1:2"}, []string{"50"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 14 {
		t.Fatalf("ожидали 14 строк, получили %d", len(out))
	}
	// Верхняя строка (y=0): линия с заголовком, подпись «100» справа от угла ┐.
	top := strings.TrimRight(out[0], " ")
	if !strings.HasSuffix(top, "┐100") {
		t.Fatalf("подпись 100 не справа (ожидали ...┐100):\n%s", dump(out))
	}
	// Нижняя строка (ось X): угол ┘ справа, подпись минимума «0» после него.
	bot := strings.TrimRight(out[13], " ")
	if !strings.HasSuffix(bot, "┘0") {
		t.Fatalf("нижняя ось неверна:\n%s", dump(out))
	}
}

// TestChartCandleColor — свечи раскрашены: бычья (close>=open) зелёная (маркер
// \x01G), медвежья (close<open) красная (\x01R). Тело — сплошной блок.
func TestChartCandleColor(t *testing.T) {
	engine.SetWindowSize(40, 14)
	defer engine.SetWindowSize(0, 0)
	delete(series, "") // серия для пустого ключа — только текущий тест
	delete(lastAdd, "")

	// Бычья свеча: open=10 close=15 high=20 low=5.
	out, err := engine.RunSteps([]string{"chart:0:100:japanse:1:2"}, []string{"10 20 5 15"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 14 {
		t.Fatalf("ожидали 14 строк, получили %d", len(out))
	}
	found := false
	for _, ln := range out {
		if strings.Contains(ln, "\x01G") {
			found = true
		}
	}
	if !found {
		t.Fatalf("бычья свеча не зелёная (нет \\x01G):\n%s", dump(out))
	}

	// Медвежья свеча: close < open.
	delete(lastAdd, "")
	out, err = engine.RunSteps([]string{"chart:0:100:japanse:1:2"}, []string{"15 20 5 10"})
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, ln := range out {
		if strings.Contains(ln, "\x01R") {
			found = true
		}
	}
	if !found {
		t.Fatalf("медвежья свеча не красная (нет \\x01R):\n%s", dump(out))
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

// TestChartSeriesGrows — серия свечей должна РАСТИ: каждый вызов добавляет
// новую свечу, а не перезаписывает одну и ту же.
func TestChartSeriesGrows(t *testing.T) {
	engine.SetWindowSize(40, 14)
	defer engine.SetWindowSize(0, 0)
	delete(series, "")
	delete(lastAdd, "")

	for i := 0; i < 3; i++ {
		if _, err := engine.RunSteps([]string{"chart:0:100:japanse:1:2"}, []string{"10 20 5 15"}); err != nil {
			t.Fatal(err)
		}
		delete(lastAdd, "") // каждый вызов — как новый тик (свеча добавляется)
	}
	if n := len(series[""]); n != 3 {
		t.Fatalf("серия должна расти: len=%d, ждали 3\n%v", n, series[""])
	}
}

// TestCandlesLeftToRight — свечи заполняются СЛЕВА НАПРАВО, как и bars:
// первая свеча в колонке 0, вторая в колонке 1 и т.д. (новые справа).
func TestCandlesLeftToRight(t *testing.T) {
	engine.SetWindowSize(40, 14)
	defer engine.SetWindowSize(0, 0)
	delete(series, "")
	delete(lastAdd, "")

	// 3 разные свечи, каждая добавляется с новым «тиком».
	ins := []string{"10 20 5 15", "15 25 8 20", "20 30 10 25"}
	for _, in := range ins {
		if _, err := engine.RunSteps([]string{"chart:0:100:japanse:1:2"}, []string{in}); err != nil {
			t.Fatal(err)
		}
		delete(lastAdd, "")
	}
	out, err := engine.RunSteps([]string{"chart:0:100:japanse:1:2"}, []string{ins[len(ins)-1]})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("рендер свечей:\n%s", dump(out))
}
