# rough Reference

Plain facts for quick lookup — don't read it top to bottom, search for what you
need. Step-by-step learning is in [`guide.md`](guide.md); here only reference
tables.

> [!NOTE]
> Every plugin has a built-in help: `action="man:name"` — it's always more
> current than this file because it comes from the plugin's code.

---

## 1. HTML tags

| Tag | What it does | Example |
|---|---|---|
| text | drawn as-is | `Hi` |
| `<h1>` / `<p>` | heading / paragraph | `<h1>Settings</h1>` |
| `<br/>` | empty line | `<br/>` |
| `<b>` / `<i>` | bold / italic | `<b>x</b>` |
| `<center>` | centered | `<center>x</center>` |
| `<div>` | block | `<div>x</div>` |
| `<row>` | row of columns | `<row><div width="50%">…` |
| `<div width="%">` | column | `<div width="30%">x</div>` |
| `<div align="center">` | centered block | `<div align="center">…` |
| `<div id="id">` | output receiver | `<div id="out"></div>` |
| `<hr/>` | line | `<hr/>` |
| `<pre>` | monospace block | `<pre>…</pre>` |
| `<button action>` | button-command | `<button action="cat:/x">` |
| `<input action label output>` | input field | `<input action="man:" label="Package"/>` |
| `<checkbox action>` | toggle | `<checkbox action="toggle:a.conf:debug">` |
| `<select action options label>` | dropdown | `<select action="set:…" options="a:b:c"/>` |
| `<table>` | table | `<table><tr><td>x</td></tr></table>` |
| `<img src>` | PPM picture | `<img src="/opt/x.ppm"/>` |
| `<a href>` | page navigation | `<a href="/main">Home</a>` |
| `<plugin name/pipe interval>` | live content | `<plugin name="clock" interval="1s"/>` |
| `output="id"` | where the result goes | `<button action="…" output="out">` |
| `| confirm` | confirmation | `<button action="… | confirm">` |

### Color

| Attribute | What | Values |
|---|---|---|
| `color="…"` | text color | `#rrggbb`, palette number `3`, theme name `header_bg`, `color_2` |
| `bg="…"` | background color | same |

---

## 2. `tiles.json`: pages and tiles

| Key | What | Example |
|---|---|---|
| `theme` | theme from `themes/` | `"theme": "orange"` |
| `menu` | bottom tabs (label, route) | `"menu": [["Main","/main"]]` |
| `pattern` | tile row schema (optional) | `["id","x","y","w","h","file"]` |
| `/route` | page: list of tiles | `"/main": [["cfg","0%","0%","40%","100%","tiles/cfg.html"]]` |

A tile is an array `[id, x, y, w, h, file]`:

| Field | What |
|---|---|
| `id` | tile name (visible on the frame) |
| `x, y` | position |
| `w, h` | size |
| `file` | path to the tile's HTML |

Size units: `%` — percent of the screen, `px` — cells, `vw`/`vh` — window fractions.

---

## 3. Quick params (plugin contract)

Each plugin defines its own params and their order (the contract). The engine
provides two input ways and their mix.

| Way | Example | Note |
|---|---|---|
| positional (in order via `:`) | `chart:0:100:1:2:CPU` | order from the plugin contract |
| skip a param (default) | `chart:0:100::2:CPU` | empty slot `::` |
| flags | `chart --min=0 --max=100 --title=CPU` | order doesn't matter |
| mix | `chart::100 --title=CPU` | |
| value with `:`/`\|` | `sed:':':1` | in quotes |
| arbitrary tail | `ssh:user:host:67:'docker compose down && up'` | last param «swallows» the rest |

> [!IMPORTANT]
> The contract is different for every plugin. What the params are and their
> defaults — always via `man:name`.

---

## 4. Ranges and loops

A range in `[...]` expands into an iteration over values, outputs are glued.

| Form | Values | Example |
|---|---|---|
| `[N-M]` | all numbers | `[1-250]` → 1..250 |
| `[N-M:S]` | with a step | `[0-1000:100]` → 0,100,…,1000 |
| `[a-c]` | letters | `[a-c]` → a,b,c |
| `[v1,v2,v3]` | list | `[1,4,9]` |
| several | cartesian product | `192.168.[1-2].[1-2]` → 4 addresses |

A range can hold a variable: `[0-$n:500]` expands when the step runs (after `$n`
is substituted).

Content that doesn't look like a range (e.g. `[foo]`) is left as-is.

**Loop `loop:N`** — repeats the remaining pipe steps N times, glues the outputs:
`loop:3 | ssh:...:sl`. `loop:1` is the same as no loop. The count can come from a
variable: `loop:$count`.

---

## 5. Variables

| Form | What |
|---|---|
| `export:name` | save the current pipe output to a variable (engine keyword) |
| `$name` | substitute the value |
| `${name}` | separate the name from neighbors |
| `${f%.log}` / `${f#/var/}` | trim suffix / prefix (like bash) |
| `\$name` | literal `$` (don't substitute) |
| `'$name'` | substitution off inside quotes |
| `unexport:name` | delete a variable (the opposite of export) |
| `export:name += [NUMBER]` | accumulator: add to the current value |

`export`/`unexport` are **engine keywords**, not plugins. A variable is available
always and everywhere; `$name` is substituted when the step runs.

---

## 6. Standard plugins

The list of available plugins and their params — via `action="man:name"` (the
engine generates help from the plugin code). Main ones:

| Plugin | What it does | Typical call |
|---|---|---|
| `cat` | reads a file | `cat:/etc/hostname` |
| `wc` | counts input lines | `cat:x | wc` |
| `grep` | keeps matching lines | `cat:x | grep:^server` |
| `head` / `tail` | first / last lines | `cat:x | tail:20` |
| `line` | a line or line range | `cat:x | line:5-10` |
| `sed` | text replacement | `cat:x | sed:old:new` |
| `sort` / `uniq` / `tr` | sort / collapse / char replace | `cat:x | sort` |
| `cut` / `awk` | fields and filters | `cat:x | cut::2` |
| `ssh` | command on a remote server | `ssh:user:host::cmd` |
| `curl` | HTTP request | `curl:https://…` |
| `man` | plugin help | `man:ssh` |
| `chart` | live bar chart | `… | chart:0:100:1:2:CPU` |
| `clock` | clock | `<plugin name="clock"/>` |
| `set` / `toggle` | edit `key=value` configs | `set:app.conf:loglevel:info` |
| `theme` | list / switch theme | `theme:list`, `theme:orange` |
| `ps` | processes/goroutines | `ps` |

Full list and per-plugin help — `man:name`.
