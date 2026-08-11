# Кукингбук: HTML

Всё, что можно нарисовать в тайле. Идём от простого к сложному — к концу ты
соберёшь целую панель управления, которая по SSH гоняет команды по куче серверов.

> **Главное правило**: тайл — это обычный HTML-файл (`tiles/что-то.html`).
> Движок понимает свой набор тегов. Всё остальное — это «кнопки», которые
> вызывают плагины через `action="имя:арг1:арг2"`.
>
> **Плагин = юникс-команда**: строки на входе, строки на выходе. Соединяй их
> пайпом `|` как в шелле.

---

## 1. Текст: h1, p, br, b, i, center

```html
<h1 color="#ffcc00">Настройки</h1>   <!-- заголовок, цвет — необязательно -->
<p>Обычный абзац.</p>
<br/>                                  <!-- пустая строка -->
<b>жирный</b> <i>курсив</i>            <!-- стили -->
<center>по центру</center>             <!-- центрирование строки -->
<div>блок текста</div>                 <!-- просто блок -->
```

Текст можно лить прямо в тайл без тегов — отрисуется как есть.

---

## 2. Кнопки: button

Кнопка вызывает плагин. Аргументы — через `:`.

```html
<!-- hello без аргументов -->
<button action="hello">Поздороваться</button>

<!-- hello с аргументом -->
<button action="hello:мир">Поздороваться (с аргументом)</button>

<!-- cat: показать файл с диска -->
<button action="cat:/etc/hostname">Показать hostname</button>
<button action="cat:/etc/os-release">Показать os-release</button>
```

Результат без `output` уходит в **статус-строку** внизу экрана.

---

## 3. Пайпы: вывод одной команды → вход другой

Пайп `|` — как в шелле: выход одного шага идёт на вход следующего.

```html
<!-- cat → tail: последние 20 строк лога -->
<button action="cat:/var/log/app.log | tail:20">Хвост лога</button>

<!-- cat → grep: только строки, начинающиеся с server -->
<button action="cat:/etc/x.conf | grep:^server">Серверы из конфига</button>

<!-- cat → head: первые 3 строки -->
<button action="cat:/etc/os-release | head:3">Первые строки</button>

<!-- cat → grep → head: цепочка -->
<button action="cat:/etc/app.conf | grep:^server | head:5">Серверы (5 шт)</button>

<!-- curl → grep: ищем "ok" в ответе API -->
<button action="curl:https://api.example.com | grep:ok">Статус API</button>

<!-- curl → head: первые 5 строк ответа -->
<button action="curl:https://example.com | head:5">Сайт (5 строк)</button>

<!-- curl → wc: сколько строк в ответе -->
<button action="curl:https://example.com | wc">Строк в ответе</button>

<!-- man: справка по плагину ssh, первые 10 строк -->
<button action="man:ssh | head:10">Справка ssh</button>
```

Справочник всех плагинов и их вызовов — в разделе 18 внизу.

---

## 4. Вывод в тайл: output="id"

Без `output` результат уходит в статус-строку. Если добавить `output="id"` —
результат уйдёт в **блок с этим id**, где бы он ни был: в этом тайле, в соседнем,
даже в другом роуте.

### Пример: поле ввода + вывод рядом

Тайл `man_in.html` — поле ввода:

```html
<h1>Справка</h1>
<p>Введи имя команды:</p>

<!--
  input: пользователь вводит "ssh", движок дописывает к action:
  action="man:" + "ssh" → выполняется "man:ssh".
  output="out" — результат пойдёт в блок id="out".
-->
<input action="man:" output="out" label="Пакет"/>
<p>Например: ssh, cat, curl, bars, grep</p>
```

Тайл `man_out.html` — блок вывода:

```html
<!-- сюда упадёт результат: man:ssh, man:cat, ... -->
<div id="out"></div>
```

Оба тайла — на одной странице (`tiles.json`), и вывод появляется в правом тайле,
пока ты печатаешь в левом.

### Поле ввода + кнопка в одну строчку, по центру

```html
<br/><br/><br/>
<row>
  <div width="15%"></div>                      <!-- левый отступ -->
  <div><input action="man:" output="out" label="Пакет"/></div>
  <div><button action="man" output="out">Документация</button></div>
  <div width="15%"></div>                      <!-- правый отступ -->
</row>
```

Поля-отступы — просто пустые `<div width="...">` внутри `<row>`.

