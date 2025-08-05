package markdown

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kolkov/gops/internal/model"
)

func (g *Generator) WriteFileSection(file *model.ProjectFile) {
	g.file.WriteString(fmt.Sprintf("### %s\n\n", file.Path))

	if file.Skipped {
		g.file.WriteString("```\n[Содержимое файла пропущено]\n```\n\n")
		g.file.WriteString("_Файл был пропущен по настройкам документации._\n\n")
		return
	}

	// Определяем язык для подсветки синтаксиса
	lang := file.Lang
	ext := strings.ToLower(filepath.Ext(file.Path))
	switch ext {
	case ".scss", ".sass", ".less":
		lang = "scss"
	case ".html", ".htm":
		lang = "html"
	}

	g.file.WriteString(fmt.Sprintf("```%s\n", lang))
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
