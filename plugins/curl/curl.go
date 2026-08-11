// Плагин curl — скачать URL и отдать тело построчно.
// Юникс-лайк: `curl https://...` — так же и здесь: action="curl:https://example.com".
// Нативно в Go (net/http) — никакого внешнего curl, работает в любом контейнере.
// ВАЖНО: URL содержит «:», а аргументы вызова режутся по «:» — поэтому плагин
// склеивает аргументы обратно через «:» (URL восстанавливается как есть).
// Можно пайпом: action="curl:https://api.example.com | grep:ok".
package curl

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"rough"
)

func init() {
	rough.AddPlugin("curl", func(in []string, args []string) ([]string, error) {
		if len(args) < 1 {
			return nil, errors.New("curl: нужен URL")
		}
		// Склеиваем обратно: curl:https://x → args=["https","//x"] → "https://x".
		url := strings.Join(args, ":")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return nil, errors.New("curl: " + resp.Status)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			return nil, nil
		}
		return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), nil
	})
}
