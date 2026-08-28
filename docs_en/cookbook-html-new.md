# Cookbook: HTML from scratch

rough renders a mini-HTML layout. This guide goes from general to specific,
with live examples.

## 1. The idea

Each tile is an HTML file. Elements are clickable when they carry an `action`.
A command plugin runs on click; its output goes to a target.

## 2. Pages and tiles (tiles.json)

The screen is split into tiles. Each tile has an `id`, position and size in
percent/pixels, and an HTML file:

```json
{ "pattern": ["id","x","y","w","h","file"],
  "/main": [["out","0%","0%","100%","100%","tiles/out.html"]] }
```

## 3. Text and markup

```html
<h1>Title</h1>
<p>Paragraph with <b>bold</b> and <i>italic</i>.</p>
<hr/>
```

## 4. Button and command (action)

```html
<button action="hello">Say hi</button>
<button action="cat:/etc/hostname">Hostname</button>
```

### 4.1. Loop over a range `[N-M]`

A range in brackets expands into a loop: the command runs for every value, and
the outputs are glued into one block. Great for a fleet of hosts:

```html
<button action="ssh:root:192.168.1.[1-250]:apt upgrade" output="out">Upgrade all</button>
```

`[1-250]` expands to `192.168.1.1 … 192.168.1.250`. Also supported:
- a **step** `[N-M:S]`: `[0-1000:500]` → `0,500,1000`;
- letters: `[a-c]` → a, b, c;
- a list: `[1,4,9]`;
- several ranges at once (cartesian product): `192.168.[1-2].[1-2]` → 4 addresses.

Pair the step with the `line` plugin (`line:N-M` — lines N..M) to process a big
file in chunks of 500 without loading it all:
`cat:data.log | line:0-500 | grep:needle | wc:lines`.

A range can hold a variable: `[0-$n:500]` expands when the step runs (after
`$n` becomes a number) — so a count from `wc` turns into a sweep.

Content that does **not** look like a range/list (e.g. `[foo]`) is left as-is.

### 4.2. Repeat the rest of the pipe `loop:N`

`loop:N` is a reserved keyword: it just runs the remaining steps of the pipe
**N times** and glues the outputs. Example — run the `sl` train over ssh once
per found word:

```html
<button action="clear && cat:lev_tolstoy.txt | wc:lines | export:ln_sum
  && line:[0-$ln_sum:500] | grep:$in | wc:lines | export:count
  && loop:$count | ssh:root:127.0.0.1:22:sl" label="Word" output="out">Word</button>
```

`$count` is already expanded to a number (e.g. `loop:7`), so the train runs
exactly 7 times. `loop:1` is the same as no loop. The range `[0-$ln_sum:500]`
holds a variable: it expands when the step runs (after `$ln_sum` becomes a
number), not at startup when the variable is empty.

## 5. Pipes

`|` chains commands: the output of one feeds the next.

```html
<button action="cat:/var/log/app.log | tail:20">Tail log</button>
```

### 5.1. Several pipes: `&&` and gluing with `clear`

A button can run several INDEPENDENT pipes in a row via `&&`; their outputs
are **glued** into one block (like `cat a b`):

```html
<button action="clear && man:ssh && cat:/etc/hosts">Help + hosts</button>
```

- `clear` empties the block and starts gluing (a clean slate);
- the following pipes **add** their output to the block.

Without `clear` only the last pipe's output lands in the target.

### 5.2. Variables: `export`, `$name`, `unexport`

`export` and `unexport` are **engine keywords** (not plugins): the engine itself
collects and stores variables. A variable saved with `export` is **available
always and everywhere** — from any button, field, tile, and even from a
**different `&&` pipe**. `$name` is substituted when the step *runs* (not when
the action is parsed), so `export` from an earlier pipe writes before a later
pipe reads it.

```html
<button action="ssh:root:srv1::hostname | export:host">Remember</button>
<button action="ssh:root:$host::uptime">Uptime on $host</button>
```

Use it across pipes:

```html
<button action="cat:data.log | wc:lines | export:n
  && hello:lines $n" >How many lines</button>
```

Delete a variable with `unexport` (the opposite of `export`):

```html
<button action="... | export:tmp | ... | unexport:tmp">temporary</button>
```

After `unexport:name` the variable `$name` is gone (expands to empty).

#### Accumulator: `export:NAME +=`

Plain `export:name` **overwrites** the variable on every call — in a multi-step
pipe only the last number survives, and there is no sum over chunks. The form
`export:NAME +=` makes the engine **add** a number to the current value instead
of replacing it. The number comes from the current pipe output (the chunk) or
is given explicitly after the operator (`export:count += 5`). The engine does
the summing itself — the `wc` plugin just counts the lines of one chunk.

```html
<button action="clear
  && cat:log.txt | line:0-499  | wc:lines | export:count +=
  && cat:log.txt | line:500-999 | wc:lines | export:count +=
  && cat:log.txt | line:1000-1499 | wc:lines | export:count +=
  && hello:total $count" >Sum over chunks</button>
```

Three 500-line chunks: after each `export:count +=` the `$count` grows rather
than overwrites, ending with the total. (Note: `line` reads the file whole —
the "chunks" are only iteration boundaries, not a memory optimization.)

## 6. Where the result goes

- `output="id"` → into a `<div id="id">` tile;
- no output → the status line.

```html
<div id="out"></div>
<button action="man:ssh" output="out">Help on ssh</button>
```

## 7. Input field

```html
<input action="man:" label="Package"/>
```
Type `ssh`, press Enter → runs `man:ssh`.

The typed value is a literal (quoted) — no injection. How it lands:
- **`$in` in the action** → substituted as an argument at that exact spot,
  anywhere in a pipe: `grep:$in | sort`, `man:$in`.
- **a pipe without `$in`** → goes as **stdin to the first plugin** (Linux style,
  `echo value | plugin | ...`).
- **a single plugin without `|`** → appended as its argument (as before):
  `man:` + "ssh" → `man:ssh`.

## 8. Configs: checkbox and select

```html
<checkbox action="toggle:app.conf:debug">Debug</checkbox>
<select action="set:app.conf:loglevel" options="info:debug:trace" label="Level"/>
```

## 9. Live content: <plugin> on a timer

```html
<plugin pipe="emu:a | chart:0:100:1:2:ALPHA" height="7" interval="2s" updateanytime="1"/>
```
`interval` — refresh rate; `updateanytime="1"` keeps it running on inactive tabs.

## 10. Cheatsheet: tags

| Tag | Meaning | Example |
|---|---|---|
| `<h1>/<p>/<b>/<i>` | text | `<b>bold</b>` |
| `<hr/>` | horizontal line | `<hr/>` |
| `<button action>` | command button | `<button action="cat:/x">` |
| `<input action label output>` | input field | `<input action="man:" label="Pkg"/>` |
| `<checkbox action>` | toggle | `<checkbox action="toggle:a:debug">` |
| `<select action options label>` | dropdown | `<select action="set:…" options="a:b:c"/>` |
| `<plugin pipe interval height updateanytime>` | live widget | `<plugin pipe="…" interval="2s"/>` |
| `output="id"` | where the result goes | `<button action="…" output="out">` |
| `\| confirm` | confirm dialog | `<button action="… \| confirm">` |
