package selector

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
)

// FileSelector интерактивный селектор файлов и папок
type FileSelector struct {
	rootDir string
	cfg     *model.ScanConfig
	logger  *logger.Logger
	reader  *bufio.Reader
	tree    *FileTree
	nodeMap map[int]*FileTree // индекс -> узел для быстрого поиска
}

// FileTree узел дерева файлов
type FileTree struct {
	Path     string
	Name     string
	IsDir    bool
	Children []*FileTree
	Parent   *FileTree
	Expanded bool
	Selected bool
	Partial  bool // частично выбрана (для папок с частью выбранных файлов)
}

// NewFileSelector создает новый селектор файлов
func NewFileSelector(rootDir string, cfg *model.ScanConfig, logger *logger.Logger) *FileSelector {
	return &FileSelector{
		rootDir: rootDir,
		cfg:     cfg,
		logger:  logger,
		reader:  bufio.NewReader(os.Stdin),
		nodeMap: make(map[int]*FileTree),
	}
}

// SelectFiles интерактивно выбирает файлы для включения в документацию
func (fs *FileSelector) SelectFiles() ([]string, error) {
	fmt.Println("\n📁 Interactive File Selection")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Print("Building project tree...")

	// Строим дерево проекта
	if err := fs.buildTree(); err != nil {
		return nil, fmt.Errorf("failed to build tree: %w", err)
	}
	fmt.Println(" Done!")

	if fs.tree == nil || len(fs.tree.Children) == 0 {
		return nil, fmt.Errorf("no files found in project")
	}

	fmt.Println("\nCommands:")
	fmt.Println("  [number]     - Toggle file/folder selection")
	fmt.Println("  e[number]    - Expand/collapse folder")
	fmt.Println("  all          - Select all")
	fmt.Println("  none         - Deselect all")
	fmt.Println("  done         - Finish selection")
	fmt.Println("  quit         - Cancel")

	// Интерактивный выбор
	for {
		fs.displayTree()
		fmt.Print("\nCommand: ")

		input, err := fs.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}
		input = strings.TrimSpace(input)

		switch input {
		case "done":
			selected := fs.collectSelected()
			if len(selected) == 0 {
				fmt.Println("⚠️  No files selected!")
				continue
			}
			return selected, nil
		case "quit", "q":
			return nil, fmt.Errorf("selection cancelled")
		case "all":
			fs.selectAll(fs.tree, true)
		case "none":
			fs.selectAll(fs.tree, false)
		default:
			fs.processCommand(input)
		}
	}
}

// QuickSelect быстрый выбор по паттернам
func (fs *FileSelector) QuickSelect(patterns []string) ([]string, error) {
	if err := fs.buildTree(); err != nil {
		return nil, fmt.Errorf("failed to build tree: %w", err)
	}

	for _, pattern := range patterns {
		fs.selectByPattern(fs.tree, pattern)
	}

	return fs.collectSelected(), nil
}

func (fs *FileSelector) buildTree() error {
	fs.tree = &FileTree{
		Path:     fs.rootDir,
		Name:     filepath.Base(fs.rootDir),
		IsDir:    true,
		Expanded: true,
		Selected: false,
	}

	return fs.scanDir(fs.tree)
}

func (fs *FileSelector) scanDir(parent *FileTree) error {
	entries, err := os.ReadDir(parent.Path)
	if err != nil {
		return err
	}

	// Сортируем: сначала папки, потом файлы
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		// Пропускаем системные папки
		if fs.shouldSkip(entry.Name(), entry.IsDir()) {
			continue
		}

		fullPath := filepath.Join(parent.Path, entry.Name())

		node := &FileTree{
			Path:     fullPath,
			Name:     entry.Name(),
			IsDir:    entry.IsDir(),
			Parent:   parent,
			Expanded: false,
			Selected: false,
		}

		if entry.IsDir() {
			// Рекурсивно сканируем подпапки
			if err := fs.scanDir(node); err != nil {
				fs.logger.Debug("Failed to scan directory", "path", fullPath, "error", err)
			}
		}

		parent.Children = append(parent.Children, node)
	}

	return nil
}

func (fs *FileSelector) shouldSkip(name string, isDir bool) bool {
	// Системные исключения
	systemExcludes := []string{
		".git", "node_modules", "vendor", "dist", "build",
		"__pycache__", ".idea", ".vscode", "coverage",
		".nx", ".angular", ".next", ".nuxt", "bin", "obj",
	}

	for _, exclude := range systemExcludes {
		if name == exclude {
			return true
		}
	}

	// Пропускаем сам файл конфигурации и вывода
	if strings.HasPrefix(name, "gops_config") || strings.HasPrefix(name, "project_docs") {
		return true
	}

	// Пользовательские исключения из конфига
	for _, pattern := range fs.cfg.ExcludedPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	return false
}

