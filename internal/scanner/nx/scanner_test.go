package nx

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kolkov/gops/internal/docgen"
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGenerator для тестирования
type MockGenerator struct {
	docgen.Generator
}

func (m *MockGenerator) WriteProjectHeader(name, ptype, root string) {}
func (m *MockGenerator) WriteProjectTree(structure string)           {}
func (m *MockGenerator) WriteModulesHeader()                         {}
func (m *MockGenerator) WriteFileSection(file *model.ProjectFile)    {}
func (m *MockGenerator) Close() error                                { return nil }

func TestNxScanner(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &model.ScanConfig{
		OutputFilename:       "output.md",
		OutputConfigFilename: "config.yaml",
	}
	logger := logger.New(logger.InfoLevel)

	// Создаем структуру проектов
	projects := []struct {
		dir  string
		name string
	}{
		{"apps/app1", "app1"},
		{"apps/app2", "app2"},
		{"libs/lib1", "lib1"},
		{"tools/tool1", "tool1"},
	}

	for _, p := range projects {
		dir := filepath.Join(tmpDir, p.dir)
		require.NoError(t, os.MkdirAll(dir, 0755))

		projectFile := filepath.Join(dir, "project.json")
		require.NoError(t, os.WriteFile(projectFile, []byte(`{"name": "`+p.name+`"}`), 0644))
	}

	rootFiles := []string{"nx.json", "package.json"}
	for _, file := range rootFiles {
		path := filepath.Join(tmpDir, file)
		require.NoError(t, os.WriteFile(path, []byte("{}"), 0644))
	}

	scanner := NewScanner(tmpDir, "output.md", cfg, logger)

	t.Run("LoadProjects", func(t *testing.T) {
		require.NoError(t, scanner.loadProjects())
		assert.Len(t, scanner.projects, 4)
	})

	t.Run("FilterProjects", func(t *testing.T) {
		// Создаем тестовые проекты
		testProjects := []*model.NxProject{
			{Name: "p1", Type: "apps"},
			{Name: "p2", Type: "apps"},
			{Name: "p3", Type: "libs"},
		}

		tests := []struct {
			input    string
			expected []string
		}{
			{"all", []string{"p1", "p2", "p3"}},
			{"1,3", []string{"p1", "p3"}},
			{"2", []string{"p2"}},
			{"invalid", []string{}},
			{"", []string{"p1", "p2", "p3"}},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result := filterProjects(testProjects, tt.input)
				assert.Len(t, result, len(tt.expected),
					"For input '%s' expected %d projects, got %d",
					tt.input, len(tt.expected), len(result))

				for i, name := range tt.expected {
					assert.Equal(t, name, result[i].Name)
				}
			})
		}
	})

	t.Run("GetRootFiles", func(t *testing.T) {
		// Настроим исключения для конфигурации
		cfg.ExcludedPatterns = []string{"*.tmp"}
		scanner := NewScanner(tmpDir, "output.md", cfg, logger)

		files, err := scanner.getRootFiles()
		require.NoError(t, err)

		assert.Len(t, files, 2)
		assert.Contains(t, files, "nx.json")
		assert.Contains(t, files, "package.json")
	})

	t.Run("ScanProject", func(t *testing.T) {
		scanner := NewScanner(tmpDir, "output.md", cfg, logger)
		require.NoError(t, scanner.loadProjects())

		project := scanner.projects[0] // Первый проект

		// Создаем временный файл для проекта
		testFile := filepath.Join(project.SourceDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		// Создаем mock генератора
		mockGen := new(MockGenerator)

		// Выполняем сканирование
		require.NoError(t, scanner.scanProject(context.Background(), mockGen, project))
	})
}

// Локальная реализация фильтрации для тестирования
func filterProjects(projects []*model.NxProject, input string) []*model.NxProject {
	if strings.EqualFold(input, "all") || input == "" {
		return projects
	}

	var selected []*model.NxProject
	indices := strings.Split(input, ",")

	for _, idxStr := range indices {
		idx, err := strconv.Atoi(strings.TrimSpace(idxStr))
		if err != nil || idx < 1 || idx > len(projects) {
			continue
		}
		selected = append(selected, projects[idx-1])
	}

	return selected
}
