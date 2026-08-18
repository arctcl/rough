# Cookbook: themes

A theme is a JSON file in `themes/*.json` — symbols for frames/buttons/fields
and colors.

## 1. Structure

```json
{
  "name": "default",
  "symbols": {
    "tile_tl": "╭", "tile_tr": "╮", "tile_bl": "╯", "tile_br": "╰",
    "tile_h": "─", "tile_v": "│",
    "button_l": "⟨", "button_r": "⟩",
    "input_l": "[", "input_r": "]", "input_icon": "✎",
    "cursor": "░",
    "status_tl": "╭", "status_tr": "╮", "status_bl": "╯", "status_br": "╰",
    "status_h": "─", "status_v": "│"
  },
  "colors": {
    "bg": "default", "fg": "color_7",
    "frame": "color_2",
    "header_bg": "color_8", "header_fg": "color_7",
    "title_fg": "color_7",
    "active_bg": "color_8", "active_fg": "color_10",
    "status_fg": "color_11", "status_bg": "default",
    "input_fg": "color_10",
    "color_0": "0", "color_1": "1", "color_2": "2", "color_3": "3",
    "color_4": "4", "color_5": "5", "color_6": "6", "color_7": "7",
    "color_8": "8", "color_9": "9", "color_10": "10", "color_11": "11",
    "color_12": "12", "color_13": "13", "color_14": "14", "color_15": "15"
  }
}
```

## 2. Colors

- `bg`/`fg` — the default background and text.
- `frame` — tile borders, scrollbars.
- `header_bg`/`header_fg` — inactive tabs and tile name bars.
- `title_fg` — tile names (text).
- `active_bg`/`active_fg` — the active tab.
- `status_fg`/`status_bg` — the status window.
- `input_fg` — input fields and select.
- `color_0..15` — the 16-color palette used by plugins (charts, bars).

A color value is: `"default"` (terminal default), an ANSI palette number
(`"2"`), a `color_N` key, or hex `"#rrggbb"`.

## 3. Selecting a theme

`tiles.json` key `"theme"` names the theme file. Default: `"default"`.
The `theme` plugin can switch themes at runtime.
