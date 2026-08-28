// Переменные сессии: глобальная память движка (как outputCache).
// Храним строки вывода по имени; в action подставляются через $имя или ${имя}.
// Запись делает плагин export (engine.SetVar), подстановку — движок (expandVars).
// Это ИНФРАСТРУКТУРА оболочки, не плагин: никакой логики, только хранилище+подстановка.
package engine

import "strings"

// vars — переменные сессии: имя → строки значения.
var vars = map[string][]string{}

// SetVar сохраняет значение переменной (строки) по имени.
// Вызывается плагином export (и любым другим, кому нужно записать в память).
// Безопасно из async-горутин.
func SetVar(name string, lines []string) {
	engineMu.Lock()
	vars[name] = lines
	engineMu.Unlock()
}

// DelVar удаляет переменную по имени (плагин unexport). Безопасно из async.
func DelVar(name string) {
	engineMu.Lock()
	delete(vars, name)
	engineMu.Unlock()
}

// VarLine возвращает переменную одной строкой для подстановки в action:
// одна строка — как есть, несколько — склеиваются пробелом.
// Безопасно из async-горутин.
func VarLine(name string) string {
	engineMu.RLock()
	v, ok := vars[name]
	engineMu.RUnlock()
	if !ok || len(v) == 0 {
		return ""
	}
	return strings.Join(v, " ")
}

// expandVars подставляет $имя и ${имя} в строку (аргументы action).
//   $имя   — имя из букв/цифр/подчёркивания;
//   ${имя} — имя до закрывающей скобки (когда нужно отделить от соседей);
//   \$     — литеральный доллар;
//   '...'  — внутри ОДИНАРНЫХ кавычек подстановка НЕ выполняется (литерал).
//            Так поле ввода (оборачивает значение в кавычки) не может
//            прочитать/подставить переменную: '...' — это защита от инъекции;
//   неизвестная переменная — пустая строка.
// Остальные символы (в т.ч. \d, [0-9]) не трогаем.
func expandVars(s string) string {
	var sb strings.Builder
	runes := []rune(s)
	inSingle := false // внутри '...' всё литерально (подстановка отключена)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if inSingle {
			sb.WriteRune(c)
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		// Одиночная кавычка открывает литеральный участок.
		if c == '\'' {
			inSingle = true
			sb.WriteRune(c)
			continue
		}
		// \$ — литеральный доллар.
		if c == '\\' && i+1 < len(runes) && runes[i+1] == '$' {
			sb.WriteRune('$')
			i++
			continue
		}
		if c != '$' {
			sb.WriteRune(c)
			continue
		}
		// ${имя} или ${имя%паттерн} / ${имя#паттерн} (Bash Parameter Expansion).
		if i+1 < len(runes) && runes[i+1] == '{' {
			j := i + 2
			start := j
			for j < len(runes) && runes[j] != '}' {
				j++
			}
			if j < len(runes) {
				sb.WriteString(expandBraced(string(runes[start:j])))
				i = j
				continue
			}
			// незакрытая ${ — оставляем как есть
			sb.WriteRune(c)
			continue
		}
		// $имя — имя начинается с буквы или _ (цифра первой не допускается,
		// иначе $1/$2 в sed/awk ошибочно станут «переменными»).
		j := i + 1
		if j >= len(runes) || !isVarStartRune(runes[j]) {
			// $ без имени или с цифры — литеральный доллар
			sb.WriteRune(c)
			continue
		}
		start := j
		for j < len(runes) && isVarRune(runes[j]) {
			j++
		}
		sb.WriteString(VarLine(string(runes[start:j])))
		i = j - 1
	}
	return sb.String()
}

// expandBraced разбирает содержимое ${...}: либо просто имя переменной,
// либо Bash Parameter Expansion — усечение по самому короткому совпадению:
//
//	${var%паттерн} — убрать суффикс (как ${var%pattern} в bash);
//	${var#паттерн} — убрать префикс (как ${var#pattern} в bash).
//
// Пока узор — ЛИТЕРАЛ (без glob-звёздочек), через TrimSuffix/TrimPrefix.
func expandBraced(inner string) string {
	for k, r := range inner {
		switch r {
		case '%', '#':
			v := VarLine(inner[:k])
			if r == '%' {
				return strings.TrimSuffix(v, inner[k+1:])
			}
			return strings.TrimPrefix(v, inner[k+1:])
		}
		if !isVarRune(r) {
			break // не имя переменной — оператора нет
		}
	}
	return VarLine(inner)
}
func isVarRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_'
}

// isVarStartRune — ПЕРВЫЙ символ имени переменной: буква или _ (НЕ цифра,
// чтобы $1/$2 в sed/awk не считались переменными).
func isVarStartRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}
