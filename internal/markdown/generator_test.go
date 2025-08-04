package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateProjectTree(t *testing.T) {
	// Создаем временную директорию
	tmpDir := t.TempDir()

	// Создаем структуру проекта:
	//   tmpDir/
	//   ├── dir1/
	//   │   ├── file1.txt
	//   │   └── subdir/
	//   │       └── file2.txt
	//   ├── dir2/
	//   │   └── file3.txt
	//   └── root_file.txt

	// Создаем директории
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	subdir := filepath.Join(dir1, "subdir")

	os.Mkdir(dir1, 0755)
	os.Mkdir(dir2, 0755)
	os.Mkdir(subdir, 0755)

	// Создаем файлы
	os.WriteFile(filepath.Join(dir1, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(subdir, "file2.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir2, "file3.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "root_file.txt"), []byte("test"), 0644)

	// Создаем генератор
	gen := &DocumentationGenerator{}

	// Тест 1: Проверка полной структуры
	t.Run("FullStructure", func(t *testing.T) {
		result := gen.GenerateProjectTree(tmpDir, tmpDir)
		expected := strings.TrimSpace(`
.
├── dir1/
│   ├── file1.txt
│   └── subdir/
│       └── file2.txt
├── dir2/
│   └── file3.txt
└── root_file.txt
`)

		// Нормализуем ожидаемый результат
		expected = "```\n" + expected + "\n```\n"

		if !strings.Contains(result, expected) {
			t.Errorf("Ожидалось:\n%s\nПолучено:\n%s", expected, result)
		}
	})

	// Тест 2: Проверка относительных путей
	t.Run("RelativePath", func(t *testing.T) {
		result := gen.GenerateProjectTree(dir1, tmpDir)
		expected := strings.TrimSpace(`
dir1
├── file1.txt
└── subdir/
    └── file2.txt
`)

		// Нормализуем ожидаемый результат
		expected = "```\n" + expected + "\n```\n"

		if !strings.Contains(result, expected) {
			t.Errorf("Ожидалось:\n%s\nПолучено:\n%s", expected, result)
		}
	})

	// Тест 3: Проверка Windows-путей
	t.Run("WindowsPaths", func(t *testing.T) {
		// Эмулируем Windows-пути
		winPath := strings.ReplaceAll(tmpDir, "/", "\\")
		result := gen.GenerateProjectTree(winPath, winPath)

		// Ожидаем Linux-формат в выводе
		expected := strings.TrimSpace(`
.
├── dir1/
│   ├── file1.txt
│   └── subdir/
│       └── file2.txt
├── dir2/
│   └── file3.txt
└── root_file.txt
`)

		expected = "```\n" + expected + "\n```\n"

		if !strings.Contains(result, expected) {
			t.Errorf("Ожидалось:\n%s\nПолучено:\n%s", expected, result)
		}
	})

	// Тест 4: Проверка исключений
	t.Run("Exclusions", func(t *testing.T) {
		// Создаем исключаемую директорию
		excludedDir := filepath.Join(tmpDir, ".git")
		os.Mkdir(excludedDir, 0755)
		os.WriteFile(filepath.Join(excludedDir, "config"), []byte("test"), 0644)

		result := gen.GenerateProjectTree(tmpDir, tmpDir)

		// Проверяем что .git отсутствует в выводе
		if strings.Contains(result, ".git") {
			t.Errorf("Исключенная директория .git присутствует в выводе:\n%s", result)
		}
	})
}

func TestShouldExcludeFromTree(t *testing.T) {
	tests := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{".git/HEAD", false, true},
		{"node_modules/react", true, true},
		{"dist/main.js", false, true},
		{"src/app.js", false, false},
		{".hidden/file", false, true},
		{"project_documentation.md", false, true},
		{"project_structure.txt", false, true},
		{"normal_dir/normal_file.txt", false, false},
	}

	for _, test := range tests {
		// Создаем фейковый FileInfo
		info := struct {
			os.FileInfo
			name  string
			isDir bool
		}{
			name:  filepath.Base(test.path),
			isDir: test.isDir,
		}

		result := shouldExcludeFromTree(test.path, info)
		if result != test.expected {
			t.Errorf("Для пути '%s' (isDir: %v) ожидалось %v, получено %v",
				test.path, test.isDir, test.expected, result)
		}
	}
}
