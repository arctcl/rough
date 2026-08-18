// Агрегатор плагинов примера — «линк» на плагины из корня репозитория (rough/plugins).
// Сами плагины (cat, hello, ssh, curl, ...) живут там; здесь только одна строчка-импорт.
package plugins

import _ "github.com/arctcl/rough/plugins" // все плагины из корня
