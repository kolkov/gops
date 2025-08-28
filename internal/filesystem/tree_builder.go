package filesystem

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/kolkov/gops/internal/model"
)

type TreeBuilder struct {
	rootDir string
	cfg     *model.ScanConfig
	mu      sync.Mutex
}

func NewTreeBuilder(rootDir string, cfg *model.ScanConfig) *TreeBuilder {
	return &TreeBuilder{
		rootDir: rootDir,
		cfg:     cfg,
	}
}

// Build (старая версия) - сохраняем для обратной совместимости
func (tb *TreeBuilder) Build() (string, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	var structure strings.Builder
	structure.WriteString(".\n")

	rootNode := &treeNode{
		name:  ".",
		isDir: true,
	}

	if err := tb.buildTree(rootNode, tb.rootDir, 0); err != nil {
		return "", err
	}

	for i, child := range rootNode.children {
		tb.renderTree(child, &structure, "", i == len(rootNode.children)-1)
	}

	return structure.String(), nil
}

// BuildStructure (новая версия) - единое сканирование для структуры и содержимого
func (tb *TreeBuilder) BuildStructure() (*model.ProjectStructure, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	structure := &model.ProjectStructure{
		Root: &model.FileNode{
			Path:  tb.rootDir,
			Name:  ".",
			IsDir: true,
		},
	}

	// Единый обход для сбора структуры и файлов
	err := tb.buildTreeAndCollectFiles(structure.Root, tb.rootDir, 0, structure)
	if err != nil {
		return nil, err
	}

	return structure, nil
}

func (tb *TreeBuilder) buildTreeAndCollectFiles(parent *model.FileNode, path string, depth int, structure *model.ProjectStructure) error {
	if depth > 20 {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Сортируем: сначала директории, потом файлы
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() == entries[j].IsDir() {
			return entries[i].Name() < entries[j].Name()
		}
		return entries[i].IsDir()
	})

	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())

		if tb.shouldSkip(childPath, entry) {
			continue
		}

		childNode := &model.FileNode{
			Path:  childPath,
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
		}

		if !entry.IsDir() {
			// Обрабатываем файл сразу
			projectFile := tb.processFile(childPath, tb.rootDir)
			if projectFile != nil {
				childNode.FileInfo = projectFile
				structure.Files = append(structure.Files, projectFile)
				structure.AllPaths = append(structure.AllPaths, projectFile.Path)
			}
		}

		parent.Children = append(parent.Children, childNode)

		if entry.IsDir() {
			if err := tb.buildTreeAndCollectFiles(childNode, childPath, depth+1, structure); err != nil {
				return err
			}
		}
	}

	return nil
}

func (tb *TreeBuilder) processFile(path, rootDir string) *model.ProjectFile {
	relPath, _ := filepath.Rel(rootDir, path)
	file := &model.ProjectFile{
		Path: relPath,
		Lang: model.GetFileLanguage(path),
	}

	// Применяем все проверки исключений
	if ShouldSkipFile(filepath.Base(path), tb.cfg.OutputConfigFilename, tb.cfg.OutputFilename, tb.cfg.ExcludedPatterns, tb.cfg.ImportantFiles) {
		return nil
	}

	// Проверяем тип файла
	if shouldSkipByType(path, tb.cfg) {
		file.Skipped = true
		return file
	}

	// Читаем содержимое файла
	content, err := ReadFile(path, tb.cfg.MaxFileSize)
	if err != nil {
		file.Skipped = true
		return file
	}

	file.Content = string(content)
	return file
}

func (tb *TreeBuilder) RenderTreeFromStructure(root *model.FileNode) string {
	var structure strings.Builder
	structure.WriteString(".\n")

	for i, child := range root.Children {
		tb.renderStructureNode(child, &structure, "", i == len(root.Children)-1)
	}

	return structure.String()
}

