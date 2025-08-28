package filesystem

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kolkov/gops/internal/model"
)

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string { return m.name }
func (m mockDirEntry) IsDir() bool  { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode {
	info, _ := m.Info()
	return info.Mode()
}
func (m mockDirEntry) Info() (fs.FileInfo, error) {
	return mockFileInfo{name: m.name, isDir: m.isDir}, nil
}

type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() fs.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func TestTreeBuilder_Build(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем структуру проекта
	dirs := []string{
		"dir1",
		"dir1/subdir",
		"dir2",
		".git",
		"node_modules/react",
		"dist",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	files := []string{
		"dir1/file1.txt",
		"dir1/subdir/file2.txt",
		"dir2/file3.txt",
		".git/config",
		"node_modules/react/index.js",
		"dist/app.js",
		"project_docs.md",
		"root_file.txt",
		"important.config",
	}

	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Конфиг с исключениями
	cfg := &model.ScanConfig{
		ExcludedPatterns: []string{"project_docs*.md"},
		ImportantFiles:   []string{"important.config"},
		IncludeConfigs:   false,
		IncludeTests:     false,
	}

	t.Run("FullStructure", func(t *testing.T) {
		tb := NewTreeBuilder(tmpDir, cfg)
		result, err := tb.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Ожидаемая структура с правильным порядком: директории -> файлы
		expected := strings.TrimSpace(`
.
├── dir1/
│   ├── subdir/
│   │   └── file2.txt
│   └── file1.txt
├── dir2/
│   └── file3.txt
├── important.config
└── root_file.txt
`)

		// Нормализуем пробельные символы для сравнения
		normalizedResult := strings.ReplaceAll(result, "\r\n", "\n")
		normalizedExpected := strings.ReplaceAll(expected, "\r\n", "\n")

		if !strings.Contains(normalizedResult, normalizedExpected) {
			t.Errorf("Ожидалось:\n%s\n\nПолучено:\n%s", normalizedExpected, normalizedResult)
		}
	})

	t.Run("Exclusions", func(t *testing.T) {
		tb := NewTreeBuilder(tmpDir, cfg)
		result, err := tb.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Проверяем, что исключенные элементы отсутствуют
		excluded := []string{
			".git",
			"node_modules",
			"dist",
			"project_docs.md",
		}

		for _, item := range excluded {
			if strings.Contains(result, item) {
				t.Errorf("Исключенный элемент '%s' присутствует в выводе", item)
			}
		}
	})

	t.Run("ImportantFiles", func(t *testing.T) {
		tb := NewTreeBuilder(tmpDir, cfg)
		result, err := tb.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if !strings.Contains(result, "important.config") {
			t.Error("Важный файл 'important.config' отсутствует в выводе")
		}
	})

	t.Run("DirectoryOrder", func(t *testing.T) {
		// Создаем дополнительные директории
		dirs := []string{"b_dir", "a_dir", "c_dir"}
		for _, dir := range dirs {
			if err := os.Mkdir(filepath.Join(tmpDir, dir), 0755); err != nil {
				t.Fatal(err)
			}
		}

		tb := NewTreeBuilder(tmpDir, cfg)
		result, err := tb.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Проверяем порядок директорий
		expectedOrder := []string{
			"a_dir/",
			"b_dir/",
			"c_dir/",
			"dir1/",
			"dir2/",
		}

		lastIndex := -1
		for _, dir := range expectedOrder {
			idx := strings.Index(result, dir)
			if idx == -1 {
				t.Errorf("Директория %s отсутствует в выводе", dir)
				continue
			}

			if idx < lastIndex {
				t.Errorf("Неправильный порядок директорий: %s должна быть после предыдущей директории", dir)
			}
			lastIndex = idx
		}
	})

	t.Run("FileOrder", func(t *testing.T) {
		// Создаем дополнительные файлы
		files := []string{"b_file.txt", "a_file.txt", "c_file.txt"}
		for _, file := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		tb := NewTreeBuilder(tmpDir, cfg)
		result, err := tb.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Проверяем порядок файлов (должны быть после директорий)
		expectedOrder := []string{
			"a_file.txt",
			"b_file.txt",
			"c_file.txt",
			"important.config",
			"root_file.txt",
		}

		lastIndex := -1
		for _, file := range expectedOrder {
			idx := strings.Index(result, file)
			if idx == -1 {
				t.Errorf("Файл %s отсутствует в выводе", file)
				continue
			}

			if idx < lastIndex {
				t.Errorf("Неправильный порядок файлов: %s должна быть после предыдущего файла", file)
			}
			lastIndex = idx
		}
	})
}

func TestTreeBuilder_shouldSkip(t *testing.T) {
	cfg := &model.ScanConfig{
		ExcludedPatterns: []string{"*.log", "temp_*"},
		ImportantFiles:   []string{"important.config"},
		IncludeConfigs:   false,
		IncludeTests:     false,
	}

	tb := NewTreeBuilder("", cfg)

	tests := []struct {
		name     string // только имя файла/директории
		isDir    bool
		expected bool
		desc     string
	}{
		{name: ".git", isDir: true, expected: true, desc: "Системное исключение: директория .git"},
		{name: "node_modules", isDir: true, expected: true, desc: "Системное исключение: директория node_modules"},
		{name: "dist", isDir: true, expected: true, desc: "Системное исключение: директория dist"},
		{name: "project_structure.txt", isDir: false, expected: true, desc: "Системное исключение: файл project_structure.txt"},
		{name: "src", isDir: true, expected: false, desc: "Нормальная директория"},
		{name: "app.js", isDir: false, expected: false, desc: "Нормальный файл"},
		{name: ".hidden", isDir: true, expected: false, desc: "Скрытая директория"},
		{name: ".hidden_file", isDir: false, expected: false, desc: "Скрытый файл"},
		{name: "project_docs.md", isDir: false, expected: true, desc: "Исключение по шаблону документации"},
		{name: "error.log", isDir: false, expected: true, desc: "Исключение по шаблону"},
		{name: "temp_file.txt", isDir: false, expected: true, desc: "Исключение по шаблону"},
		{name: "important.config", isDir: false, expected: false, desc: "Важный файл"},
		{name: "config.yml", isDir: true, expected: false, desc: "Директория не исключается по типу"},
		{name: "config.yml", isDir: false, expected: true, desc: "Файл конфига исключается"},
		{name: "test.spec.js", isDir: false, expected: true, desc: "Тестовый файл исключается"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			entry := mockDirEntry{
				name:  tt.name,
				isDir: tt.isDir,
			}

			// Для проверки используем только имя, так как системные исключения работают по имени
			result := tb.shouldSkip(tt.name, entry)
			if result != tt.expected {
				t.Errorf("Для элемента '%s' (isDir: %v) ожидалось %v, получено %v",
					tt.name, tt.isDir, tt.expected, result)
			}
		})
	}
}

func TestTreeBuilder_RenderTree(t *testing.T) {
	tb := NewTreeBuilder("", nil)

	// Создаем тестовую структуру дерева
	root := &treeNode{
		name:  ".",
		isDir: true,
		children: []*treeNode{
			{
				name:  "dir1",
				isDir: true,
				children: []*treeNode{
					{name: "file1.txt", isDir: false},
					{
						name:  "subdir",
						isDir: true,
						children: []*treeNode{
							{name: "file2.txt", isDir: false},
						},
					},
				},
			},
			{name: "file3.txt", isDir: false},
		},
	}

	var builder strings.Builder
	tb.renderTree(root, &builder, "", true)

	result := builder.String()
	expected := strings.TrimSpace(`
└── ./
    ├── dir1/
    │   ├── file1.txt
    │   └── subdir/
    │       └── file2.txt
    └── file3.txt
`)

	// Нормализуем пробельные символы
	normalizedResult := strings.ReplaceAll(result, "\r\n", "\n")
	normalizedExpected := strings.ReplaceAll(expected, "\r\n", "\n")

	if !strings.Contains(normalizedResult, normalizedExpected) {
		t.Errorf("Ожидалось:\n%s\n\nПолучено:\n%s", normalizedExpected, normalizedResult)
	}
}

func TestTreeBuilder(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем структуру проекта
	dirs := []string{
		"src",
		"src/components",
		"test",
		"node_modules",
	}

	for _, dir := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, dir), 0755))
	}

	files := []string{
		"src/main.go",
		"src/components/button.go",
		"test/main_test.go",
		"go.mod",
		"node_modules/module/index.js",
	}

	for _, file := range files {
		filePath := filepath.Join(tmpDir, file)
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
		require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))
	}

	cfg := &model.ScanConfig{
		ExcludedPatterns: []string{"test", "node_modules"}, // Исправлено
		ImportantFiles:   []string{"go.mod"},
	}

	t.Run("BuildTree", func(t *testing.T) {
		builder := NewTreeBuilder(tmpDir, cfg)
		tree, err := builder.Build()
		require.NoError(t, err)

		assert.Contains(t, tree, "src/")
		assert.Contains(t, tree, "go.mod")
		assert.NotContains(t, tree, "test/")
		assert.NotContains(t, tree, "node_modules/")
	})

	t.Run("DirectoryOrder", func(t *testing.T) {
		// Создаем дополнительные директории
		newDirs := []string{"b_dir", "a_dir", "c_dir"}
		for _, dir := range newDirs {
			require.NoError(t, os.Mkdir(filepath.Join(tmpDir, dir), 0755))
		}

		builder := NewTreeBuilder(tmpDir, cfg)
		tree, err := builder.Build()
		require.NoError(t, err)

		// Проверяем порядок: a_dir, b_dir, c_dir, src, ...
		aIdx := strings.Index(tree, "a_dir/")
		bIdx := strings.Index(tree, "b_dir/")
		cIdx := strings.Index(tree, "c_dir/")

		assert.True(t, aIdx < bIdx && bIdx < cIdx,
			"Директории должны быть в алфавитном порядке")
	})

	t.Run("FileOrder", func(t *testing.T) {
		// Создаем дополнительные файлы
		newFiles := []string{"b_file.txt", "a_file.txt", "c_file.txt"}
		for _, file := range newFiles {
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, file), []byte("content"), 0644))
		}

		builder := NewTreeBuilder(tmpDir, cfg)
		tree, err := builder.Build()
		require.NoError(t, err)

		// Проверяем порядок файлов: a_file, b_file, c_file
		aIdx := strings.Index(tree, "a_file.txt")
		bIdx := strings.Index(tree, "b_file.txt")
		cIdx := strings.Index(tree, "c_file.txt")

		assert.True(t, aIdx < bIdx && bIdx < cIdx,
			"Файлы должны быть в алфавитном порядке")
	})
}

