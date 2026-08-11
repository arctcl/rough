# Кукингбук: проект

Как собрать свой проект на rough — от нуля до рабочего интерфейса.

## 1. Подключить (4 строчки)

`main.go`:

```go
package main

import (
	"embed"

	"rough"
	_ "myproject/rough/plugins" // твои плагины (линк на готовые или свои)
)

//go:embed rough
var roughDir embed.FS

func main() { rough.TUI(roughDir) }
```

`go.mod`:

```
module myproject

go 1.25.0

require rough v0.0.0

replace rough => ../rough   // или версия с GitHub
```

## 2. Структура папки rough/

```
rough\
  tiles.json            # страницы, тайлы, вкладки, тема
  tiles\*.html          # вёрстка тайлов
  themes\*.json         # темы
  plugins\plugins.go    # агрегатор: линк на готовые плагины
```

`rough/plugins/plugins.go` — подключить готовый набор плагинов репозитория:

```go
package plugins

import _ "rough/plugins" // cat, hello, ssh, curl, man, grep, tail, head, wc, ...
```

Свои плагины — тоже здесь (см. [cookbook-plugins](cookbook-plugins.md)).

## 3. tiles.json — страницы, тайлы, вкладки

```json
{
  "theme": "default",
  "menu": [["Главная", "/main"], ["Справка", "/man"]],
  "паттерн": ["id", "x", "y", "w", "h", "файл"],
  "/main": [
    ["hello", "0%",  "0%",  "40%", "40%", "tiles/hello.html"],
    ["cfg",   "40%", "0%",  "60%", "100%", "tiles/cfg.html"]
  ]
}
```

- `"menu"` — вкладки внизу (пары [имя, роут]), переключение кликом / Tab / Ctrl+цифры;
- `"паттерн"` — схема строк данных (один раз), дальше только массивы;
- координаты: `%` (от экрана), `px` (клетки), `vw/vh` — доли окна;
- `theme` — имя темы из `themes/`.

## 4. HTML тайлов

Каждый тайл — отдельный HTML-файл. Движок понимает свой набор тегов:

```html
<h1 color="#ffcc00">Настройки</h1>
<p>Кнопки и поля:</p>

<button action="hello">Поздороваться</button>
<button action="cat:/etc/hostname | grep:^host">Hostname</button>
<a href="/man">→ Справка</a>

<input action="man:" output="out" label="Пакет"/>
```

**Формы и данные:**

```html
<!-- чекбокс: [x]/[ ], состояние из плагина toggle -->
<checkbox action="toggle:/etc/app.conf:logging">Логирование</checkbox>

<!-- выпадающий список: клик — меню, выбор дописывает :вариант -->
<select action="set:/etc/app.conf:loglevel" options="info:debug:trace" label="Уровень"/>

<!-- таблица: колонки выравниваются по ширине, th — жирный -->
<table>
  <tr><th>Сервис</th><th>Статус</th></tr>
  <tr><td>api</td><td>работает</td></tr>
</table>

<!-- картинка PPM (P6) половинчатыми блоками ▀▄█ -->
<img src="/opt/app/logo.ppm"/>

<hr/>   <!-- горизонтальная линия -->
```

Контент длиннее тайла — скроллится колесом мыши.

### Колонки и центрирование

```html
<row>
  <div width="50%">левая половина</div>
  <div width="50%">правая половина</div>
</row>

<div align="center"><plugin name="clock" interval="1s"/></div>
```

Плагины внутри колонки видят её реальный размер через `engine.Window()` —
поменял процент, вёрстка перестроилась сама.

### Вывод команды в тайл

`output="id"` направляет результат действия в блок с этим id (в любом тайле):

```html
<input action="man:" output="out" label="Пакет"/>
```
```html
<!-- в другом тайле -->
<div id="out"></div>
```

Без `output` результат уходит в статус-строку внизу.

### Живой контент (плагин по таймеру)

```html
<plugin name="clock" interval="1s"/>
<plugin pipe="cat:data.log | tail:10 | bars" interval="1s"/>
```

## 5. Сборка и запуск

```
go build -o myapp.exe .
myapp.exe -tui
```

- `-tui` — включить интерфейс; без флага CLI работает как обычно;
- всё вшивается в **один бинарник** — на проде не нужен ни один внешний файл;
- терминал должен быть **UTF-8** (Windows: `chcp 65001` или Windows Terminal).

## Примеры готовых проектов

- `example\` — минимальный живой пример;
- `testv001\` — проверочная сборка (справка man с выводом в тайл, инструменты).
