// Демо-плагин flag — чекбокс в памяти: flag:KEY переключает, flag:KEY:get читает.
package plugins

import "github.com/arctcl/rough"

func init() {
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
