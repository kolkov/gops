package markdown

import (
	"fmt" // Добавляем импорт fmt
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

	// Режим заголовков
	if g.docsMode == "headers" && file.Header != nil {
		g.writeFileHeader(file)
		return
	}

	// Полный режим
	g.writeFullContent(file)
}

func (g *Generator) writeFileHeader(file *model.ProjectFile) {
	header := file.Header

	g.file.WriteString("**File Type:** ")
	g.file.WriteString(strings.ToUpper(file.Lang))
	g.file.WriteString("\n\n")

	g.file.WriteString("**Summary:** ")
	if header.Summary != "" {
		g.file.WriteString(header.Summary)
	} else {
		g.file.WriteString("No summary available")
	}
	g.file.WriteString("\n\n")

	// Показываем информацию только для Go файлов (остальные - заглушки)
	if file.Lang == "go" {
		if len(header.ExportedFunctions) > 0 {
			g.file.WriteString("**Exported Functions:**\n```go\n")
			for _, fn := range header.ExportedFunctions {
				g.file.WriteString(fn + "\n")
			}
			g.file.WriteString("```\n\n")
		}

		if len(header.ExportedTypes) > 0 {
			g.file.WriteString("**Exported Types:**\n```go\n")
			for _, typ := range header.ExportedTypes {
				g.file.WriteString(typ + "\n")
			}
			g.file.WriteString("```\n\n")
		}

		if len(header.Dependencies) > 0 {
			g.file.WriteString("**Dependencies:**\n")
			for _, dep := range header.Dependencies {
				g.file.WriteString(fmt.Sprintf("- `%s`\n", dep))
			}
			g.file.WriteString("\n")
		}
	}

	// Исправляем форматирование строки
	g.file.WriteString(fmt.Sprintf("_To view full file content, use the tag: `[FILE:%s]`_\n\n", file.Path))
}

func (g *Generator) writeFullContent(file *model.ProjectFile) {
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