func (fs *FileSelector) displayTree() {
	fmt.Println("\n📂 Project Structure (selected items marked with [x]):")
	fmt.Println("──────────────────────────────────────────────────────")

	// Очищаем карту индексов
	fs.nodeMap = make(map[int]*FileTree)

	index := 1
	for i, child := range fs.tree.Children {
		index = fs.displayNode(child, "", index, i == len(fs.tree.Children)-1)
	}
}

func (fs *FileSelector) displayNode(node *FileTree, prefix string, index int, isLast bool) int {
	if node == nil {
		return index
	}

	// Сохраняем узел в карте
	fs.nodeMap[index] = node

	// Формируем строку для отображения
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	checkbox := "[ ]"
	if node.Selected {
		checkbox = "[x]"
	} else if node.Partial {
		checkbox = "[~]"
	}

	expandIcon := ""
	if node.IsDir {
		if node.Expanded {
			expandIcon = "▼ "
		} else {
			expandIcon = "▶ "
		}
	}

	fmt.Printf("%s%s %s %s%s", prefix, connector, checkbox, expandIcon, node.Name)

	if node.IsDir {
		fmt.Printf("/ (%d)", index)
	} else {
		fmt.Printf(" (%d)", index)
	}
	fmt.Println()

	currentIndex := index + 1

	// Обновляем префикс для детей
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Отображаем детей если папка раскрыта
	if node.IsDir && node.Expanded && len(node.Children) > 0 {
		for i, child := range node.Children {
			currentIndex = fs.displayNode(child, childPrefix, currentIndex, i == len(node.Children)-1)
		}
	}

	return currentIndex
}

func (fs *FileSelector) processCommand(input string) {
	// Проверяем команду expand/collapse
	if strings.HasPrefix(input, "e") {
		if num, err := strconv.Atoi(input[1:]); err == nil && num > 0 {
			fs.toggleExpand(num)
		}
		return
	}

	// Проверяем команду выбора
	if num, err := strconv.Atoi(input); err == nil && num > 0 {
		fs.toggleSelect(num)
	}
}

func (fs *FileSelector) toggleExpand(index int) {
	node, exists := fs.nodeMap[index]
	if exists && node.IsDir {
		node.Expanded = !node.Expanded
	}
}

func (fs *FileSelector) toggleSelect(index int) {
	node, exists := fs.nodeMap[index]
	if !exists {
		return
	}

	node.Selected = !node.Selected

	// Если это папка, выбираем/снимаем выбор со всех вложенных файлов
	if node.IsDir {
		fs.selectAll(node, node.Selected)
	}

	// Обновляем состояние родительских папок
	fs.updateParentStates(node)
}

func (fs *FileSelector) selectAll(node *FileTree, selected bool) {
	if node == nil {
		return
	}

	node.Selected = selected
	node.Partial = false

	for _, child := range node.Children {
		fs.selectAll(child, selected)
	}
}

func (fs *FileSelector) updateParentStates(node *FileTree) {
	parent := node.Parent
	if parent == nil {
		return
	}

	selectedCount := 0
	partialCount := 0
	totalCount := len(parent.Children)

	// Подсчитываем состояния детей
	for _, child := range parent.Children {
		if child.Selected {
			selectedCount++
		} else if child.IsDir && child.Partial {
			partialCount++
		}
	}

	// Обновляем состояние родителя на основе состояний детей
	if selectedCount == totalCount {
		// Все дети выбраны - родитель полностью выбран
		parent.Selected = true
		parent.Partial = false
	} else if selectedCount > 0 || partialCount > 0 {
		// Некоторые дети выбраны или частично выбраны - родитель частично выбран
		parent.Selected = false
		parent.Partial = true
	} else {
		// Никто не выбран - родитель не выбран
		parent.Selected = false
		parent.Partial = false
	}

	// Рекурсивно обновляем вышестоящие папки
	fs.updateParentStates(parent)
}

func (fs *FileSelector) collectSelected() []string {
	var selected []string
	fs.collectFromNode(fs.tree, &selected)
	return selected
}

func (fs *FileSelector) collectFromNode(node *FileTree, selected *[]string) {
	if node == nil {
		return
	}

	// Добавляем файл если он выбран
	if !node.IsDir && node.Selected {
		relPath, _ := filepath.Rel(fs.rootDir, node.Path)
		*selected = append(*selected, relPath)
	}

	// Рекурсивно обходим детей
	for _, child := range node.Children {
		fs.collectFromNode(child, selected)
	}
}

