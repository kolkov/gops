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
func (m *MockGenerator) WriteRootConfigHeader()                      {} // НОВЫЙ МЕТОД
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

// MockGeneratorWithTracking для отслеживания вызовов методов
type MockGeneratorWithTracking struct {
	rootConfigCalled bool
	rootFiles        []*model.ProjectFile
}

func (m *MockGeneratorWithTracking) WriteHeader(meta *model.ProjectMeta) {}
func (m *MockGeneratorWithTracking) WriteTree(structure string)          {}
func (m *MockGeneratorWithTracking) WriteNxStructure(projects []*model.NxProject, rootFiles []string) {
}
func (m *MockGeneratorWithTracking) WriteProjectHeader(name, ptype, root string) {}
func (m *MockGeneratorWithTracking) WriteProjectTree(structure string)           {}
func (m *MockGeneratorWithTracking) WriteModulesHeader()                         {}
func (m *MockGeneratorWithTracking) Close() error                                { return nil }

func (m *MockGeneratorWithTracking) WriteRootConfigHeader() {
	m.rootConfigCalled = true
}

func (m *MockGeneratorWithTracking) WriteFileSection(file *model.ProjectFile) {
	// Сохраняем файлы для проверки
	if m.rootFiles == nil {
		m.rootFiles = []*model.ProjectFile{}
	}
	m.rootFiles = append(m.rootFiles, file)
}

func TestNxScanner_ProcessRootImportantFiles(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(logger.InfoLevel)

	// Создаем корневые конфигурационные файлы
	packageJSON := []byte(`{
  "name": "test-monorepo",
  "version": "1.0.0",
  "license": "MIT"
}`)

	nxJSON := []byte(`{
  "npmScope": "test",
  "affected": {
    "defaultBase": "main"
  }
}`)

	tsconfigJSON := []byte(`{
  "compilerOptions": {
    "target": "es2020",
    "module": "esnext"
  }
}`)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), packageJSON, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "nx.json"), nxJSON, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tsconfig.base.json"), tsconfigJSON, 0644))

	// Создаем структуру проектов
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "apps", "app1"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "apps", "app1", "project.json"),
		[]byte(`{"name": "app1"}`),
		0644,
	))

	t.Run("ImportantFilesProcessing", func(t *testing.T) {
		cfg := &model.ScanConfig{
			ImportantFiles: []string{
				"package.json",
				"nx.json",
				"tsconfig.base.json",
			},
			MaxFileSize:          2 * 1024 * 1024,
			OutputFilename:       "output.md",
			OutputConfigFilename: "project_docs.md",
		}

		scanner := NewScanner(tmpDir, "output.md", cfg, log)
		mockGen := &MockGeneratorWithTracking{}

		// Вызываем метод обработки корневых файлов напрямую
		err := scanner.processRootImportantFiles(mockGen)
		require.NoError(t, err)

		// Проверяем что был вызван WriteRootConfigHeader
		assert.True(t, mockGen.rootConfigCalled, "WriteRootConfigHeader should be called")

		// Проверяем что все файлы были обработаны
		assert.Len(t, mockGen.rootFiles, 3, "Should process 3 root files")

		// Отладочная информация - выводим что получили
		t.Logf("Processed files count: %d", len(mockGen.rootFiles))
		for i, file := range mockGen.rootFiles {
			t.Logf("File %d: Path=%s, Content length=%d, Lang=%s",
				i, file.Path, len(file.Content), file.Lang)
			if len(file.Content) > 50 {
				t.Logf("  Content preview: %s...", string(file.Content[:50]))
			} else {
				t.Logf("  Content: %s", string(file.Content))
			}
		}

		// Проверяем содержимое файлов - ищем каждый файл по имени
		fileMap := make(map[string]*model.ProjectFile)
		for _, file := range mockGen.rootFiles {
			fileMap[file.Path] = file
		}

		// Проверяем package.json
		if file, ok := fileMap["package.json"]; ok {
			assert.NotNil(t, file.Content, "File content should not be nil for package.json")
			assert.False(t, file.Skipped, "File should not be skipped for package.json")
			assert.Contains(t, string(file.Content), "test-monorepo", "package.json should contain 'test-monorepo'")
			assert.Equal(t, "json", file.Lang)
		} else {
			t.Error("package.json not found in processed files")
		}

		// Проверяем nx.json
		if file, ok := fileMap["nx.json"]; ok {
			assert.NotNil(t, file.Content, "File content should not be nil for nx.json")
			assert.False(t, file.Skipped, "File should not be skipped for nx.json")
			assert.Contains(t, string(file.Content), "npmScope", "nx.json should contain 'npmScope'")
			assert.Equal(t, "json", file.Lang)
		} else {
			t.Error("nx.json not found in processed files")
		}

		// Проверяем tsconfig.base.json
		if file, ok := fileMap["tsconfig.base.json"]; ok {
			assert.NotNil(t, file.Content, "File content should not be nil for tsconfig.base.json")
			assert.False(t, file.Skipped, "File should not be skipped for tsconfig.base.json")
			assert.Contains(t, string(file.Content), "compilerOptions", "tsconfig.base.json should contain 'compilerOptions'")
			assert.Equal(t, "json", file.Lang)
		} else {
			t.Error("tsconfig.base.json not found in processed files")
		}
	})

	t.Run("NoImportantFiles", func(t *testing.T) {
		cfg := &model.ScanConfig{
			ImportantFiles:       []string{}, // Нет важных файлов
			MaxFileSize:          2 * 1024 * 1024,
			OutputFilename:       "output.md",
			OutputConfigFilename: "project_docs.md",
		}

		scanner := NewScanner(tmpDir, "output.md", cfg, log)
		mockGen := &MockGeneratorWithTracking{}

		err := scanner.processRootImportantFiles(mockGen)
		require.NoError(t, err)

		// Не должен вызываться WriteRootConfigHeader если нет файлов
		assert.False(t, mockGen.rootConfigCalled, "WriteRootConfigHeader should not be called when no files")
		assert.Len(t, mockGen.rootFiles, 0, "Should not process any files")
	})

	t.Run("NonExistentImportantFiles", func(t *testing.T) {
		cfg := &model.ScanConfig{
			ImportantFiles: []string{
				"package.json",
				"non-existent.json", // Несуществующий файл
			},
			MaxFileSize:          2 * 1024 * 1024,
			OutputFilename:       "output.md",
			OutputConfigFilename: "project_docs.md",
		}

		scanner := NewScanner(tmpDir, "output.md", cfg, log)
		mockGen := &MockGeneratorWithTracking{}

		err := scanner.processRootImportantFiles(mockGen)
		require.NoError(t, err)

		// Должен обработать только существующий файл
		assert.True(t, mockGen.rootConfigCalled)
		assert.Len(t, mockGen.rootFiles, 1, "Should process only existing file")
		assert.Equal(t, "package.json", mockGen.rootFiles[0].Path)
	})
}
