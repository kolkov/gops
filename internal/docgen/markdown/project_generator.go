package markdown

import "fmt"

func (g *Generator) WriteProjectHeader(name, ptype, root string) {
	g.file.WriteString(fmt.Sprintf("\n## Проект: %s (%s)\n\n", name, ptype))
	g.file.WriteString(fmt.Sprintf("**Путь:** %s\n\n", root))
}

func (g *Generator) WriteProjectTree(structure string) {
	g.file.WriteString("### Структура проекта\n\n")
	g.file.WriteString("```\n")
	g.file.WriteString(structure)
	g.file.WriteString("\n```\n\n")
}

func (g *Generator) WriteModulesHeader() {
	g.file.WriteString("## Основные модули\n\n")
	g.file.WriteString("В этом разделе представлены ключевые файлы проекта с их исходным кодом:\n\n")
}
