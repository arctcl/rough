# Architecture

**rough** is a tiled, resizable engine that renders a mini-HTML layout straight
into the terminal. It is built in layers — the engine knows nothing about
plugins; all behaviour lives in command plugins.

## Layers

```
rough.go (public API) → engine/ (core) → plugins/ (commands)
```

- **rough.go** — the public surface: `TUI(embed.FS)`, `AddPlugin`, `AddMan`.
- **engine/** — pure core: HTML parsing, rendering, input, widgets, tiles,
  tabs, themes, click zones. No plugins here.
- **plugins/** — commands: strings in, strings out. The project picks which ones
  to import (dead code is not pulled in).

## Key ideas

1. **Tiles** — the screen is split into rectangles by percentage/pixels in
   `tiles.json`. Each tile draws an HTML file. Resize the window — tiles
   re-stretch (the "rubber" part).
2. **Actions** — every interactive element carries an `action` string. The
   engine parses it into a pipe (`a | b`), executes it, and routes the output to
   a target (`output="id"` or the status line).
3. **Pipes** — `|` is a pipeline (output → input). `&&` runs several
   independent pipes in a row, and `clear` starts gluing them into one block.
4. **Plugins are commands** — no engine-side behaviour. A plugin takes `in
   []string`, `args []string` and returns `[]string` (or an error). It registers
   itself in `init()` via `rough.AddPlugin`.
5. **Themes** — symbols and colors in `themes/*.json`, switched at runtime.
6. **Diff rendering** — only changed cells are redrawn (`frontend_buffer.go`),
   so static UI does not flicker.

## Data flow

```
HTML → ParseHTML (x/net/html) → convertNode → renderTile → Buffer → Blit (diff) → screen
click/Enter → HitTest → execAction(action, output) → RunSteps (pipe) → putOutput
```

Live widgets: `<plugin interval="2s" updateanytime="1">` keeps running in the
background for inactive tabs (`renderBackgroundPages`).

## Conventions

- Engine files are split by concern (`backend_*`, `frontend_*`, `people_input_*`).
- `curTheme`, `curW/curH`, `pluginCache` are package globals reset per frame.
- Syntax is validated at startup (`syntax_checker.go`) — a wrong action/plugin
  link prevents the UI from starting.
