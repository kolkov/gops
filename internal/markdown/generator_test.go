package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m mockFileInfo) Name() string { return m.name }
func (m mockFileInfo) Size() int64  { return m.size }
func (m mockFileInfo) Mode() os.FileMode {
	if m.isDir {
		return os.ModeDir | 0755
	}
	return 0644
}
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func TestGenerateProjectTree(t *testing.T) {
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

	gen := &DocumentationGenerator{}

	t.Run("FullStructure", func(t *testing.T) {
		result := gen.GenerateProjectTree(tmpDir, tmpDir)

		// Ожидаем порядок: сначала директории, потом файлы
		expected := strings.TrimSpace(`
.
├── dir1/
│   ├── subdir/
│   │   └── file2.txt
│   └── file1.txt
├── dir2/
│   └── file3.txt
└── root_file.txt
`)

		expected = "```\n" + expected + "\n```\n"

		if !strings.Contains(result, expected) {
			t.Errorf("Ожидалось:\n%s\nПолучено:\n%s", expected, result)
		}
	})

	t.Run("RelativePath", func(t *testing.T) {
		result := gen.GenerateProjectTree(dir1, tmpDir)

		expected := strings.TrimSpace(`
dir1
├── subdir/
│   └── file2.txt
└── file1.txt
`)

		expected = "```\n" + expected + "\n```\n"

		if !strings.Contains(result, expected) {
			t.Errorf("Ожидалось:\n%s\nПолучено:\n%s", expected, result)
		}
	})

	t.Run("WindowsPaths", func(t *testing.T) {
		winPath := strings.ReplaceAll(tmpDir, "/", "\\")
		result := gen.GenerateProjectTree(winPath, winPath)

		expected := strings.TrimSpace(`
.
├── dir1/
│   ├── subdir/
│   │   └── file2.txt
│   └── file1.txt
├── dir2/
│   └── file3.txt
└── root_file.txt
`)

		expected = "```\n" + expected + "\n```\n"

		if !strings.Contains(result, expected) {
			t.Errorf("Ожидалось:\n%s\nПолучено:\n%s", expected, result)
		}
	})

	t.Run("Exclusions", func(t *testing.T) {
		excludedDir := filepath.Join(tmpDir, ".git")
		os.Mkdir(excludedDir, 0755)
		os.WriteFile(filepath.Join(excludedDir, "config"), []byte("test"), 0644)

		result := gen.GenerateProjectTree(tmpDir, tmpDir)

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
		{".hidden/file", false, true}, // Исправлено: должно быть true
		{"project_documentation.md", false, true},
		{"project_structure.txt", false, true},
		{"normal_dir/normal_file.txt", false, false},
	}

	for _, test := range tests {
		info := mockFileInfo{
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