---

## 5. Живой контент: plugin по таймеру

`<plugin>` сам себя обновляет по таймеру — без кликов.

```html
<!-- часы, обновление раз в секунду -->
<div align="center"><plugin name="clock" interval="1s"/></div>

<!-- живой график: cat файла → tail → bars, каждую секунду -->
<plugin pipe="cat:data.log | tail:10 | bars" interval="1s"/>

<!-- график CPU по маске (группа-число), каждую секунду -->
<plugin pipe="file:/tmp/cpu.log | bars:cpu=(\d+)" interval="1s"/>

<!-- ssh → bars: нагрузка на сервере живым графиком -->
<plugin pipe="ssh:root@srv:cat /proc/loadavg | bars" interval="1s"/>
```

`bars` подстраивается под ширину тайла/колонки через `engine.Window()`:
поменял `width="50%"` на `40%` — график сам стал уже.

---

## 6. Колонки и центрирование: row, div width, align

```html
<row>
  <div width="50%">левая половина</div>
  <div width="50%">правая половина</div>
</row>

<row>
  <div width="30%">левый столбец</div>
  <div width="40%">середина</div>
  <div width="30%">правый столбец</div>
</row>

<div align="center"><button action="hello">Кнопка по центру</button></div>
```

- `width` — в `%`, `px` (клетки) или `vw/vh`.
- Пустые `<div width="x%">` — это отступы.
- Плагин внутри колонки видит её реальный размер через `engine.Window()`.

---

## 7. Чекбокс: checkbox

`[x]` / `[ ]` — переключатель. Состояние читается у плагина автоматически
(к вызову дописывается `:get`), клик — переключает.

```html
<!--
  toggle:файл:ключ — переключает 0↔1/on↔off в конфиге вида ключ=значение.
  Движок сам спросит текущее значение (toggle:файл:ключ:get) и нарисует [x]/[ ].
-->
<checkbox action="toggle:app.conf:debug">Отладка</checkbox>
<checkbox action="toggle:/etc/app.conf:logging">Логирование</checkbox>
```

Файл `app.conf`:

```
debug=1
logging=0
```

Клик по «Логирование» → `logging=1`, галочка появится. Клик ещё раз → `logging=0`.

---

## 8. Выпадающий список: select

Клик — выпадающее меню ПОД элементом (как в HTML): стрелки + Enter выбирают,
Esc или клик мимо — закрыть. Выбор дописывается к вызову через `:`. Можно
кликнуть вариант мышью.

```html
<!--
  set:файл:ключ:значение — ставит значение ключа в конфиге.
  options="info:debug:trace" — варианты, разделённые ":". Выбор дописывается:
  выбрал debug → выполнится set:/etc/app.conf:loglevel:debug
-->
<select action="set:/etc/app.conf:loglevel" options="info:debug:trace" label="Уровень"/>

<select action="set:/etc/app.conf:max_users" options="10:50:100:500" label="Лимит"/>
```

---

## 9. Таблица: table

```html
<table>
  <tr><th>Сервис</th><th>Статус</th></tr>   <!-- th — жирный заголовок -->
  <tr><td>api</td><td>работает</td></tr>
  <tr><td>db</td><td>работает</td></tr>
  <tr><td>queue</td><td>стоп</td></tr>
</table>
```

Колонки выравниваются по самой широкой ячейке, между ними разделитель `│`.

---

## 10. Картинка: img (PPM)

```html
<!-- картинка PPM (P6) рисуется половинчатыми блоками ▀▄█ с цветами -->
<img src="/opt/app/logo.ppm"/>

<!-- ширина подстроится под тайл -->
<img src="/opt/app/diagram.ppm"/>
```

Масштаб — по ширине тайла/колонки. Формат пока один: **PPM P6** (бинарный).

---

## 11. Линии и блоки: hr, pre

```html
<hr/>                        <!-- горизонтальная линия на всю ширину -->

<pre>                        <!-- моноширинный блок (для логов, ASCII-арта) -->
   ____              __
  / __ \__  ______  / /_
 / /_/ / / / / __ \/ __/
/ ____/ /_/ / /_/ / /_
/_/    \__, / .___/\__/
      /____/_/
</pre>
```

`pre` не переносит строки и не ломает пробелы — идеально для логов и арта.

---

## 12. Ссылки: a

Переход между роутами (страницами):

