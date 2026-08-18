// for_gifs — демо-проект специально для записи GIF-ок в README.
// Четыре вкладки: 10 живых графиков, конструктор запросов, справка, о проекте.
package main

import (
	"embed"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/arctcl/rough"

	_ "for_gifs/rough/plugins" // встроенные плагины
)

//go:embed rough
var roughDir embed.FS

// init — демо-плагины (эмуляторы), строго здесь, в main демо-проекта.
func init() {
	// demoOpts — выбранные значения select-«крутилок»; demoFlags — чекбоксы.
	demoOpts := map[string]string{}
	demoFlags := map[string]bool{}

	// opt — память для выпадающих списков: opt:KEY:VALUE — запомнить выбор,
	// opt:KEY:get — вернуть текущий (для подписи select). Без файлов.
	rough.AddPlugin("opt", func(in []string, args []string) ([]string, error) {
		if len(args) == 0 {
			return []string{"?"}, nil
		}
		key := args[0]
		if len(args) > 1 && args[1] == "get" {
			if v, ok := demoOpts[key]; ok && v != "" {
				return []string{v}, nil
			}
			return []string{"—"}, nil
		}
		if len(args) > 1 {
			demoOpts[key] = args[1]
			return []string{demoOpts[key]}, nil
		}
		return []string{"?"}, nil
	})

	// flag — чекбокс в памяти: flag:KEY — переключить, flag:KEY:get — прочитать.
	rough.AddPlugin("flag", func(in []string, args []string) ([]string, error) {
		if len(args) == 0 {
			return []string{"off"}, nil
		}
		key := args[0]
		if len(args) > 1 && args[1] == "get" {
			if demoFlags[key] {
				return []string{"on"}, nil
			}
			return []string{"off"}, nil
		}
		demoFlags[key] = !demoFlags[key]
		if demoFlags[key] {
			return []string{"on"}, nil
		}
		return []string{"off"}, nil
	})

	// emu — генератор плавного «живого» значения (для 10 графиков).
	// Имя в аргументе задаёт свою частоту/фазу — каждый график уникален.
	rough.AddPlugin("emu", func(in []string, args []string) ([]string, error) {
		name := "x"
		if len(args) > 0 {
			name = args[0]
		}
		seed := 0
		for _, r := range name {
			seed += int(r)
		}
		now := float64(time.Now().UnixMilli()) / 1000.0
		v := 50 +
			35*math.Sin(now/float64(3+seed%5)+float64(seed)) +
			12*math.Sin(now*(1.2+float64(seed%3)*0.4)) +
			4*rand.Float64() - 2
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		return []string{fmt.Sprintf("%.1f", v)}, nil
	})

	// query — имитация SQL-запроса: возвращает «таблицу»-результат.
	rough.AddPlugin("query", func(in []string, args []string) ([]string, error) {
		q := "SELECT"
		if len(args) > 0 {
			q = strings.Join(args, " ")
		}
		return []string{
			"запрос: " + q,
			"→ 42 строки, 3 столбца, 12ms",
			"",
			" id | name          | status ",
			"----+---------------+--------",
			"  1 | api           | ok     ",
			"  2 | worker        | ok     ",
			"  3 | cache         | warn   ",
			"  4 | db-master     | ok     ",
		}, nil
	})
}

func main() {
	rough.TUI(roughDir)
}
