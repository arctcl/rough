// Агрегатор плагинов примера — «линк» на плагины из корня репозитория (rough/plugins).
// Сами плагины (cat, hello, nyan) живут там; здесь только одна строчка-импорт.
package plugins

import _ "rough/plugins" // все плагины из корня: cat, hello, nyan
