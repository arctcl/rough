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

func init() {
	// ssh: host user@host + команда (склеивается через «:» обратно)
	rough.AddPlugin("ssh", func(in []string, args []string) ([]string, error) {
		if len(args) < 2 {
			return nil, errors.New("ssh: нужен хост user@host и команда")
		}
		addr := args[0]
		cmd := strings.Join(args[1:], ":")
		return runSSH(addr, cmd)
	})
}

// runSSH подключается к серверу, выполняет команду и возвращает вывод построчно.
func runSSH(addr, cmd string) ([]string, error) {
	user, host := splitHost(addr)
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods(),
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

// authMethods собирает способы аутентификации: ssh-agent и стандартные ключи.
func authMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	// 1) ssh-agent (SSH_AUTH_SOCK) — если запущен агент.
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}

	// 2) стандартные ключи из ~/.ssh (без пароля — пароль не храним нигде).
	for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
		p := filepath.Join(homeDir(), ".ssh", name)
		if b, err := os.ReadFile(p); err == nil {
			if signer, err := ssh.ParsePrivateKey(b); err == nil {
				methods = append(methods, ssh.PublicKeys(signer))
			}
		}
	}
	return methods
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
