# How it works

There are two ways to use rough: as a **library** in your project, or as a
**standalone app**.

## 1. As a library

Your project imports the engine, embeds a folder with the UI, and calls `TUI`:

```go
package main

import (
	"embed"

	"github.com/arctcl/rough"

	_ "example_project/rough/plugins" // my plugins
)

//go:embed rough
var roughDir embed.FS

func main() {
	rough.TUI(roughDir)
}
```

The embedded folder `rough/` holds `tiles.json`, `tiles/*.html`, `themes/*.json`
and `plugins/`. At build time everything — engine + plugins + folder — becomes
one binary. Nothing external is needed in production.

## 2. As a standalone app

Build a binary that embeds the UI, run it — you get the interface without
writing any UI code yourself.

## What happens at runtime

1. `TUI` loads `tiles.json` (pages, tiles, menu), the theme, and checks syntax.
2. Each plugin's `init()` registered it in the engine registry.
3. The main loop renders frames on a timer (500ms) and handles input
   (keyboard + mouse).
4. A click on an element runs its `action` pipe; the result goes to the target
   tile or the status line.
5. Live `<plugin interval="...">` renders on the active page. Cyclic plugins
   without `async` stop when the tab is unloaded; with `async` they keep running
   in their own goroutine.