func TestTreeBuilder_ExcludesGopsConfigFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем файлы конфигурации gops, которые должны быть исключены
	gopsFiles := []string{
		"gops_config.yaml",
		"gops_config_old.yaml",
		"custom_gops_config.yml",
	}

	for _, file := range gopsFiles {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, file), []byte("config content"), 0644))
	}

	// Создаем нормальные файлы, которые должны быть включены
	normalFiles := []string{
		"main.go",
		"README.md",
		"documentation.md",
	}

	for _, file := range normalFiles {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, file), []byte("content"), 0644))
	}

	// Создаем полный конфиг с включенной документацией и явно задаем OutputConfigFilename
	cfg := &model.ScanConfig{
		MaxFileSize:          2 * 1024 * 1024,
		IncludeTests:         true,
		IncludeConfigs:       true,
		IncludeMarkup:        true,
		IncludeStyles:        true,
		IncludeDocs:          true, // Явно включаем документацию
		ExcludedPatterns:     []string{},
		ParallelWorkers:      4,
		SelectionMode:        "all",
		IncludedPaths:        []string{},
		IncludePatterns:      []string{},
		ImportantFiles:       []string{},
		ShowFullStructure:    false,
		DocumentationMode:    "full",
		OutputFilename:       "output.md",
		OutputConfigFilename: "project_docs.md", // Явно задаем, чтобы шаблон был "project_docs*.md"
	}

	builder := NewTreeBuilder(tmpDir, cfg)
	structure, err := builder.BuildStructure()
	require.NoError(t, err)

	// Отладочная информация
	t.Logf("Found %d files in structure:", len(structure.Files))
	for i, file := range structure.Files {
		t.Logf("  %d: %s (skipped: %v, lang: %s)", i, file.Path, file.Skipped, file.Lang)
	}

	// Проверяем, что файлы gops исключены
	for _, gopsFile := range gopsFiles {
		found := false
		for _, projectFile := range structure.Files {
			if projectFile.Path == gopsFile {
				found = true
				break
			}
		}
		assert.False(t, found, "Gops config file %s should be excluded", gopsFile)
	}

	// Проверяем, что нормальные файлы включены и не пропущены
	for _, normalFile := range normalFiles {
		found := false
		for _, projectFile := range structure.Files {
			if projectFile.Path == normalFile && !projectFile.Skipped {
				found = true
				break
			}
		}
		assert.True(t, found, "Normal file %s should be included and not skipped", normalFile)
	}
}

