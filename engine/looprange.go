package engine

import (
	"strconv"
	"strings"
)

// expandRanges разворачивает диапазоны [N-M], [a-b] и списки [v1,v2] в списки
// конкретных действий — перебор значений (например адресов). Пример:
//
//	"ssh:192.168.1.[1-3]:apt upgrade"
//	  → "ssh:192.168.1.1:apt upgrade"
//	    "ssh:192.168.1.2:apt upgrade"
//	    "ssh:192.168.1.3:apt upgrade"
//
// Несколько диапазонов — декартово произведение. Без диапазонов возвращает
// исходное действие одним элементом. Содержимое, которое не похоже на диапазон
// или список (например "[foo]"), не трогаем — оставляем как есть.
func expandRanges(raw string) []string {
	type rng struct {
		open, close int // индексы '[' и ']'
		vals        []string
	}
	var ranges []rng
	for i := 0; i < len(raw); i++ {
		if raw[i] != '[' {
			continue
		}
		j := strings.IndexByte(raw[i:], ']')
		if j < 0 {
			break
		}
		j += i
		if v := rangeValues(raw[i+1 : j]); v != nil {
			ranges = append(ranges, rng{open: i, close: j, vals: v})
		}
		i = j
	}
	if len(ranges) == 0 {
		return []string{raw}
	}

	// Собираем все комбинации (декартово произведение).
	var out []string
	var rec func(idx int, buf []byte)
	rec = func(idx int, buf []byte) {
		if idx == len(ranges) {
			// хвост после последнего диапазона
			tail := raw[ranges[len(ranges)-1].close+1:]
			out = append(out, string(buf)+tail)
			return
		}
		start := 0
		if idx > 0 {
			start = ranges[idx-1].close + 1
		}
		prefix := raw[start:ranges[idx].open]
		for _, v := range ranges[idx].vals {
			next := append(append([]byte{}, buf...), prefix...)
			next = append(next, v...)
			rec(idx+1, next)
		}
	}
	rec(0, nil)
	return out
}

// rangeValues разбирает содержимое "[...]" в список значений.
//
//	"1-3"   → 1,2,3
//	"a-c"   → a,b,c
//	"1,4,9" → 1,4,9
//
// Если не похоже ни на то, ни на другое — возвращает nil (не диапазон).
func rangeValues(spec string) []string {
	if strings.Contains(spec, ",") {
		return strings.Split(spec, ",")
	}
	if i := strings.IndexByte(spec, '-'); i > 0 {
		lo, hi := spec[:i], spec[i+1:]
		// Числовой диапазон N-M.
		if a, err := strconv.Atoi(lo); err == nil {
			if b, err := strconv.Atoi(hi); err == nil {
				var out []string
				if a <= b {
					for v := a; v <= b; v++ {
						out = append(out, strconv.Itoa(v))
					}
				} else {
					for v := a; v >= b; v-- {
						out = append(out, strconv.Itoa(v))
					}
				}
				return out
			}
		}
		// Буквенный диапазон a-c.
		if len(lo) == 1 && len(hi) == 1 {
			var out []string
			for r := lo[0]; ; {
				out = append(out, string(r))
				if r == hi[0] {
					break
				}
				if r < hi[0] {
					r++
				} else {
					r--
				}
			}
			return out
		}
	}
	return nil
}
