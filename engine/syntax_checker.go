package engine

import (
	"io/fs"
	"strings"
)

// Проверяльщик синтаксиса: валидирует tiles.json и HTML до запуска интерфейса.
// Каждое action, <plugin> и ссылка должны ссылаться на существующее.
// Пока есть ошибки — интерфейс не стартует, движок «шлёт нахуй выдумщиков».

// CheckSyntax проверяет интерфейс и возвращает список ошибок (пустой = всё ок).
func CheckSyntax(fsys fs.FS, pages Pages) []string {
	var errs []string
	add := func(where, msg string) {
		errs = append(errs, where+": "+msg)
	}

	// Каждая страница и каждый тайл.
	for route, tiles := range pages {
		for _, t := range tiles {
			if t.File == "" {
				add(route+" / "+t.ID, "нет файла тайла")
				continue
			}
			if _, err := fs.Stat(fsys, t.File); err != nil {
				add(route+" / "+t.ID, "нет файла: "+t.File)
				continue
			}
			checkHTMLFile(fsys, t.File, pages, add)
		}
	}
	return errs
}

// checkHTMLFile парсит HTML-файл тайла и проверяет все узлы.
func checkHTMLFile(fsys fs.FS, file string, pages Pages, add func(where, msg string)) {
	f, err := fsys.Open(file)
	if err != nil {
		add(file, "не открылся: "+err.Error())
		return
	}
	root, perr := ParseHTML(f)
	f.Close()
	if perr != nil {
		add(file, "не распарсился: "+perr.Error())
		return
	}
	checkNodes(root, file, pages, add)
}

// checkNodes рекурсивно проверяет узлы HTML-дерева.
func checkNodes(n *Node, file string, pages Pages, add func(where, msg string)) {
	switch n.Tag {
	case "button", "input", "checkbox", "select":
		// Все шаги action (или пайпа) должны существовать в реестре.
		act := n.Attrs["action"]
		if act == "" {
			add(file, "<"+n.Tag+"> без атрибута action")
			break
		}
		for _, s := range SplitSteps(act) {
			name, _ := SplitAction(s)
			if name == "confirm" {
				continue // гейт подтверждения, не плагин
			}
			if !HasPlugin(name) {
				add(file, "нет такого плагина: "+name+"  (action=\""+act+"\")")
			}
		}
		// У select должен быть options (иначе выпадать нечему).
		if n.Tag == "select" && n.Attrs["options"] == "" {
			add(file, "<select> без атрибута options  (action=\""+act+"\")")
		}
	case "a":
		// Ссылка должна вести на существующую страницу.
		href := n.Attrs["href"]
		if href == "" {
			add(file, "<a> без атрибута href")
			break
		}
		if _, ok := pages[href]; !ok {
			add(file, "нет такой страницы: "+href)
		}
	case "plugin":
		// Все шаги пайпа (или собранного из атрибутов) должны существовать.
		for _, s := range pluginSteps(n) {
			name, _ := SplitAction(s)
			if !HasPlugin(name) {
				add(file, "нет такого плагина: "+name+"  (в <plugin>)")
			}
		}
	}
	for _, c := range n.Children {
		checkNodes(c, file, pages, add)
	}
}

// syntaxErrorsOneLine склеивает ошибки в одну строку для вывода.
func syntaxErrorsOneLine(errs []string) string {
	return strings.Join(errs, "\n  ")
}
