// Плагин ssh — выполнить команду на удалённом сервере по SSH.
// Юникс-лайк: `ssh user@host команда` — здесь единые quick-параметры:
// action="ssh:user:host::команда" (порт — пустой слот или --port, дефолт 22).
// Нативно в Go (golang.org/x/crypto/ssh) — никаких внешних ssh-бинарников,
// работает в любом контейнере. Аутентификация: ssh-agent + стандартные ключи ~/.ssh
// (или --keys=ПУТЬ), проверка хоста — по known_hosts (если файл есть).
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
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"rough"
	"rough/engine"
)

// man_ssh — справка по плагину (для man).
const man_ssh = `ssh — выполнить команду на удалённом сервере по SSH.

Использование (единые quick-параметры: позиционные или флаги, можно микс):
  action="ssh:user:host::команда"             — порт по дефолту 22
  action="ssh:user:host:2222:команда"         — порт позиционно
  action="ssh:user:host::команда --port=2222" — порт флагом
  action="ssh:user:host::команда --keys=ПУТЬ" — свои ключи

Аргументы:
  user    — пользователь на сервере.
  host    — адрес сервера.
  port    — порт (по умолчанию 22): пустой слот, позиционно или флаг --port.
  команда — что выполнить (остаток через «:»).
  --keys=ПУТЬ — папка с ключами или конкретный файл ключа (иначе ~/.ssh).

Аутентификация: ssh-agent (SSH_AUTH_SOCK), затем ключи из --keys или ~/.ssh:
id_ed25519, id_rsa, id_ecdsa. Проверка хоста — по known_hosts (если файл есть).

Примеры:
  action="ssh:root:srv1:uptime"
  action="ssh:root:srv1::systemctl status nginx"
  action="ssh:root:srv1:2222:uptime"
  action="ssh:root:srv1::hostname --keys=/root/keys"
  action="ssh:root:srv1::df -h | grep:/data"`

// sshParams — единые quick-параметры ssh. Порядок = позиции:
// user, host, port (дефолт 22), cmd (глотает остаток).
var sshParams = []engine.Param{
	{Name: "user"},
	{Name: "host"},
	{Name: "port", Default: "22"},
	{Name: "cmd"},
}

func init() {
	rough.AddMan("ssh", man_ssh)

	rough.AddPlugin("ssh", func(in []string, args []string) ([]string, error) {
		// Ключи — флаг --keys=ПУТЬ. Вырезаем вручную, чтобы не занимал
		// позиционный слот (иначе сдвинул бы user:host:port:cmd).
		keyPath := ""
		rest := args[:0]
		for _, a := range args {
			if strings.HasPrefix(a, "--keys=") {
				keyPath = strings.TrimPrefix(a, "--keys=")
				continue
			}
			rest = append(rest, a)
		}
		// Остальное — единый разбор quick-параметров.
		vals, err := engine.ParseArgs(rest, sshParams)
		if err != nil {
			return nil, err
		}
		user, host, port, cmd := vals["user"], vals["host"], vals["port"], vals["cmd"]
		if user == "" || host == "" || cmd == "" {
			return nil, errors.New("ssh: нужен user, host и команда")
		}
		return runSSH(user, host, port, keyPath, cmd)
	})
}

// runSSH подключается к серверу, выполняет команду и возвращает вывод построчно.
func runSSH(user, host, port, keyPath, cmd string) ([]string, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods(keyPath),
		HostKeyCallback: hostKeyCallback(),
		Timeout:         10 * time.Second, // 10 секунд на установку соединения
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, port), config)
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
