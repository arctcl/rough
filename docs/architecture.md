# Rough — Архитектура

> Движок TUI-интерфейсов: HTML-разметка рендерится нативно в клетки терминала.
> Без веб-сервера и браузера. `project -tui` → один статический бинарник
> (CGO_ENABLED=0): движок + твои плагины + вшитая папка `rough/`.

---

## 1. Идея и интеграция

Проект подключает `rough` как Go-модуль (`replace` в go.mod). Вся интеграция — 4 строчки в `main.go`:

```go
//go:embed rough
var roughDir embed.FS          // tiles.json / tiles/*.html / themes/ (вшивается в бинарник)

func main() {
	rough.TUI(roughDir)        // вызвано с -tui → открывает интерфейс, иначе false
}
```

Плагины проекта подключаются одним blank-import (`_ "myproject/rough/plugins"`).
Движок при старте читает всё из вшитой папки (fs.FS) — на проде снаружи ничего не нужно.

## 2. Как работает

```
Run(fsys) → LoadUI (tiles.json → страницы/тайлы, тема)
  → главный цикл (for {}):
       renderFrame: HTML → Buffer (backend) → копия на экран (frontend)
                    + статус, вкладки, виджеты, курсор
       ждём событие: клавиша | мышь | таймер (0.5с) | ресайз
       реагируем:    клик → hit-test → action → плагин (пайп) → вывод
```

Движок — **пустая оболочка**: рисует HTML в клетки, ловит ввод, держит реестр плагинов.
Вся функциональность — только плагины (`строки → строки`), вызываются по клику или таймеру.

## 3. Карта файлов движка (engine/)

Все файлы — один пакет `engine`, разбиты по слоям (по имени видно, что где).

**Загрузка (loader_*)**
- `loader_tiles.go` — tiles.json → страницы/тайлы: `Tile`, `Pages`, `LoadPages`, `LoadMenu`
- `loader_theme.go` — темы: `Theme`, `LoadTheme`, `ResolveColor`, `SwitchTheme`, `ThemeColor`

**Контент тайла (backend_*, HTML → клетки)**
- `backend_html.go` — вёрстка HTML: `RenderHTML`, `renderNode`, `flowState`, `renderRow`,
  `renderCenteredDiv`, `renderTable` (текст/кнопки/таблицы/колонки)
- `backend_plugin.go` — тег `<plugin>`: `renderPlugin`, `pluginSteps`, кэш по `interval`
- `backend_img.go` — картинки PPM: `renderImg`, `decodePPM`, блоки ▀▄█
- `backend_clickzones.go` — кликабельные зоны: `Hotzone`, `HitTest`, `HitSelect`

**Виджеты (frontend_widget_*, инпут внутри интерфейса)**
- `frontend_widget_input_area.go` — поле ввода
- `frontend_widget_selecter_list.go` — выпадающий список + подменю
- `frontend_widget_modal_confirm.go` — модалка подтверждения
- `frontend_widget_status_window.go` — статус-строка + показ ошибок
- `frontend_widget_tabs.go` — вкладки + кнопка «Закрыть»

**Фронт (раскладка и холст)**
- `frontend_buffer.go` — холст: `Buffer`/`Style`/`Cell` (2D-буфер клеток → экран tcell)
- `frontend_stretcher.go` — растягиватель: `renderFrame`, `renderTile`, `scrollTile`,
  `(Tile)Rect` — тайлы тянутся за ресайзом терминала
- `frontend_tiles_borders.go` — рамки тайлов: `drawFrame`, `drawFrameStyled`
- `frontend_focus.go` — фокус: `focusIdx`, `moveFocus`, `activateFocus`

**Сырой ввод от человека (people_input_*)**
- `people_input_keyboard.go` — клавиши: `handleKey` (раздаёт виджетам или глобальным клавишам)
- `people_input_mouse_events.go` — `MouseEvent` + состояние мыши
- `people_input_mouse_desktop.go` — мышь через терминал (tcell): `handleMouse`
- `people_input_mouse_teletype.go` — сырая мышь `/dev/input/mice` (Linux, протокол PS/2)

**Ядро**
- `engine.go` — `Run` (главный цикл), `execAction`, crash.log
- `plugin_registry.go` — реестр плагинов, пайпы, `ParseArgs`, `LoadUI`
- `syntax_checker.go` — проверка action/плагинов/ссылок до старта

**Тесты (testing_*_test.go)** — по тем же областям (backend, theme, centered, plugin_args, teletype_mouse).

## 4. Поток рендера и ввода

- **Рендер:** `renderFrame` собирает кадр: фон ← тема, тайлы ← backend (HTML→клетки),
  поверх — статус/вкладки/виджеты, вниз — `Buffer.Blit` на экран.
- **Ресайз (растягиватель):** терминал сменил размер → новое `w/h` → `(Tile)Rect`
  пересчитывает `%` → тайлы растягиваются, плагины адаптируются через `Window()`.
- **Ввод:** `handleKey`/`handleMouse` получают сырые события и раздают активному виджету
  (поле/select/confirm) или глобальным клавишам; клик → `HitTest` → `action` → плагин.
- **Изоляция:** каждый шаг пайпа в `recover()` (`callSafe`) — паника плагина не валит
  интерфейс, трасса уходит в статус-окно; верхнеуровневый перехват пишет `crash.log`.

## 5. Плагины

Единый контракт: `func(in []string, args []string) ([]string, error)` — юникс-команда.
Реестр один: `AddPlugin`. Всё через пайпы `|`; параметры — гибрид (двоеточия по порядку
или `--флаги`, дефолты/обязательные). `run` запрещён движком. Каждый плагин несёт
`man_<имя>` для справки (`man:имя`). Цвета плагинов — из темы (`color_0..color_15`,
маркеры `\x01{имя}` в выводе).

## 6. Темы

Темы — `themes/<имя>.json`: символы рамок/кнопок и цвета (ссылка на палитру терминала,
hex или имя темы). Выбор — ключ `"theme"` в tiles.json. Переключение на лету —
плагин `theme` (`theme:list`, `theme:ИМЯ`). Плагинам доступны цвета через `engine.ThemeColor`.
