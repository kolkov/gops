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
	projectName := filepath.Base(basePath)
	builder.WriteString("```\n" + projectName + "\n")

	type treeNode struct {
		path  string
		isDir bool
	}
	var nodes []treeNode

	// Собираем все элементы (директории и файлы)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Пропускаем корневую директорию
		if path == root {
			return nil
		}

		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return nil
		}

		// Пропускаем нежелательные элементы
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

	// Сортируем узлы по полному пути
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].path < nodes[j].path
	})

	// Строим карту родительских директорий
	childrenMap := make(map[string][]treeNode)
	for _, node := range nodes {
		parent := filepath.Dir(node.path)
		childrenMap[parent] = append(childrenMap[parent], node)
	}

	// Рекурсивная функция для построения дерева
	var buildTree func(parent string, prefix string)
	buildTree = func(parent string, prefix string) {
		children := childrenMap[parent]

		// Разделяем на директории и файлы
		var dirs []treeNode
		var files []treeNode
		for _, child := range children {
			if child.isDir {
				dirs = append(dirs, child)
			} else {
				files = append(files, child)
			}
		}

		// Сортируем директории и файлы отдельно
		sort.Slice(dirs, func(i, j int) bool {
			return dirs[i].path < dirs[j].path
		})
		sort.Slice(files, func(i, j int) bool {
			return files[i].path < files[j].path
		})

		// Объединяем: сначала директории, потом файлы
		sortedChildren := append(dirs, files...)

		for i, child := range sortedChildren {
			isLast := i == len(sortedChildren)-1
			name := filepath.Base(child.path)

			// Текущая строка
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

			// Рекурсия для поддиректорий
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

	// Начинаем с корневой директории
	buildTree(".", "")

	builder.WriteString("```\n")
	return builder.String()
}

// shouldExcludeFromTree определяет, нужно ли исключить элемент из дерева
func shouldExcludeFromTree(relPath string, info os.FileInfo) bool {
	// Пропускаем .git и его содержимое
	if strings.HasPrefix(relPath, ".git") {
		return true
	}

	// Пропускаем сгенерированные файлы документации
	if strings.HasPrefix(info.Name(), "project_documentation") &&
		strings.HasSuffix(info.Name(), ".md") {
		return true
	}

	// Пропускаем бинарные файлы git
	if strings.Contains(relPath, "objects") && len(info.Name()) == 38 {
		return true
	}

	// Пропускаем файлы git
	gitFiles := []string{"HEAD", "COMMIT_EDITMSG", "config", "description", "index"}
	for _, file := range gitFiles {
		if info.Name() == file {
			return true
		}
	}

	// Пропускаем скрытые файлы/папки (начинающиеся с точки)
	if strings.HasPrefix(info.Name(), ".") {
		return true
	}

	// Пропускаем системные папки
	skipDirs := []string{"hooks", "info", "logs", "refs", "objects", "pack", "smartgit", "node_modules"}
	for _, dir := range skipDirs {
		if info.IsDir() && info.Name() == dir {
			return true
		}
	}

	// Пропускаем IDE-специфичные файлы
	ideFiles := []string{".idea", ".vscode", "workspace.xml", "modules.xml", "*.iml"}
	for _, file := range ideFiles {
		if strings.Contains(relPath, file) {
			return true
		}
	}

	return false
}
