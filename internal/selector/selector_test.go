package selector

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSelector_BuildTree(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(logger.InfoLevel)

	// Создаем тестовую структуру проекта
	createTestProjectStructure(t, tmpDir)

	cfg := &model.ScanConfig{
		MaxFileSize:      1024 * 1024,
		ExcludedPatterns: []string{"*.tmp"},
	}

	fs := NewFileSelector(tmpDir, cfg, log)
	err := fs.buildTree()
	require.NoError(t, err)

	assert.NotNil(t, fs.tree)
	assert.True(t, fs.tree.IsDir)
	assert.Equal(t, filepath.Base(tmpDir), fs.tree.Name)
	assert.Greater(t, len(fs.tree.Children), 0)

	// Проверяем что системные папки исключены
	for _, child := range fs.tree.Children {
		assert.NotEqual(t, "node_modules", child.Name)
		assert.NotEqual(t, ".git", child.Name)
	}
}

func TestFileSelector_QuickSelect(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(logger.InfoLevel)

	createTestProjectStructure(t, tmpDir)

	cfg := &model.ScanConfig{
		MaxFileSize: 1024 * 1024,
	}

	fs := NewFileSelector(tmpDir, cfg, log)

	t.Run("SelectByPattern", func(t *testing.T) {
		patterns := []string{"src/*.go", "*.md"}
		selected, err := fs.QuickSelect(patterns)
		require.NoError(t, err)

		assert.Contains(t, selected, filepath.Join("src", "main.go"))
		assert.Contains(t, selected, filepath.Join("src", "util.go"))
		assert.Contains(t, selected, "README.md")
		assert.NotContains(t, selected, filepath.Join("test", "test.go"))
	})

	t.Run("EmptyPattern", func(t *testing.T) {
		selected, err := fs.QuickSelect([]string{})
		require.NoError(t, err)
		assert.Empty(t, selected)
	})
}

