// Демо-плагин deploy — симулирует МЕДЛЕННУЮ фоновую задачу (для async-демо).
// Без async кнопка с deploy:4 заморозит интерфейс на 4 секунды.
// С async — задача уходит в фон, интерфейс живёт, результат придёт сам.
package plugins

import (
	"fmt"
	"time"

	"github.com/arctcl/rough"
)

const man_deploy = `deploy: simulate a slow background task (async demo).
Usage: deploy:seconds
Example: deploy:4   (sleeps 4s, then returns a message)`

func init() {
	rough.AddMan("deploy", man_deploy)
	rough.AddPlugin("deploy", func(in []string, args []string) ([]string, error) {
		n := 3
		if len(args) > 0 {
			fmt.Sscanf(args[0], "%d", &n)
		}
		if n < 1 {
			n = 1
		}
		if n > 10 {
			n = 10
		}
		time.Sleep(time.Duration(n) * time.Second)
		return []string{fmt.Sprintf("deploy ok after %ds", n)}, nil
	})
}
