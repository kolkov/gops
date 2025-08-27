// markdown/generator.go
package markdown

import (
	"fmt"
	"os"
	"time"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
	"go.uber.org/zap"
)

// Generator реализует генерацию Markdown документации
type Generator struct {
	file       *os.File
	logger     *logger.Logger
	outputFile string
}

func NewGenerator(outputFile string, logger *logger.Logger) *Generator {
	file, err := os.Create(outputFile)
	if err != nil {
		logger.Fatal("Failed to create output file", zap.Error(err))
	}
	return &Generator{
		file:       file,
		logger:     logger,
		outputFile: outputFile,
	}
}

func (g *Generator) WriteHeader(meta *model.ProjectMeta) {
	g.file.WriteString(fmt.Sprintf("# Проект: %s\n\n", meta.Name))
	g.file.WriteString(fmt.Sprintf("**Дата генерации:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	g.file.WriteString("**Полнота:** Полная структура проекта и исходный код\n\n")
	g.file.WriteString(fmt.Sprintf("**Тип:** %s\n\n", meta.Type))
	g.file.WriteString("## Содержание\n")
	g.file.WriteString("- [Структура проекта](#структура-проекта)\n")
	g.file.WriteString("- [Основные модули](#основные-модули)\n")
}

func (g *Generator) WriteTree(structure string) {
	g.file.WriteString("## Структура проекта\n\n")
	g.file.WriteString("```\n")
	g.file.WriteString(structure)
	g.file.WriteString("\n```\n\n")
}