```html
<a href="/main">← На главную</a>
<a href="/man">→ Справка</a>
<a href="/tools">Инструменты</a>
```

Роуты задаются в `tiles.json` (см. [cookbook-project](cookbook-project.md)).
Переключение — клик, Tab, или Ctrl+Tab / Ctrl+цифра (как вкладки браузера).

---

## 13. Скролл

Контент длиннее тайла — **скроллится колесом мыши**. Кнопки, поля и хотзоны
сдвигаются вместе с контентом, так что всё остаётся кликабельным. Справа
в тайле появляется **полоса прокрутки** (бегунок `█` на треке `│`) — видно,
сколько контента ещё ниже. Статус-блок тоже показывает полосу, если строк
больше трёх.

Просто не думай об этом: напиши больше строк, чем влезает, — и прокрутишь.

---

## 14. Подтверждение опасных действий: | confirm

`| confirm` в конце — движок покажет окно «Выполнить?» (Enter — да, Esc — нет).

```html
<button action="deploy:all | confirm">Раскатать прод</button>
<button action="loop:172.0.0.[1-127] | ssh:root:reboot | confirm">Перезагрузить все</button>
```

---

## 15. SSH-оркестрация: loop | ssh

Вот ради чего всё это. Два режима у `ssh`:

### Режим А: один хост — пишем адрес прямо в вызове (с `@`)

```html
<!-- ssh:user@host:команда — команда склеивается из остатка через ":" -->
<button action="ssh:root@srv1:uptime">Uptime srv1</button>
<button action="ssh:root@srv1:systemctl status nginx">Статус nginx</button>
<button action="ssh:root@srv1:df -h | grep:/data">Диск /data</button>
<button action="ssh:root@srv1:journalctl -u api | tail:50">Лог api (50 строк)</button>
<button action="ssh:root@srv1:cat /proc/loadavg | bars">Нагрузка</button>
```

Если в первом аргументе есть `@` — это обычный режим: хост указан явно.

### Режим Б: хост из пайпа — подставляем loop (без `@`)

**`loop` разворачивает шаблон с диапазонами `[a-b]` в список адресов**, и каждый
адрес идёт в `ssh` как отдельный хост. **Если `@` в первом аргументе нет — ssh
берёт хост из каждой строки входа (из loop).**

```html
<!-- адреса 172.0.0.1 .. 172.0.0.127, на каждом: apt update && apt upgrade -->
<button action="loop:172.0.0.[1-127] | ssh:root:apt update && apt upgrade -y">Апдейт всей сети</button>

<!-- диапазоны в любых местах -->
<button action="loop:172.0.[0-255].[1-254] | ssh:root:hostname">Hostname по всей сети</button>

<!-- не только IP: пирожок_номер_1 .. пирожок_номер_999 -->
<button action="loop:пирожок_номер_[1-999] | ssh:root:hostname">Пирожки</button>



### Ключи: -i ПУТЬ

По умолчанию ssh ищет ключи в `~/.ssh` (id_ed25519, id_rsa, id_ecdsa), работает
и ssh-agent. Своя папка/файл ключей — флаг `-i` (как у настоящего ssh):

```html
<!-- обычный режим + папка ключей -->
<button action="ssh:root@srv1:-i:/root/keys:hostname">Hostname (ключи)</button>

<!-- обычный режим + конкретный файл ключа -->
<button action="ssh:root@srv1:-i:~/.ssh/id_rsa:uptime">Uptime (файл)</button>

<!-- пайп-режим + ключи: loop → ssh по всем хостам с ключами из /root/keys -->
<button action="loop:172.0.0.[1-127] | ssh:root:-i:/root/keys:uptime">Uptime всей сети</button>
```

### Как это работает

```mermaid
graph LR
    A[loop:172.0.0.[1-3]] -->|строка 1: 172.0.0.1| B[ssh:root:uptime]
    A -->|строка 2: 172.0.0.2| B
    A -->|строка 3: 172.0.0.3| B
    B --> C[результаты по всем хостам]
    C --> D[в тайл / статус]
```

`loop` выдаёт адреса построчно → каждый уходит в `ssh` как хост → результаты
склеиваются и показываются.

---

## 16. Сборный пример: панель управления

Все рецепты в одном тайле `tiles/dash.html`:

```html
<h1 color="#ffcc00">Панель управления</h1>

