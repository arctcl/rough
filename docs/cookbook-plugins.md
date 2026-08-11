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

Ключи из `~/.ssh` по дефолту, или `-i:ПУТЬ` для своей папки/файла ключей.

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

## Правила

- Имя плагина = имя команды Linux. Поведение = поведение команды.
- Нативно в Go: **не дёргаем внешние бинарники** — работает в любом контейнере.
- `run` запрещён движком — произвольный запуск команд через кнопку невозможен.
