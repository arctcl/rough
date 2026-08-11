// Плагин loop — перебор адресов диапазона (для пайпа с ssh).
// Выдаёт строки адресов от БАЗЫ до конца; каждая строка уходит в следующий
// шаг пайпа как хост. Вместе с ssh — «раскатать команду по всем хостам».
// Вызов: loop:БАЗА:КОНЕЦ | ssh:user:команда
package loop

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"rough"
)

// man_loop — справка по плагину (для man).
const man_loop = `loop — перебрать адреса диапазона (для пайпа с ssh).

Использование:
  часть пайпа: loop:БАЗА:КОНЕЦ | ssh:user:команда

Аргументы:
  БАЗА  — стартовый адрес (например 172.0.0.1).
  КОНЕЦ — последний октет (25) или полный адрес (172.0.0.25).

Выдаёт строки адресов от БАЗЫ до конца — каждая идёт в ssh как хост.

Примеры:
  action="loop:172.0.0.1:25 | ssh:root:apt update && apt upgrade -y"
  action="loop:172.0.0.1:25 | ssh:root:-i:/root/keys:uptime"
  action="loop:10.0.0.10:20 | ssh:admin:hostname"`

func init() {
	rough.AddMan("loop", man_loop)
	rough.AddPlugin("loop", func(in []string, args []string) ([]string, error) {
		if len(args) < 2 {
			return nil, errors.New("loop: нужен стартовый адрес и конец")
		}
		return genIPs(args[0], args[1])
	})
}

// genIPs генерирует адреса от start до end (конец — число последнего октета
// или полный адрес). Возвращает строки адресов.
func genIPs(start, end string) ([]string, error) {
	ip := net.ParseIP(start).To4()
	if ip == nil {
		return nil, fmt.Errorf("loop: плохой стартовый адрес %q", start)
	}
	last, err := strconv.Atoi(end)
	if err != nil {
		// Конец — полный адрес.
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