<!-- Сеть: один хост и вся подсеть -->
<row>
  <div width="50%">
    <h1>Один сервер</h1>
    <button action="ssh:root@srv1:uptime">Uptime</button>
    <button action="ssh:root@srv1:df -h | grep:/data">Диск /data</button>
  </div>
  <div width="50%">
    <h1>Вся сеть</h1>
    <button action="loop:172.0.0.[1-127] | ssh:root:-i:/root/keys:uptime">Uptime всех</button>
    <button action="loop:172.0.0.[1-127] | ssh:root:apt update && apt upgrade -y | confirm">Апдейт всех</button>
  </div>
</row>

<hr/>

<!-- Настройки приложения -->
<h1>Настройки</h1>
<checkbox action="toggle:app.conf:debug">Отладка</checkbox>
<checkbox action="toggle:app.conf:logging">Логирование</checkbox>
<select action="set:/app.conf:loglevel" options="info:debug:trace" label="Уровень"/>

<hr/>

<!-- Статус сервисов -->
<h1>Сервисы</h1>
<table>
  <tr><th>Сервис</th><th>Статус</th></tr>
  <tr><td>api</td><td>работает</td></tr>
  <tr><td>db</td><td>работает</td></tr>
  <tr><td>queue</td><td>стоп</td></tr>
</table>

<hr/>

<!-- Живой график -->
<h1>Нагрузка</h1>
<plugin pipe="ssh:root@srv1:cat /proc/loadavg | bars" interval="1s"/>

<hr/>

<!-- Справка: ввод → вывод в другой тайл -->
<input action="man:" output="out" label="Справка по плагину"/>
```

---

## 17. Шпаргалка: все теги

| Тег | Что делает | Пример |
|---|---|---|
| текст | просто текст | `Привет` |
| `<h1 color="...">` | заголовок | `<h1>Настройки</h1>` |
| `<p>` | абзац | `<p>текст</p>` |
| `<br/>` | пустая строка | `<br/>` |
| `<b>` / `<i>` | жирный / курсив | `<b>x</b>` |
| `<center>` | по центру | `<center>x</center>` |
| `<div>` | блок | `<div>x</div>` |
| `<row>` | ряд колонок | `<row><div width="50%">…` |
| `<div width="%">` | колонка | `<div width="30%">x</div>` |
| `<div align="center">` | центрированный блок | `<div align="center"><button…>` |
| `<button action="…">` | кнопка-вызов | `<button action="cat:/etc/hostname">` |
| `<a href="/route">` | ссылка на роут | `<a href="/man">Справка</a>` |
| `<input action="…" output="…" label="…"/>` | поле ввода | `<input action="man:" output="out" label="Пакет"/>` |
| `<checkbox action="…">` | переключатель [x]/[ ] | `<checkbox action="toggle:a.conf:debug">` |
| `<select action="…" options="…" label="…"/>` | выпадающий список | `<select action="set:…" options="a:b:c"/>` |
| `<table><tr><th>/<td>` | таблица | см. раздел 9 |
| `<img src="…"/>` | картинка PPM | `<img src="/opt/x.ppm"/>` |
| `<hr/>` | линия | `<hr/>` |
| `<pre>` | моноширинный блок | `<pre>…</pre>` |
| `<plugin name="…" interval="…"/>` | живой плагин | `<plugin name="clock" interval="1s"/>` |
| `<plugin pipe="…" interval="…"/>` | живой пайп | `<plugin pipe="cat:x | tail | bars" interval="1s"/>` |
| `output="id"` | куда вывести результат | `<button action="…" output="out">` |
| `| confirm` | подтверждение | `<button action="… | confirm">` |

---

## 18. Шпаргалка: все плагины и их вызовы

| Плагин | Вызов | Что делает |
|---|---|---|
| `hello` | `hello[:ИМЯ]` | поздороваться |
| `cat` | `cat:ФАЙЛ` | показать файл |
| `man` | `man[:ИМЯ]` | справка по плагинам (без имени — список) |
| `grep` | `… \| grep:МАСКА` | оставить строки под регулярку |
| `head` | `… \| head[:N]` | первые N строк (по умолчанию 10) |
| `tail` | `… \| tail[:N]` | последние N строк (по умолчанию 10) |
| `wc` | `… \| wc` | сколько строк во входе |
| `line` | `… \| line:N` или `line:ФАЙЛ:N` | строка по номеру |
| `append` | `append:ФАЙЛ:строка` | дописать строку в файл |
| `set` | `set:ФАЙЛ:КЛЮЧ:ЗНАЧЕНИЕ` | поставить значение ключа |
| `toggle` | `toggle:ФАЙЛ:КЛЮЧ` | переключить флаг 0↔1/on↔off |
| `curl` | `curl:URL` | скачать URL и отдать тело |
| `ssh` | `ssh:user@host:[ -i ПУТЬ ]:команда` | выполнить команду по SSH |
| `loop` | `loop:ШАБЛОН` или `loop:БАЗА:КОНЕЦ` | развернуть диапазоны в адреса |
| `bars` | `… \| bars[:МАСКА]` | полосковый график из чисел |
| `clock` | `<plugin name="clock" interval="1s"/>` | живые часы |
| `tobotom` | `… \| tobotom:pass` / `… \| tobotom:stop` | отладка: вывод в статус-строку |

### Примеры каждого

```html
<!-- hello -->
<button action="hello">Привет</button>
<button action="hello:мир">Привет, мир</button>

