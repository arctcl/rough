// Плагин ssh — выполнить команду на удалённом сервере по SSH.
// Юникс-лайк: `ssh user@host команда` — так же и здесь:
// action="ssh:user@host:команда" (команда — остаток после второго «:»).
// Нативно в Go (golang.org/x/crypto/ssh) — никаких внешних ssh-бинарников,
// работает в любом контейнере. Аутентификация: ssh-agent + стандартные ключи ~/.ssh,
// проверка хоста — по known_hosts (если файл есть).
package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"rough"
)

// man_ssh — справка по плагину (для man).
const man_ssh = `ssh — выполнить команду на удалённом сервере по SSH.

Использование:
  action="ssh:user@host:команда"
  action="ssh:user@host:-i:ПУТЬ:команда"   — с указанием папки или файла ключей

Аргументы:
  user@host   — куда подключаться.
  -i ПУТЬ     — папка с ключами или конкретный файл ключа (как у настоящего ssh).
                Если не указан — ключи ищутся в ~/.ssh (дефолт ssh).
  команда     — что выполнить (склеивается из остатка через «:»).

Аутентификация: ssh-agent (SSH_AUTH_SOCK), затем ключи из указанной папки
или ~/.ssh по дефолту: id_ed25519, id_rsa, id_ecdsa. Проверка хоста —
по known_hosts (если файл есть).

Примеры:
  action="ssh:root@srv1:systemctl status nginx"
  action="ssh:root@srv1:uptime"
  action="ssh:root@srv1:-i:/root/keys:hostname"          — ключи из папки /root/keys
  action="ssh:root@srv1:-i:~/.ssh/id_rsa:uptime"         — конкретный файл ключа
  action="ssh:root@srv1:df -h | grep:/data"              — вывод идёт в пайп
  action="ssh:root@srv1:-i:/root/keys:df -h | head:3"    — ключи + пайп`

func init() {
	// ssh: user@host [ -i ПУТЬ ] : команда (команда склеивается через «:» обратно)
	rough.AddMan("ssh", man_ssh)

	rough.AddPlugin("ssh", func(in []string, args []string) ([]string, error) {
		if len(args) < 2 {
			return nil, errors.New("ssh: нужен хост user@host и команда")
		}
		addr := args[0]
		rest := args[1:]

		// Необязательный флаг -i: папка с ключами или конкретный файл ключа.
		// Не указан — ключи из ~/.ssh по дефолту (как у настоящего ssh).
		keyPath := ""
		if len(rest) >= 2 && rest[0] == "-i" {
			keyPath = rest[1]
			rest = rest[2:]
		}

		cmd := strings.Join(rest, ":")
		if cmd == "" {
			return nil, errors.New("ssh: нужна команда")
		}
		return runSSH(addr, keyPath, cmd)
	})
}

// runSSH подключается к серверу, выполняет команду и возвращает вывод построчно.
func runSSH(addr, keyPath, cmd string) ([]string, error) {
	user, host := splitHost(addr)
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods(keyPath),
		HostKeyCallback: hostKeyCallback(),
		Timeout:         10 * 1e9, // 10 секунд на установку соединения
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, "22"), config)
	if err != nil {
		return nil, fmt.Errorf("ssh: %w", err)
	}
	defer conn.Close()

	sess, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh: %w", err)
	}
	defer sess.Close()

	var buf bytes.Buffer
	sess.Stdout = &buf
	sess.Stderr = &buf
	if err := sess.Run(cmd); err != nil {
		// Команда вернула ненулевой код — отдаём и вывод, и ошибку,
		// чтобы в статусе было видно, что произошло.
		return splitLines(buf.String()), fmt.Errorf("ssh %s: %v", cmd, err)
	}
	return splitLines(buf.String()), nil
}

// splitHost разбирает "user@host" на пользователя и хост (без порта).
// Без «@» — текущий пользователь, хост как есть.
func splitHost(addr string) (usr, host string) {
	if i := strings.Index(addr, "@"); i >= 0 {
		return addr[:i], addr[i+1:]
	}
	u, err := user.Current()
	if err != nil {
		return "", addr
	}
	return u.Username, addr
}

// authMethods собирает способы аутентификации: ssh-agent и ключи.
func authMethods(keyPath string) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	// 1) ssh-agent (SSH_AUTH_SOCK) — если запущен агент.
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}

	// 2) ключи: указанная папка/файл или ~/.ssh по дефолту.
	for _, p := range keyFiles(keyPath) {
		if b, err := os.ReadFile(p); err == nil {
			if signer, err := ssh.ParsePrivateKey(b); err == nil {
				methods = append(methods, ssh.PublicKeys(signer))
			}
		}
	}
	return methods
}

// keyFiles возвращает файлы ключей для аутентификации. keyPath может быть:
// пустой — дефолт ~/.ssh (как у настоящего ssh);
// файл — используется только он; папка — стандартные имена внутри неё.
func keyFiles(keyPath string) []string {
	if keyPath == "" {
		keyPath = filepath.Join(homeDir(), ".ssh")
	}
	// Если указан конкретный файл — берём только его.
	if fi, err := os.Stat(keyPath); err == nil && !fi.IsDir() {
		return []string{keyPath}
	}
	// Иначе — стандартные имена ключей в указанной (или дефолтной) папке.
	var out []string
	for _, n := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
		out = append(out, filepath.Join(keyPath, n))
	}
	return out
}

// hostKeyCallback проверяет хост по known_hosts. Если файла нет — не проверяем
// (свой инвентарь), иначе MITM был бы возможен на чужих серверах.
func hostKeyCallback() ssh.HostKeyCallback {
	if hk, err := knownhosts.New(filepath.Join(homeDir(), ".ssh", "known_hosts")); err == nil {
		return hk
	}
	return ssh.InsecureIgnoreHostKey()
}

// homeDir возвращает домашнюю директорию текущего пользователя.
func homeDir() string {
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return os.Getenv("HOME")
}

// splitLines разбивает вывод на строки (без хвостового перевода строки).
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}
