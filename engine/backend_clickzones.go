package engine

import "strings"

// Hotzone — кликабельная зона (кнопка/ссылка/поле ввода) в абсолютных координатах экрана.
type Hotzone struct {
	X, Y, W, H int
	Action     string // что выполнить по клику
	Href       string // куда перейти (роут)
	Kind       string // вид зоны: "" (действие), "nav" (ссылка), "input" (поле ввода), "select"
	Label      string // подпись для окна ввода
	Output     string // id блока, куда направить вывод (пусто = статус-строка)
	Options    string // варианты для select (через ":", подменю — в [квадратных скобках])
	SelLevel   int    // уровень меню select (для хотзон вариантов "selopt")
	SelIdx     int    // индекс варианта в уровне меню select
}

// hotzones — хотзоны последнего кадра, по ним делаем hit-test мыши.
var hotzones []Hotzone

// outputCache — результаты команд, направленных в блоки с id (id блока → строки).
// Блок <div id="..."> или <col id="..."> при рендере рисует эти строки.
var outputCache = map[string][]string{}

// HitTest ищет хотзону по координатам клика.
func HitTest(x, y int) (kind, action, href, label, output, options string) {
	for _, hz := range hotzones {
		if x >= hz.X && x < hz.X+hz.W && y >= hz.Y && y < hz.Y+hz.H {
			return hz.Kind, hz.Action, hz.Href, hz.Label, hz.Output, hz.Options
		}
	}
	return "", "", "", "", "", ""
}

// HitSelect ищет вариант выпадающего меню select по координатам клика.
// Пока меню открыто, карта экрана ПОЛНОСТЬЮ отключена — клик обрабатывается
// только по вариантам меню (хотзоны "selopt"). Всё, что лежит под меню
// (поле ввода, кнопки, сам элемент select), не срабатывает вообще.
// Возвращает уровень и индекс варианта в стеке меню (ok=false — промах).
func HitSelect(x, y int) (level, idx int, ok bool) {
	for i := range hotzones {
		hz := &hotzones[i]
		if hz.Kind != "selopt" {
			continue
		}
		if x >= hz.X && x < hz.X+hz.W && y >= hz.Y && y < hz.Y+hz.H {
			return hz.SelLevel, hz.SelIdx, true
		}
	}
	return 0, 0, false
}

// checkboxOn читает состояние чекбокса: вызывает action с ":get" — плагин
// (например toggle) возвращает текущее значение; true — если включено.
func checkboxOn(act string) bool {
	steps := SplitSteps(act)
	if len(steps) == 0 {
		return false
	}
	out, err := RunSteps([]string{steps[len(steps)-1] + ":get"}, nil)
	if err != nil || len(out) == 0 {
		return false
	}
	v := strings.TrimSpace(out[len(out)-1])
	return v == "1" || strings.EqualFold(v, "on") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// selectValue — выбранные значения select по action (для показа в подписи).
var selectValue = map[string]string{}

// currentSelect возвращает текущее значение select: сначала — выбранное в сессии,
// затем пробует прочитать через плагин (последний шаг action + ":get", как у
// toggle/set). Если не знаем — "?".
func currentSelect(act string) string {
	if v, ok := selectValue[act]; ok && v != "" {
		return v
	}
	steps := SplitSteps(act)
	if len(steps) > 0 {
		if out, err := RunSteps([]string{steps[len(steps)-1] + ":get"}, nil); err == nil && len(out) > 0 {
			return strings.TrimSpace(out[len(out)-1])
		}
	}
	return "?"
}
