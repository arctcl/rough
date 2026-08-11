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
| [cookbook-html](cookbook-html.md) | ВСЕ примеры вёрстки: кнопки, пайпы, вывод в тайл, таблицы, чекбоксы, ssh+loop |
| [cookbook-plugins](cookbook-plugins.md) | как писать плагины: контракт, рецепты, пайпы, справка `man` |
| [cookbook-themes](cookbook-themes.md) | как делать темы: символы, цвета, примеры |
| [how-it-works](how-it-works.md) | как устроен движок изнутри: рендер, события, вывод в тайл |
| [architecture](architecture.md) | архитектура и принципы (подробно) |
| [demo](demo.md) | описание шаблона проекта |
| [systemprompt](systemprompt.md) | правила разработки (для ИИ и людей) |

## Структура репозитория

```
rough\
  rough.go               # публичное API: TUI(embed.FS), AddPlugin, AddMan
  engine\                # ДВИЖОК: чистый, без плагинов
    backend.go           # HTML → DOM → клетки, колонки, вывод в блок
    backend_conf_loader.go  # tiles.json: страницы, тайлы, menu
    frontend.go          # холст: буфер клеток → экран tcell
    people_input.go      # главный цикл: мышь/клавиши/таймер, вкладки, фокус
    plugin_registry.go   # реестр плагинов + пайпы + справки + LoadUI
    theme.go             # темы: символы и цвета
    syntax_checker.go    # проверка: action/plugin/ссылки существуют
  plugins\               # ВСЕ плагины: cat, hello, ssh, curl, man, grep, ...
  example\               # живой пример (отдельный модуль)
  testv001\              # проверочная сборка (тоже отдельный модуль)
  docs\                  # эта документация
```

Важно: `example\` и `testv001\` — **отдельные модули** с `replace rough => ../`.
Оба используют один и тот же корневой движок — дублирования кода нет.
