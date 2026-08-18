# Документация rough

**rough** — простой тайловый резиновый движок HTML-интерфейсов прямо в терминале. Пишешь HTML —
получаешь кликабельный интерфейс в консоли. Без веб-сервера, без браузера.
Подходит и для прямой интеграции в проекты на ГО и для отдельного использования

## Быстрый старт

Четыре строчки — и интерфейс вшит в твой бинарник:

```go
//go:embed rough
var roughDir embed.FS

func main() { rough.TUI(roughDir) }
```

Запуск: `myapp -tui`. Полный шаблон — [cookbook-project](cookbook-project.md).

## Как читать документацию

| Документ | О чём |
|---|---|
| [README](../README.md) | лицо проекта: что это и зачем (для людей) |
| [cookbook-project](cookbook-project.md) | как собрать свой проект: 4 строчки, `tiles.json`, HTML тайлов, вкладки |
| [cookbook-html-new](cookbook-html-new.md) | HTML с нуля: от общего к частному, живые примеры (рекомендуется начать здесь) |
| [cookbook-plugins](cookbook-plugins.md) | как писать плагины: контракт, живой пример cat, рецепты, пайпы, справка `man` |
| [cookbook-themes](cookbook-themes.md) | как делать темы: символы, цвета, примеры |
| [how-it-works](how-it-works.md) | как устроен движок изнутри: 2 способа использования (библиотека/standalone), рендер, события |
| [architecture](architecture.md) | архитектура и принципы (подробно) |
| [demo](demo.md) | описание шаблона проекта |
| [systemprompt](systemprompt.md) | правила разработки (для ИИ и людей) |

## Структура репозитория

```
rough\
  rough.go               # публичное API: TUI(embed.FS), AddPlugin, AddMan
  engine\                # ДВИЖОК: чистый, без плагинов (слои по файлам)
    engine.go            # Run (главный цикл), execAction, crash.log
    loader_tiles.go      # tiles.json: страницы, тайлы, menu
    loader_theme.go      # темы: символы и цвета
    backend_html.go      # HTML → DOM → клетки, колонки, вывод в блок
    backend_plugin.go    # тег <plugin>: пайп + кэш по интервалу
    backend_img.go       # картинки PPM
    backend_clickzones.go# кликабельные зоны, hit-test, чекбокс/select-состояние
    frontend_buffer.go   # холст: буфер клеток → экран (дифф-рендеринг)
    frontend_stretcher.go# растягиватель тайлов + скролл
    frontend_tiles_borders.go # рамки тайлов
    frontend_tabs.go     # вкладки + кнопка «Закрыть»
    frontend_focus.go    # фокус стрелками
    frontend_widget_*.go # виджеты: поле ввода, select, модалка, статус-блок
    people_input_*.go    # клавиатура + ЕДИНАЯ мышь (десктоп/телетайп)
    plugin_registry.go   # реестр плагинов + пайпы + ParseArgs (quick) + LoadUI
    vars.go              # переменные сессии: SetVar/VarLine, подстановка $имя
    syntax_checker.go    # проверка: action/plugin/ссылки существуют
  plugins\               # ВСЕ плагины: cat, hello, ssh, curl, man, grep, cut, awk, sed, export, ...
  example\               # живой пример (отдельный модуль)
  testv001\              # проверочная сборка (тоже отдельный модуль)
  docs\                  # эта документация
```

Важно: `example\` и `testv001\` — **отдельные модули** с `replace rough => ../`.
Оба используют один и тот же корневой движок — дублирования кода нет.
