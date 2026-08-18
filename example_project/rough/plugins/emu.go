// Demo plugin emu — a live metric scaled to the chart range: emu:NAME:SCALE.
// wave() is shared with stats so the chart and the numbers match.
package plugins

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/arctcl/rough"
)

// wave generates a smooth "live" value 0..scale for the name-seed.
func wave(now float64, name string, scale float64) float64 {
	seed := 0
	for _, r := range name {
		seed += int(r)
	}
	v := 0.5 +
		0.35*math.Sin(now/float64(3+seed%5)+float64(seed)) +
		0.12*math.Sin(now*(1.2+float64(seed%3)*0.4)) +
		0.04*rand.Float64() - 0.02
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return v * scale
}

func init() {
	rough.AddPlugin("emu", func(in []string, args []string) ([]string, error) {
		name := "x"
		scale := 1.0
		if len(args) > 0 {
			name = args[0]
		}
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%f", &scale)
		}
		if scale <= 0 {
			scale = 1
		}
		now := float64(time.Now().UnixMilli()) / 1000.0
		return []string{fmt.Sprintf("%.3f", wave(now, name, scale))}, nil
	})
}
