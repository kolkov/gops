package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"project_scanner/internal/projecttype"
	"project_scanner/internal/utils"
)

type DocumentationGenerator struct {
	file       *os.File
	outputFile string
}

func NewDocumentationGenerator(filename string) *DocumentationGenerator {
	file, err := os.Create(filename)
	if err != nil {
		panic(fmt.Sprintf("Ошибка создания файла: %v", err))
	}
	return &DocumentationGenerator{file: file, outputFile: filename}
}

func (d *DocumentationGenerator) Close() {
	d.file.Close()
}

func (d *DocumentationGenerator) WriteHeader(
	projectName string,
	currentTime time.Time,
	isNx bool,
	projectType string,
	subType string,
) {
	d.file.WriteString(fmt.Sprintf("# Проект: %s\n\n", projectName))
	d.file.WriteString(fmt.Sprintf("**Дата генерации:** %s\n\n", currentTime.Format("2006-01-02 15:04:05")))
	d.file.WriteString("**Полнота:** Полная структура проекта и исходный код\n\n")

	if isNx {
		d.file.WriteString("**Тип:** NX Monorepo\n\n")
	} else {
		typeDescription := "Стандартный проект"
		if subType != "" {
			typeDescription = subType
		} else if projectType != "" {
			if projectType == projecttype.Go {
				typeDescription = "Go проект"
			} else if projectType == projecttype.JS {
				typeDescription = "JavaScript проект"
			} else if projectType == projecttype.Angular {
				typeDescription = "Angular проект"
			} else if projectType == projecttype.BrowserExtension {
				typeDescription = "Browser Extension"
			} else {
				typeDescription = strings.ToUpper(projectType[:1]) + projectType[1:] + " проект"
			}
		}
		d.file.WriteString(fmt.Sprintf("**Тип:** %s\n\n", typeDescription))
	}

	d.file.WriteString("## Содержание\n")
	d.file.WriteString("- [Полная структура проекта](#полная-структура-проекта)\n")
	d.file.WriteString("- [Основные модули](#основные-модули)\n")
	if isNx {
		d.file.WriteString("- [Общая структура Nx Monorepo](#общая-структура-nx-monorepo)\n")
	}
	d.file.WriteString("\n")
}

func (d *DocumentationGenerator) WriteNxStructure(projects []utils.NxProject, rootDir string) {
	d.file.WriteString("## Общая структура Nx Monorepo\n\n")
	d.file.WriteString(d.GenerateNxStructure(projects, rootDir))
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

func (d *DocumentationGenerator) GenerateNxStructure(projects []utils.NxProject, rootDir string) string {
	var builder strings.Builder
	builder.WriteString("```\nnx-monorepo/\n")

	// Сначала выводим папки проектов
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
			prefix := "├── "
			if i == len(projects)-1 {
				prefix = "└── "
			}
			builder.WriteString(fmt.Sprintf("│   %s %s\n", prefix, p.Name))
		}
	}

	// Добавляем разделительную линию после проектов, если они есть
	if len(projects) > 0 {
		builder.WriteString("│\n")
	}

	// Динамически сканируем корневую директорию на наличие файлов
	rootFiles, err := d.scanRootFiles(rootDir)
	if err != nil {
		fmt.Printf("Ошибка сканирования корневой директории: %v\n", err)
		// Продолжаем работу, даже если возникла ошибка
	}

	sort.Strings(rootFiles)

	for i, file := range rootFiles {
		isLast := i == len(rootFiles)-1
		if isLast {
			builder.WriteString(fmt.Sprintf("└── %s\n", file))
		} else {
			builder.WriteString(fmt.Sprintf("├── %s\n", file))
		}
	}

	builder.WriteString("```\n")
	return builder.String()
}

// Новая функция для сканирования корневых файлов
func (d *DocumentationGenerator) scanRootFiles(rootDir string) ([]string, error) {
	files, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	var rootFiles []string
	for _, file := range files {
		// Пропускаем директории
		if file.IsDir() {
			continue
		}

		// Проверяем, должен ли файл быть включен в структуру
		if d.shouldIncludeRootFile(file.Name()) {
			rootFiles = append(rootFiles, file.Name())
		}
	}

	return rootFiles, nil
}

// Функция проверки, должен ли файл быть включен
func (d *DocumentationGenerator) shouldIncludeRootFile(name string) bool {
	// Исключаем lock-файлы
	if name == "package-lock.json" || name == "yarn.lock" {
		return false
	}

	// Исключаем сгенерированные файлы документации
	if utils.IsGeneratedFile(name, d.outputFile) {
		return false
	}

	// Используем существующую логику фильтрации из utils
	if !utils.ShouldIncludeFile(name) {
		return false
	}

	// Дополнительные проверки для корневых файлов
	excludePatterns := []string{
		".DS_Store", "Thumbs.db", ".env", ".env.local",
		".env.development", ".env.production",
	}

	for _, pattern := range excludePatterns {
		if name == pattern {
			return false
		}
	}

	return true
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
		parent := filepath.ToSlash(filepath.Dir(node.path))
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
			return dirs[i].path < dirs[j].path
		})
		sort.Slice(files, func(i, j int) bool {
			return files[i].path < files[j].path
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
				newPrefix := prefix
				if isLast {
					newPrefix += "    "
				} else {
					newPrefix += "│   "
				}
				buildTree(child.path, newPrefix)
			} else {
				builder.WriteString(name + "\n")
			}
		}
	}

	buildTree(relRoot, "")
	builder.WriteString("```\n")
	return builder.String()
}

func shouldExcludeFromTree(relPath string, info os.FileInfo) bool {
	parts := strings.Split(relPath, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}

	excludePatterns := []string{
		".git", ".vscode", ".idea", "node_modules", "dist", "build",
		".angular", ".nx", "coverage", "__pycache__", "bin", "obj",
		"project_documentation", "project_structure.txt", "target", "out",
		"__tests__", "__snapshots__", ".next", ".nuxt", ".cache", "cypress",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(relPath, pattern) {
			return true
		}
	}

	return false
}
