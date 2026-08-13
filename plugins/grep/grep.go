// Плагин grep — оставить строки, подходящие под регулярное выражение.
// Юникс-лайк: `grep error file` — здесь строки приходят из пайпа.
// Вызов: action="cat:x | grep:маска"
package grep

import (
	"regexp"

	"rough"
)

// man_grep — справка по плагину (для man).
const man_grep = `grep — оставить строки, подходящие под регулярку.

Использование:
  часть пайпа: ... | grep:МАСКА

Аргументы:
  МАСКА — регулярное выражение (как в Go).

Примеры:
  action="cat:/etc/x.conf | grep:^server"     — строки, начинающиеся с server
  action="ssh:root:srv1::uptime | grep:load"    — строка с load
  action="man:ssh | grep:-i"                  — где в справке упомянут -i`

func init() {
	rough.AddMan("grep", man_grep)
	rough.AddPlugin("grep", func(in []string, args []string) ([]string, error) {
		if len(args) < 1 {
			return in, nil
		}
		re, err := regexp.Compile(args[0])
		if err != nil {
			return nil, err
		}
		var out []string
		for _, ln := range in {
			if re.MatchString(ln) {
				out = append(out, ln)
			}
		}
		return out, nil
	})
}
