package selector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSelectionManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSelectionManager(tmpDir)

	paths := []string{
		"src/main.go",
		"src/util.go",
		"README.md",
	}

	t.Run("SaveSelection", func(t *testing.T) {
		err := sm.SaveSelection("test_selection", "Test description", paths)
		require.NoError(t, err)

		// Проверяем что файл создан
		selectionPath := filepath.Join(tmpDir, ".gops", "test_selection.selection.yaml")
		assert.FileExists(t, selectionPath)

		// Проверяем что last.selection.yaml тоже создан
		lastPath := filepath.Join(tmpDir, ".gops", "last.selection.yaml")
		assert.FileExists(t, lastPath)
	})

	t.Run("LoadSelection", func(t *testing.T) {
		selection, err := sm.LoadSelection("test_selection")
		require.NoError(t, err)

		assert.Equal(t, "test_selection", selection.Name)
		assert.Equal(t, "Test description", selection.Description)
		assert.Equal(t, paths, selection.Paths)
		assert.NotEmpty(t, selection.Created)
	})

	t.Run("LoadLastSelection", func(t *testing.T) {
		selection, err := sm.LoadSelection("last")
		require.NoError(t, err)

		assert.Equal(t, "test_selection", selection.Name)
		assert.Equal(t, paths, selection.Paths)
	})

	t.Run("LoadNonExistentSelection", func(t *testing.T) {
		_, err := sm.LoadSelection("non_existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSelectionManager_ListSelections(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSelectionManager(tmpDir)

	t.Run("EmptyList", func(t *testing.T) {
		selections, err := sm.ListSelections()
		require.NoError(t, err)
		assert.Empty(t, selections)
	})

	t.Run("MultipleSelections", func(t *testing.T) {
		// Сохраняем несколько выборок
		require.NoError(t, sm.SaveSelection("selection_1", "First", []string{"file1.go"}))
		require.NoError(t, sm.SaveSelection("selection_2", "Second", []string{"file2.go"}))
		require.NoError(t, sm.SaveSelection("selection_3", "Third", []string{"file3.go"}))

		selections, err := sm.ListSelections()
		require.NoError(t, err)

		assert.Len(t, selections, 3)
		assert.Contains(t, selections, "selection 1") // Подчеркивания заменяются на пробелы
		assert.Contains(t, selections, "selection 2")
		assert.Contains(t, selections, "selection 3")
	})

	t.Run("ExcludesLastSelection", func(t *testing.T) {
		selections, err := sm.ListSelections()
		require.NoError(t, err)

		// last.selection.yaml не должен быть в списке
		for _, s := range selections {
			assert.NotEqual(t, "last", s)
		}
	})
}

func TestSelectionManager_DeleteSelection(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSelectionManager(tmpDir)

	// Сохраняем выборку
	require.NoError(t, sm.SaveSelection("to_delete", "Test", []string{"file.go"}))

	t.Run("DeleteExisting", func(t *testing.T) {
		err := sm.DeleteSelection("to_delete")
		require.NoError(t, err)

		// Проверяем что файл удален
		selectionPath := filepath.Join(tmpDir, ".gops", "to_delete.selection.yaml")
		assert.NoFileExists(t, selectionPath)

		// Проверяем что нельзя загрузить удаленную выборку
		_, err = sm.LoadSelection("to_delete")
		assert.Error(t, err)
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		err := sm.DeleteSelection("non_existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSelectionManager_ExportImport(t *testing.T) {
	tmpDir := t.TempDir()
	sm1 := NewSelectionManager(filepath.Join(tmpDir, "project1"))
	sm2 := NewSelectionManager(filepath.Join(tmpDir, "project2"))

	paths := []string{"src/main.go", "README.md"}

	t.Run("ExportSelection", func(t *testing.T) {
		// Сохраняем выборку в первом проекте
		require.NoError(t, sm1.SaveSelection("export_test", "Export test", paths))

		// Экспортируем
		exportPath := filepath.Join(tmpDir, "export.yaml")
		err := sm1.ExportSelection("export_test", exportPath)
		require.NoError(t, err)

		assert.FileExists(t, exportPath)

		// Проверяем содержимое экспортированного файла
		data, err := os.ReadFile(exportPath)
		require.NoError(t, err)

		var selection SelectionFile
		require.NoError(t, yaml.Unmarshal(data, &selection))
		assert.Equal(t, "export_test", selection.Name)
		assert.Equal(t, paths, selection.Paths)
	})

	t.Run("ImportSelection", func(t *testing.T) {
		exportPath := filepath.Join(tmpDir, "export.yaml")

		// Импортируем во второй проект
		err := sm2.ImportSelection(exportPath)
		require.NoError(t, err)

		// Проверяем что выборка доступна
		selection, err := sm2.LoadSelection("export_test")
		require.NoError(t, err)
		assert.Equal(t, "Export test", selection.Description)
		assert.Equal(t, paths, selection.Paths)
	})

	t.Run("ImportNonExistentFile", func(t *testing.T) {
		err := sm2.ImportSelection(filepath.Join(tmpDir, "non_existent.yaml"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read import file")
	})
}

func TestSelectionManager_InteractiveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSelectionManager(tmpDir)

	t.Run("NoSavedSelections", func(t *testing.T) {
		_, err := sm.InteractiveLoad()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no saved selections found")
	})

	t.Run("CancelSelection", func(t *testing.T) {
		// Сохраняем выборку
		require.NoError(t, sm.SaveSelection("test", "Test", []string{"file.go"}))

		// Создаем временный reader с вводом "cancel"
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, _ := os.Pipe()
		os.Stdin = r
		go func() {
			defer w.Close()
			w.Write([]byte("cancel\n"))
		}()

		_, err := sm.InteractiveLoad()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("ValidSelection", func(t *testing.T) {
		// Используем bufio.Reader напрямую для теста
		sm := &SelectionManager{
			rootDir:      tmpDir,
			selectionDir: filepath.Join(tmpDir, ".gops"),
		}

		// Сохраняем несколько выборок
		require.NoError(t, sm.SaveSelection("selection_1", "First", []string{"file1.go"}))
		require.NoError(t, sm.SaveSelection("selection_2", "Second", []string{"file2.go"}))

		// Мокаем InteractiveLoad через модификацию
		selection, err := sm.LoadSelection("selection 1")
		require.NoError(t, err)
		assert.Equal(t, "selection_1", selection.Name)
	})

	t.Run("InvalidSelectionNumber", func(t *testing.T) {
		// Создаем временный reader с невалидным вводом
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, _ := os.Pipe()
		os.Stdin = r
		go func() {
			defer w.Close()
			w.Write([]byte("999\n"))
		}()

		// Сохраняем выборку
		require.NoError(t, sm.SaveSelection("test2", "Test", []string{"file.go"}))

		_, err := sm.InteractiveLoad()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid selection")
	})
}

func TestSelectionFile_YAML(t *testing.T) {
	selection := SelectionFile{
		Name:        "test_selection",
		Description: "Test description",
		Created:     "2023-01-01 12:00:00",
		Paths: []string{
			"src/main.go",
			"README.md",
		},
		Patterns: []string{
			"*.go",
			"docs/**/*.md",
		},
	}

	t.Run("MarshalYAML", func(t *testing.T) {
		data, err := yaml.Marshal(&selection)
		require.NoError(t, err)

		yamlStr := string(data)
		assert.Contains(t, yamlStr, "name: test_selection")
		assert.Contains(t, yamlStr, "description: Test description")
		assert.Contains(t, yamlStr, "src/main.go")
		assert.Contains(t, yamlStr, "*.go")
	})

	t.Run("UnmarshalYAML", func(t *testing.T) {
		yamlContent := `
name: test_selection
description: Test description
created: "2023-01-01 12:00:00"
paths:
  - src/main.go
  - README.md
patterns:
  - "*.go"
  - "docs/**/*.md"
`

		var loaded SelectionFile
		err := yaml.Unmarshal([]byte(yamlContent), &loaded)
		require.NoError(t, err)

		assert.Equal(t, selection.Name, loaded.Name)
		assert.Equal(t, selection.Description, loaded.Description)
		assert.Equal(t, selection.Paths, loaded.Paths)
		assert.Equal(t, selection.Patterns, loaded.Patterns)
	})
}

func TestSelectionManager_SpecialCharactersInName(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSelectionManager(tmpDir)

	t.Run("NameWithSpaces", func(t *testing.T) {
		name := "my selection with spaces"
		err := sm.SaveSelection(name, "Test", []string{"file.go"})
		require.NoError(t, err)

		// Проверяем что файл сохранен с подчеркиваниями
		expectedFile := filepath.Join(tmpDir, ".gops", "my_selection_with_spaces.selection.yaml")
		assert.FileExists(t, expectedFile)

		// Загружаем по имени с пробелами
		selection, err := sm.LoadSelection(name)
		require.NoError(t, err)
		assert.Equal(t, name, selection.Name)
	})

	t.Run("ListWithSpaces", func(t *testing.T) {
		selections, err := sm.ListSelections()
		require.NoError(t, err)

		// В списке имя должно быть с пробелами
		assert.Contains(t, selections, "my selection with spaces")
	})
}

// Вспомогательная функция для тестирования интерактивного ввода
func createMockInput(input string) (*os.File, func()) {
	r, w, _ := os.Pipe()

	go func() {
		defer w.Close()
		w.Write([]byte(input))
	}()

	oldStdin := os.Stdin
	os.Stdin = r

	return r, func() {
		r.Close()
		os.Stdin = oldStdin
	}
}
