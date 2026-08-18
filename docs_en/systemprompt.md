# Development rules (system prompt)

Rules for developing rough — for AI agents and humans.

## Goals

- Keep the engine pure: no plugins inside `engine/`.
- All behaviour lives in plugins (strings in, strings out).
- No dead code: projects import only the plugins they use.

## Layout

- `rough.go` + `engine/` — the engine (public API, renderer, input, widgets).
- `plugins/` — ready-made plugins, one package per folder.
- `example_project/` — the demo project (separate module, `replace => ../`).
- `docs_ru/`, `docs_en/` — documentation in Russian and English.

## Conventions

- Comments, docs and errors: keep UI strings in English (the interface is
  English); internal comments may be in Russian or English.
- `AddPlugin` in `init()`, add a `man` page via `AddMan`.
- A panic is caught by the engine; never crash the UI.
- Preserve UTF-8; the terminal must handle it.

## Git commits

- Imperative title (~60 chars): `Added`, `Renamed`, `Fixed`, `Documented`.
- Group by folder: engine → `engine/` + `rough.go`, plugins → `plugins/`,
  example → `example_project/`, docs → `docs_ru/` + `docs_en/`.
- No `wip`/`фикс` — if you cannot explain it in the title, it is not one commit.
