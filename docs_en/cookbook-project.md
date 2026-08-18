# Cookbook: build your own project

Building a project on rough is four lines of code and a folder with configs.

## 1. The four lines

```go
package main

import (
	"embed"

	"github.com/arctcl/rough"

	_ "myproject/rough/plugins" // my plugins (ready-made or your own)
)

//go:embed rough
var roughDir embed.FS

func main() {
	rough.TUI(roughDir)
}
```

`go.mod`:

```
module myproject

go 1.25.0

require github.com/arctcl/rough v0.0.0

replace github.com/arctcl/rough => ../rough   // or a GitHub version
```

## 2. The rough/ folder structure

```
rough\
  tiles.json            # pages, tiles, tabs, theme
  tiles\*.html          # tile markup
  themes\*.json         # themes
  plugins\plugins.go    # aggregator: lists the plugins used
```

`rough/plugins/plugins.go` imports ONLY the plugins actually used (selective
import — no dead code). Each plugin is a separate import:

```go
package plugins

import (
	_ "rough/plugins/cat"    // only what you need
	_ "rough/plugins/man"
	_ "rough/plugins/set"
)
```

Your own plugins go here too (see [cookbook-plugins](cookbook-plugins.md)).

## 3. tiles.json — pages and tiles

```json
{
  "theme": "default",
  "menu": [["Home", "/main"], ["Settings", "/cfg"]],
  "pattern": ["id", "x", "y", "w", "h", "file"],
  "/main": [
    ["cfg", "0%", "0%", "40%", "100%", "tiles/cfg.html"],
    ["out", "40%", "0%", "60%", "100%", "tiles/out.html"]
  ]
}
```

## 4. HTML tiles

```html
<!-- tiles/cfg.html -->
<h1>Settings</h1>
<button action="cat:/etc/hostname">Hostname</button>
<input action="man:" output="out" label="Package"/>
```

## 5. The result

Everything is embedded into **one binary** — no external files needed in
production. The terminal must be **UTF-8**.

## 6. Ready-made projects

- `example_project\` — the demo project (4 tabs) for recording README GIFs.