<!-- cat -->
<button action="cat:/etc/hostname">Hostname</button>

<!-- man -->
<button action="man">Список плагинов</button>
<button action="man:ssh">Справка по ssh</button>
<input action="man:" label="Пакет"/>   <!-- вводишь имя → справка -->

<!-- grep -->
<button action="cat:/etc/x.conf | grep:^server">Серверы</button>

<!-- head / tail -->
<button action="cat:/etc/os-release | head:3">Первые 3</button>
<button action="cat:/var/log/app.log | tail:20">Хвост лога</button>

<!-- wc -->
<button action="cat:/var/log/x.log | wc">Строк в логе</button>

<!-- line -->
<button action="cat:/etc/x.conf | line:5">5-я строка</button>
<button action="line:/etc/x.conf:5">5-я строка файла</button>

<!-- append -->
<button action="append:/etc/app.conf:debug=1">Включить debug</button>

<!-- set -->
<button action="set:/etc/app.conf:loglevel:debug">Уровень = debug</button>
<select action="set:/etc/app.conf:loglevel" options="info:debug:trace" label="Уровень"/>

<!-- toggle -->
<checkbox action="toggle:/etc/app.conf:debug">Отладка</checkbox>
<button action="toggle:/etc/app.conf:logging">Переключить логирование</button>

<!-- curl -->
<button action="curl:https://api.example.com/status">Статус API</button>

<!-- ssh: один хост -->
<button action="ssh:root@srv1:uptime">Uptime</button>

<!-- ssh: пайп-режим с loop -->
<button action="loop:172.0.0.[1-127] | ssh:root:uptime">Uptime всей сети</button>

<!-- bars -->
<plugin pipe="cat:data.log | tail:10 | bars" interval="1s"/>

<!-- clock -->
<plugin name="clock" interval="1s"/>

<!-- tobotom: отладка пайпа — увидеть, что реально приходит на вход -->
<button action="cat:x | tobotom:pass | grep:y">cat → показать → grep</button>
<button action="cat:x | tobotom:stop">cat → показать и стоп</button>`

---

## 19. Частые вопросы

**Почему `cat:`, а не `file:`?** Потому что плагин называется как команда Linux
и ведёт себя как она. `cat` — показать файл, `grep` — фильтр, `tail` — хвост.

**Что будет без `output="id"`?** Результат уйдёт в статус-строку внизу экрана.

**Можно ли в `output` указать тайл другого роута?** Да — движок находит блок с
id по всему интерфейсу, не только в текущем тайле.

**Почему у `ssh` команда склеивается через `:`?** Потому что аргументы разделяются
`:` — а в команде могут быть пробелы. `ssh:root@srv:df -h` → команда `df -h`.

**Что если ssh-команда сама содержит `|`?** Внутри одного `action` движок тоже
режет по `|` — это уже следующий шаг пайпа. Если на сервере нужен шелл-пайп —
оберни команду в `sh -c '...'`.

**`loop` vs `loop:БАЗА:КОНЕЦ`?** Шаблон с `[a-b]` — новый формат, работает для
всего (`пирожок_[1-5]`). Старый `loop:172.0.0.1:25` — только IP от базы до конца.
Используй шаблон.

**У меня не работает кириллица.** Терминал должен быть UTF-8: Windows — `chcp
65001` или Windows Terminal / VS Code.

Версия 0.1 для разработки
