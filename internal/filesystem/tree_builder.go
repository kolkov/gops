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

	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())
		childNode := &treeNode{
			name:  entry.Name(),
			isDir: entry.IsDir(),
		}

		if tb.shouldSkip(childPath, entry) {
			continue
		}

		if entry.IsDir() {
			if err := tb.buildTree(childNode, childPath, depth+1); err != nil {
				return err
			}
		}

		parent.children = append(parent.children, childNode)
	}

	sort.Slice(parent.children, func(i, j int) bool {
		if parent.children[i].isDir == parent.children[j].isDir {
			return parent.children[i].name < parent.children[j].name
		}
		return parent.children[i].isDir
	})

	return nil
}

func (tb *TreeBuilder) shouldSkip(path string, entry os.DirEntry) bool {
	name := entry.Name()

	// Системные исключения (всегда)
	systemExcludes := []string{".idea", ".vscode", ".git", "node_modules"}
	for _, excl := range systemExcludes {
		if name == excl {
			return true
		}
	}

	// Проверка по шаблонам исключений
	for _, pattern := range tb.cfg.ExcludedPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	// Исключение файлов документации
	if strings.HasPrefix(name, "project_docs") && strings.HasSuffix(name, ".md") {
		return true
	}
	if strings.HasPrefix(name, "project_documentation_") && strings.HasSuffix(name, ".md") {
		return true
	}
	if name == "project_docs.md" {
		return true
	}

	// Исключение по типу файла (только для файлов)
	if !entry.IsDir() {
		// Проверяем настройки сканера
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
		}
	}

	return false
}

type treeNode struct {
	name     string
	isDir    bool
	children []*treeNode
}
