// Плагин konami — easter egg «GLHF mate!» и страница-пасхалка.
// Сам код (↑ ↑ ↓ ↓ ← → ← → B A) больше НЕ регистрируется здесь жёстко:
// его ведёт инжектор chch (chch.json → /konami), открывая секретную страницу
// konami_page.html с этим плагином. Плагин glhf остаётся для вывода приветствия.
package konami

import (
	"github.com/arctcl/rough"
)

const man_glhf = `glhf: easter egg — returns "GLHF mate!"`

func init() {
	// Плагин, который выводит приветствие (живёт на странице /konami).
	rough.AddPlugin("glhf", func(in []string, args []string) ([]string, error) {
		return []string{"GLHF mate!"}, nil
	})
}

