// Агрегатор плагинов демо — подключаем ТОЛЬКО те плагины из корня репозитория,
// что реально используются в этом проекте (не тянем мёртвый код).
// Свои демо-плагины (emu, stats, nginx) — рядом, в этом же пакете.
package plugins

import (
	// Графики и справка.
	_ "github.com/arctcl/rough/plugins/chart"
	_ "github.com/arctcl/rough/plugins/man"
	// Склейка пайпов и пауза (кнопка Pipeline).
	_ "github.com/arctcl/rough/plugins/clear"
	_ "github.com/arctcl/rough/plugins/sleep"
	// Плагины, по которым на вкладке Man показываем справку.
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
)
