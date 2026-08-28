# Кукингбук: синтаксис rough

Шпаргалка в формате **«ЧТО ХОЧУ → ПРИМЕР»**: на каждый замысел — готовый код.
Бери пример и вставляй в свой `tiles.json` / `tiles/*.html`. В самом низу —
огромный пример, где всё сразу.

---

## 🚂 Пример-монстр: весь синтаксис в одном элементе

Один `action` — сразу и `&&`, и `|`, и переменные, и ssh, и подтверждение.
Читаем `lev_tolstoy.txt`, считаем вхождения слова, и отправляем по ssh в докер
столько паровозиков `sl`, сколько слов нашлось. Всё — в одной кнопке:

```html
<button action="
  clear
  && cat:lev_tolstoy.txt | head:500 | grep:мир | wc:lines | export:count
  && cat:lev_tolstoy.txt | head:500 | grep:мир | tail:3
  && ssh:root:srv:22::docker exec tolstoy bash -c 'for i in $(seq 1 $count); do sl; done'
  && chart:0:500:$count | bars
  | confirm
" output="out">
Паровозик Толстого 🚂
</button>
```

Что здесь задействовано, построчно (слово «мир» — просто пример, подставь своё):

- **`clear`** — очистить блок вывода и начать склейку с чистого листа.
- **`&&`** — каждый кусок выполняется отдельно, а выводы склеиваются в один блок.
- **`cat:lev_tolstoy.txt | head:500 | grep:мир | wc:lines | export:count`** —
  взять 500 строк, оставить строки со словом «мир», посчитать их, запомнить
  число в переменную `$count`.
- **`cat:... | grep:мир | tail:3`** — показать последние 3 найденные строки
  (для наглядности).
- **`ssh:root:srv:22::docker exec tolstoy bash -c '...$count...'`** — по ssh в
  докер-контейнер запустить `sl` (паровозик) столько раз, сколько слов нашли;
  `$count` — переменная из прошлого шага.
- **`chart:0:500:$count | bars`** — нарисовать график «сколько нашли», прямо с
  переменной в середине пайпа.
- **`| confirm`** — перед выполнением спросить подтверждение (Enter — да,
  Esc — нет): паровозик в докере — дело серьёзное.

А вот тот же замысел, но слово приходит **от юзера через поле ввода** — через
`$in` встаёт ровно в место поиска:

```html
<input action="
  clear
  && cat:lev_tolstoy.txt | head:500 | grep:$in | wc:lines | export:count
  && ssh:root:srv:22::docker exec tolstoy bash -c 'for i in $(seq 1 $count); do sl; done'
  | confirm
" label="Слово" output="out"/>
```

---

## 1. Раскладка: `tiles.json`

**ЧТО ХОЧУ:** одна страница из двух тайлов (слева настройки, справа вывод).
**ПРИМЕР:**
```json
{
  "theme": "default",
  "menu": [["Главная", "/main"]],
  "/main": [
    ["cfg", "0%", "0%", "40%", "100%", "tiles/cfg.html"],
    ["out", "40%", "0%", "60%", "100%", "tiles/out.html"]
  ]
}
```

**ЧТО ХОЧУ:** несколько страниц и вкладки внизу.
**ПРИМЕР:**
```json
{
  "menu": [
    ["Главная", "/main"],
    ["Настройки", "/cfg"]
  ],
  "/main":  [["main", "0%", "0%", "100%", "100%", "tiles/main.html"]],
  "/cfg":   [["cfg",  "0%", "0%", "100%", "100%", "tiles/cfg.html"]]
}
```

**ЧТО ХОЧУ:** тайл, растягивающийся при изменении окна (проценты).
**ПРИМЕР:**
```json
["panel", "0%", "0%", "100%", "100%", "tiles/panel.html"]
```

**ЧТО ХОЧУ:** тайл фиксированного размера (клетки терминала).
**ПРИМЕР:**
```json
["side", "0", "0", "40", "20", "tiles/side.html"]
```

**ЧТО ХОЧУ:** свою тему вместо `default`.
**ПРИМЕР:**
```json
{ "theme": "orange" }
```

---

## 2. Текст и разметка

