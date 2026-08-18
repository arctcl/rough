# Demo project (integration template)

The live example lives in `example_project/` (a separate Go module,
`go build` + `go run . -tui`). Here is how it is put together and how to
repeat it in your own project.

## A user project — 4 lines of code at most

```go
// main.go
package main

import (
	"embed"

	"github.com/arctcl/rough"

	_ "example_project/rough/plugins" // 2. my plugins
)

//go:embed rough
var roughDir embed.FS // 3. the folder with html/themes/data (embedded)

func main() {
	rough.TUI(roughDir) // 4. one call
}
```

The user does 4 things:
1. `import "github.com/arctcl/rough"` — the engine;
2. `_ "example_project/rough/plugins"` — where the plugins are (embedded at build);
3. `//go:embed rough` — embed the UI data folder;
4. `rough.TUI(roughDir)` — run.

## Project structure

```
myproject\
  go.mod                  # require github.com/arctcl/rough + replace ... => ../rough
  main.go                 # the 4 lines above
  rough\                  # EVERYTHING about the project: data + plugins
    tiles.json            # pages and tiles
    tiles\*.html          # tile markup
    themes\*.json         # themes
    plugins\              # plugins: your own or imports of ready-made ones
      plugins.go          # aggregator: lists the plugins used
```

**The library (rough) = engine only.** No plugins or themes inside it — all of
that lives in your project's `rough/` folder. Ready-made plugins (`cat`,
`hello`, `ssh`, `curl`, `man`, `grep`, `tail`, `head`, `wc`, `clear`, `sleep`,
`opt`, `flag`, ...) live in the root `plugins/` folder; your project imports
only the ones it uses (selective import — no dead code).

## How it works

```
go build
  → the compiler puts into the binary: engine + your plugins + the rough/ folder (embed)
  → at startup each plugin's init() registers it in the engine registry
  → rough.TUI() reads tiles.json, html, themes from the embedded folder
  → a click on action="name:..." → the engine finds the plugin in the registry → calls it
```
