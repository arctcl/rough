# rough

ROUGH outlines UI — go html

A simple tiled, resizable engine for quickly building a terminal UI with a mix of HTML markup and linux-like commands.

No web server, no browser. Use it as a library in your project, or as a standalone utility (with ssh and curl as plugins, among others) — and whatever else you come up with: hook it up to a COM port and drive a machine, or build a config editor so no clueless admin can click anything extra on the servers.

## What's going on

![UI demo](docs_ru/gifs/stiky.gif)

I'm tired of endless web servers in projects where they're not needed at all, and I'm equally tired of endless config files.
So: the same HTML, the same linux-like commands, and it all renders right in the terminal with buttons, fields and spinners. On top of that — the engine stretches tiles and their content to its own size by itself: resize the window and nothing breaks.

(and u can click the mouse)

![mouse](docs_ru/gifs/mouse.gif)

## Using it in a project

![Создание тайлов](docs_ru/gifs/tiles.gif)

Three steps:

**1. Draw tiles** — divide the screen into rectangles with percentages or pixels in `tiles.json`:
```json
{
  "pattern": ["id", "x", "y", "w", "h", "file"],
  "/main": [
    ["cfg",  "0%",  "0%",  "40%", "100%", "tiles/cfg.html"],
    ["out",  "40%", "0%",  "60%", "100%", "tiles/out.html"]
  ]
}
```
Left — settings tile, right — output.

**2. Write HTML markup** inside each tile — headings, buttons, fields:

![html-simly](docs_ru/gifs/html.gif)

```html
<!-- tiles/cfg.html -->
<h1>Settings</h1>
<button action="cat:/etc/hostname">Hostname</button>
```

**3. Call plugins right from HTML** — like linux commands:
```html
<button action="cat:/etc/hostname">Hostname</button>
<button action="ssh:root:srv1::uptime">Uptime on srv1</button>
```

The "guts" of the buttons are written in Go — they're command plugins: **strings in, strings out**. The engine itself can't do anything — all the logic lives in plugins. Contract requirements are in the cookbook.

#Quick Start

```bash
git clone https://github.com/arctcl/rough
cd rough/example_project
go run . -tui
```

## What it looks like

Type a package name in a field — the help appears in a neighbouring tile:

```html
<!-- "input" tile -->
<input action="man:" output="out" label="Package"/>

<!-- "output" tile (id="out" (IMPORTANT)): the engine draws the command help here -->
```

`man` is an output command, like `cat`: whatever it runs, it shows. A button, an input field, a pipe — all of them are commands, and the result of any of them can be routed to your tile.

> Demo: `example_project/` — full demo (4 tabs: live charts, nginx builder, man, about). Run: `cd example_project && go run . -tui`.
> GIF source: `example_project/` — demo with 4 tabs (live charts, nginx config builder, man, about) to record `docs_ru/gifs/demo.gif`. Run: `cd example_project && go run . -tui`.

## Live example:

Say your project lives in `/opt/my_docker_project/conf.conf` and you want to give an admin a "set max users" button. Easy:

```html
<!-- field: type 100 and press Enter → set writes max_users=100 -->
<input action="set:/opt/my_docker_project/conf.conf:max_users" label="max_users"/>

<!-- or a button with a fixed value right away -->
<button action="set:/opt/my_docker_project/conf.conf:max_users:100 | confirm">max_users = 100</button>
<button action="ssh:user:localhost:67:docker compose down && docker compose --project-directory /opt/my_docker_project/ up -d | confirm">REBOOT DOCKER</button>
<button action="ssh:user:localhost::docker compose down && docker compose --project-directory /opt/my_docker_project/ up -d | confirm">THIS BUTTON USE DEFAULT PORT - "::" its default/empty - </button>
```

## Quick parameters

You can use a quick method for entering parameters—they are positional and support skipping. An example of the same action:
```html
 action="ssh:user:localhost:67:docker compose down"
 action="shh::docker compose down --user=user --host=localhost --port=67
 (yep this is ipv6 logic)
```
The parameters, their placement, and their default values ​​are defined within the plugin itself—the engine simply passes them through—see the cookbooks.

## How to use in your Go project

1. Add the module: `go get github.com/arctcl/rough`
2. Put a `rough/` folder next to `main.go`: `tiles.json`, `tiles/*.html`, `themes/*.json` and `plugins/plugins.go` (link to the plugins).
3. Embed the folder and run:

```go
//go:embed rough
var roughDir embed.FS

func main() { rough.TUI(roughDir) }
```

Run with the UI — `myapp -tui` (without the flag the program works as usual).
Your own plugins — `rough.AddPlugin("name", func(...) ...)` in your `main.go`.
Everything is embedded into a single file — no `rough/` folder needed in production.

## What you can build

**Admin panel.** Settings checkboxes, tables, log output:

```html
<checkbox action="toggle:app.conf:logging">Logging</checkbox>
<button action="cat:/var/log/app/errors.log | grep:ERROR">Errors</button>
```

**SSH orchestrator.** A "deploy update" button — `ssh` with keys from a folder, `| confirm` asks before running:

```html
<button action="ssh:root:srv1::apt update && apt upgrade -y | confirm">Deploy update</button>
```

`ssh` runs a command on a host (keys from `/root/keys`), `| confirm` — a confirmation window before deploying.

**Any automation.** Timed charts, panic buttons:

```html
<plugin pipe="cat:/tmp/core_temp | cut::1 | bars" interval="1s"/>
```

## Idea

- engine = empty frame + cursor; all the logic — plugins
- button = plugin call: `<button action="cat:/etc/hostname">`
- everything natively in Go — ssh, curl, sqlite; works even in a scratch container
- plugin help — right in the UI (`man`)

More: [docs (EN)](docs_en/), [docs (RU)](docs_ru/) — cookbooks (plugins, themes, project), how-it-works, architecture.

## License

Engine and plugins — [MIT](LICENSE). Dependencies are permissive (Apache-2.0, MIT, BSD-3) — no copyleft, safe for closed-source projects.