**ЧТО ХОЧУ:** заголовок.
**ПРИМЕР:**
```html
<h1>Заголовок</h1>
```

**ЧТО ХОЧУ:** обычный абзац.
**ПРИМЕР:**
```html
<p>Обычный текст.</p>
```

**ЧТО ХОЧУ:** пустая строка.
**ПРИМЕР:**
```html
<br/>
```

**ЧТО ХОЧУ:** жирный и курсив.
**ПРИМЕР:**
```html
<b>жирный</b> <i>курсив</i>
```

**ЧТО ХОЧУ:** текст по центру.
**ПРИМЕР:**
```html
<center>по центру</center>
```

**ЧТО ХОЧУ:** горизонтальная линия на всю ширину.
**ПРИМЕР:**
```html
<hr/>
```

**ЧТО ХОЧУ:** моноширинный блок «как есть» (пробелы сохраняются).
**ПРИМЕР:**
```html
<pre>
  1     running   main.main
  5     sleeping  time.Sleep
</pre>
```

**ЧТО ХОЧУ:** блок (группировка), в т.ч. колонки в ряд.
**ПРИМЕР:**
```html
<div>один блок</div>
<row>
  <div width="50%">левая колонка</div>
  <div width="50%">правая колонка</div>
</row>
```

---

## 3. Цвет

**ЧТО ХОЧУ:** цвет текста.
**ПРИМЕР:**
```html
<p color="color_2">зелёный текст</p>
```

**ЧТО ХОЧУ:** цвет фона.
**ПРИМЕР:**
```html
<p bg="#000000">чёрный фон</p>
```

**ЧТО ХОЧУ:** цвет из палитры терминала.
**ПРИМЕР:**
```html
<h1 color="3">жёлтый (палитра)</h1>
```

**ЧТО ХОЧУ:** взять цвет из темы.
**ПРИМЕР:**
```html
<p bg="header_bg">цвет из темы</p>
```

---

## 4. Кнопка и команда `action`

**ЧТО ХОЧУ:** кнопка без аргументов.
**ПРИМЕР:**
```html
<button action="hello">Поздороваться</button>
```

**ЧТО ХОЧУ:** кнопка с аргументами через двоеточие.
**ПРИМЕР:**
```html
<button action="hello:мир">Поздороваться, мир</button>
```

**ЧТО ХОЧУ:** кнопка с аргументами-флагами (как в Linux).
**ПРИМЕР:**
```html
<button action="chart --min=0 --max=100 --title=CPU">График</button>
```

**ЧТО ХОЧУ:** кнопка, результат которой идёт в тайл-приёмник.
**ПРИМЕР:**
```html
<button action="man:ssh" output="out">Справка по ssh</button>
```

**ЧТО ХОЧУ:** несколько действий одной кнопкой (по очереди).
**ПРИМЕР:**
```html
<button action="clear" action="man:ssh" action="cat:/etc/hosts">Всё сразу</button>
```

**ЧТО ХОЧУ:** кнопка, не морозящая интерфейс (медленный плагин в фоне).
**ПРИМЕР:**
```html
<button async action="ssh:root:srv::apt upgrade" output="out">Деплой</button>
```

**ЧТО ХОЧУ:** кнопка с подтверждением (опасное действие).
**ПРИМЕР:**
```html
<button action="ssh:root:srv::reboot | confirm">Перезагрузить</button>
```

**ЧТО ХОЧУ:** ссылка на другую страницу.
**ПРИМЕР:**
```html
<a href="/main">← На главную</a>
```

---

## 5. Аргументы плагина

**ЧТО ХОЧУ:** передать параметры по порядку.
**ПРИМЕР:**
```text
chart:0:100:1:2:CPU
```

**ЧТО ХОЧУ:** пропустить параметр (берётся дефолт) — пустой слот `::`.
**ПРИМЕР:**
```text
chart:0:100::2:CPU
```

**ЧТО ХОЧУ:** передать параметры флагами (порядок не важен).
**ПРИМЕР:**
```text
chart --min=0 --max=100 --title=CPU
```

**ЧТО ХОЧУ:** смешать слоты и флаги.
**ПРИМЕР:**
```text
chart::100 --title=CPU
```

