package markdown

import (
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerator(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test_output.md")
	log := logger.New(logger.InfoLevel)

	t.Run("WriteHeader", func(t *testing.T) {
		gen := NewGenerator(outputFile, log)
		defer gen.Close()

		meta := &model.ProjectMeta{
			Name:    "Test Project",
			Type:    "Go",
			RootDir: "/path/to/project",
		}
		gen.WriteHeader(meta)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		assert.Contains(t, string(content), "# Проект: Test Project")
		assert.Contains(t, string(content), "**Тип:** Go")
		assert.Contains(t, string(content), "## Содержание")
	})

	t.Run("WriteTree", func(t *testing.T) {
		gen := NewGenerator(outputFile, log)
		defer gen.Close()

		treeStructure := ".\n├── dir1\n└── file1.txt"
		gen.WriteTree(treeStructure)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		expected := "## Структура проекта\n\n```\n" + treeStructure + "\n```\n\n"
		assert.Contains(t, string(content), expected)
	})

	t.Run("WriteFileSection", func(t *testing.T) {
		gen := NewGenerator(outputFile, log)
		defer gen.Close()

		file := &model.ProjectFile{
			Path:    "src/main.go",
			Content: []byte("package main\n\nfunc main() {}"),
			Lang:    "go",
		}
		gen.WriteFileSection(file)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		expected := "### src/main.go\n\n```go\npackage main\n\nfunc main() {}\n```\n\n"
		assert.Contains(t, string(content), expected)
	})

	t.Run("WriteSkippedFile", func(t *testing.T) {
		gen := NewGenerator(outputFile, log)
		defer gen.Close()

		file := &model.ProjectFile{
			Path:    "large.bin",
			Skipped: true,
		}
		gen.WriteFileSection(file)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		assert.Contains(t, string(content), "### large.bin\n\n```\n[Содержимое файла пропущено]\n```")
		assert.Contains(t, string(content), "_Файл был пропущен по настройкам документации._")
	})

	t.Run("WriteNxStructure", func(t *testing.T) {
		gen := NewGenerator(outputFile, log)
		defer gen.Close()

		projects := []*model.NxProject{
			{Name: "app1", Type: "apps"},
			{Name: "lib1", Type: "libs"},
		}
		rootFiles := []string{"nx.json", "package.json"}
		gen.WriteNxStructure(projects, rootFiles)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		assert.Contains(t, string(content), "## Общая структура Nx Monorepo")
		assert.Contains(t, string(content), "apps/")
		assert.Contains(t, string(content), "libs/")
		assert.Contains(t, string(content), "nx.json")
	})

	t.Run("Close", func(t *testing.T) {
		gen := NewGenerator(outputFile, log)
		require.NoError(t, gen.Close())

		_, err := os.Stat(outputFile)
		require.NoError(t, err)
	})
}
