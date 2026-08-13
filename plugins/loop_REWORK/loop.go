// Плагин loop — перебор адресов по шаблону с диапазонами.
// Шаблон: 172.0.0.[1-127] или 172.0.[0-255].[1-254] — диапазоны в квадратных
// скобках в любых октетах. Выдаёт строки адресов — каждая уходит в пайп как хост.
// Вызов: loop:ШАБЛОН | ssh:user:команда   (или старый формат loop:БАЗА:КОНЕЦ)
package loop

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"rough"
)

// man_loop — справка по плагину (для man).
const man_loop = `loop — перебрать по шаблону с диапазонами (для пайпа с ssh).

Использование:
  часть пайпа: loop:ШАБЛОН | ssh:user:команда

ШАБЛОН — любая строка с диапазонами [a-b]:
  172.0.0.[1-127]          — адреса 172.0.0.1 .. 172.0.0.127
  172.0.[0-255].[1-254]    — диапазоны в любых местах
  пирожок_номер_[1-999]    — не только IP: пирожок_номер_1 .. пирожок_номер_999

Выдаёт все варианты — каждая строка идёт в ssh как хост.

Примеры:
  action="loop:172.0.0.[1-127] | ssh:root:apt update && apt upgrade -y"
  action="loop:пирожок_номер_[1-5] | ssh:root:hostname"`

func init() {
	rough.AddMan("loop", man_loop)
	rough.AddPlugin("loop", func(in []string, args []string) ([]string, error) {
		if len(args) < 1 {
			return nil, errors.New("loop: нужен шаблон")
		}
		// Шаблон с диапазонами [a-b] — универсальный разворачиватель.
		if strings.Contains(args[0], "[") {
			return genTemplate(args[0])
		}
		// Старый формат: loop:БАЗА:КОНЕЦ (только IP-диапазон).
		if len(args) < 2 {
			return nil, errors.New("loop: нужен шаблон или стартовый адрес с концом")
		}
		return genIPs(args[0], args[1])
	})
}

// part — кусок шаблона: фиксированный текст или диапазон lo..hi.
type part struct {
	fixed   string
	lo, hi  int
	isRange bool
}

// genTemplate разворачивает любой шаблон с диапазонами [a-b] во все варианты.
// Примеры: 172.0.0.[1-3] → 172.0.0.1..3;  пирожок_номер_[1-999] → пирожок_номер_1..999.
func genTemplate(pat string) ([]string, error) {
	parts, err := parseTemplate(pat)
	if err != nil {
		return nil, err
	}
	var out []string
	cur := ""
	var walk func(i int)
	walk = func(i int) {
		if i == len(parts) {
			out = append(out, cur)
			return
		}
		p := parts[i]
		if !p.isRange {
			cur += p.fixed
			walk(i + 1)
			cur = cur[:len(cur)-len(p.fixed)]
			return
		}
		for v := p.lo; v <= p.hi; v++ {
			s := strconv.Itoa(v)
			cur += s
			walk(i + 1)
			cur = cur[:len(cur)-len(s)]
		}
	}
	walk(0)
	return out, nil
}

// parseTemplate разбирает шаблон на фиксированные куски и диапазоны [a-b].
func parseTemplate(pat string) ([]part, error) {
	var parts []part
	rest := pat
	for len(rest) > 0 {
		i := strings.Index(rest, "[")
		if i < 0 {
			if rest != "" {
				parts = append(parts, part{fixed: rest})
			}
			break
		}
		if i > 0 {
			parts = append(parts, part{fixed: rest[:i]})
		}
		j := strings.Index(rest[i:], "]")
		if j < 0 {
			return nil, fmt.Errorf("loop: незакрытый диапазон в %q", pat)
		}
		j += i
		inner := rest[i+1 : j]
		pp := strings.SplitN(inner, "-", 2)
		if len(pp) != 2 {
			return nil, fmt.Errorf("loop: плохой диапазон [%s]", inner)
		}
		lo, err1 := strconv.Atoi(strings.TrimSpace(pp[0]))
		hi, err2 := strconv.Atoi(strings.TrimSpace(pp[1]))
		if err1 != nil || err2 != nil || lo < 0 || lo > hi {
			return nil, fmt.Errorf("loop: плохой диапазон [%s]", inner)
		}
		parts = append(parts, part{lo: lo, hi: hi, isRange: true})
		rest = rest[j+1:]
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("loop: пустой шаблон %q", pat)
	}
	return parts, nil
}

// genIPs — старый формат: адреса от start до end (конец — число или полный адрес).
func genIPs(start, end string) ([]string, error) {
	ip := net.ParseIP(start).To4()
	if ip == nil {
		return nil, fmt.Errorf("loop: плохой стартовый адрес %q", start)
	}
	last, err := strconv.Atoi(end)
	if err != nil {
		e := net.ParseIP(end).To4()
		if e == nil {
			return nil, fmt.Errorf("loop: плохой конец %q", end)
		}
		last = int(e[3])
	}
	first := int(ip[3])
	if last < first {
		return nil, errors.New("loop: конец меньше старта")
	}
	var out []string
	base := append(net.IP(nil), ip...)
	for i := first; i <= last; i++ {
		b := append(net.IP(nil), base...)
		b[3] = byte(i)
		out = append(out, b.String())
	}
	return out, nil
}
