// Пример интеграции rough «по гайдбуку»: максимум, что пишет пользователь.
//
//  1. import "github.com/arctcl/rough"          — подключил библиотеку
//  2. _ "example_project/rough/plugins"        — вот мои плагины (встроенные + демо)
//  3. //go:embed rough                          — вшил папку с html/плагинами/данными
//  4. rough.TUI(roughDir)                       — один вызов
//
// Демо-плагины (opt, flag, emu, stats, nginx, clear, sleep) лежат в
// example_project/rough/plugins/ — как и положено по гайдбуку (cookbook-project).
package main

import (
	"embed"

	"github.com/arctcl/rough"

	_ "example_project/rough/plugins" // мои плагины (встроенные + демо)
)

//go:embed rough
var roughDir embed.FS // папка с html и всем остальным (вшивается в бинарник)

func main() {
	rough.TUI(roughDir)
}
