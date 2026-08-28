package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"runtime/debug"
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

// isReserved — зарезервированные слова движка, которые НЕ плагины: их
// обрабатывает сам движок (RunSteps / PrepareAction), а не реестр плагинов.
// Плагин с таким именем зарегистрировать нельзя (AddPlugin молча отбросит).
func isReserved(name string) bool {
	switch strings.ToLower(name) {
	case "run", "loop", "export", "unexport", "confirm":
		return true
	}
	return false
}

// AddPlugin регистрирует плагин. Зарезервированные имена молча отбрасываются.
func AddPlugin(name string, fn PluginFunc) {
	if isReserved(name) {
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
	if strings.EqualFold(name, forbiddenName) {
		return
	}
	mans[name] = text
}

// manExport / manUnexport — справка по резервным словам движка. Это НЕ плагины:
// их обрабатывает сам движок в RunSteps, поэтому и справку он кладёт сам.
const manExport = `export — резервное слово движка: сохранить текущий вывод пайпа
в переменную сессии (для $подстановки в action).

Использование:
  часть пайпа: ... | export:ИМЯ

Аргументы:
  ИМЯ — имя переменной. Дальше в любом action её можно подставить через $ИМЯ
        (или ${ИМЯ}). Переменная доступна всегда и отовсюду, в т.ч. из другого
        "&&"-пайпа: $ИМЯ подставляется в момент выполнения шага.

Примеры:
  action="ssh:root:srv1::hostname | export:host"    — запомнить хост
  action="ssh:root:$host::uptime"                   — подставить переменную
  action="cat:app.conf | cut::2 | export:val | bars" — и сохранить, и показать`

const manUnexport = `unexport — резервное слово движка: удалить переменную сессии
(антипод export).

Использование:
  часть пайпа: ... | unexport:ИМЯ

Аргументы:
  ИМЯ — имя переменной. После удаления $ИМЯ больше нигде не подставляется.

Примеры:
  ... | unexport:tmp     — выкинуть временную переменную
  export:count | ... | unexport:count — записать, использовать, удалить`

func init() {
	AddMan("export", manExport)
	AddMan("unexport", manUnexport)
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
// "demo --a=1 --b=2" → demo + ["--a=1 --b=2"] (разбор флагов — в ParseArgs).
func SplitAction(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	i := indexColonOutsideQuotes(raw)
	if i < 0 {
		// Двоеточий вне кавычек нет: имя до первого пробела, остальное — аргументы
		// (иначе флаги ушли бы в имя).
		if j := strings.IndexAny(raw, " \t"); j > 0 {
			return raw[:j], []string{strings.TrimSpace(raw[j:])}
		}
		return raw, nil
	}
	name := raw[:i]
	rest := raw[i+1:]
	// Аргументы режем по ":" с учётом кавычек '...' и "...": внутри кавычек
	// ":" — обычный символ (например разделитель для cut/sed: sed:':':1).
	return name, splitQuoted(rest, ':')
}

// indexColonOutsideQuotes возвращает индекс первого ":" вне кавычек
// ('...' / "...") или -1. Нужно, чтобы ":" внутри значения флага
// (--sep=':') не резал имя действия.
func indexColonOutsideQuotes(s string) int {
	var quote rune
	for i, r := range s {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ':':
			return i
		}
	}
	return -1
}

// splitQuoted режет строку по sep, уважая кавычки '...' и "...": внутри кавычек
// sep — обычный символ, сами кавычки снимаются. Пустые слоты сохраняются
// (как strings.Split), чтобы "::" оставалось пустым слотом для quick-параметров.
func splitQuoted(s string, sep rune) []string {
	var parts []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		parts = append(parts, cur.String())
		cur.Reset()
	}
	for _, r := range s {
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case sep:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return parts
}

// stripQuotes снимает парные кавычки вокруг значения ('...' / "...").
// Используется для значений флагов и позиционных токенов, пришедших с кавычками.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// SplitSteps разбирает пайп "шаг1 | шаг2 | шаг3" на шаги.
// "|" внутри кавычек ('...' / "...") — часть шага, не разделитель:
// так ввод из поля ввода (обёрнутый в кавычки) не создаёт новый шаг
// (защита от инъекций: поле не может протащить произвольную команду).
func SplitSteps(raw string) []string {
	var steps []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			steps = append(steps, s)
		}
		cur.Reset()
	}
	for _, r := range raw {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			cur.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
			cur.WriteRune(r)
		case '|':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return steps
}

