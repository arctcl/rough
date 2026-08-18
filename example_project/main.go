// Example of integrating rough (by the guidebook): everything the user writes.
//
//  1. import "github.com/arctcl/rough"   — the library
//  2. _ "example_project/rough/plugins" — my plugins (built-in + demo)
//  3. //go:embed rough                   — embed the folder with html/plugins/data
//  4. rough.TUI(roughDir)                — one call
//
// Demo plugins (emu, stats, nginx) live in example_project/rough/plugins/
// as the guidebook says (cookbook-project).
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
