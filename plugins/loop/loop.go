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
const man_loop = `loop — перебрать адреса по шаблону (для пайпа с ssh).

Использование:
  часть пайпа: loop:ШАБЛОН | ssh:user:команда

ШАБЛОН — IPv4 с диапазонами [a-b] в любых октетах:
  172.0.0.[1-127]
  172.0.[0-255].[1-254]
  10.0.[0-1].[0-255]

Выдаёт все адреса из диапазонов — каждая строка идёт в ssh как хост.

Примеры:
  action="loop:172.0.0.[1-127] | ssh:root:apt update && apt upgrade -y"
  action="loop:172.0.0.[1-127] | ssh:root:-i:/root/keys:uptime"`

func init() {
	rough.AddMan("loop", man_loop)
	rough.AddPlugin("loop", func(in []string, args []string) ([]string, error) {
		if len(args) < 1 {
			return nil, errors.New("loop: нужен шаблон адреса")
		}
		// Шаблон с диапазонами: 172.0.0.[1-127]
		if strings.Contains(args[0], "[") {
			return genPattern(args[0])
		}
		// Старый формат: loop:БАЗА:КОНЕЦ
		if len(args) < 2 {
			return nil, errors.New("loop: нужен шаблон или стартовый адрес с концом")
		}
		return genIPs(args[0], args[1])
	})
}

// octet — диапазон одного октета (lo..hi).
type octet struct {
	lo, hi int
}

// genPattern разворачивает шаблон вида 172.0.[0-255].[1-254] во все адреса.
func genPattern(pat string) ([]string, error) {
	parts := strings.Split(pat, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("loop: ждём IPv4-шаблон, получили %q", pat)
	}
	octs := make([]octet, 4)
	for i, p := range parts {
		o, err := parseOctet(p)
		if err != nil {
			return nil, err
		}
		octs[i] = o
	}
	var out []string
	cur := make([]int, 4)
	var walk func(i int)
	walk = func(i int) {
		if i == 4 {
			out = append(out, net.IPv4(byte(cur[0]), byte(cur[1]), byte(cur[2]), byte(cur[3])).String())
			return
		}
		for v := octs[i].lo; v <= octs[i].hi; v++ {
			cur[i] = v
			walk(i + 1)
		}
	}
	walk(0)
	return out, nil
}

// parseOctet разбирает октет: "10" — фиксированный, "[1-127]" — диапазон.
func parseOctet(s string) (octet, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := s[1 : len(s)-1]
		pp := strings.SplitN(inner, "-", 2)
		if len(pp) != 2 {
			return octet{}, fmt.Errorf("loop: плохой диапазон %q", s)
		}
		lo, err1 := strconv.Atoi(strings.TrimSpace(pp[0]))
		hi, err2 := strconv.Atoi(strings.TrimSpace(pp[1]))
		if err1 != nil || err2 != nil || lo < 0 || hi > 255 || lo > hi {
			return octet{}, fmt.Errorf("loop: плохой диапазон %q", s)
		}
		return octet{lo: lo, hi: hi}, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 || v > 255 {
		return octet{}, fmt.Errorf("loop: плохой октет %q", s)
	}
	return octet{lo: v, hi: v}, nil
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