// FlagValue вытаскивает из args флаг --name=value. Флаг может быть «приклеен»
// к значению через пробел (например "val --sep=space") — разбираем
// каждый аргумент по пробелам. Возвращает значение флага и остаток аргументов
// (значения склеены обратно с пробелами). Нужно плагинам, которые не используют
// ParseArgs (значение глотает остаток двоеточий), — например set/toggle.
func FlagValue(args []string, name string) (string, []string) {
	flag := "--" + name + "="
	val := ""
	found := false
	var rest []string
	for _, a := range args {
		if strings.TrimSpace(a) == "" {
			continue
		}
		var plain []string
		for _, tok := range strings.Fields(a) {
			if strings.HasPrefix(tok, flag) {
				val = strings.TrimPrefix(tok, flag)
				found = true
				continue
			}
			plain = append(plain, tok)
		}
		if len(plain) > 0 {
			rest = append(rest, strings.Join(plain, " "))
		}
	}
	if !found {
		return "", args
	}
	return val, rest
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
//	demo:a:b:c          — позиционные, по порядку объявления;
//	demo --a=1 --b=3    — именованные флаги (порядок не важен, в одном
//	                      аргументе может быть несколько через пробел);
//	demo::b --a=1       — микс: пустой слот ":" — подставится флаг/дефолт.
//
// Последний объявленный параметр «глотает» остаток двоеточий (как в
// ssh:host:command или set:file:key:value:with:colons). Неизвестный
// флаг — ошибка; обязательный параметр без значения — ошибка.
// Возвращает карту имя→значение для всех параметров.
func ParseArgs(args []string, params []Param) (map[string]string, error) {
	// Отделяем флаги --name=value от позиционных слотов. В одном аргументе
	// может быть всё сразу: "" (пустой слот), "10", "off --mode=off",
	// "--place=back --time=7" — поэтому режем каждый аргумент по пробелам.
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
					flags[kv[:i]] = stripQuotes(kv[i+1:])
				} else {
					return nil, fmt.Errorf("флаг %s требует =значение", tok)
				}
				continue
			}
			plain = append(plain, stripQuotes(tok))
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
	// Последний параметр глотает остаток двоеточий (только ЛИШНИЕ слоты:
	// слот самого последнего параметра уже записан в vals выше).
	if len(params) > 0 && len(pos) > len(params) {
		last := params[len(params)-1].Name
		var parts []string
		for _, t := range pos[len(params):] {
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
//	chart:MIN:MAX[:WIDTH[:SECONDS[:TITLE]]]
//	chart [--min=VAL] [--max=VAL] [--width=1] [--seconds=2] [--title=VAL]
//
// В флаговой форме показывается дефолтное значение (--width=1); без дефолта —
// плейсхолдер VAL. Микс: позиционные слоты + флаги, пустой слот ":" —
// берётся флаг или дефолт. Частичный ввод работает: только первый
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
	// значение (--width=1), без дефолта — плейсхолдер VAL.
	flagForm := name
	for _, p := range params {
		dv := "VAL"
		if p.Default != "" {
			dv = p.Default
		}
		flagForm += " [--" + p.Name + "=" + dv + "]"
	}
	return posForm + "\n  " + flagForm
}

// PrepareAction разбирает action на шаги и отделяет confirm-гейт.
// Возвращает шаги к выполнению и флаг «нужно подтверждение».
// PrepareAction разбирает action на пайпы и отделяет confirm-гейт.
// "a && b" — два НЕЗАВИСИМЫХ пайпа (выполняются последовательно, каждый
// своим выводом), внутри пайпа "|" — конвейер (выход → вход).
// Возвращает список пайпов (каждый — шаги) и флаг «нужно подтверждение».
func PrepareAction(raw string) (pipes [][]string, needConfirm bool) {
	for _, seg := range splitAnd(raw) {
		var steps []string
		for _, s := range SplitSteps(seg) {
			name, _ := SplitAction(s)
			if name == "confirm" {
				needConfirm = true
				continue
			}
			// Шаг храним БЕЗ подстановки переменных: $имя раскроется в RunSteps на
			// момент выполнения, чтобы export из раннего "&&"-пайпа был виден в
			// позднем. Разворачивать здесь — рано, значение ещё пустое.
			steps = append(steps, s)
		}
		if len(steps) > 0 {
			pipes = append(pipes, steps)
		}
	}
	return pipes, needConfirm
}

// splitAnd разбивает action на несколько пайпов по "&&" (вне кавычек).
// Одиночный "&" — обычный символ. Кавычки '...'/"..." защищают "&&" внутри.
func splitAnd(raw string) []string {
	var parts []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			parts = append(parts, s)
		}
		cur.Reset()
	}
	runes := []rune(raw)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			cur.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
			cur.WriteRune(r)
		case '&':
			if i+1 < len(runes) && runes[i+1] == '&' {
				flush()
				i++
			} else {
				cur.WriteRune(r)
			}
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return parts
}

