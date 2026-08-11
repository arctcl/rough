package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
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

// mans — реестр справок (man): имя плагина → текст описания.
// У каждого плагина внутри лежит переменная man_<имя> (см. контракт в
// systemprompt.md), и init() регистрирует её здесь через AddMan.
var mans = map[string]string{}

// AddMan регистрирует справку по плагину (юникс-like man).
// Обязательный вызов из init() плагина — рядом с AddPlugin.
func AddMan(name, text string) {
	if name == forbiddenName {
		return
	}
	mans[name] = text
}

// ManText возвращает справку по имени плагина.
func ManText(name string) (string, bool) {
	t, ok := mans[name]
	return t, ok
}

// ManNames возвращает отсортированный список имён, у которых есть справка.
func ManNames() []string {
	names := make([]string, 0, len(mans))
	for n := range mans {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// SplitAction разбирает "имя:арг1:арг2" на имя и аргументы.
// Флаги "--name=value" идут через пробел после имени (без двоеточий):
// "имя --а=1 --б=2" → имя + ["--а=1 --б=2"] (разбор флагов — в ParseArgs).
func SplitAction(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	i := strings.Index(raw, ":")
	if i < 0 {
		// Двоеточий нет: имя до первого пробела, остальное — аргументы
		// (иначе флаги ушли бы в имя).
		if j := strings.IndexAny(raw, " \t"); j > 0 {
			return raw[:j], []string{strings.TrimSpace(raw[j:])}
		}
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

// Param — описание одного параметра плагина (гибридный ввод).
// Порядок объявления = позиция при вводе двоеточиями; Name — имя для
// флага --name=value; Default — значение по умолчанию (пусто = нет дефолта);
// Required — обязательный (ошибка, если не задан и нет дефолта).
type Param struct {
	Name     string
	Default  string
	Required bool
}

// ParseArgs разбирает аргументы вызова плагина по объявленным параметрам.
// Принимает три формы и их микс:
//
//	имя:а:б:в          — позиционные, по порядку объявления;
//	имя --а=1 --в=3    — именованные флаги (порядок не важен, в одном
//	                      аргументе может быть несколько через пробел);
//	имя::б --а=1       — микс: пустой слот ":" — подставится флаг/дефолт.
//
// Последний объявленный параметр «глотает» остаток двоеточий (как в
// ssh:host:команда или set:файл:ключ:значение:с:двоеточиями). Неизвестный
// флаг — ошибка; обязательный параметр без значения — ошибка.
// Возвращает карту имя→значение для всех параметров.
func ParseArgs(args []string, params []Param) (map[string]string, error) {
	// Отделяем флаги --name=value от позиционных слотов. В одном аргументе
	// может быть всё сразу: "" (пустой слот), "10", "нет --смазка=нет",
	// "--место=зад --время=7" — поэтому режем каждый аргумент по пробелам.
	flags := map[string]string{}
	var pos []string
	for _, a := range args {
		if strings.TrimSpace(a) == "" {
			// Пустой слот ":" — берётся флагом/дефолтом.
			pos = append(pos, "")
			continue
		}
		var plain []string
		for _, tok := range strings.Fields(a) {
			if strings.HasPrefix(tok, "--") {
				kv := strings.TrimPrefix(tok, "--")
				if i := strings.Index(kv, "="); i >= 0 {
					flags[kv[:i]] = kv[i+1:]
				} else {
					return nil, fmt.Errorf("флаг %s требует =значение", tok)
				}
				continue
			}
			plain = append(plain, tok)
		}
		// Позиционная часть аргумента — не-флаговые токены через пробел.
		// Аргумент, состоящий только из флагов, слота НЕ даёт.
		if len(plain) > 0 {
			pos = append(pos, strings.Join(plain, " "))
		}
	}
	// Неизвестный флаг — ошибка (опечатка не проходит молча).
	known := map[string]bool{}
	for _, p := range params {
		known[p.Name] = true
	}
	for name := range flags {
		if !known[name] {
			return nil, fmt.Errorf("неизвестный параметр --%s", name)
		}
	}
	// Слот i ↔ параметр i; пустой слот пропускаем (возьмётся флагом/дефолтом).
	vals := map[string]string{}
	for i, p := range params {
		if i < len(pos) && pos[i] != "" {
			vals[p.Name] = pos[i]
		}
		if v, ok := flags[p.Name]; ok {
			vals[p.Name] = v
		}
	}
	// Последний параметр глотает остаток двоеточий (лишние слоты).
	if len(params) > 0 && len(pos) > len(params) {
		last := params[len(params)-1].Name
		var parts []string
		for _, t := range pos[len(params)-1:] {
			if t != "" {
				parts = append(parts, t)
			}
		}
		if len(parts) > 0 {
			if cur, ok := vals[last]; ok && cur != "" {
				vals[last] = cur + ":" + strings.Join(parts, ":")
			} else {
				vals[last] = strings.Join(parts, ":")
			}
		}
	}
	// Дефолты и проверка обязательных.
	var missing []string
	for _, p := range params {
		if _, ok := vals[p.Name]; !ok {
			if p.Default != "" {
				vals[p.Name] = p.Default
			} else if p.Required {
				missing = append(missing, p.Name)
			} else {
				vals[p.Name] = "" // необязательный без дефолта — пусто
			}
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("не заданы параметры: %s", strings.Join(missing, ", "))
	}
	return vals, nil
}

// ParamsUsage собирает для man обе формы ввода параметров плагина:
//
//	chart:МИН:МАКС[:ШИРИНА[:СЕКУНД[:ЗАГОЛОВОК]]]
//	chart [--мин=ЗНАЧ] [--макс=ЗНАЧ] [--ширина=1] [--секунд=2] [--заголовок=ЗНАЧ]
//
// В флаговой форме показывается дефолтное значение (--ширина=1); без дефолта —
// плейсхолдер ЗНАЧ. Микс: позиционные слоты + флаги, пустой слот ":" —
// берётся флаг или дефолт. Частичный ввод работает: fuck:ass — только первый
// параметр, остальные уходят в дефолты (или пустое, их обрабатывает разраб).
func ParamsUsage(name string, params []Param) string {
	up := func(s string) string { return strings.ToUpper(s) }
	var required, optional []string
	for _, p := range params {
		if p.Required && p.Default == "" {
			required = append(required, up(p.Name))
		} else {
			optional = append(optional, up(p.Name))
		}
	}
	// Позиционная форма: имя:ОБЯЗ... + вложенные [:] для опциональных.
	posForm := name
	for _, r := range required {
		posForm += ":" + r
	}
	tail := ""
	for i := len(optional) - 1; i >= 0; i-- {
		tail = "[:" + optional[i] + tail + "]"
	}
	posForm += tail
	// Флаговая форма: все параметры по имени; с дефолтом показываем его
	// значение (--ширина=1), без дефолта — плейсхолдер ЗНАЧ.
	flagForm := name
	for _, p := range params {
		dv := "ЗНАЧ"
		if p.Default != "" {
			dv = p.Default
		}
		flagForm += " [--" + p.Name + "=" + dv + "]"
	}
	return posForm + "\n  " + flagForm
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
			// Плагин отладки попросил остановить пайп — это не ошибка, вывод уже в статусе.
			if errors.Is(err, ErrStop) {
				return nil, ErrStop
			}
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		cur = out
	}
	return cur, nil
}

// ErrNeedConfirm — сигнал движку: в action есть confirm, нужна модалка.
var ErrNeedConfirm = errors.New("нужно подтверждение")

// ErrStop — сигнал движку остановить пайп без ошибки (плагин отладки tobotom:stop):
// вывод уже показан в статус-блоке, дальше пайп не работает.
var ErrStop = errors.New("стоп")

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

// SetWindowSize вручную задаёт размер окна тайла для рисовалок.
// Нужна тестам плагинов (без запуска всего интерфейса).
func SetWindowSize(w, h int) { curW, curH = w, h }

// curW, curH — размер окна тайла для рисовалок (глобал, как curTheme).
var curW, curH int

// curPluginKey — сигнатура текущего <plugin> (для stateful-плагинов вроде chart).
var curPluginKey string

// PluginKey возвращает сигнатуру текущего выполняемого <plugin> (пайп).
// Нужна stateful-плагинам (chart), чтобы хранить свою серию отдельно.
func PluginKey() string { return curPluginKey }

// curViewH — видимая высота тайла. Нужна для вертикального центрирования
// <div align="center"> внутри запасного буфера скролла (ставит renderTile).
var curViewH int

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

// UI — всё, что движок вытащил из папки /rough проекта: страницы, вкладки и тема.
type UI struct {
	Pages Pages
	Menu  [][]string // вкладки [имя, роут] — навбар внизу
	Theme *Theme
}

// LoadUI — единый загрузчик: читает из вшитой папки (fs.FS) всё, что нужно
// интерфейсу — tiles.json (страницы/тайлы/вкладки) и тему. HTML тайлов читается
// по ходу отрисовки из той же папки. Больше никаких загрузчиков нет.
func LoadUI(fsys fs.FS) (*UI, error) {
	pages, err := LoadPages(fsys)
	if err != nil {
		return nil, err
	}
	return &UI{Pages: pages, Menu: LoadMenu(fsys), Theme: LoadTheme(fsys, ConfigTheme(fsys))}, nil
}
