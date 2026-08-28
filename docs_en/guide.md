# Cookbook: rough from simple to complex

A single guide: how the engine works, how to write tiles and HTML, how to build
commands. We go from the simplest («here's how a tile is written») to the
complex («here's how to write a 50-step pipeline for the button that reboots a
nuclear reactor on Linux»). Each section — a short explanation and a ready example.

> [!TIP]
> This is a **Tutorial** — read it in order. Reference tables (tags, quick
> params, range forms) are in [`reference.md`](reference.md); architecture is
> in [`architecture.md`](architecture.md).

---

## Contents

1. [What is rough](#1-what-is-rough) — idea, pages, tiles, HTML
2. [Here's how a tile is written (tiles.json)](#2-heres-how-a-tile-is-written-tilesjson) — tiles.json, pages, menu
3. [Here's how HTML is written (text and markup)](#3-heres-how-html-is-written-text-and-markup) — text, markup, color
4. [Here's how a button works (action)](#4-heres-how-a-button-works-action) — action, args, flags, man
5. [Here's how a button with confirmation works (confirm)](#5-heres-how-a-button-with-confirmation-works-confirm) — dangerous actions
6. [Here's how a picture and a table work](#6-heres-how-a-picture-and-a-table-work) — img, table
7. [Here's how a pipe works](#7-heres-how-a-pipe-works) — `|`, command chains
8. [Here's how multiple pipes work (`&&` and `clear`)](#8-heres-how-multiple-pipes-work--and-clear) — `&&`, clear, glue
9. [Here's how variables work](#9-heres-how-variables-work) — export, `$name`, accumulator
10. [Here's how ranges and loops work](#10-heres-how-ranges-and-loops-work) — `[N-M]`, step, `loop:N`
11. [Here's how quick params work (plugin contract)](#11-heres-how-quick-params-work-plugin-contract) — plugin contract, slots, flags
12. [Here's how inputs, checkboxes and selects work](#12-heres-how-inputs-checkboxes-and-selects-work) — input, toggles
13. [Here's how live content works (plugin)](#13-heres-how-live-content-works-plugin) — plugin on a timer, async
14. [Here's where the result goes (output)](#14-heres-where-the-result-goes-output) — status, output block
15. [Here's how themes work](#15-heres-how-themes-work) — list, switch
16. [Everything together: a panel](#16-everything-together-a-panel) — building a whole page

Reference (tags, params, ranges): [`reference.md`](reference.md).

---

## 1. What is rough

The engine draws pages, lays tiles out on them, and inside the tiles places
plain HTML — and the interface is ready. You don't write interface code or
assemble windows: you write HTML, and the engine turns it into terminal cells
and reacts to clicks. A button in HTML carries a command, a click on the button
runs the command, and its result is shown on screen — in the status line at the
bottom or in an output block.

The interface is divided into **pages** (routes). Each page is a set of **tiles**,
rectangles with HTML inside. Which pages exist, which tiles are on them and
where — all of this is described in a single file `tiles.json`.

---

## 2. Here's how a tile is written (tiles.json)

One page of two tiles — settings on the left, output on the right:

```json
{
  "theme": "default",
  "menu": [["Main", "/main"]],
  "/main": [
    ["cfg", "0%", "0%", "40%", "100%", "tiles/cfg.html"],
    ["out", "40%", "0%", "60%", "100%", "tiles/out.html"]
  ]
}
```

The `theme` key sets the theme from the `themes/` folder (default is `default`).
The `menu` key describes the tabs at the bottom: each tab is a «label and route»
pair. Then come the pages: the route name (e.g. `/main`) and its list of tiles.

Each tile is an array of six values: `[id, x, y, w, h, file]`. `id` is the tile
name (visible on its frame). `x`, `y`, `w`, `h` are position and size: percent of
the screen (`%`), cells (`px`) or window fractions (`vw`/`vh`). `file` is the
path to the tile's HTML file. The engine stretches tiles to fit the terminal:
resize the window — percents recalculate, and nothing breaks.

Multiple pages and tabs at the bottom:

```json
{
  "menu": [
    ["Main", "/main"],
    ["Settings", "/cfg"]
  ],
  "/main":  [["main", "0%", "0%", "100%", "100%", "tiles/main.html"]],
  "/cfg":   [["cfg",  "0%", "0%", "100%", "100%", "tiles/cfg.html"]]
}
```

A tile that stretches on window resize (percents) and a fixed-size tile (cells):

```json
["panel", "0%", "0%", "100%", "100%", "tiles/panel.html"]
["side",  "0",   "0",  "40",   "20",   "tiles/side.html"]
```

---

## 3. Here's how HTML is written (text and markup)

Everything written in a tile's HTML is drawn as-is — the tags make the text
meaningful. Heading, paragraph, empty line, bold and italic, centering, line and
monospace block:

```html
<h1>Heading</h1>
<p>An ordinary paragraph.</p>
<br/>                  <!-- empty line -->
<b>bold</b> <i>italic</i>
<center>centered</center>
<div>a block of text</div>
<hr/>                  <!-- line -->
<pre>
   monospace block
   spaces are kept
</pre>
```

The `<div>` block groups text, and `<row>` lays block-columns in a row:

```html
<row>
  <div width="50%">left column</div>
  <div width="50%">right column</div>
</row>
```

Color — with the `color` attribute for text and `bg` for background. You can use
hex (`#rrggbb`), a terminal palette number (`3`) or a theme name (`header_bg`):

```html
<h1 color="#ffcc00">Yellow heading</h1>
<p bg="4">Blue background from the palette</p>
<p color="color_2">green text</p>
<p bg="header_bg">color from the theme</p>
```

---

## 4. Here's how a button works (action)

A button runs a **command** — just like a command in Linux. The format is simple:
`name:arg1:arg2` or `name --arg=value`. The name is a plugin, arguments are
separated by colons or flags:

```html
<button action="hello">Say hello</button>
<button action="hello:world">Say hello, world</button>
<button action="cat:/etc/hostname">Show hostname</button>
```

The first button calls the `hello` plugin with no arguments. The second calls the
same `hello` with the argument `world`. The third calls `cat` with a file path:
`cat` reads the file from disk and returns its contents.

Parameters can also be given as flags, like in Linux (`cat --file=/x`), or
quick-positionally, via colons. What exactly a plugin accepts is defined by the
plugin itself; you can always find out via help: a button `action="man:name"`
shows the description.

> [!NOTE]
> Which parameters a plugin accepts and in what order — see `man:name` (or the
> «Here's how quick params work» section below).

The result of a command without an `output` attribute is shown in the status
line at the bottom of the screen.

---

## 5. Here's how a button with confirmation works (confirm)

A dangerous action (reboot, delete) is worth gating with a confirmation:
`| confirm` at the end of the command opens a window — Enter «yes», Esc «no»:

```html
<button action="ssh:root:srv::reboot | confirm">Reboot</button>
<button action="ssh:root:srv1::'apt update && apt upgrade -y' | confirm">Update</button>
```

> [!WARNING]
> `| confirm` does not cancel the action — it only asks for confirmation. Think
> about what you're running before pressing Enter.

---

## 6. Here's how a picture and a table work

A table is built from `<tr>` rows and `<td>` (normal) and `<th>` (header) cells.
Columns align themselves to the widest cell, so a table is always even:

```html
<table>
  <tr><th>Service</th><th>Status</th></tr>
  <tr><td>api</td><td>running</td></tr>
  <tr><td>db</td><td>running</td></tr>
</table>
```

A picture is drawn from a PPM P6 file — in half blocks, two pixels per terminal
cell. Just give the path in `src`:

```html
<img src="/opt/app/logo.ppm"/>
```

A link switches the page to another route — that's how you make navigation:

```html
<a href="/main">← Home</a>
```

---

## 7. Here's how a pipe works

The pipe `|` joins commands into a chain: the output of one command goes into
the input of the next, like in a shell. For example, take the tail of a log or
filter the lines of a config:

```html
<button action="cat:/var/log/app.log | tail:20">Log tail</button>
<button action="cat:/etc/x.conf | grep:^server | head:5">Servers (5)</button>
```

Here `cat` reads the file and returns lines, `grep` keeps only the needed ones,
and `head` cuts to the first five. The first plugin can be a «source» — fetch
data itself, for example over ssh:

```html
<button action="ssh:root:srv::uptime | head:3">Uptime</button>
```

If an error happens at some step — the pipe stops and the error message is shown
in the status line.

---

## 8. Here's how multiple pipes work (`&&` and `clear`)

One button can run several INDEPENDENT pipes in a row — via `&&`. Each `&&`
chunk runs separately (not as an output→input pipeline), and the results are
**glued** into one output block (like `cat a b`):

```html
<button action="clear && man:ssh && cat:/etc/hosts">Help + hosts</button>
```

- `clear` clears the output block and turns on gluing mode (starts from a clean slate);
- the following pipes **add** their output to the block, not overwrite it.

Without `clear` several pipes also run, but only the last one lands in the
receiver (the previous ones are overwritten).

---

## 9. Here's how variables work

`export` and `unexport` are **engine keywords** (not plugins): the engine itself
collects and stores variables. A variable saved with `export` is **available
always and everywhere** — from any button, field, tile, and even from another
`&&` pipe. `$name` is substituted when the step runs (not when the action is
parsed), so `export` from an earlier pipe writes before a later pipe reads it.

Save a result and substitute it later:

```html
<button action="ssh:root:srv1::hostname | export:host">Remember</button>
<button action="ssh:root:$host::uptime">Uptime on $host</button>
```

Use a variable from another `&&` pipe:

```html
<button action="cat:data.log | wc | export:n
  && hello:lines $n" >How many lines</button>
```

Delete a variable — `unexport` (the opposite of `export`):

```html
<button action="... | export:tmp | ... | unexport:tmp">temporary</button>
```

Separate the variable name from neighbors — `${name}`; trim suffix/prefix — like
in bash (`${f%.log}`, `${f#/var/}`); a literal `$` — `\$`; don't substitute at
all — quotes `'$name'`.

**Accumulator.** Plain `export:name` overwrites the variable on every call — in
a multi-step pipe only the last number survives, and there's no sum over chunks.
The form `export:NAME +=` makes the engine **add** a number to the current value.
The number comes from the current pipe output (the chunk) or is given explicitly
(`export:count += 5`). The engine sums itself — the `wc` plugin only counts the
lines of one chunk:

```html
<button action="clear
  && cat:log.txt | line:0-499  | wc | export:count +=
  && cat:log.txt | line:500-999 | wc | export:count +=
  && cat:log.txt | line:1000-1499 | wc | export:count +=
  && hello:total $count" >Sum over chunks</button>
```

(Note: `line:0-499` takes lines 0–499 from the pipe input; `cat` reads the
whole file.)

---

## 10. Here's how ranges and loops work

A range in square brackets expands into an iteration: the command runs on every
value, outputs are glued. That's how you upgrade a fleet of servers:

```html
<button action="ssh:root:192.168.1.[1-250]::apt upgrade" output="out">Upgrade all</button>
```

`[1-250]` → `192.168.1.1 … 192.168.1.250`.

Also supported:
- a step `[N-M:S]`: `[0-1000:100]` → `0,100,200,…,1000`;
- letters `[a-c]` → a, b, c;
- a list `[1,4,9]`;
- several ranges at once (cartesian product): `192.168.[1-2].[1-2]` → 4 addresses.

A range can hold a variable: `[0-$n:500]` expands when the step runs (after `$n`
becomes a number), not at startup when the variable is empty.

Content that doesn't look like a range (e.g. `[foo]`) is left as-is.

**Loop `loop:N`** — a keyword: runs the remaining pipe steps N times and glues
the outputs. `$count` is already expanded to a number, so the command runs
exactly that many times:

```html
<button action="clear && cat:lev_tolstoy.txt | wc | export:ln_sum
  && line:[0-$ln_sum:500] | grep:$in | wc | export:count +=
  && loop:$count | ssh:root:127.0.0.1:22:sl" label="Word" output="out">Word</button>
```

`loop:1` is the same as no loop.

---

## 11. Here's how quick params work (plugin contract)

Each plugin defines which parameters it accepts and in what order — that's the
**plugin contract**. The engine only provides a common input mechanism:
parameters can be passed **positionally** (via colons, in order) or **as flags**
(`--name=value`), and you can mix them.

Positionally — in order:

```text
chart:0:100:1:2:CPU
```

Skip a parameter (default is used) — an empty slot `::`:

```text
chart:0:100::2:CPU
```

As flags — order doesn't matter:

```text
chart --min=0 --max=100 --title=CPU
```

Mix slots and flags:

```text
chart::100 --title=CPU
```

A value that contains `:` or `|` — put it in quotes:

```text
sed:':':1
```

The last parameter «swallows» the rest of the colons — this is how you pass a
command with an arbitrary tail:

```text
ssh:user:host:67:'docker compose down && up'
```

`&&` inside a command value is also a special character (a pipe separator), so
a command with it goes in quotes, like `:` / `|`.

What exactly a plugin accepts and its defaults — look in `man:name` (a button
`action="man:name"`).

> [!IMPORTANT]
> The parameter contract is different for every plugin: the engine only gives
> two input ways (positional and flags), but what the params are and their
> defaults — each has its own, always via `man:name`.

---

## 12. Here's how inputs, checkboxes and selects work

An input field builds a command from its `action` and the text the user types.
Clicking the field activates it, the user types, presses Enter — and the engine
joins the `action` with the input via a colon. Type `ssh` into a field with
`action="man:"` — `man:ssh` runs:

```html
<input action="man:" label="Package"/>   <!-- typed ssh → man:ssh -->
```

The input can be placed into a specific spot in the pipe via `$in`, and if there
is no `$in` — the input goes as «input» to the first plugin:

```html
<input action="grep:$in | sort" label="Pattern"/>
<input action="cat | grep | sort" label="Data"/>
```

Esc cancels the input.

> [!IMPORTANT]
> Input from a field is always a single value. The engine wraps it in quotes, so
> `|`, `:`, `$` inside the input don't work as syntax: you can't type `cat |
> ssh:...` into a field — it stays plain text, not a command.

A checkbox — a toggle: clicking switches a key's value in the file (0↔1, on↔off):

```html
<checkbox action="toggle:app.conf:logging">Logging</checkbox>
```

A dropdown — a choice of one value from a set; the chosen variant is appended to
the command:

```html
<select action="set:app.conf:loglevel" options="info:debug:trace" label="Level"/>
```

Lists support nesting: a variant with a submenu is written as
`Label [children:variants]`:

```html
<select action="set:app.conf:theme"
        options="Dark:Light:Colorful [Red:Green:Blue]" label="Theme"/>
```

If the config isn't in `key=value` form but space-separated — add the flag
`--sep=space` (for `toggle` and `set`):

```html
<checkbox action="toggle:/etc/mailcow:debug --sep=space">Debug</checkbox>
```

---

## 13. Here's how live content works (plugin)

The `<plugin>` tag updates itself, on a timer — no clicks at all. The interval is
set with the `interval` attribute (default is two seconds). That's how clocks,
load graphs or sparklines live in a tile:

```html
<plugin name="clock" interval="1s"/>                   <!-- clock -->
<plugin pipe="emu:alpha:100 | chart:0:100:1:2:CPU" height="14" interval="2s"/>
<plugin pipe="cat:data.log | tail:10 | bars" interval="2s"/>
```

`chart` is a live bar chart with axes. Its params go via colons:
`chart:MIN:MAX[:WIDTH[:SECONDS[:TITLE]]]` — the range is required, the rest uses
defaults. Height comes from `height`, width from the tile, so the chart fits the
space on its own.

> [!NOTE]
> The live source `emu` (`emu:name:scale`) is a demo plugin from
> `example_project`; the core set has no live source of its own — write yours as
> a normal plugin.

A slow source (e.g. ssh) — run it as `async` so it doesn't freeze the interface:

```html
<plugin pipe="ssh:user:host::vmstat | cut::2" interval="1s" async/>
```

---

## 14. Here's where the result goes (output)

A command result can be shown in two ways. Without an `output` attribute the
result goes to the status line at the bottom. If you specify `output="id"` — the
result lands in the `<div id="id">` block, and this block can be anywhere: in
this tile, in a neighbor, or on another page:

```html
<button action="man:ssh" output="out">ssh help</button>
<!-- in any tile: -->
<div id="out"></div>
```

---

## 15. Here's how themes work

List themes and switch on the fly — via the `theme` plugin:

```text
theme:list
theme:orange
```

---

## 16. Everything together: a panel

Let's build a small panel: one page, a settings tile on the left, an output tile
on the right. It all starts with `tiles.json`:

```json
{
  "theme": "default",
  "menu": [
    ["Panel", "/main"],
    ["Help", "/man"]
  ],
  "/main": [
    ["cfg", "0%", "0%", "40%", "100%", "tiles/cfg.html"],
    ["out", "40%", "0%", "60%", "100%", "tiles/out.html"]
  ],
  "/man": [
    ["man", "0%", "0%", "100%", "100%", "tiles/man.html"]
  ]
}
```

In the settings tile live a checkbox, a dropdown, an input field, confirmation
buttons and a live chart:

```html
<h1 color="color_2">Panel</h1>

<checkbox action="toggle:app.conf:debug">Debug</checkbox>
<select action="set:app.conf:loglevel" options="info:debug:trace" label="Level"/>

<input action="man:" output="out" label="Plugin help"/>

<button action="cat:app.conf" output="out">Show config</button>
<button action="clear && man:ssh && cat:app.conf" output="out">Help + config</button>
<button action="ssh:root:srv::reboot | confirm">Restart</button>

<plugin pipe="emu:alpha:100 | chart:0:100:1:2:CPU" height="10" interval="1s"/>
<plugin pipe="ps --track=1" interval="1s" async/>

<hr/>
<button action="ssh:root:srv::hostname | export:host">Remember host</button>
<button action="ssh:root:$host::uptime" output="out">Uptime on $host</button>

<a href="/man">Help →</a>
```

And in `tiles/out.html` — just a heading and the output receiver:

```html
<h1>Output</h1>
<div id="out"></div>
```

Done: buttons change the config, checkboxes toggle flags, and the result of any
command is shown in the right tile.

---

## Next

This was the Tutorial. Now, when you forget something or look for a specific
fact (a tag, a param slot, a range form) — open [`reference.md`](reference.md):
plain reference tables without explanations live there.