func (tb *TreeBuilder) renderStructureNode(node *model.FileNode, builder *strings.Builder, prefix string, isLast bool) {
	builder.WriteString(prefix)
	if isLast {
		builder.WriteString("└── ")
	} else {
		builder.WriteString("├── ")
	}

	builder.WriteString(node.Name)
	if node.IsDir {
		builder.WriteString("/")
	}
	builder.WriteString("\n")

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	for i, child := range node.Children {
		tb.renderStructureNode(child, builder, childPrefix, i == len(node.Children)-1)
	}
}

func (tb *TreeBuilder) renderTree(node *treeNode, builder *strings.Builder, prefix string, isLast bool) {
	builder.WriteString(prefix)
	if isLast {
		builder.WriteString("└── ")
	} else {
		builder.WriteString("├── ")
	}

	builder.WriteString(node.name)
	if node.isDir {
		builder.WriteString("/")
	}
	builder.WriteString("\n")

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	for i, child := range node.children {
		tb.renderTree(child, builder, childPrefix, i == len(node.children)-1)
	}
}

func (tb *TreeBuilder) buildTree(parent *treeNode, path string, depth int) error {
	if depth > 20 {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Сортируем: сначала директории, потом файлы
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() == entries[j].IsDir() {
			return entries[i].Name() < entries[j].Name()
		}
		return entries[i].IsDir() // Директории всегда перед файлами
	})

	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())

		if tb.shouldSkip(childPath, entry) {
			continue
		}

		childNode := &treeNode{
			name:  entry.Name(),
			isDir: entry.IsDir(),
		}

		if entry.IsDir() {
			if err := tb.buildTree(childNode, childPath, depth+1); err != nil {
				return err
			}
		}

		parent.children = append(parent.children, childNode)
	}

	return nil
}

func (tb *TreeBuilder) shouldSkip(path string, entry os.DirEntry) bool {
	name := entry.Name()

	// Всегда пропускать конфигурационные файлы gops
	if strings.HasPrefix(name, "gops_config") || strings.Contains(strings.ToLower(name), "gops_config") {
		return true
	}

	// Проверка важных файлов (переопределяют исключения)
	for _, important := range tb.cfg.ImportantFiles {
		if name == important {
			return false
		}
	}

	// Полный список системных исключений
	systemExcludes := []string{
		"node_modules", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"go.sum", ".git", ".idea", ".vscode", "dist", "build", "out", "bin", "obj",
		"__pycache__", ".pytest_cache", "coverage", ".nyc_output", ".angular", ".nx", ".next", ".nuxt",
		"__tests__", "__snapshots__", "e2e", "*.log", "*.tmp", "*.bak",
		"*.png", "*.jpg", "*.jpeg", "*.gif", "*.ico", "*.svg", "*.bmp", "*.webp",
		"project_structure.txt",
	}

	// Проверка системных исключений
	for _, pattern := range systemExcludes {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	// Проверка пользовательских исключений из конфига
	for _, pattern := range tb.cfg.ExcludedPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	// Убрать проверку на project_docs*.md и project_documentation_*.md
	// так как они не должны влиять на обычные markdown-файлы

	// Для файлов применяем дополнительные проверки на основе типа
	if !entry.IsDir() {
		ext := strings.ToLower(filepath.Ext(path))
		fileName := strings.ToLower(name)

		switch {
		case !tb.cfg.IncludeStyles && (ext == ".css" || ext == ".scss" || ext == ".sass" || ext == ".less"):
			return true
		case !tb.cfg.IncludeMarkup && (ext == ".html" || ext == ".htm"):
			return true
		case !tb.cfg.IncludeConfigs && (ext == ".json" || ext == ".yaml" || ext == ".yml" || strings.Contains(fileName, "config")):
			return true
		case !tb.cfg.IncludeTests && (strings.Contains(fileName, ".spec.") || strings.Contains(fileName, ".test.")):
			return true
		case !tb.cfg.IncludeDocs && (ext == ".md" || ext == ".markdown"):
			return true
		}
	}

	return false
}

type treeNode struct {
	name     string
	isDir    bool
	children []*treeNode
}
