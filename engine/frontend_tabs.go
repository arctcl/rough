package engine

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// drawTabs рисует вкладки внизу (под всеми тайлами). Активная — подсвечена.
// Каждая вкладка — хотзона-ссылка (Kind "nav", Href=роут), клик переключает страницу.
func drawTabs(b *Buffer, menu [][]string, route string, out *[]Hotzone) {
	tabFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	tabBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	// Фон активной вкладки — отдельный ключ active_bg (в title_fg теперь
	// только текст заголовков тайлов). Fallback — title_fg (старые темы).
	actBg := curTheme.ResolveColor(themeColor("active_bg"),
		curTheme.ResolveColor(themeColor("title_fg"), tcell.ColorGreen))
	// Цвет текста активной вкладки — отдельный ключ active_fg.
	// Неактивная вкладка красится tabFg (header_fg). Fallback — header_fg.
	actFg := curTheme.ResolveColor(themeColor("active_fg"),
		curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite))
	x := 0
	for _, m := range menu {
		label := " " + m[0] + " "
		lw := uniseg.StringWidth(label)
		st := Style{Fg: tabFg, Bg: tabBg}
		if m[1] == route {
			st = Style{Fg: actFg, Bg: actBg, Bold: true}
		}
		b.SetString(x, b.H-1, label, st)
		*out = append(*out, Hotzone{X: x, Y: b.H - 1, W: lw, H: 1, Href: m[1], Kind: "nav"})
		x += lw + 1 // пробел в 1 клетку между вкладками
	}
}

// drawQuit рисует кнопку «Закрыть» на правом краю нижней строки (рядом с вкладками).
// Показывается ТОЛЬКО когда открыта строка вывода (statusMsg != ""). Это хотзона
// Kind "quit": клик или Enter закрывают строку вывода (как клавиша q).
// Скобки — из темы (button_l/button_r), цвета — как у вкладок.
func drawQuit(b *Buffer, out *[]Hotzone) {
	tabFg := curTheme.ResolveColor(themeColor("header_fg"), tcell.ColorWhite)
	tabBg := curTheme.ResolveColor(themeColor("header_bg"), tcell.ColorDarkBlue)
	bl := curTheme.Sym("button_l", "⟨")
	br := curTheme.Sym("button_r", "⟩")
	text := string(bl) + " Close " + string(br)
	tw := uniseg.StringWidth(text)
	x := b.W - tw
	if x < 0 {
		x = 0
	}
	b.SetString(x, b.H-1, text, Style{Fg: tabFg, Bg: tabBg})
	*out = append(*out, Hotzone{X: x, Y: b.H - 1, W: tw, H: 1, Kind: "quit"})
}
