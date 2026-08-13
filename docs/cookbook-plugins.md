# Кукингбук: плагины

## 1. Суть

Плагин — это юникс-команда: строки на входе, строки на выходе. Он не трогает
экран и мышь — это чистая функция. Всё, что умеет интерфейс, делают плагины:
показать файл, отфильтровать лог, переключить флаг в конфиге, зайти по ssh,
нарисовать график. Движок — пустая оболочка: он только зовёт плагин по имени
из HTML и показывает результат.

Живой пример — плагин `cat`: читает файл и отдаёт строки.

```html
<button action="cat:/etc/hostname">Показать hostname</button>
```

Кнопка вызывает `cat` с аргументом `/etc/hostname`, плагин читает файл,
возвращает строки, движок показывает их в статус-строке внизу.

## 2. Контракт

```go
type PluginFunc func(in []string, args []string) ([]string, error)
```

- `in` — строки с предыдущего шага пайпа (или nil, если плагин первый);
- `args` — аргументы из вызова (`имя:арг1:арг2`);
- возврат — строки результата и ошибка. Ошибка показывается в статус-строке,
  а пайп останавливается.

## 3. Первый плагин

Каждый плагин — маленький пакет с `init()`, в котором он сам записывается
в реестр и несёт справку для `man`:

```go
// plugins/mycmd/mycmd.go
package mycmd

import "rough"

// man_mycmd — справка по плагину (обязательно, для man).
const man_mycmd = `mycmd — что делает.

Использование:
  action="mycmd:ARG"

Примеры:
  action="mycmd:x"`

func init() {
	rough.AddMan("mycmd", man_mycmd) // 1) справка (обязательное правило)
	rough.AddPlugin("mycmd", func(in []string, args []string) ([]string, error) {
		return []string{"готово"}, nil // 2) сам плагин
	})
}
```

## 4. Подключение

Движок сам ничего не ищет — плагин записывается в реестр в `init()`. Чтобы он
попал в бинарник, добавь его в агрегатор `plugins/plugins.go` (одна строчка):

```go
// plugins/plugins.go
package plugins

import _ "rough/plugins/mycmd" // ← добавил плагин — добавил строчку
```

А в своём проекте подключи весь пакет `plugins` одним импортом
(см. [cookbook-project](cookbook-project.md)).

## 5. Живой пример: cat

`cat` — самый простой плагин: читает файл, отдаёт строки. `strings.TrimRight`
убирает последний перенос строки, иначе в выводе появится пустая строка внизу:

```go
rough.AddPlugin("cat", func(in []string, args []string) ([]string, error) {
	b, err := os.ReadFile(args[0])
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), nil
})
```

В HTML это кнопки «показать файл»:

```html
<button action="cat:/etc/os-release">Показать os-release</button>
<button action="cat:/var/log/app.log | tail:20">Хвост лога</button>
```

Вторая кнопка сразу с пайпом: `cat` отдаёт строки, `tail` режет до последних 20.

## 6. Обработка текста в пайпе

Большинство плагинов — фильтры и обработчики: берут строки из `in`, меняют,
возвращают. Из таких кирпичиков собираются цепочки.

**Фильтр строк (как grep)** — оставить только строки, где есть регулярка:

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

```html
<button action="cat:/etc/x.conf | grep:^server | head:5">Серверы (5)</button>
```

Рядом живут `head`/`tail` (первые/последние строки), `wc` (счётчики),
`cut`/`awk` (вырезать поля по разделителю), `sed` (замена подстроки). Всё это —
обычные фильтры по контракту выше.

## 7. Конфиги: set и toggle

Конфиги приложений бывают двух форматов: `ключ=значение` (по умолчанию) и
`ключ значение` (через пробел — как у mailcow). Плагин `set` ставит значение
ключа, `toggle` переключает флаг (0↔1, on↔off, true↔false).

```html
<button action="set:/etc/app.conf:loglevel:debug">Уровень: debug</button>
<checkbox action="toggle:app.conf:debug">Отладка</checkbox>
```

Какой символ отделяет ключ от значения — необязательный флаг `--sep=CHAR`.
По умолчанию `=`; для пробела укажи слово `space`:

```html
<button action="set:/etc/mailcow:limit:100 --sep=space">Лимит 100</button>
<checkbox action="toggle:/etc/mailcow:debug --sep=space">Отладка</checkbox>
```

Флаг вырезается из аргументов функцией `engine.FlagValue`, значение
нормализуется `normSep` (пусто/`=` → `=`, `space` → пробел). Такой же `--sep`
понимает плагин `set` (и для чтения `...:file:key:get`). Про сами элементы
в HTML (чекбокс, select) — [cookbook-html-new, раздел 8](cookbook-html-new.md).

## 8. Сеть: curl и ssh

`curl` — HTTP-запрос нативно в Go (никаких внешних бинарников):

```go
rough.AddPlugin("curl", func(in []string, args []string) ([]string, error) {
	url := strings.Join(args, ":") // URL содержит «:», склеиваем обратно
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(url)
	// ... тело → строки
})
```

`ssh` — подключение по golang.org/x/crypto/ssh. Ключи из `~/.ssh` по дефолту,
или `--keys=PATH` для своей папки/файла ключей. Параметры — как у консольной
команды: `ssh:USER:HOST:PORT::COMMAND`:

