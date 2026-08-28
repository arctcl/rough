# Cookbook: write plugins

A plugin is a Go function: **strings in, strings out**. The engine has no
behaviour of its own — everything is a plugin.

## Contract

```go
func(in []string, args []string) ([]string, error)
```

- `in` — lines from the previous pipe step;
- `args` — the plugin's arguments (after the name, split by `:`);
- return `[]string` (output lines) or an error.

## Register a plugin

```go
package plugins // your project's plugins package (or plugins/<name> in the root)

import "github.com/arctcl/rough"

func init() {
	rough.AddMan("mycmd", `mycmd — does something.`) // optional help
	rough.AddPlugin("mycmd", func(in []string, args []string) ([]string, error) {
		// ...
		return []string{"ok"}, nil
	})
}
```

Because it lives in the `plugins` package your project imports, `init()` runs at
startup and registers the plugin. Then you call it from HTML by name:
`action="mycmd:..."`.

## Args parsing

Use `engine.ParseArgs` for named params and `--flags`, or `args` directly.
Quotes `'...'` / `"..."` protect `:` and `|` inside values.

## Pipes

A plugin doesn't know about pipes: it gets `in` (lines from the previous step)
and returns lines, while gluing steps into a chain `|`, confirmation `| confirm`
and `&&` is done by the engine. How it works from the HTML side —
[guide, sections 7–8](guide.md). Keep each plugin simple and composable.

## Man

`AddMan("name", "...")` gives the plugin a help page shown by the `man` plugin.

## Live widgets

Plugins returning data over time (charts, monitors) are re-run by `<plugin
interval="...">`. Add `updateanytime="1"` to keep them running on inactive tabs.

## Inject into the engine (key sequences)

A plugin is not only "called from HTML" — it can register **hooks** into the
engine through public APIs.

`AddCheatRoute(seq, route)` — a key sequence **navigates** to a page (like a
tab). Keys: `'U','D','L','R'` — arrows; letter/digit — that key; `'+'` — the
plus key. Secret pages can be registered programmatically with `AddPage` —
so `tiles.json` is never touched and the page has no tab button; reachable only
by the code. On the page you put normal plugins, e.g. an immediately-printed
greeting:

```html
<plugin pipe="cat GLHF mate!" async interval="1s"/>
```

`AddPage(route string, tiles []Tile)` — registers a page (route → tiles)
programmatically. Used by injectors (`chch`) for secret pages: they live in the
injector's config, not in `tiles.json`, and are added to the page list at
startup.

`OnReady(fn func(fs.FS))` — a hook run right after the engine loads the embedded
folder (in `init()` the folder is not ready yet). Use it to read config files.

The **`chch` injector plugin** makes this data-driven. It reads `chch.json`
from the project's `/rough` folder and registers every page and code:

```json
{
  "title": "chch — secret pages",
  "description": "Type a code to open a hidden page.",
  "pages": {
    "/ps": [
      { "id": "ps", "x": "0%", "y": "0%", "w": "100%", "h": "100%", "file": "tiles/ps_page.html" }
    ]
  },
  "cheats": {
    "ps+": "/ps",
    "UUDDLRLRba": "/konami"
  }
}
```

```go
func init() { rough.OnReady(load) }

func load(fsys fs.FS) {
	b, err := fs.ReadFile(fsys, "chch.json")
	if err != nil { return }
	var c struct {
		Pages  map[string][]rough.Tile `json:"pages"`
		Cheats map[string]string       `json:"cheats"`
	}
	if json.Unmarshal(b, &c) != nil { return }
	for route, tiles := range c.Pages {
		rough.AddPage(route, tiles)
	}
	for seq, route := range c.Cheats {
		rough.AddCheatRoute(seq, route)
	}
}
```

The injector is optional: drop the `chch` import and the engine works as usual —
just no secret pages. Everything goes through public engine APIs, no hacking.

## Tips

- Do not touch files unless needed — `opt`/`flag` keep state in memory.
- Return an error for bad input; the engine shows it in the status line.
- A panic is caught by the engine and turned into a readable error.
