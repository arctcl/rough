package engine

import "strings"

// Hotzone — кликабельная зона (кнопка/ссылка/поле ввода) в абсолютных координатах экрана.
type Hotzone struct {
	X, Y, W, H int
	Action     string   // что выполнить по клику (первый action)
	Actions    []string // все action="..." по порядку (если их несколько)
	Href       string   // куда перейти (роут)
	Kind       string   // вид зоны: "" (действие), "nav" (ссылка), "input" (поле ввода), "select"
	Label      string   // подпись для окна ввода
	Output     string   // id блока, куда направить вывод (пусто = статус-строка)
	Async      bool     // async: действие выполняется в фоновой горутине (не блокирует ядро)
	Options    string   // варианты для select (через ":", подменю — в [квадратных скобках])
	SelLevel   int      // уровень меню select (для хотзон вариантов "selopt")
	SelIdx     int      // индекс варианта в уровне меню select
}

// hotzones — хотзоны последнего кадра, по ним делаем hit-test мыши.
var hotzones []Hotzone

// outputCache — результаты команд, направленных в блоки с id (id блока → строки).
// Блок <div id="..."> или <col id="..."> при рендере рисует эти строки.
var outputCache = map[string][]string{}

// HitTest ищет хотзону по координатам клика (nil — промах).
func HitTest(x, y int) *Hotzone {
	for i := range hotzones {
		hz := &hotzones[i]
		if x >= hz.X && x < hz.X+hz.W && y >= hz.Y && y < hz.Y+hz.H {
			return hz
		}
	}
	return nil
}

// runHotzone выполняет действия хотзоны: одна кнопка может нести несколько
// атрибутов action="..." — выполняются последовательно в один приёмник.
// Если Actions пусто — выполняется одиночный Action (старое поведение).
// Если у хотзоны async — действие уходит в фоновую горутину (см. async.go).
func runHotzone(hz *Hotzone) {
	if len(hz.Actions) > 0 {
		for _, a := range hz.Actions {
			if a != "" {
				runAction(hz, a)
			}
		}
		return
	}
	if hz.Action != "" {
		runAction(hz, hz.Action)
	}
}

// runAction выполняет одно действие хотзоны: синхронно (по умолчанию) или
// в фоне (если async).
func runAction(hz *Hotzone, a string) {
	if hz.Async {
		execActionAsync(a, hz.Output)
		return
	}
	execAction(a, hz.Output)
}

// execActionAsync выполняет действие в фоне: ключ = РАСКРЫТАЯ команда.
// Повторный клик на «занятой» async-кнопке игнорируется (защита от двойного
// запуска). Пока поддерживаем один пайп без confirm.
func execActionAsync(raw, target string) {
	key := expandVars(raw)
	steps, need := PrepareAction(raw)
	if need || len(steps) == 0 {
		// confirm или пусто — пока не async: выполняем синхронно.
		execAction(raw, target)
		return
	}
	startAsyncJob(key, target, steps[0])
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

// stateCache — кэш состояния чекбоксов/селектов (action+get → значение).
// Плагин :get вызывается ОДИН раз и не дёргается на каждый кадр; кэш
// сбрасывается при выполнении действия (execAction) — после этого состояние
// перечитывается. Для сверхлёгкой отрисовки.
var stateCache = map[string]string{}

// clearStateCache сбрасывает кэш состояния (вызывается при действии).
func clearStateCache() {
	stateCache = map[string]string{}
}

// checkboxOn читает состояние чекбокса: вызывает action с ":get" — плагин
// (например toggle) возвращает текущее значение; true — если включено.
// Значение кэшируется (см. stateCache).
func checkboxOn(act string) bool {
	steps := SplitSteps(act)
	if len(steps) == 0 {
		return false
	}
	key := steps[len(steps)-1] + ":get"
	v, ok := stateCache[key]
	if !ok {
		out, err := RunSteps([]string{key}, nil)
		if err != nil || len(out) == 0 {
			return false
		}
		v = strings.TrimSpace(out[len(out)-1])
		stateCache[key] = v
	}
	return v == "1" || strings.EqualFold(v, "on") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// selectValue — выбранные значения select по action (для показа в подписи).
var selectValue = map[string]string{}

// currentSelect возвращает текущее значение select: сначала — выбранное в сессии,
// затем — из кэша состояния, и только если ничего нет — читает через плагин
// (последний шаг action + ":get", как у toggle/set). Если не знаем — "?".
func currentSelect(act string) string {
	if v, ok := selectValue[act]; ok && v != "" {
		return v
	}
	steps := SplitSteps(act)
	if len(steps) > 0 {
		key := steps[len(steps)-1] + ":get"
		if v, ok := stateCache[key]; ok {
			return v
		}
		if out, err := RunSteps([]string{key}, nil); err == nil && len(out) > 0 {
			v := strings.TrimSpace(out[len(out)-1])
			stateCache[key] = v
			return v
		}
	}
	return "?"
}
