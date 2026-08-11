// Пакет plugins — ВСЕ плагины rough в одном месте (корень репозитория).
// Это НЕ движок: движок (rough) их не знает и не подключает. Их подключает проект
// пользователя: в его example/rough/plugins/plugins.go стоит «линк» на этот пакет.
//
// Добавил плагин — добавь строчку сюда.
package plugins

import (
	_ "rough/plugins/append"
	_ "rough/plugins/bars"
	_ "rough/plugins/cat"
	_ "rough/plugins/clock"
	_ "rough/plugins/curl"
	_ "rough/plugins/grep"
	_ "rough/plugins/head"
	_ "rough/plugins/hello"
	_ "rough/plugins/line"
	_ "rough/plugins/man"
	_ "rough/plugins/set"
	_ "rough/plugins/ssh"
	_ "rough/plugins/tail"
	_ "rough/plugins/toggle"
	_ "rough/plugins/wc"
)
