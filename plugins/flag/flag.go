// Плагин flag — чекбокс в памяти: flag:KEY переключает, flag:KEY:get читает.
// Без файлов и рабочей директории — работает где угодно.
package flag

import "github.com/arctcl/rough"

func init() {
	rough.AddMan("flag", `flag — a checkbox in memory (no files).

Usage:
  flag:KEY      — toggle (for checkbox action="flag:KEY")
  flag:KEY:get  — return "on"/"off" (engine calls it for the check mark)

Example:
  <checkbox action="flag:verbose">Verbose output</checkbox>`)
	demoFlags := map[string]bool{}
	rough.AddPlugin("flag", func(in []string, args []string) ([]string, error) {
		if len(args) == 0 {
			return []string{"off"}, nil
		}
		key := args[0]
		if len(args) > 1 && args[1] == "get" {
			if demoFlags[key] {
				return []string{"on"}, nil
			}
			return []string{"off"}, nil
		}
		demoFlags[key] = !demoFlags[key]
		if demoFlags[key] {
			return []string{"on"}, nil
		}
		return []string{"off"}, nil
	})
}
