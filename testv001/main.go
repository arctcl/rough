// testv001 — проверочное приложение: справка man с выводом в тайл.
// 4 строчки: import rough + плагины + embed + TUI.
package main

import (
	"embed"

	"rough"
	_ "testv001/rough/plugins"
)

//go:embed rough
var roughDir embed.FS

func main() { rough.TUI(roughDir) }
