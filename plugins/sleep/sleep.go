// Плагин sleep — пауза: sleep:N ждёт N секунд и ничего не выводит.
// Полезно в склейке "clear && sleep:1 && man:ssh" — задаёт темп демо.
package sleep

import (
	"fmt"
	"time"

	"github.com/arctcl/rough"
)

func init() {
	rough.AddMan("sleep", `sleep — pause for N seconds.

Usage: sleep:N (N is whole seconds).

Outputs nothing, just waits. Handy in a "&&" chain to space out the output
of several pipes over time.

Example:
  action="clear && sleep:1 && man:ssh && sleep:1 && nginx:get"`)
	rough.AddPlugin("sleep", func(in []string, args []string) ([]string, error) {
		n := 1
		if len(args) > 0 {
			fmt.Sscanf(args[0], "%d", &n)
		}
		if n < 0 {
			n = 0
		}
		time.Sleep(time.Duration(n) * time.Second)
		return nil, nil
	})
}
