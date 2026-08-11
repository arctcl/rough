// testv001 — ТЕСТЕР: проверочное приложение со всеми плагинами.
// Кнопки не привязаны к Linux — вместо реальных /proc и /etc здесь эмуляторы:
// они генерируют данные (нагрузку CPU, память, лог, инфу о системе) прямо
// в main.go на чистом Go. Так можно проверять плагины на любой ОС.
package main

import (
	"bytes"
	"embed"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	"rough"
	_ "testv001/rough/plugins"
)

//go:embed rough
var roughDir embed.FS

// init — эмуляторы плагинов (строго здесь, в main.go тестового проекта).
// Каждый — обычный плагин rough: строки на входе, строки на выходе.
func init() {
	// emu_cpu — эмуляция нагрузки CPU: синусоида + шум, последние N точек.
	rough.AddMan("emu_cpu", `emu_cpu — эмуляция нагрузки CPU (тестовый плагин).

Использование:
  часть пайпа: emu_cpu[:N] | bars

Аргументы:
  N — сколько точек истории выдать (по умолчанию 30).

Примеры:
  <plugin pipe="emu_cpu | bars" interval="1s"/>   — живой график нагрузки`)

	rough.AddPlugin("emu_cpu", func(in []string, args []string) ([]string, error) {
		n := 30
		if len(args) > 0 {
			if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
				n = v
			}
		}
		out := make([]string, 0, n)
		now := time.Now()
		for i := n - 1; i >= 0; i-- {
			t := float64(now.Unix())/8 + float64(i)/8
			v := 30 + 40*math.Sin(t/3) + 15*rand.Float64()
			out = append(out, fmt.Sprintf("%.1f", v))
		}
		return out, nil
	})

	// emu_mem — эмуляция занятой памяти: плавно растёт и падает.
	rough.AddMan("emu_mem", `emu_mem — эмуляция памяти (тестовый плагин).

Использование:
  часть пайпа: emu_mem[:N] | bars

Примеры:
  <plugin pipe="emu_mem | bars" interval="1s"/>   — живой график памяти`)

	rough.AddPlugin("emu_mem", func(in []string, args []string) ([]string, error) {
		n := 30
		if len(args) > 0 {
			if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
				n = v
			}
		}
		out := make([]string, 0, n)
		now := time.Now()
		for i := n - 1; i >= 0; i-- {
			t := float64(now.Unix())/6 + float64(i)/6
			v := 45 + 30*math.Sin(t/5) + 10*rand.Float64()
			out = append(out, fmt.Sprintf("%.1f", v))
		}
		return out, nil
	})

	// emu_log — эмуляция лога приложения: метки времени и уровни INFO/WARN/ERROR.
	rough.AddMan("emu_log", `emu_log — эмуляция лога приложения (тестовый плагин).

Использование:
  часть пайпа: emu_log | grep:ERROR    — только ошибки
                emu_log | tail:5       — последние строки
                emu_log | wc           — сколько строк

Примеры:
  action="emu_log | grep:ERROR"        — только ERROR
  action="emu_log | tail:3"            — последние 3 строки`)

	rough.AddPlugin("emu_log", func(in []string, args []string) ([]string, error) {
		levels := []string{"INFO", "WARN", "ERROR"}
		out := make([]string, 0, 40)
		for i := 0; i < 40; i++ {
			lvl := levels[rand.Intn(len(levels))]
			ts := time.Now().Add(-time.Duration(i) * 3 * time.Second).Format("15:04:05")
			out = append(out, fmt.Sprintf("%s %s запрос #%d обработан", ts, lvl, i))
		}
		return out, nil
	})

	// emu_sys — кроссплатформенная инфа о системе (без Linux-файлов).
	rough.AddMan("emu_sys", `emu_sys — инфа о системе (тестовый плагин, кроссплатформенно).

Использование:
  action="emu_sys"

Примеры:
  action="emu_sys"   — ОС, архитектура, хост, ядра CPU`)

	rough.AddPlugin("emu_sys", func(in []string, args []string) ([]string, error) {
		host, _ := os.Hostname()
		if host == "" {
			host = "неизвестно"
		}
		return []string{
			"ОС:        " + runtime.GOOS,
			"Архитектура: " + runtime.GOARCH,
			"Хост:      " + host,
			"Ядер CPU:  " + strconv.Itoa(runtime.NumCPU()),
			"Время:     " + time.Now().Format("02.01.2006 15:04:05"),
		}, nil
	})

	// emu_candle — эмуляция японских свечей: непрерывный мартингейл.
	// За вызов генерирует 16 точек (в 8 раз больше, чем обновление раз в 2с),
	// агрегирует их в OHLC: open — первая цена, close — последняя, high/low —
	// экстремумы. open следующей = close предыдущей → свечи стыкуются без гепов.
	rough.AddMan("emu_candle", `emu_candle — эмуляция OHLC для свечей (тестовый плагин).

Использование:
  часть пайпа: emu_candle | chart:0:100:japanse

Примеры:
  <plugin pipe="emu_candle | chart:0:100:japanse:1:2:ETH" height="14" interval="2s"/>`)

	rough.AddPlugin("emu_candle", func(in []string, args []string) ([]string, error) {
		open := emuCandleLast
		high, low := open, open
		prev := open
		for i := 0; i < 16; i++ {
			next := prev + 6*rand.Float64() - 3
			if next > high {
				high = next
			}
			if next < low {
				low = next
			}
			prev = next
		}
		close := prev
		emuCandleLast = close
		return []string{fmt.Sprintf("%.2f %.2f %.2f %.2f", open, high, low, close)}, nil
	})
}

// emuCandleLast — последняя цена (open следующей свечи = close предыдущей).
var emuCandleLast = 50.0

func main() {
	// Тестовые файлы на диске (рядом с exe): конфиг для toggle/set/append и PPM для img.
	ensureConfig()
	ensureLogo()

	rough.TUI(roughDir)
}

// ensureConfig создаёт app.conf (ключ=значение), если его ещё нет.
// На него смотрят checkbox/toggle/set/append/cat на вкладке «Формы».
func ensureConfig() {
	if _, err := os.Stat("app.conf"); os.IsNotExist(err) {
		os.WriteFile("app.conf", []byte("debug=1\nlogging=0\nloglevel=info\nmax_users=10\n"), 0644)
	}
}

// ensureLogo генерирует PPM-картинку (P6, градиент 40x20), если её нет.
// На неё смотрит тег <img src="logo.ppm"/> на вкладке «Формы».
func ensureLogo() {
	if _, err := os.Stat("logo.ppm"); os.IsNotExist(err) {
		w, h := 40, 20
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "P6\n%d %d\n255\n", w, h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r := byte(255 * x / w)
				g := byte(128 + 100*y/h)
				b := byte(255 * y / h)
				buf.Write([]byte{r, g, b})
			}
		}
		os.WriteFile("logo.ppm", buf.Bytes(), 0644)
	}
}