func TestFileSelector_ShouldSkip(t *testing.T) {
	log := logger.New(logger.InfoLevel)
	cfg := &model.ScanConfig{
		ExcludedPatterns: []string{"*.log", "temp_*"},
	}

	fs := NewFileSelector(".", cfg, log)

	tests := []struct {
		name     string
		isDir    bool
		expected bool
	}{
		{"node_modules", true, true},
		{".git", true, true},
		{"vendor", true, true},
		{"gops_config.yaml", false, true},
		{"project_docs.md", false, true},
		{"error.log", false, true},
		{"temp_file.txt", false, true},
		{"main.go", false, false},
		{"src", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.shouldSkip(tt.name, tt.isDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileSelector_SelectAll(t *testing.T) {
	log := logger.New(logger.InfoLevel)
	cfg := &model.ScanConfig{}
	fs := NewFileSelector(".", cfg, log)

	// Создаем тестовое дерево
	root := &FileTree{
		Name:  "root",
		IsDir: true,
		Children: []*FileTree{
			{
				Name:  "file1.go",
				IsDir: false,
			},
			{
				Name:  "dir1",
				IsDir: true,
				Children: []*FileTree{
					{
						Name:  "file2.go",
						IsDir: false,
					},
				},
			},
		},
	}

	// Устанавливаем родительские связи
	for _, child := range root.Children {
		child.Parent = root
		if child.IsDir {
			for _, grandchild := range child.Children {
				grandchild.Parent = child
			}
		}
	}

	// Выбираем все
	fs.selectAll(root, true)
	assert.True(t, root.Selected)
	assert.True(t, root.Children[0].Selected)
	assert.True(t, root.Children[1].Selected)
	assert.True(t, root.Children[1].Children[0].Selected)

	// Снимаем выбор со всех
	fs.selectAll(root, false)
	assert.False(t, root.Selected)
	assert.False(t, root.Children[0].Selected)
	assert.False(t, root.Children[1].Selected)
	assert.False(t, root.Children[1].Children[0].Selected)
}

func TestFileSelector_UpdateParentStates(t *testing.T) {
	log := logger.New(logger.InfoLevel)
	cfg := &model.ScanConfig{}
	fs := NewFileSelector(".", cfg, log)

	// Создаем тестовое дерево
	root := &FileTree{
		Name:  "root",
		IsDir: true,
	}

	dir1 := &FileTree{
		Name:   "dir1",
		IsDir:  true,
		Parent: root,
	}

	file1 := &FileTree{
		Name:   "file1.go",
		IsDir:  false,
		Parent: dir1,
	}

	file2 := &FileTree{
		Name:   "file2.go",
		IsDir:  false,
		Parent: dir1,
	}

	dir1.Children = []*FileTree{file1, file2}
	root.Children = []*FileTree{dir1}

	t.Run("AllSelected", func(t *testing.T) {
		// Сбрасываем состояние
		file1.Selected = false
		file2.Selected = false
		dir1.Selected = false
		dir1.Partial = false
		root.Selected = false
		root.Partial = false

		// Выбираем все файлы
		file1.Selected = true
		file2.Selected = true
		fs.updateParentStates(file1)

		assert.True(t, dir1.Selected)
		assert.False(t, dir1.Partial)
		assert.True(t, root.Selected)
		assert.False(t, root.Partial)
	})

	t.Run("PartialSelected", func(t *testing.T) {
		// Сбрасываем состояние
		file1.Selected = false
		file2.Selected = false
		dir1.Selected = false
		dir1.Partial = false
		root.Selected = false
		root.Partial = false

		// Выбираем только один файл
		file1.Selected = true
		file2.Selected = false
		fs.updateParentStates(file1)

		// Когда выбран только один файл из двух, папка должна быть в частичном состоянии
		assert.False(t, dir1.Selected)
		assert.True(t, dir1.Partial)
		assert.False(t, root.Selected)
		assert.True(t, root.Partial)
	})

	t.Run("NoneSelected", func(t *testing.T) {
		// Сбрасываем состояние
		file1.Selected = false
		file2.Selected = false
		dir1.Selected = false
		dir1.Partial = false
		root.Selected = false
		root.Partial = false

		fs.updateParentStates(file1)

		assert.False(t, dir1.Selected)
		assert.False(t, dir1.Partial)
		assert.False(t, root.Selected)
		assert.False(t, root.Partial)
	})
}

func TestFileSelector_ProcessCommand(t *testing.T) {
	log := logger.New(logger.InfoLevel)
	cfg := &model.ScanConfig{}
	fs := NewFileSelector(".", cfg, log)

	// Создаем тестовое дерево с индексами
	fs.tree = &FileTree{
		Name:  "root",
		IsDir: true,
		Children: []*FileTree{
			{Name: "dir1", IsDir: true},
			{Name: "file1.go", IsDir: false},
		},
	}

	fs.nodeMap = map[int]*FileTree{
		1: fs.tree.Children[0],
		2: fs.tree.Children[1],
	}

	t.Run("ToggleSelect", func(t *testing.T) {
		fs.processCommand("2")
		assert.True(t, fs.tree.Children[1].Selected)

		fs.processCommand("2")
		assert.False(t, fs.tree.Children[1].Selected)
	})

	t.Run("ToggleExpand", func(t *testing.T) {
		fs.processCommand("e1")
		assert.True(t, fs.tree.Children[0].Expanded)

		fs.processCommand("e1")
		assert.False(t, fs.tree.Children[0].Expanded)
	})

	t.Run("InvalidCommand", func(t *testing.T) {
		// Не должно вызывать панику
		fs.processCommand("invalid")
		fs.processCommand("999")
		fs.processCommand("e999")
	})
}

func TestFileSelector_InteractiveCommands(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(logger.InfoLevel)

	createTestProjectStructure(t, tmpDir)

	cfg := &model.ScanConfig{
		MaxFileSize: 1024 * 1024,
	}

	t.Run("SelectAllCommand", func(t *testing.T) {
		// Симулируем команды: all, done
		input := "all\ndone\n"
		fs := &FileSelector{
			rootDir: tmpDir,
			cfg:     cfg,
			logger:  log,
			reader:  bufio.NewReader(strings.NewReader(input)),
			nodeMap: make(map[int]*FileTree),
		}

		selected, err := fs.SelectFiles()
		require.NoError(t, err)
		assert.Greater(t, len(selected), 0)
	})

	t.Run("QuitCommand", func(t *testing.T) {
		// Симулируем команду quit
		input := "quit\n"
		fs := &FileSelector{
			rootDir: tmpDir,
			cfg:     cfg,
			logger:  log,
			reader:  bufio.NewReader(strings.NewReader(input)),
			nodeMap: make(map[int]*FileTree),
		}

		_, err := fs.SelectFiles()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("NoneCommand", func(t *testing.T) {
		// Симулируем команды: all, none, done (должен требовать выбор файлов)
		input := "all\nnone\nall\ndone\n"
		fs := &FileSelector{
			rootDir: tmpDir,
			cfg:     cfg,
			logger:  log,
			reader:  bufio.NewReader(strings.NewReader(input)),
			nodeMap: make(map[int]*FileTree),
		}

		selected, err := fs.SelectFiles()
		require.NoError(t, err)
		assert.Greater(t, len(selected), 0)
	})
}

// Вспомогательная функция для создания тестовой структуры проекта
func createTestProjectStructure(t *testing.T, rootDir string) {
	dirs := []string{
		"src",
		"test",
		"docs",
		"node_modules",
		".git",
	}

	files := map[string]string{
		"README.md":                 "# Test Project",
		"src/main.go":               "package main",
		"src/util.go":               "package main",
		"test/test.go":              "package test",
		"docs/api.md":               "# API",
		"node_modules/package.json": "{}",
		".git/config":               "[core]",
		"temp.tmp":                  "temp",
	}

	for _, dir := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(rootDir, dir), 0755))
	}

	for path, content := range files {
		fullPath := filepath.Join(rootDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}
}
