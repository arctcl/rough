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

## Tips

- Do not touch files unless needed — `opt`/`flag` keep state in memory.
- Return an error for bad input; the engine shows it in the status line.
- A panic is caught by the engine and turned into a readable error.
