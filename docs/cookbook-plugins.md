# Кукингбук: плагины

Плагин — это **юникс-команда**: строки на входе, строки на выходе. Экран и мышь
он не трогает — это чистая функция. Движок зовёт плагин по имени из HTML:
`<button action="имя:арг1:арг2">`.

## Контракт

```go
type PluginFunc func(in []string, args []string) ([]string, error)
```

- `in` — строки с предыдущего шага пайпа (или nil, если плагин первый);
- `args` — аргументы из вызова (`имя:арг1:арг2`);
- возврат — строки результата и ошибка.

## Шаблон плагина

```go
// plugins/mycmd/mycmd.go
package mycmd

import "rough"

// man_mycmd — справка по плагину (обязательно, для man).
const man_mycmd = `mycmd — что делает.

Использование:
  action="mycmd:АРГ"

Примеры:
  action="mycmd:x"`

func init() {
	// 1) Справка (обязательное правило).
	rough.AddMan("mycmd", man_mycmd)
	// 2) Сам плагин.
	rough.AddPlugin("mycmd", func(in []string, args []string) ([]string, error) {
		return []string{"готово"}, nil
	})
}
```

## Подключение плагина

Движок сам ничего не ищет — плагин **сам записывается в реестр** в `init()`.
Чтобы он попал в бинарник, добавь его в агрегатор:

```go
// plugins/plugins.go
package plugins

import _ "rough/plugins/mycmd" // ← добавил плагин — добавил строчку
```

А в своём проекте подключи весь пакет `plugins` одним импортом
(см. [cookbook-project](cookbook-project.md)).

## Рецепты

### Показать файл (как cat)

```go
rough.AddPlugin("cat", func(in []string, args []string) ([]string, error) {
	b, err := os.ReadFile(args[0])
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), nil
})
```

### Фильтр строк (как grep) — часть пайпа

```go
rough.AddPlugin("grep", func(in []string, args []string) ([]string, error) {
	re, err := regexp.Compile(args[0])
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ln := range in {
		if re.MatchString(ln) {
			out = append(out, ln)
		}
	}
	return out, nil
})
```

### Переключить флаг в конфиге (как toggle)

```go
rough.AddPlugin("toggle", func(in []string, args []string) ([]string, error) {
	// args = [файл, ключ]; читаем key=value, инвертируем 0↔1/on↔off, пишем обратно
	return toggleKey(args[0], args[1])
})
```

### Сетевой плагин (как curl) — нативно в Go

```go
rough.AddPlugin("curl", func(in []string, args []string) ([]string, error) {
	url := strings.Join(args, ":") // URL содержит «:», склеиваем обратно
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(url)
	// ... тело → строки
})
```

### SSH (как ssh) — golang.org/x/crypto/ssh

Ключи из `~/.ssh` по дефолту, или `--keys=ПУТЬ` для своей папки/файла ключей.

### Рисовалка с адаптивом (как bars/clock)

Плагину доступен реальный размер окна/колонки:

```go
rough.AddPlugin("clock", func(in []string, args []string) ([]string, error) {
	now := time.Now()
	return []string{now.Format("02.01.2006"), now.Format("15:04:05")}, nil
})
```

А чтобы подстраиваться под ширину — `engine.Window()`:

```go
import "rough/engine"

w, h := engine.Window() // размер тайла/колонки в клетках
```

Поменял `width="50%"` на `40%` — плагин через `Window()` сам стал уже.

## Гибридные параметры (двоеточия + `--флаги`)

Плагин объявляет параметры один раз, а `engine.ParseArgs` сам разберёт любую
форму ввода. Порядок объявления = позиция при вводе двоеточиями, `Name` — имя
для флага `--имя=значение`, `Default` — дефолт (пусто = нет), `Required` —
обязательный.

```go
// plugins/chart/chart.go
import "rough/engine"

var chartParams = []engine.Param{
	{Name: "мин", Required: true},  // обязательный
	{Name: "макс", Required: true}, // обязательный
	{Name: "ширина", Default: "1"}, // дефолт, если не задан
	{Name: "секунд", Default: "2"}, // дефолт
	{Name: "заголовок"},            // необязательный, без дефолта
}

rough.AddPlugin("chart", func(in []string, args []string) ([]string, error) {
	vals, err := engine.ParseArgs(args, chartParams)
	if err != nil {
		return nil, err
	}
	lo, _ := strconv.ParseFloat(vals["мин"], 64)
	hi, _ := strconv.ParseFloat(vals["макс"], 64)
	colW, _ := strconv.Atoi(vals["ширина"])
	title := vals["заголовок"]
	// ...
})
```

