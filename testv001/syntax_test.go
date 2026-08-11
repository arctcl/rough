// syntax_test.go — проверка тестера: вся вёрстка (tiles.json + HTML) должна
// проходить синтаксическую проверку движка, а все плагины в action — существовать.
package main

import (
	"io/fs"
	"strings"
	"testing"

	"rough/engine"
)

// TestSyntax — грузит вшитую папку /rough и проверяет HTML и плагины.
func TestSyntax(t *testing.T) {
	sub, err := fs.Sub(roughDir, "rough")
	if err != nil {
		t.Fatal(err)
	}
	ui, err := engine.LoadUI(sub)
	if err != nil {
		t.Fatal(err)
	}
	errs := engine.CheckSyntax(sub, ui.Pages)
	if len(errs) > 0 {
		t.Fatalf("синтаксис тестера:\n  %s", strings.Join(errs, "\n  "))
	}
}

// TestRender — каждый HTML-тайл рендерится в буфер без паник
// (ловит падения в новых тегах: table, checkbox, img, pre и т.п.).
func TestRender(t *testing.T) {
	sub, err := fs.Sub(roughDir, "rough")
	if err != nil {
		t.Fatal(err)
	}
	fs.WalkDir(sub, "tiles", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		f, err := sub.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		root, err := engine.ParseHTML(f)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		buf := engine.NewBuffer(80, 24)
		var hz []engine.Hotzone
		// Рендер без темы (nil-safe) — ловим панику, если она есть.
		engine.RenderHTML(root, buf, 0, 0, &hz)
		t.Logf("%s: ок (%d хотзон)", path, len(hz))
		return nil
	})
}
