package markdown

import (
	"fmt"
	"github.com/kolkov/gops/internal/model"
)

func (g *Generator) WriteFileSection(file *model.ProjectFile) {
	g.file.WriteString(fmt.Sprintf("### %s\n\n", file.Path))

	if file.Skipped {
		g.file.WriteString("```\n[Содержимое файла пропущено]\n```\n\n")
		g.file.WriteString("_Файл был пропущен по настройкам документации._\n\n")
		return
	}

	g.file.WriteString(fmt.Sprintf("```%s\n", file.Lang))
	g.file.Write(file.Content)
	if len(file.Content) > 0 && file.Content[len(file.Content)-1] != '\n' {
		g.file.WriteString("\n")
	}
	g.file.WriteString("```\n\n")
}

func (g *Generator) Close() error {
	if g.file != nil {
		return g.file.Close()
	}
	return nil
}
