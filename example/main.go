// Пример интеграции rough: максимум, что пишет пользователь.
//
//  1. import "github.com/arctcl/rough"                     — подключил библиотеку
//  2. _ "example/rough/plugins"          — показал, где его плагины
//  3. //go:embed rough                    — вшил папку с html/плагинами/данными
//  4. rough.TUI(roughDir)                 — один вызов
//
// При сборке: движок + плагины + вся папка rough/ вшиваются в бинарник.
// На проде лежит один файл — ничего снаружи не нужно.
package main

import (
	"embed"

	"github.com/arctcl/rough"

	_ "example/rough/plugins" // вот мои плагины
)

//go:embed rough
var roughDir embed.FS // вот папка с html и всем остальным (вшивается в бинарник)

func main() {
	rough.TUI(roughDir)
}
