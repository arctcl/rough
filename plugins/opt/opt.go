// Плагин opt — память для <select>: opt:KEY:VALUE запоминает выбор,
// opt:KEY:get читает. Всё в памяти, файлов не трогает.
package opt

import "github.com/arctcl/rough"

func init() {
	rough.AddMan("opt", `opt — a select value kept in memory (no files).

Usage:
  opt:KEY:VALUE — remember a choice (for select action="opt:KEY")
  opt:KEY:get   — return the current value (engine calls it for the select label)

Example:
  <select action="opt:theme" label="Theme" options="day:night"/>`)
	demoOpts := map[string]string{}
	rough.AddPlugin("opt", func(in []string, args []string) ([]string, error) {
		if len(args) == 0 {
			return []string{"?"}, nil
		}
		key := args[0]
		if len(args) > 1 && args[1] == "get" {
			if v, ok := demoOpts[key]; ok && v != "" {
				return []string{v}, nil
			}
			return []string{"—"}, nil
		}
		if len(args) > 1 {
			demoOpts[key] = args[1]
			return []string{demoOpts[key]}, nil
		}
		return []string{"?"}, nil
	})
}
