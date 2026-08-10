package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
)

// PluginFunc — единый контракт плагина (юникс-команда):
// строки на входе (in) → строки на выходе (out).
// args — аргументы из вызова (имя:арг1:арг2).
// Плагин не знает про экран и мышь — чистая функция.
type PluginFunc func(in []string, args []string) ([]string, error)

// plugins — единый реестр. Движок сам ничего не умеет — только эта карта.
var plugins = map[string]PluginFunc{}

// forbiddenName — зарезервированные движком имена.
// "run" запрещён изначально: произвольный запуск команд через кнопку невозможен.
const forbiddenName = "run"

// AddPlugin регистрирует плагин. Зарезервированные имена молча отбрасываются.
func AddPlugin(name string, fn PluginFunc) {
	if name == forbiddenName {
		return
	}
	plugins[name] = fn
}

// HasPlugin проверяет наличие плагина в реестре.
func HasPlugin(name string) bool { _, ok := plugins[name]; return ok }

// SplitAction разбирает "имя:арг1:арг2" на имя и аргументы.
func SplitAction(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	i := strings.Index(raw, ":")
	if i < 0 {
		return raw, nil
	}
	name := raw[:i]
	rest := raw[i+1:]
	return name, strings.Split(rest, ":")
}

// SplitSteps разбирает пайп "шаг1 | шаг2 | шаг3" на шаги.
func SplitSteps(raw string) []string {
	var steps []string
	for _, s := range strings.Split(raw, "|") {
		s = strings.TrimSpace(s)
		if s != "" {
			steps = append(steps, s)
		}
	}
	return steps
}

// PrepareAction разбирает action на шаги и отделяет confirm-гейт.
// Возвращает шаги к выполнению и флаг «нужно подтверждение».
func PrepareAction(raw string) (steps []string, needConfirm bool) {
	for _, s := range SplitSteps(raw) {
		name, _ := SplitAction(s)
		if name == "confirm" {
			needConfirm = true
			continue
		}
		steps = append(steps, s)
	}
	return steps, needConfirm
}

// RunSteps выполняет пайп: выход одного шага — вход следующего.
func RunSteps(steps []string, in []string) ([]string, error) {
	cur := in
	for _, s := range steps {
		name, args := SplitAction(s)
		if name == forbiddenName {
			return nil, fmt.Errorf("%s запрещён движком", name)
		}
		fn, ok := plugins[name]
		if !ok {
			return nil, fmt.Errorf("нет такого плагина: %s", name)
		}
		out, err := fn(cur, args)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		cur = out
	}
	return cur, nil
}

// ErrNeedConfirm — сигнал движку: в action есть confirm, нужна модалка.
var ErrNeedConfirm = errors.New("нужно подтверждение")

// DoAction выполняет action из HTML (кнопка/поле) и возвращает текст для статуса.
// Если в action есть confirm — возвращает ErrNeedConfirm (движок сам покажет модалку).
func DoAction(raw string) (string, error) {
	steps, need := PrepareAction(raw)
	if need {
		return "", ErrNeedConfirm
	}
	out, err := RunSteps(steps, nil)
	if err != nil {
		return "", err
	}
	return strings.Join(out, " | "), nil
}

// Window возвращает текущий размер окна тайла в клетках.
// Движок устанавливает его перед запуском пайпа в <plugin>, чтобы рисовалки
// (bars и т.п.) адаптировали вывод под свой размер.
func Window() (int, int) {
	return curW, curH
}

// curW, curH — размер окна тайла для рисовалок (глобал, как curTheme).
var curW, curH int

// ApplyMask извлекает числа из строк по регулярке (первая группа захвата).
func ApplyMask(lines []string, mask string) []float64 {
	if mask == "" {
		return nil
	}
	re, err := regexp.Compile(mask)
	if err != nil {
		return nil
	}
	var out []float64
	for _, ln := range lines {
		m := re.FindStringSubmatch(ln)
		if len(m) < 2 {
			continue
		}
		if f, err := strconv.ParseFloat(m[1], 64); err == nil {
			out = append(out, f)
		}
	}
	return out
}

// UI — всё, что движок вытащил из папки /rough проекта: страницы и тема.
type UI struct {
	Pages Pages
	Theme *Theme
}

// LoadUI — единый загрузчик: читает из вшитой папки (fs.FS) всё, что нужно
// интерфейсу — tiles.json (страницы/тайлы) и тему. HTML тайлов читается по ходу
// отрисовки из той же папки. Больше никаких загрузчиков нет.
func LoadUI(fsys fs.FS) (*UI, error) {
	pages, err := LoadPages(fsys)
	if err != nil {
		return nil, err
	}
	return &UI{Pages: pages, Theme: LoadTheme(fsys, ConfigTheme(fsys))}, nil
}