Все формы работают одинаково:

```
chart:0:100:1:2:CPU                             # двоеточия по порядку
chart --мин=0 --макс=100 --заголовок=CPU        # флаги, порядок не важен
chart:0:100:1:2 --заголовок=CPU                 # микс
chart::1:2 --мин=0 --макс=100                   # пустой слот «:» — флаг/дефолт
chart:0:100                                     # частичный ввод: остальное — дефолты
```

Частичный ввод работает: `fuck:ass` — задаёшь только нужные первые параметры,
остальные уходят в дефолты (или пустое). Что и в каком порядке ждёт плагин —
определяет разраб, отсутствие остальных он тоже обрабатывает сам.

Правила:

- Обязательный параметр без значения — ошибка; неизвестный флаг — ошибка
  (опечатка не проходит молча).
- Последний объявленный параметр «глотает» остаток двоеточий
  (как `ssh:host:команда`, `set:файл:ключ:значение:с:двоеточиями`).
- В `man` обе формы собираются через `engine.ParamsUsage("chart", chartParams)`
  — строки использования не разъезжаются с кодом; во флагах виден дефолт
  (`--ширина=1`), без дефолта — `--имя=ЗНАЧ`.
- У каждого параметра в `man` пишется дефолтное значение, если применимо,
  иначе — «пустое».

## Цветной вывод (цвета из темы)

Плагин может красить свой вывод цветами из темы. Цвета — `color_0`…`color_15`
(палитра терминала) и любые ключи темы. Два способа:

- **маркер в строке**: оберни фрагмент в `\x01{имя}…\x01{}` — движок раскрасит
  его цветом из темы, маркеры невидимы и не ломают ширину:

```go
// chart: красим только столбики, рамку/ось — нет.
return "\x01{color_2}" + bars + "\x01{}" + axis
```

- **движковый `engine.ThemeColor`** для цветов прямо в коде:

```go
import "rough/engine"
c := engine.ThemeColor("color_2", tcell.ColorGreen)
```

В статус-строку и блоки вывода маркеры не просачиваются (движок их вычищает).

## Переменные сессии (запомнить и подставить)

Движок хранит глобальную память — переменные (имя → строки). Запись — плагин `export`,
подстановку `$имя` делает движок в любом action. Плагин может читать/писать сам:

```go
import "rough/engine"

engine.SetVar("имя", []string{"значение"})   // записать
v, ok := engine.GetVar("имя")                // прочитать ([]string, есть ли)
line := engine.VarLine("имя")                // одной строкой (для $подстановки)
```

В HTML это выглядит так:

```html
<!-- сохранить вывод в переменную host -->
<button action="ssh:root:srv1::hostname | export:host">Запомнить хост</button>
<!-- подставить переменную в другой action -->
<button action="ssh:root:$host::uptime">Uptime</button>
```

`export:имя` сохраняет входящие строки и пропускает их дальше по пайпу (как `tee`).
`$имя` / `${имя}` — подстановка; `\$` — литеральный доллар; неизвестная — пусто.

## Пайпы

`action="cat:x | grep:err | head:5"` — выход одного шага идёт на вход следующего.
Разделитель `|`, аргументы через `:`.

## Подтверждение

`| confirm` в конце — движок покажет окно «Выполнить?» (Enter — да, Esc — нет).
Для опасных действий: `<button action="deploy:all | confirm">Раскатать</button>`.

## Справка man

Каждый плагин обязан нести переменную `man_<имя>` и регистрировать её через
`rough.AddMan`. В интерфейсе её показывает плагин `man`:
`action="man"` — список, `action="man:ssh"` — справка по плагину.

Если у плагина есть параметры — в `man` обязательны **обе формы ввода**
(двоеточия и `--флаги`). Строки использования собираются через
`engine.ParamsUsage("имя", params)` — тогда они не разъезжаются с кодом.

## Правила

- Имя плагина = имя команды Linux. Поведение = поведение команды.
- Нативно в Go: **не дёргаем внешние бинарники** — работает в любом контейнере.
- `run` запрещён движком — произвольный запуск команд через кнопку невозможен.
