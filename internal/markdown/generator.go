package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"project_scanner/internal/utils"
)

type DocumentationGenerator struct {
	file *os.File
}

func NewDocumentationGenerator(filename string) *DocumentationGenerator {
	file, err := os.Create(filename)
	if err != nil {
		panic(fmt.Sprintf("Ошибка создания файла: %v", err))
	}
	return &DocumentationGenerator{file: file}
}

func (d *DocumentationGenerator) Close() {
	d.file.Close()
}

func (d *DocumentationGenerator) WriteHeader(projectName string, currentTime time.Time, isNx bool) {
	d.file.WriteString(fmt.Sprintf("# Проект: %s\n\n", projectName))
	d.file.WriteString(fmt.Sprintf("**Дата генерации:** %s\n", currentTime.Format("2006-01-02 15:04:05")))
	d.file.WriteString("**Полнота:** Полная документация проекта\n")

	if isNx {
		d.file.WriteString("**Тип:** NX Monorepo\n\n")
	} else {
		d.file.WriteString("**Тип:** Стандартный проект\n\n")
	}
}

func (d *DocumentationGenerator) WriteNxStructure(projects []utils.NxProject) {
	d.file.WriteString("## Общая структура Nx Monorepo\n\n")
	d.file.WriteString(d.GenerateNxStructure(projects))
	d.file.WriteString("\n")
}

func (d *DocumentationGenerator) WriteProjectHeader(name, ptype, root string) {
	d.file.WriteString(fmt.Sprintf("\n## Проект: %s (%s)\n\n", name, ptype))
	d.file.WriteString(fmt.Sprintf("**Путь:** %s\n\n", root))
	d.file.WriteString("### Структура проекта\n\n")
}

func (d *DocumentationGenerator) WriteProjectTree(sourceDir, basePath string) {
	d.file.WriteString(d.GenerateProjectTree(sourceDir, basePath))
	d.file.WriteString("\n")
}

func (d *DocumentationGenerator) WriteStandardProjectTree(rootDir string) {
	d.file.WriteString("## Полная структура проекта\n\n")
	d.file.WriteString(d.GenerateProjectTree(rootDir, rootDir))
	d.file.WriteString("\n")
}

func (d *DocumentationGenerator) WriteFileSection(filePath string, content []byte, lang string) {
	d.file.WriteString(fmt.Sprintf("### %s\n\n", filePath))
	d.file.WriteString(fmt.Sprintf("```%s\n", lang))
	d.file.Write(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		d.file.WriteString("\n")
	}
	d.file.WriteString("```\n\n")
}

func GenerateOutputFilename(currentTime time.Time) string {
	return fmt.Sprintf("project_documentation_%s.md", currentTime.Format("20060102_150405"))
}

// GenerateNxStructure создает структуру Nx Monorepo в формате Markdown
func (d *DocumentationGenerator) GenerateNxStructure(projects []utils.NxProject) string {
	var builder strings.Builder
	builder.WriteString("```\nnx-monorepo/\n")

	// Группировка проектов по типу
	projectsByType := make(map[string][]utils.NxProject)
	for _, p := range projects {
		projectsByType[p.Type] = append(projectsByType[p.Type], p)
	}

	// Сортировка типов для детерминированного вывода
	var types []string
	for t := range projectsByType {
		types = append(types, t)
	}
	sort.Strings(types)

	// Построение структуры
	for _, t := range types {
		projects := projectsByType[t]
		builder.WriteString(fmt.Sprintf("├── %ss/\n", t))
		for i, p := range projects {
			prefix := "├──"
			if i == len(projects)-1 {
				prefix = "└──"
			}
			builder.WriteString(fmt.Sprintf("│   %s %s\n", prefix, p.Name))
		}
	}
	builder.WriteString("```\n")
	return builder.String()
}

// GenerateProjectTree создает древовидную структуру проекта
func (d *DocumentationGenerator) GenerateProjectTree(root, basePath string) string {
	var builder strings.Builder
	builder.WriteString("```\n")

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(basePath, path)
		if err != nil || relPath == "." {
			return nil
		}

		depth := strings.Count(relPath, string(filepath.Separator))
		prefix := strings.Repeat("│   ", depth)

		// Обработка директорий
		if info.IsDir() {
			// Пропускаем исключенные директории
			if utils.ShouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}

			if depth > 0 {
				prefix = strings.Repeat("│   ", depth-1) + "├── "
			}
			builder.WriteString(fmt.Sprintf("%s%s/\n", prefix, info.Name()))
		} else {
			// Пропускаем исключенные файлы
			if !utils.ShouldIncludeFile(info.Name()) {
				return nil
			}

			if depth > 0 {
				prefix = strings.Repeat("│   ", depth-1) + "├── "
			}
			builder.WriteString(fmt.Sprintf("%s%s\n", prefix, info.Name()))
		}

		return nil
	})

	builder.WriteString("```\n")
	return builder.String()
}
