package engine

import (
	"testing"
	"testing/fstest"
)

// TestConfirmBlocksMouse — модалка подтверждения блокирует клики мыши:
// клик по кнопке ПОЗАДИ модалки не выполняет действие, а клик по «Да» — подтверждает.
func TestConfirmBlocksMouse(t *testing.T) {
	var calls int
	AddPlugin("__confirm_probe", func(in []string, args []string) ([]string, error) {
		calls++
		return nil, nil
	})

	// Клик по обычной кнопке позади модалки — игнорируется.
	hotzones = hotzones[:0]
	hotzones = append(hotzones, Hotzone{X: 10, Y: 10, W: 5, H: 1, Action: "__confirm_probe"})
	mouseBtn1 = false
	confirmMode = true
	route := "/main"
	handleMouseEvent(MouseEvent{X: 12, Y: 10, Left: true}, Pages{}, &route, 40, 20)
	if calls != 0 {
		t.Fatalf("клик при confirmMode выполнил действие: calls=%d", calls)
	}

	// Клик по кнопке «Да» модалки — подтверждает и выполняет отложенные шаги.
	hotzones = hotzones[:0]
	hotzones = append(hotzones, Hotzone{X: 10, Y: 10, W: 5, H: 1, Kind: "confirm_yes"})
	mouseBtn1 = false
	confirmMode = true
	pendingSteps = []string{"__confirm_probe"}
	handleMouseEvent(MouseEvent{X: 12, Y: 10, Left: true}, Pages{}, &route, 40, 20)
	if calls != 1 {
		t.Fatalf("клик по «Да» не подтвердил: calls=%d", calls)
	}
	if confirmMode {
		t.Fatal("confirmMode остался открытым после «Да»")
	}

	// Сброс глобалов.
	confirmMode = false
	mouseBtn1 = false
	hotzones = hotzones[:0]
}

// TestSelectQuotesLabel — выбранное значение select оборачивается в кавычки:
// пайп | в подписи не разрывает action, плагин получает значение целиком.
func TestSelectQuotesLabel(t *testing.T) {
	var got []string
	AddPlugin("__select_probe", func(in []string, args []string) ([]string, error) {
		got = args
		return nil, nil
	})

	selectStack = []selLevel{{nodes: []*selNode{{label: "значение|с пайпом"}}}}
	selectMode = true
	selectAction = "__select_probe"
	selectOutput = ""
	selectValue = map[string]string{}
	selectOption(0, 0)

	if len(got) != 1 || got[0] != "значение|с пайпом" {
		t.Fatalf("плагин получил args=%v, ждали ['значение|с пайпом']", got)
	}
	// Сброс глобалов.
	selectMode = false
	selectStack = nil
}

// TestExpandVarsDigitNotVar — $1/$2 (поля sed/awk) НЕ трактуются как переменные,
// а настоящая переменная $host подставляется.
func TestExpandVarsDigitNotVar(t *testing.T) {
	if got := expandVars("sed:s/$1//g"); got != "sed:s/$1//g" {
		t.Fatalf("$1 изменился: %q", got)
	}
	if got := expandVars("awk '{print $2}'"); got != "awk '{print $2}'" {
		t.Fatalf("$2 изменился: %q", got)
	}
	SetVar("host", []string{"srv1"})
	if got := expandVars("ssh:root:$host"); got != "ssh:root:srv1" {
		t.Fatalf("$host не подставился: %q", got)
	}
}

// TestBackgroundSkipsNonInterval — фоновый рендер неактивных страниц запускает
// ТОЛЬКО плагины с явным interval; разовые плагины (без interval) пропускает.
func TestBackgroundSkipsNonInterval(t *testing.T) {
	var calls int
	AddPlugin("__bg_probe", func(in []string, args []string) ([]string, error) {
		calls++
		return nil, nil
	})

	// Без interval — в фоне не запускается.
	fsys := fstest.MapFS{
		"tiles/bg.html": &fstest.MapFile{Data: []byte(`<plugin pipe="__bg_probe"/>`)},
	}
	pages := Pages{"/bg": []Tile{{ID: "b", X: "0%", Y: "0%", W: "100%", H: "100%", File: "tiles/bg.html"}}}
	renderBackgroundPages(pages, "/active", 40, 10, fsys)
	if calls != 0 {
		t.Fatalf("фон запустил плагин без interval: calls=%d", calls)
	}

	// С interval — запускается (живой виджет).
	fsys2 := fstest.MapFS{
		"tiles/bg2.html": &fstest.MapFile{Data: []byte(`<plugin pipe="__bg_probe" interval="2s"/>`)},
	}
	pages2 := Pages{"/bg2": []Tile{{ID: "b2", X: "0%", Y: "0%", W: "100%", H: "100%", File: "tiles/bg2.html"}}}
	renderBackgroundPages(pages2, "/active", 40, 10, fsys2)
	if calls != 1 {
		t.Fatalf("фон НЕ запустил плагин с interval: calls=%d", calls)
	}
}

// TestFocusResetOnNav — переход по ссылке (мышь) сбрасывает фокус.
func TestFocusResetOnNav(t *testing.T) {
	pages := Pages{"/a": nil, "/b": nil}
	hotzones = hotzones[:0]
	hotzones = append(hotzones, Hotzone{X: 5, Y: 5, W: 3, H: 1, Href: "/b", Kind: "nav"})
	mouseBtn1 = false
	route := "/a"
	focusIdx = 3

	handleMouseEvent(MouseEvent{X: 6, Y: 5, Left: true}, pages, &route, 40, 20)
	if route != "/b" {
		t.Fatalf("маршрут не сменился: %q", route)
	}
	if focusIdx != -1 {
		t.Fatalf("фокус не сброшен при переходе: %d", focusIdx)
	}

	// Сброс глобалов.
	focusIdx = -1
	mouseBtn1 = false
	hotzones = hotzones[:0]
}