// RunSteps выполняет пайп: выход одного шага — вход следующего.
// Каждый шаг защищён от паники (callSafe): падение одного плагина не валит
// весь интерфейс — пайп останавливается, а трасса уходит в ошибку.
func RunSteps(steps []string, in []string) ([]string, error) {
	cur := in
	for i := 0; i < len(steps); i++ {
		s := steps[i]
		// Переменные $имя подставляем ЗДЕСЬ, на момент выполнения шага, а не при
		// разборе: export в раннем "&&"-пайпе уже успел записать значение, и оно
		// видно в позднем ($ln_sum в следующем куске). Это и есть "переменная
		// доступна всегда и отовсюду".
		s = expandVars(s)
		name, args := SplitAction(s)
		// loop:N — ключевое слово (как confirm): повторить ОСТАЛЬНЫЕ шаги пайпа
		// N раз, выводы склеить. Например "loop:$count | ssh:...:sl" запустит
		// ssh-паровозик $count раз. $count уже раскрыт выше в число.
		if strings.EqualFold(name, "loop") {
			n := 1
			if len(args) > 0 {
				if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
					n = v
				}
			}
			rest := steps[i+1:]
			var acc []string
			for k := 0; k < n; k++ {
				r, err := RunSteps(rest, cur)
				if err != nil {
					return nil, err
				}
				acc = append(acc, r...)
			}
			return acc, nil
		}
		// export:ИМЯ — резервное слово движка: сохранить ТЕКУЩИЙ вывод пайпа (cur)
		// в переменную сессии. Движок сам собирает переменные — плагин про них
		// не знает. Дальше строки проходят как есть (как tee).
		if strings.EqualFold(name, "export") {
			if len(args) == 0 || args[0] == "" {
				return nil, fmt.Errorf("export: нужно имя переменной")
			}
			SetVar(args[0], cur)
			continue
		}
		// unexport:ИМЯ — резервное слово движка: удалить переменную (антипод
		// export). $имя больше нигде не подставится.
		if strings.EqualFold(name, "unexport") {
			if len(args) == 0 || args[0] == "" {
				return nil, fmt.Errorf("unexport: нужно имя переменной")
			}
			DelVar(args[0])
			continue
		}
		if strings.EqualFold(name, forbiddenName) {
			return nil, fmt.Errorf("%s запрещён движком", name)
		}
		fn, ok := plugins[name]
		if !ok {
			return nil, fmt.Errorf("нет такого плагина: %s", name)
		}
		out, err := callSafe(fn, cur, args, name)
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

// callSafe вызывает плагин с защитой от паники. Паника (кривой параметр,
// пустая переменная, выход за границы) превращается в обычную ошибку с
// трассой — пайп останавливается, но интерфейс живёт, а где косяк — видно.
func callSafe(fn PluginFunc, in []string, args []string, name string) (out []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			out = nil
			err = fmt.Errorf("паника: %v\n%s", r, debug.Stack())
		}
	}()
	return fn(in, args)
}

// ErrStop — сигнал движку остановить пайп без ошибки (плагин отладки tobotom:stop):
// вывод уже показан в статус-блоке, дальше пайп не работает.
var ErrStop = errors.New("стоп")

// Window возвращает текущий размер окна тайла в клетках.
// Движок устанавливает его перед запуском пайпа в <plugin>, чтобы рисовалки
// (bars и т.п.) адаптировали вывод под свой размер. Безопасно из async-горутин.
func Window() (int, int) {
	engineMu.RLock()
	defer engineMu.RUnlock()
	return curW, curH
}

// SetWindowSize вручную задаёт размер окна тайла для рисовалок.
// Нужна тестам плагинов (без запуска всего интерфейса).
func SetWindowSize(w, h int) {
	engineMu.Lock()
	curW, curH = w, h
	engineMu.Unlock()
}

// curW, curH — размер окна тайла для рисовалок (глобал, как curTheme).
var curW, curH int

// curPluginKey — сигнатура текущего <plugin> (для stateful-плагинов вроде chart).
var curPluginKey string

// PluginKey возвращает сигнатуру текущего выполняемого <plugin> (пайп).
// Нужна stateful-плагинам (chart), чтобы хранить свою серию отдельно.
// Безопасно из async-горутин.
func PluginKey() string {
	engineMu.RLock()
	defer engineMu.RUnlock()
	return curPluginKey
}

// curViewH — видимая высота тайла. Нужна для вертикального центрирования
// <div align="center"> внутри запасного буфера скролла (ставит renderTile).
var curViewH int

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
