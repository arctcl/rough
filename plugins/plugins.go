// Пакет plugins — ВСЕ плагины rough в одном месте (корень репозитория).
// Это НЕ движок: движок (rough) их не знает и не подключает. Их подключает проект
// пользователя: в его example/rough/plugins/plugins.go стоит «линк» на этот пакет.
//
// Добавил плагин — добавь строчку сюда.
package plugins

import (
	_ "github.com/arctcl/rough/plugins/append"
	_ "github.com/arctcl/rough/plugins/awk"
	_ "github.com/arctcl/rough/plugins/bars"
	_ "github.com/arctcl/rough/plugins/cat"
	_ "github.com/arctcl/rough/plugins/chart"
	_ "github.com/arctcl/rough/plugins/clock"
	_ "github.com/arctcl/rough/plugins/curl"
	_ "github.com/arctcl/rough/plugins/cut"
	_ "github.com/arctcl/rough/plugins/export"
	_ "github.com/arctcl/rough/plugins/grep"
	_ "github.com/arctcl/rough/plugins/head"
	_ "github.com/arctcl/rough/plugins/hello"
	_ "github.com/arctcl/rough/plugins/line"
	_ "github.com/arctcl/rough/plugins/man"
	_ "github.com/arctcl/rough/plugins/sed"
	_ "github.com/arctcl/rough/plugins/set"
	_ "github.com/arctcl/rough/plugins/sort"
	_ "github.com/arctcl/rough/plugins/ssh"
	_ "github.com/arctcl/rough/plugins/tail"
	_ "github.com/arctcl/rough/plugins/theme"
	_ "github.com/arctcl/rough/plugins/tobotom"
	_ "github.com/arctcl/rough/plugins/toggle"
	_ "github.com/arctcl/rough/plugins/tr"
	_ "github.com/arctcl/rough/plugins/uniq"
	_ "github.com/arctcl/rough/plugins/wc"
)