func TestTreeBuilder_ExcludesGoSum(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем go.sum и другие системные файлы
	systemFiles := []string{
		"go.sum",
		"package-lock.json",
		"yarn.lock",
	}

	for _, file := range systemFiles {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, file), []byte("content"), 0644))
	}

	// Создаем нормальные файлы
	normalFiles := []string{
		"go.mod",
		"main.go",
	}

	for _, file := range normalFiles {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, file), []byte("content"), 0644))
	}

	cfg := &model.ScanConfig{
		ExcludedPatterns: []string{},
	}

	builder := NewTreeBuilder(tmpDir, cfg)
	structure, err := builder.BuildStructure()
	require.NoError(t, err)

	// Проверяем, что системные файлы исключены
	for _, systemFile := range systemFiles {
		excluded := true
		for _, projectFile := range structure.Files {
			if projectFile.Path == systemFile {
				excluded = false
				break
			}
		}
		assert.True(t, excluded, "System file %s should be excluded", systemFile)
	}

	// Проверяем, что нормальные файлы включены
	for _, normalFile := range normalFiles {
		included := false
		for _, projectFile := range structure.Files {
			if projectFile.Path == normalFile {
				included = true
				break
			}
		}
		assert.True(t, included, "Normal file %s should be included", normalFile)
	}
}
