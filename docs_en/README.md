# rough documentation (EN)

**rough** — a tiled, resizable engine for terminal UIs. Write HTML — get a
clickable interface right in the console. No web server, no browser. Use it as
a library in your project or as a standalone utility.

> Русская версия — в [docs_ru/](../docs_ru/README.md).

## Quick start

Four lines and the interface is embedded into your binary:

```go
//go:embed rough
var roughDir embed.FS

func main() { rough.TUI(roughDir) }
```

Run: `myapp -tui`. Full template — [cookbook-project](cookbook-project.md).

## How to read the docs

| Doc | About |
|---|---|
| [README](../README.md) | the project face: what and why (for people) |
| [cookbook-project](cookbook-project.md) | build your own project: 4 lines, `tiles.json`, HTML tiles, tabs |
| [cookbook-html-new](cookbook-html-new.md) | HTML from scratch: general → specific, live examples (start here) |
| [cookbook-plugins](cookbook-plugins.md) | write plugins: contract, live `cat` example, pipes, `man` |
| [cookbook-themes](cookbook-themes.md) | themes: symbols, colors, examples |
| [how-it-works](how-it-works.md) | engine internals: 2 ways to use, render, events |
| [architecture](architecture.md) | architecture and principles (detailed) |
| [demo](demo.md) | the `example_project` template |
| [systemprompt](systemprompt.md) | development rules (for AI and humans) |

## Repo structure

```
rough\
  rough.go               # public API: TUI(embed.FS), AddPlugin, AddMan
  engine\                # the ENGINE: clean, no plugins (layers by file)
    engine.go            # Run (main loop), execAction, crash.log
    loader_tiles.go      # tiles.json: pages, tiles, menu
    loader_theme.go      # themes: symbols and colors
    backend_html.go      # HTML → DOM → cells, columns, output to block
    backend_plugin.go    # <plugin> tag: pipe + interval cache
    backend_img.go       # PPM images
    backend_clickzones.go# clickable zones, hit-test, checkbox/select state
    frontend_*.go        # canvas, tile stretcher, borders, tabs, focus, widgets
    people_input_*.go    # keyboard + unified mouse (desktop/teletype)
    plugin_registry.go   # plugin registry + pipes + ParseArgs + LoadUI
    vars.go              # session vars: SetVar/VarLine, $name substitution
    syntax_checker.go    # check: action/plugin/links exist
  plugins\               # ready-made plugins: cat, hello, ssh, curl, man, grep, ...
  example_project\       # demo project for README GIFs (separate module)
  docs_ru\               # this documentation, Russian
  docs_en\               # this documentation, English
```

Important: `example_project\` is a separate module with
`replace github.com/arctcl/rough => ../`. Everything uses the same core engine —
no code duplication.
