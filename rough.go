// Пакет rough — чисто движок TUI.
// Внутри НЕТ ни одного плагина: движок только рисует HTML в терминал, ловит
// мышь и держит реестр. Всё остальное (плагины, темы, данные) — в проекте
// пользователя: он импортирует свои плагины и передаёт движку вшитую папку /rough.
package rough

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/arctcl/rough/engine"
)

// TUI запускает интерфейс, если в аргументах есть -tui.
// Пользователь передаёт папку /rough, вшитую через //go:embed:
//
//	//go:embed rough
//	var roughDir embed.FS
//
//	func main() { rough.TUI(roughDir) }
//
// Возвращает true, если интерфейс был запущен и закрыт, иначе false,
// чтобы основной CLI проекта продолжил работать как обычно.
func TUI(fsys fs.FS) bool {
	if !hasTUI() {
		return false
	}
	if err := Run(fsys); err != nil {
		fmt.Fprintln(os.Stderr, "rough:", err)
	}
	return true
}

// hasTUI проверяет флаг -tui в аргументах командной строки.
func hasTUI() bool {
	for _, a := range os.Args[1:] {
		if a == "-tui" || a == "--tui" {
			return true
		}
	}
	return false
}

// Run запускает движок: берёт из вшитой папки /rough страницы, темы, html.
func Run(fsys fs.FS) error {
	// Папка /rough вшита с префиксом "github.com/arctcl/rough/..." — срезаем его, дальше пути
	// относительные: tiles.json, tiles/*.html, themes/*.json.
	sub, err := fs.Sub(fsys, "github.com/arctcl/rough")
	if err != nil {
		return err
	}
	return engine.Run(sub)
}

// PluginFunc — единый контракт плагина (юникс-команда): строки на входе, строки на выходе.
type PluginFunc = engine.PluginFunc

// AddPlugin регистрирует плагин (юникс-команда: строки → строки).
// Из HTML вызывается как action="имя:арг" или в пайпе "имя:арг | ...".
func AddPlugin(name string, fn engine.PluginFunc) { engine.AddPlugin(name, fn) }

// AddMan регистрирует справку по плагину (юникс-like man).
// Обязательный вызов из init() плагина — рядом с AddPlugin. Внутри плагина
// справка лежит в переменной man_<имя> и передаётся сюда целиком.
func AddMan(name, text string) { engine.AddMan(name, text) }

// ManText возвращает справку по плагину.
func ManText(name string) (string, bool) { return engine.ManText(name) }

// ManNames возвращает отсортированный список плагинов со справкой.
func ManNames() []string { return engine.ManNames() }
