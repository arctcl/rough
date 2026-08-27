// Demo plugin aggregator — imports ONLY the root plugins this project actually
// uses (no dead code pulled in). Own demo plugins (emu, stats, nginx) sit here.
package plugins

import (
	// Charts and help.
	_ "github.com/arctcl/rough/plugins/chart"
	_ "github.com/arctcl/rough/plugins/man"

	// Pipe gluing and pause (Pipeline button).
	_ "github.com/arctcl/rough/plugins/clear"
	_ "github.com/arctcl/rough/plugins/sleep"

	// Plugins whose help is shown on the Man tab.
	_ "github.com/arctcl/rough/plugins/awk"
	_ "github.com/arctcl/rough/plugins/bars"
	_ "github.com/arctcl/rough/plugins/cut"
	_ "github.com/arctcl/rough/plugins/grep"
	_ "github.com/arctcl/rough/plugins/sed"
	_ "github.com/arctcl/rough/plugins/set"
	_ "github.com/arctcl/rough/plugins/sort"
	_ "github.com/arctcl/rough/plugins/ssh"
	_ "github.com/arctcl/rough/plugins/toggle"
	_ "github.com/arctcl/rough/plugins/tr"
	_ "github.com/arctcl/rough/plugins/uniq"

	// Killer-фичи: просмотр горутин (ps) и инжектор секретных кодов (chch) —
	// читает chch.json и регистрирует коды как действия (пайпы).
	_ "github.com/arctcl/rough/plugins/ps"
	_ "github.com/arctcl/rough/plugins/chch"
)
