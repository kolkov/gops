package selector

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SelectionFile структура файла выбора
type SelectionFile struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Created     string   `yaml:"created"`
	Paths       []string `yaml:"paths"`
	Patterns    []string `yaml:"patterns,omitempty"`
}

// SelectionManager управляет сохраненными выборками
type SelectionManager struct {
	rootDir      string
	selectionDir string
}

// NewSelectionManager создает новый менеджер выборок
func NewSelectionManager(rootDir string) *SelectionManager {
	return &SelectionManager{
		rootDir:      rootDir,
		selectionDir: filepath.Join(rootDir, ".gops"),
	}
}

// SaveSelection сохраняет выборку файлов
func (sm *SelectionManager) SaveSelection(name, description string, paths []string) error {
	// Создаем директорию .gops если её нет
	if err := os.MkdirAll(sm.selectionDir, 0755); err != nil {
		return fmt.Errorf("failed to create .gops directory: %w", err)
	}

	selection := SelectionFile{
		Name:        name,
		Description: description,
		Created:     time.Now().Format("2006-01-02 15:04:05"),
		Paths:       paths,
	}

	// Сохраняем в YAML файл
	filename := fmt.Sprintf("%s.selection.yaml", strings.ReplaceAll(name, " ", "_"))
	filePath := filepath.Join(sm.selectionDir, filename)

	data, err := yaml.Marshal(&selection)
	if err != nil {
		return fmt.Errorf("failed to marshal selection: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write selection file: %w", err)
	}

	// Также сохраняем как "последний выбор"
	lastPath := filepath.Join(sm.selectionDir, "last.selection.yaml")
	if err := os.WriteFile(lastPath, data, 0644); err != nil {
		// Не критично если не удалось сохранить последний выбор
		fmt.Printf("Warning: Could not save last selection: %v\n", err)
	}

	return nil
}

// LoadSelection загружает сохраненную выборку
func (sm *SelectionManager) LoadSelection(name string) (*SelectionFile, error) {
	// Если имя не указано, пытаемся загрузить последнюю выборку
	if name == "" || name == "last" {
		name = "last"
	}

	filename := fmt.Sprintf("%s.selection.yaml", strings.ReplaceAll(name, " ", "_"))
	filePath := filepath.Join(sm.selectionDir, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("selection '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to read selection file: %w", err)
	}

	var selection SelectionFile
	if err := yaml.Unmarshal(data, &selection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal selection: %w", err)
	}

	return &selection, nil
}

// ListSelections возвращает список сохраненных выборок
func (sm *SelectionManager) ListSelections() ([]string, error) {
	// Проверяем существование директории
	if _, err := os.Stat(sm.selectionDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(sm.selectionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read selection directory: %w", err)
	}

	var selections []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".selection.yaml") && file.Name() != "last.selection.yaml" {
			name := strings.TrimSuffix(file.Name(), ".selection.yaml")
			name = strings.ReplaceAll(name, "_", " ")
			selections = append(selections, name)
		}
	}

	return selections, nil
}

// DeleteSelection удаляет сохраненную выборку
func (sm *SelectionManager) DeleteSelection(name string) error {
	filename := fmt.Sprintf("%s.selection.yaml", strings.ReplaceAll(name, " ", "_"))
	filePath := filepath.Join(sm.selectionDir, filename)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("selection '%s' not found", name)
		}
		return fmt.Errorf("failed to delete selection: %w", err)
	}

	return nil
}

// InteractiveLoad интерактивно выбирает и загружает сохраненную выборку
func (sm *SelectionManager) InteractiveLoad() (*SelectionFile, error) {
	selections, err := sm.ListSelections()
	if err != nil {
		return nil, err
	}

	if len(selections) == 0 {
		return nil, fmt.Errorf("no saved selections found")
	}

	fmt.Println("\n📋 Saved Selections:")
	fmt.Println("──────────────────")

	for i, name := range selections {
		// Загружаем для отображения описания
		if sel, err := sm.LoadSelection(name); err == nil {
			if sel.Description != "" {
				fmt.Printf("%2d. %s - %s (%d files)\n", i+1, name, sel.Description, len(sel.Paths))
			} else {
				fmt.Printf("%2d. %s (%d files)\n", i+1, name, len(sel.Paths))
			}
		} else {
			fmt.Printf("%2d. %s\n", i+1, name)
		}
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nSelect number (or 'cancel'): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	if input == "cancel" || input == "" {
		return nil, fmt.Errorf("selection cancelled")
	}

	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(selections) {
		return nil, fmt.Errorf("invalid selection")
	}

	return sm.LoadSelection(selections[index-1])
}

// ExportSelection экспортирует выборку в файл
func (sm *SelectionManager) ExportSelection(name string, outputPath string) error {
	selection, err := sm.LoadSelection(name)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(selection)
	if err != nil {
		return fmt.Errorf("failed to marshal selection: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportSelection импортирует выборку из файла
func (sm *SelectionManager) ImportSelection(inputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var selection SelectionFile
	if err := yaml.Unmarshal(data, &selection); err != nil {
		return fmt.Errorf("failed to unmarshal selection: %w", err)
	}

	return sm.SaveSelection(selection.Name, selection.Description, selection.Paths)
}