func (fs *FileSelector) selectByPattern(node *FileTree, pattern string) {
	if node == nil {
		return
	}

	relPath, _ := filepath.Rel(fs.rootDir, node.Path)
	// Нормализуем путь для корректной работы на Windows
	relPath = filepath.ToSlash(relPath)
	pattern = filepath.ToSlash(pattern)

	// Обрабатываем паттерны с **
	if strings.Contains(pattern, "**") {
		if matchDoublestar(pattern, relPath) && !node.IsDir {
			node.Selected = true
			fs.updateParentStates(node)
		}
	} else {
		// Для обычных паттернов используем matchSimplePattern
		if matchSimplePattern(pattern, relPath) && !node.IsDir {
			node.Selected = true
			fs.updateParentStates(node)
		}
	}

	for _, child := range node.Children {
		fs.selectByPattern(child, pattern)
	}
}

// matchSimplePattern проверяет соответствие пути простому паттерну (без **)
// Учитывает структуру директорий: * не может соответствовать /
func matchSimplePattern(pattern, path string) bool {
	// Нормализуем пути
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Разбиваем паттерн и путь на сегменты
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	// Количество сегментов должно совпадать
	if len(patternParts) != len(pathParts) {
		return false
	}

	// Проверяем каждый сегмент
	for i := 0; i < len(patternParts); i++ {
		// Для каждого сегмента используем filepath.Match
		matched, err := filepath.Match(patternParts[i], pathParts[i])
		if err != nil || !matched {
			return false
		}
	}

	return true
}

// matchDoublestar - вспомогательная функция для обработки паттернов с **
// Копирует логику из model.matchDoublestar для консистентности
func matchDoublestar(pattern, path string) bool {
	// Нормализуем пути
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Если паттерн не содержит **, это ошибка использования функции
	// но для обратной совместимости обрабатываем как обычный паттерн
	if !strings.Contains(pattern, "**") {
		// Для простых паттернов проверяем точное соответствие структуре
		// Например, "*.md" должен соответствовать только файлам в корне
		// "docs/*.md" - только файлам непосредственно в docs/

		// Если паттерн содержит слеш, проверяем точное соответствие пути
		if strings.Contains(pattern, "/") {
			matched, _ := filepath.Match(pattern, path)
			return matched
		} else {
			// Паттерн без слеша (например "*.md") должен соответствовать только файлам в корне
			if strings.Contains(path, "/") {
				return false // файл не в корне
			}
			matched, _ := filepath.Match(pattern, path)
			return matched
		}
	}

	// Обработка паттернов вида **/*.ext
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]

		// Проверяем соответствие суффиксу на любом уровне
		// Проверяем имя файла и все возможные подпути
		parts := strings.Split(path, "/")

		// Проверяем каждый возможный суффикс пути
		for i := 0; i < len(parts); i++ {
			subpath := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(suffix, subpath); matched {
				return true
			}
		}
		// Также проверяем просто имя файла
		filename := parts[len(parts)-1]
		if matched, _ := filepath.Match(suffix, filename); matched {
			return true
		}
		return false
	}

	// Обработка паттернов вида path/**
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}

	// Обработка паттернов вида prefix/**/suffix
	if idx := strings.Index(pattern, "/**/"); idx >= 0 {
		prefix := pattern[:idx]
		suffix := pattern[idx+4:]

		// Проверяем префикс
		if path == prefix {
			return false // prefix/**/ требует хотя бы одну директорию после prefix
		}

		// Путь должен начинаться с prefix/
		if !strings.HasPrefix(path, prefix+"/") {
			return false
		}

		// Остаток пути после prefix/
		remainder := path[len(prefix)+1:]

		// Для паттерна вида src/**/test/*.go
		// ** в середине означает "одна или более директорий"
		// Проверяем все возможные подпути
		parts := strings.Split(remainder, "/")

		// Проверяем соответствие суффиксу, начиная с уровня 1 (пропускаем 0)
		// так как ** должен соответствовать хотя бы одной директории
		for i := 1; i <= len(parts); i++ {
			subpath := strings.Join(parts[i-1:], "/")
			if subpath != "" {
				if matched, _ := filepath.Match(suffix, subpath); matched {
					return true
				}
			}
		}

		// Специальная обработка для паттернов с путями в суффиксе
		if strings.Contains(suffix, "/") {
			suffixParts := strings.Split(suffix, "/")
			// Начинаем с i=1, чтобы между prefix и suffix была хотя бы одна директория
			for i := 1; i <= len(parts)-len(suffixParts)+1; i++ {
				// Проверяем, что перед suffix есть хотя бы одна директория
				if i < 1 {
					continue
				}
				match := true
				for j, suffixPart := range suffixParts {
					if i-1+j >= len(parts) {
						match = false
						break
					}
					if strings.Contains(suffixPart, "*") || strings.Contains(suffixPart, "?") {
						if matched, _ := filepath.Match(suffixPart, parts[i-1+j]); !matched {
							match = false
							break
						}
					} else {
						if suffixPart != parts[i-1+j] {
							match = false
							break
						}
					}
				}
				if match {
					return true
				}
			}
		}
	}

	return false
}
