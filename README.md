# rough — HTML-интерфейсы прямо в терминале

[место для гифки]

**Одна строка:** движок рендерит HTML-разметку в клетки терминала. Пишешь HTML —
получаешь кликабельный интерфейс в любом терминале. Без веб-сервера, без браузера,
без ssh-сервера. Один бинарник — и всё.

## Зачем

Надоело лезть в веб-морды и файлы конфигов, чтобы включить логирование или
посмотреть статус? `rough` — интерфейс, который живёт прямо в консоли, где ты
и так работаешь. Пишешь HTML — админ получает кнопки, чекбоксы и таблицы
в любом терминале.

## Как подключить

Четыре строчки — и интерфейс вшит в твой бинарник:

```go
import (
	"rough"
	_ "example/rough/plugins" // твои плагины
)

//go:embed rough
var roughDir embed.FS

func main() { rough.TUI(roughDir) }
```

[место для гифки]

Запуск: `myproject -tui`. Обычный CLI работает как работал — TUI включается
только по флагу `-tui`. Всё (движок, плагины, темы, данные) компилируется
в **один исполняемый файл** — на проде не нужен ни один внешний файл.

## Философия

- **Движок ничего не умеет.** Всё, что делает интерфейс, — это плагины:
  юникс-команды «строки на входе, строки на выходе».
- **Кнопка = вызов плагина:** `<button action="cat:/etc/hostname">`.
- **Никаких внешних бинарников** — ssh, curl, sqlite — всё нативно в Go,
  работает даже в `scratch`-контейнере.
- **Справка прямо в интерфейсе:** `action="man"` — список, `action="man:ssh"` — по плагину.

## Примеры

### 1. Админка сервера (библиотека)

Свой плагин-редактор конфига, чекбокс логирования и SQL-запросы в sqlite:

```html
<h1 color="#ffcc00">Настройки приложения</h1>

<input action="app:set:loglevel" label="Уровень логов"/>

<checkbox action="toggle:app.conf:logging">Логирование</checkbox>
<checkbox action="toggle:app.conf:debug">Режим отладки</checkbox>

<table>
  <tr><th>Сервис</th><th>Статус</th></tr>
  <tr><td>api</td>    <td><i>работает</i></td></tr>
  <tr><td>worker</td> <td><b>упал</b></td></tr>
</table>

<button action="sql:SELECT count(*) FROM users">Пользователей</button>
```

[место для гифки]

### 2. SSH-оркестратор (standalone)

Кнопка «раскатать апдейт» — циклом по всей подсети, ключи в папке:

```html
<h1>Оркестратор</h1>
<button action="deploy:172.0.0.1:25 | confirm">Раскатать апдейт</button>
<button action="ssh:root@srv1:systemctl status nginx">Статус nginx</button>
<button action="deploy:log | tail:50">Последние логи деплоя</button>
```

Плагин `deploy` на Go сам перебирает `172.0.0.1..172.0.0.25`, ходит по SSH
с ключами из папки и гонит апдейт. `| confirm` спрашивает перед раскаткой.

[место для гифки]

### 3. Маленький ядерный реактор

Графики температуры и автоподдержание зоны:

```html
<h1>Реактор</h1>
<plugin pipe="file:/tmp/core_temp | bars:temp=(\d+)" interval="1s"/>
<plugin pipe="file:/tmp/zone_temp | bars:zone=(\d+)" interval="1s"/>
<button action="reactor:scram | confirm">Аварийная остановка</button>
```

[место для гифки]

Плагин `bars` рисует график из чисел, основной код держит температуру зоны
в целевом диапазоне. Не дай ему перегреться.

## Плагины

Единый контракт — юникс-команда:

```go
rough.AddPlugin("mycmd", func(in []string, args []string) ([]string, error) {
	return []string{"сделано"}, nil
})
```

В HTML: `<button action="mycmd:арг">`. Можно пайпом: `action="cat:x | grep:err"`.
Справка — прямо в интерфейсе: `action="man"`.

## Теги

`h1, p, div, br, center, row, b, i, button, a, input, plugin` — и цвета
`color`/`bg` на любом теге. На очереди: `checkbox`, `table`, `select`, `img`, скролл.

## Документация

- `docs/architecture.md` — как всё устроено
- `docs/demo.md` — шаблон проекта
- `docs/systemprompt.md` — правила разработки

