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

The output of one plugin feeds the next. Keep each plugin simple and
composable.

## Man

`AddMan("name", "...")` gives the plugin a help page shown by the `man` plugin.

## Live widgets

Plugins returning data over time (charts, monitors) are re-run by `<plugin
interval="...">`. Add `updateanytime="1"` to keep them running on inactive tabs.

## Inject into the engine (secret keys)

A plugin is not only "called from HTML" — it can register **hooks** into the
engine through public APIs.

`AddCheat(seq, action)` — a secret key sequence runs an action (a pipe, like a
button `action`):
- `'U','D','L','R'` — arrows; letter/digit — that key; `'+'` — the plus key.

`AddCheatRoute(seq, route)` — a secret key sequence **navigates** to a page
(like a tab). The page is an ordinary tile in `tiles.json` with its own html
(in `tiles/`), just **not in `menu`**: no tab button, reachable only by the
secret code. On the page you put normal plugins, e.g. an immediately-printed
greeting:

```html
<plugin pipe="cat GLHF mate!" async interval="1s"/>
```

`OnReady(fn func(fs.FS))` — a hook run right after the engine loads the embedded
folder (in `init()` the folder is not ready yet). Use it to read config files.

The **`chch` injector plugin** makes this data-driven. It reads `chch.json`
from the project's `/rough` folder and registers every code as a page jump:

```json
{
  "title": "chch — secret pages",
  "description": "Type a code to open a hidden page.",
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
	var c struct{ Cheats map[string]string `json:"cheats"` }
	if json.Unmarshal(b, &c) != nil { return }
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