**ЧТО ХОЧУ:** передать значение, в котором есть `:` или `|` (кавычки).
**ПРИМЕР:**
```text
sed:':':1
```

**ЧТО ХОЧУ:** команда с произвольным хвостом (последний параметр глотает `:`).
**ПРИМЕР:**
```text
ssh:user:host:67:docker compose down && up
```

---

## 6. Пайп `|`

**ЧТО ХОЧУ:** вывод одной команды → на вход следующей.
**ПРИМЕР:**
```html
<button action="cat:/var/log/app.log | tail:20">Хвост лога</button>
```

**ЧТО ХОЧУ:** цепочка из трёх шагов.
**ПРИМЕР:**
```html
<button action="cat:/etc/x.conf | grep:^server | head:5">Серверы (5)</button>
```

**ЧТО ХОЧУ:** первый плагин — «источник» (сам добывает данные).
**ПРИМЕР:**
```html
<button action="ssh:root:srv::uptime | head:3">Аптайм</button>
```

---

## 7. Несколько пайпов: `&&` и `clear`

**ЧТО ХОЧУ:** выполнить несколько независимых пайпов и склеить вывод.
**ПРИМЕР:**
```html
<button action="clear && man:ssh && cat:/etc/hosts">Справка + hosts</button>
```

**ЧТО ХОЧУ:** несколько пайпов без `clear` (в приёмник попадёт только последний).
**ПРИМЕР:**
```html
<button action="man:ssh && cat:/etc/hosts">Только hosts в итоге</button>
```

---

## 8. Подтверждение `| confirm`

**ЧТО ХОЧУ:** опасная кнопка с окном подтверждения (Enter — да, Esc — нет).
**ПРИМЕР:**
```html
<button action="ssh:root:srv1::apt update && apt upgrade -y | confirm">Обновить</button>
```

---

## 9. Переменные `$имя`

**ЧТО ХОЧУ:** запомнить результат и подставить позже (`export` + `$имя`).
**ПРИМЕР:**
```html
<button action="ssh:root:srv1::hostname | export:host">Запомнить</button>
<button action="ssh:root:$host::uptime">Uptime на $host</button>
```

**ЧТО ХОЧУ:** отделить имя переменной от соседей (`${имя}`).
**ПРИМЕР:**
```text
${host}:8080
```

**ЧТО ХОЧУ:** убрать суффикс / префикс (как в bash).
**ПРИМЕР:**
```text
${f%.log}   → /var/log/app      (убрали суффикс .log)
${f#/var/}  → log/app.log       (убрали префикс /var/)
```

**ЧТО ХОЧУ:** буквальный `$` (не подставлять).
**ПРИМЕР:**
```text
\$имя
```

**ЧТО ХОЧУ:** чтобы `$имя` НЕ подставилось (кавычки).
**ПРИМЕР:**
```text
'$имя'
```

---

## 10. Куда идёт результат: `output="id"`

**ЧТО ХОЧУ:** вывод в статус-строку (по умолчанию).
**ПРИМЕР:**
```html
<button action="hello">Результат внизу</button>
```

**ЧТО ХОЧУ:** вывод в конкретный тайл.
**ПРИМЕР:**
```html
<!-- кнопка: -->
<button action="man:ssh" output="out">Справка</button>
<!-- в HTML тайла: -->
<div id="out"></div>
```

---

## 11. Живой контент: `<plugin interval>`

**ЧТО ХОЧУ:** часы, обновляются сами.
**ПРИМЕР:**
```html
<plugin name="clock" interval="1s"/>
```

**ЧТО ХОЧУ:** живой график из пайпа.
**ПРИМЕР:**
```html
<plugin pipe="emu_cpu | chart:0:100:1:2:CPU" height="14" interval="2s"/>
```

**ЧТО ХОЧУ:** живой плагин из медленного источника (не морозит интерфейс).
**ПРИМЕР:**
```html
<plugin pipe="ssh:host::vmstat | cut::2" interval="1s" async/>
```

**ЧТО ХОЧУ:** перезапускать и вне интервала.
**ПРИМЕР:**
```html
<plugin pipe="cat:data.log | tail:10 | bars" interval="2s" updateanytime="1"/>
```

