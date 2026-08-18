// Demo plugin stats — a plain-text monitor: the same metrics as the charts (emu).
package plugins

import (
	"fmt"
	"time"

	"github.com/arctcl/rough"
)

func init() {
	rough.AddPlugin("stats", func(in []string, args []string) ([]string, error) {
		now := float64(time.Now().UnixMilli()) / 1000.0
		return []string{
			fmt.Sprintf("alpha = %.2f", wave(now, "alpha", 1)),
			fmt.Sprintf("beta  = %.2f", wave(now, "beta", 10)),
			fmt.Sprintf("gamma = %.0f", wave(now, "gamma", 100)),
			fmt.Sprintf("delta = %.0f", wave(now, "delta", 1000)),
		}, nil
	})
}
