// Плагин konami — возвращает конами-код как пасхалку: набираешь
// ↑ ↑ ↓ ↓ ← → ← → B A → в статус-строке появляется "GLHF mate!".
// Регистрирует последовательность через engine.AddCheat (движок отслеживает
// ввод) и плагин glhf (вывод приветствия).
package konami

import (
	"github.com/arctcl/rough"
)

const man_glhf = `glhf: easter egg — returns "GLHF mate!"`

func init() {
	// Плагин, который выводит приветствие (запускается конами-кодом).
	rough.AddPlugin("glhf", func(in []string, args []string) ([]string, error) {
		return []string{"GLHF mate!"}, nil
	})
	// Конами-код: ↑ ↑ ↓ ↓ ← → ← → B A.
	rough.AddCheat("UUDDLRLRba", "glhf")
}
