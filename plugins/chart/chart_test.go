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

	out, err := engine.RunSteps([]string{"chart:0:100:bars:1:2"}, []string{"50"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 14 {
		t.Fatalf("ожидали 14 строк, получили %d", len(out))
	}
	// Верхняя строка (y=0): подпись «100» справа от оси │ (т.е. ...│100).
	top := out[0]
	if !strings.HasSuffix(top, "│100") {
		t.Fatalf("подпись 100 не справа (ожидали ...│100):\n%s", dump(out))
	}
	// Нижняя строка (ось X): угол ┘ справа, подпись минимума «0» после него.
	bot := strings.TrimRight(out[13], " ")
	if !strings.HasSuffix(bot, "┘0") {
		t.Fatalf("нижняя ось неверна:\n%s", dump(out))
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