---

## 12. Поле ввода / чекбокс / select

**ЧТО ХОЧУ:** поле ввода, значение уходит аргументом (один плагин).
**ПРИМЕР:**
```html
<input action="man:" label="Пакет"/>   <!-- ввёл ssh → man:ssh -->
```

**ЧТО ХОЧУ:** поле ввода, значение в конкретное место пайпа (`$in`).
**ПРИМЕР:**
```html
<input action="grep:$in | sort" label="Шаблон"/>
```

**ЧТО ХОЧУ:** поле ввода, значение как stdin первому плагину (пайп без `$in`).
**ПРИМЕР:**
```html
<input action="cat | grep | sort" label="Данные"/>
```

**ЧТО ХОЧУ:** чекбокс (вкл/выкл ключ в конфиге).
**ПРИМЕР:**
```html
<checkbox action="toggle:app.conf:logging">Логирование</checkbox>
```

**ЧТО ХОЧУ:** выпадающий список из вариантов.
**ПРИМЕР:**
```html
<select action="set:app.conf:loglevel" options="info:debug:trace" label="Уровень"/>
```

**ЧТО ХОЧУ:** вложенные подменю в селекте.
**ПРИМЕР:**
```html
<select action="set:app.conf:theme"
        options="Тёмная:Светлая:Цветная [Красная:Зелёная:Синяя]" label="Тема"/>
```

---

## 13. Таблицы, картинки

**ЧТО ХОЧУ:** таблица (колонки выравниваются сами).
**ПРИМЕР:**
```html
<table>
  <tr><th>Сервис</th><th>Статус</th></tr>
  <tr><td>api</td><td>работает</td></tr>
  <tr><td>db</td><td>работает</td></tr>
</table>
```

**ЧТО ХОЧУ:** картинка из PPM-файла.
**ПРИМЕР:**
```html
<img src="/opt/app/logo.ppm"/>
```

---

## 14. Темы

**ЧТО ХОЧУ:** список тем.
**ПРИМЕР:**
```text
theme:list
```

**ЧТО ХОЧУ:** переключить тему на лету.
**ПРИМЕР:**
```text
theme:orange
```

---

## 15. Один пример, где всё сразу

Панель: слева настройки (чекбокс, select, поле ввода, кнопка с подтверждением,
живой график), справа — вывод и справка. Всё в одном.

`tiles.json`:
```json
{
  "theme": "default",
  "menu": [
    ["Панель", "/main"],
    ["Справка", "/man"]
  ],
  "/main": [
    ["cfg", "0%", "0%", "40%", "100%", "tiles/cfg.html"],
    ["out", "40%", "0%", "60%", "100%", "tiles/out.html"]
  ],
  "/man": [
    ["man", "0%", "0%", "100%", "100%", "tiles/man.html"]
  ]
}
```

`tiles/cfg.html`:
```html
<h1 color="color_2">Панель</h1>

<checkbox action="toggle:app.conf:debug">Отладка</checkbox>
<select action="set:app.conf:loglevel" options="info:debug:trace" label="Уровень"/>

<input action="man:" output="out" label="Справка по плагину"/>

<button action="cat:app.conf" output="out">Показать конфиг</button>
<button action="clear && man:ssh && cat:app.conf" output="out">Справка + конфиг</button>
<button action="ssh:root:srv::reboot | confirm">Перезапустить</button>

<plugin pipe="emu_cpu | chart:0:100:1:2:CPU" height="10" interval="1s"/>
<plugin pipe="ps --track=1" interval="1s" async/>

<hr/>
<button action="ssh:root:srv::hostname | export:host">Запомнить хост</button>
<button action="ssh:root:$host::uptime" output="out">Uptime на $host</button>

<a href="/man">Справка →</a>
```

`tiles/out.html`:
```html
<h1>Вывод</h1>
<div id="out"></div>
```

`tiles/man.html`:
```html
<h1>Справка</h1>
<button action="man:chart" output="out">chart</button>
<button action="man:ssh" output="out">ssh</button>
<button action="man:set" output="out">set</button>
<div id="out"></div>
```
