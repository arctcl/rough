// Demo plugin nginx — an in-memory nginx config editor:
// nginx:KEY:VAL sets a key, nginx:KEY:get reads it,
// nginx:toggle:KEY flips on/off, nginx:reload / nginx:reboot / nginx:backup act.
package plugins

import "github.com/arctcl/rough"

func init() {
	nginxData := map[string]string{
		"worker_processes":     "4",
		"server_name":          "example.com",
		"listen":               "8080",
		"log_level":            "info",
		"gzip":                 "on",
		"access_log":           "on",
		"client_max_body_size": "10m",
		"keepalive_timeout":    "65",
		"ssl_protocols":        "TLSv1.2",
		"proxy_read_timeout":   "60",
		"upstream":             "round_robin",
		"error_log":            "/var/log/nginx/error.log",
	}
	nginxConf := func() []string {
		return []string{
			"worker_processes " + nginxData["worker_processes"] + ";",
			"server_name " + nginxData["server_name"] + ";",
			"listen " + nginxData["listen"] + ";",
			"log_level " + nginxData["log_level"] + ";",
			"gzip " + nginxData["gzip"] + ";",
			"access_log " + nginxData["access_log"] + ";",
			"client_max_body_size " + nginxData["client_max_body_size"] + ";",
			"keepalive_timeout " + nginxData["keepalive_timeout"] + ";",
			"ssl_protocols " + nginxData["ssl_protocols"] + ";",
			"proxy_read_timeout " + nginxData["proxy_read_timeout"] + ";",
			"upstream " + nginxData["upstream"] + ";",
			"error_log " + nginxData["error_log"] + ";",
		}
	}
	rough.AddPlugin("nginx", func(in []string, args []string) ([]string, error) {
		if len(args) == 0 {
			return nginxConf(), nil
		}
		// Ключ напрямую: nginx:KEY:VAL — установить, nginx:KEY:get — прочитать.
		// Так <select>/<input> работают чисто: currentSelect делает :get,
		// а выбор варианта/ввод — KEY:VAL (без лишнего шага "set").
		if _, ok := nginxData[args[0]]; ok {
			if len(args) > 1 && args[1] == "get" {
				return []string{nginxData[args[0]]}, nil
			}
			if len(args) > 1 {
				nginxData[args[0]] = args[1]
			}
			return nginxConf(), nil
		}
		switch args[0] {
		case "toggle":
			if len(args) >= 3 && args[2] == "get" {
				if nginxData[args[1]] == "on" {
					return []string{"on"}, nil
				}
				return []string{"off"}, nil
			}
			if nginxData[args[1]] == "on" {
				nginxData[args[1]] = "off"
			} else {
				nginxData[args[1]] = "on"
			}
			return nginxConf(), nil
		case "reload":
			return append(nginxConf(), "nginx reloaded OK"), nil
		case "backup":
			return []string{"config backed up to nginx.conf.bak"}, nil
		case "reboot":
			return []string{"docker container restarted OK", "state: healthy (uptime 0.4s)"}, nil
		default: // "get" и всё неизвестное — просто показать конфиг
			return nginxConf(), nil
		}
	})
}