```html
<button action="ssh:root:srv1::uptime">Uptime на srv1</button>
<button action="ssh:root:srv1 --keys=/root/.ssh/id_ed25519::free -m">Память</button>
```

## 9. Рисование: bars, clock, chart

Рисовалки отличаются от фильтров тем, что им нужен реальный размер окна/колонки.
Плагин получает его через `engine.Window()`:

```go
import "rough/engine"

w, h := engine.Window() // размер тайла/колонки в клетках
```

Поменял `width="50%"` на `40%` — плагин через `Window()` сам стал уже. Так
живут часы, спарклайны и графики:

```go
rough.AddPlugin("clock", func(in []string, args []string) ([]string, error) {
	now := time.Now()
	return []string{now.Format("02.01.2006"), now.Format("15:04:05")}, nil
})
```

```html
<plugin name="clock" interval="1s"/>
<plugin pipe="emu_cpu | chart:0:100:1:2:CPU" height="14" interval="2s"/>
<plugin pipe="cat:data.log | tail:10 | bars" interval="2s"/>
```

## 10. Гибридные параметры (двоеточия + `--флаги`)

Плагин объявляет параметры один раз, а `engine.ParseArgs` сам разберёт любую
форму ввода. Порядок объявления = позиция при вводе двоеточиями, `Name` — имя
для флага `--name=value`, `Default` — дефолт (пусто = нет), `Required` —
обязательный.

```go
// plugins/chart/chart.go
import "rough/engine"

var chartParams = []engine.Param{
	{Name: "min", Required: true},    // обязательный
	{Name: "max", Required: true},    // обязательный
	{Name: "width", Default: "1"},   // дефолт, если не задан
	{Name: "seconds", Default: "2"}, // дефолт
	{Name: "title"},                  // необязательный, без дефолта
}

rough.AddPlugin("chart", func(in []string, args []string) ([]string, error) {
	vals, err := engine.ParseArgs(args, chartParams)
	if err != nil {
		return nil, err
	}
	lo, _ := strconv.ParseFloat(vals["min"], 64)
	hi, _ := strconv.ParseFloat(vals["max"], 64)
	title := vals["title"]
	// ...
})
```

Все формы работают одинаково:

```
chart:0:100:1:2:CPU                             # двоеточия по порядку
chart --min=0 --max=100 --title=CPU             # флаги, порядок не важен
chart:0:100:1:2 --title=CPU                     # микс
chart::1:2 --min=0 --max=100                    # пустой слот «:» — флаг/дефолт
chart:0:100                                     # частичный ввод: остальное — дефолты
```

Частичный ввод работает: `chart:0:100` — задаёшь только нужные первые параметры,
остальные уходят в дефолты. Что и в каком порядке ждёт плагин — определяет
разраб, отсутствие остальных он тоже обрабатывает сам.

Правила:

- Обязательный параметр без значения — ошибка; неизвестный флаг — ошибка
  (опечатка не проходит молча).
- Последний объявленный параметр «глотает» остаток двоеточий
  (как `ssh:host:command`, `set:file:key:value:with:colons`).
- В `man` обе формы собираются через `engine.ParamsUsage("chart", chartParams)`
  — строки использования не разъезжаются с кодом; во флагах виден дефолт
  (`--width=1`), без дефолта — `--name=VAL`.

## 11. Цветной вывод (цвета из темы)

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

## 12. Переменные сессии (запомнить и подставить)

Движок хранит глобальную память — переменные (имя → строки). Запись — плагин
`export`, подстановку `$имя` делает движок в любом action. Плагин может
читать/писать сам:

```go
import "rough/engine"

engine.SetVar("name", []string{"value"})   // записать
v, ok := engine.GetVar("name")              // прочитать ([]string, есть ли)
line := engine.VarLine("name")              // одной строкой (для $подстановки)
```

В HTML это выглядит так:

```html
<!-- сохранить вывод в переменную host -->
<button action="ssh:root:srv1::hostname | export:host">Запомнить хост</button>
<!-- подставить переменную в другой action -->
<button action="ssh:root:$host::uptime">Uptime</button>
```

`export:имя` сохраняет входящие строки и пропускает их дальше по пайпу (как
`tee`). `$имя` / `${имя}` — подстановка; `\$` — литеральный доллар; неизвестная
переменная — пусто.

## 13. Пайпы и подтверждение

`action="cat:x | grep:err | head:5"` — выход одного шага идёт на вход следующего.
Разделитель `|`, аргументы через `:`.

`| confirm` в конце — движок покажет окно «Выполнить?» (Enter — да, Esc — нет).
Для опасных действий: `<button action="deploy:all | confirm">Раскатать</button>`.

## 14. Справка man

Каждый плагин обязан нести переменную `man_<имя>` и регистрировать её через
`rough.AddMan`. В интерфейсе её показывает плагин `man`: `action="man"` —
список, `action="man:ssh"` — справка по плагину.

Если у плагина есть параметры — в `man` обязательны обе формы ввода
(двоеточия и `--флаги`). Строки использования собираются через
`engine.ParamsUsage("имя", params)` — тогда они не разъезжаются с кодом.

## 15. Правила

- Имя плагина = имя команды Linux. Поведение = поведение команды.
- Нативно в Go: не дёргаем внешние бинарники — работает в любом контейнере.
- `run` запрещён движком — произвольный запуск команд через кнопку невозможен.
