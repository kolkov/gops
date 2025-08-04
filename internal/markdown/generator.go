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

	// Автоматическое оглавление
	d.file.WriteString("## Содержание\n")
	d.file.WriteString("- [Полная структура проекта](#полная-структура-проекта)\n")
	d.file.WriteString("- [Основные модули](#основные-модули)\n")
	if isNx {
		d.file.WriteString("- [Общая структура Nx Monorepo](#общая-структура-nx-monorepo)\n")
	}
	d.file.WriteString("\n")
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

func (d *DocumentationGenerator) WriteFileSection(filePath string, content []byte, lang string, skipContent bool) {
	d.file.WriteString(fmt.Sprintf("### %s\n\n", filePath))

	if skipContent {
		d.file.WriteString("```\n[Содержимое файла пропущено]\n```\n\n")
		d.file.WriteString("_Файл был пропущен по настройкам документации. Для просмотра запросите у пользователя._\n\n")
		return
	}

	d.file.WriteString(fmt.Sprintf("```%s\n", lang))
	d.file.Write(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		d.file.WriteString("\n")
	}
	d.file.WriteString("```\n\n")
}

func (d *DocumentationGenerator) WriteModulesHeader() {
	d.file.WriteString("## Основные модули\n\n")
	d.file.WriteString("В этом разделе представлены ключевые файлы проекта с их исходным кодом:\n\n")
}

func (d *DocumentationGenerator) WriteSubHeader(text string) {
	d.file.WriteString(fmt.Sprintf("### %s\n\n", text))
}

func (d *DocumentationGenerator) WriteSkippedFilesNote() {
	d.file.WriteString("\n## Примечание о пропущенных файлах\n\n")
	d.file.WriteString("По настройкам документации были пропущены:\n")
	d.file.WriteString("- Файлы стилей (CSS, SCSS)\n")
	d.file.WriteString("- Файлы разметки (HTML)\n")
	d.file.WriteString("- Конфигурационные файлы\n")
	d.file.WriteString("- Тестовые файлы\n\n")
	d.file.WriteString("Если вам необходимо содержание этих файлов для анализа или исправления кода, запросите у пользователя конкретные файлы по их путям.\n\n")
	d.file.WriteString("**Не предполагайте содержимое пропущенных файлов!** Всегда запрашивайте точные данные у пользователя.\n")
}

func GenerateOutputFilename(currentTime time.Time) string {
	return fmt.Sprintf("project_documentation_%s.md", currentTime.Format("20060102_150405"))
}

func (d *DocumentationGenerator) GenerateNxStructure(projects []utils.NxProject) string {
	var builder strings.Builder
	builder.WriteString("```\nnx-monorepo/\n")

	projectsByType := make(map[string][]utils.NxProject)
	for _, p := range projects {
		projectsByType[p.Type] = append(projectsByType[p.Type], p)
	}

	var types []string
	for t := range projectsByType {
		types = append(types, t)
	}
	sort.Strings(types)

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

func (d *DocumentationGenerator) GenerateProjectTree(root, basePath string) string {
	var builder strings.Builder
	relRoot, err := filepath.Rel(basePath, root)
	if err != nil {
		relRoot = "."
	} else {
		relRoot = filepath.ToSlash(relRoot)
	}
	builder.WriteString("```\n" + relRoot + "\n")

	type treeNode struct {
		path  string
		isDir bool
	}
	var nodes []treeNode

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if path == root {
			return nil
		}

		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		if shouldExcludeFromTree(relPath, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		nodes = append(nodes, treeNode{
			path:  relPath,
			isDir: info.IsDir(),
		})

		return nil
	})

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].path < nodes[j].path
	})

	childrenMap := make(map[string][]treeNode)
	for _, node := range nodes {
		parent := filepath.Dir(node.path)
		childrenMap[parent] = append(childrenMap[parent], node)
	}

	var buildTree func(parent string, prefix string)
	buildTree = func(parent string, prefix string) {
		children, exists := childrenMap[parent]
		if !exists {
			return
		}

		var dirs []treeNode
		var files []treeNode
		for _, child := range children {
			if child.isDir {
				dirs = append(dirs, child)
			} else {
				files = append(files, child)
			}
		}

		sort.Slice(dirs, func(i, j int) bool {
			return filepath.Base(dirs[i].path) < filepath.Base(dirs[j].path)
		})
		sort.Slice(files, func(i, j int) bool {
			return filepath.Base(files[i].path) < filepath.Base(files[j].path)
		})

		sortedChildren := append(dirs, files...)

		for i, child := range sortedChildren {
			isLast := i == len(sortedChildren)-1
			name := filepath.Base(child.path)

			builder.WriteString(prefix)
			if isLast {
				builder.WriteString("└── ")
			} else {
				builder.WriteString("├── ")
			}

			if child.isDir {
				builder.WriteString(name + "/\n")
			} else {
				builder.WriteString(name + "\n")
			}

			if child.isDir {
				newPrefix := prefix
				if isLast {
					newPrefix += "    "
				} else {
					newPrefix += "│   "
				}
				buildTree(child.path, newPrefix)
			}
		}
	}

	buildTree(".", "")
	builder.WriteString("```\n")
	return builder.String()
}

func shouldExcludeFromTree(relPath string, info os.FileInfo) bool {
	// Пропускаем системные и временные файлы
	excludePatterns := []string{
		".git", ".vscode", ".idea", "node_modules", "dist", "build",
		".angular", ".nx", "coverage", "__pycache__", "bin", "obj",
		"project_documentation", "project_structure.txt",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(relPath, pattern) {
			return true
		}
	}

	// Пропускаем скрытые файлы/папки
	if strings.HasPrefix(filepath.Base(relPath), ".") {
		return true
	}

	return false
}
