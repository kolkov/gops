package filesystem

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

func ReadFile(path string, maxSize int64) ([]byte, error) {
	if strings.HasSuffix(strings.ToLower(path), "gops_config.yaml") {
		return nil, fmt.Errorf("gops config file should be skipped")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.Size() > maxSize {
		return nil, model.ErrFileTooLarge
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(buf, file)
	if err != nil {
		return nil, err
	}

	// ВАЖНО: Возвращаем КОПИЮ байтов, а не ссылку на буфер!
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func ShouldSkipFile(name, outputConfigFilename, outputFilename string, excludedPatterns, importantFiles []string) bool {
	// Всегда пропускать конфигурационные файлы gops
	if strings.HasPrefix(name, "gops_config") || strings.Contains(strings.ToLower(name), "gops_config") {
		return true
	}

	if name == "go.sum" {
		return true
	}

	// Проверяем outputFilename только если он не пустой
	if outputFilename != "" && name == filepath.Base(outputFilename) {
		return true
	}

	// Проверяем шаблон документации только если outputConfigFilename не пустой
	if outputConfigFilename != "" {
		baseName := strings.TrimSuffix(filepath.Base(outputConfigFilename), filepath.Ext(outputConfigFilename))
		if baseName != "" {
			docPattern := baseName + "*.md"
			if matched, _ := filepath.Match(docPattern, name); matched {
				return true
			}
		}
	}

	systemExcludes := []string{
		"package-lock.json", "yarn.lock", "pnpm-lock.yaml",
	}
	for _, excl := range systemExcludes {
		if name == excl {
			return true
		}
	}

	if strings.HasPrefix(name, "~$") {
		return true
	}

	for _, pattern := range excludedPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	// Проверка важных файлов должна переопределять исключения
	for _, important := range importantFiles {
		if name == important {
			return false
		}
	}

	return false
}

func shouldSkipDir(name string, excludedPatterns []string) bool {
	systemExcludes := []string{
		".idea", ".vscode", ".git", "node_modules",
		"vendor", "dist", "build", "__pycache__",
		"bin", "obj", "coverage", ".angular",
		".nx", "target", "out", "__tests__",
		"__snapshots__", ".next", ".nuxt", ".cache",
		"cypress", "e2e",
	}
	for _, excl := range systemExcludes {
		if name == excl {
			return true
		}
	}

	for _, pattern := range excludedPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

func shouldSkipByType(path string, cfg *model.ScanConfig) bool {
	for _, important := range cfg.ImportantFiles {
		if strings.HasSuffix(path, important) {
			return false
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	fileName := strings.ToLower(filepath.Base(path))

	switch {
	case !cfg.IncludeStyles && (ext == ".css" || ext == ".scss" || ext == ".sass" || ext == ".less"):
		return true
	case !cfg.IncludeMarkup && (ext == ".html" || ext == ".htm"):
		return true
	case !cfg.IncludeConfigs && (ext == ".json" || ext == ".yaml" || ext == ".yml" || strings.Contains(fileName, "config")):
		return true
	case !cfg.IncludeTests && (strings.Contains(fileName, ".spec.") || strings.Contains(fileName, ".test.")):
		return true
	case !cfg.IncludeDocs && (ext == ".md" || ext == ".markdown"):
		return true
	default:
		return false
	}
}

// ScanProject сканирует проект с учетом настроек выбора файлов
func ScanProject(root string, cfg *model.ScanConfig, logger *logger.Logger, processFile func(*model.ProjectFile)) error {
	// Если есть режим выбора файлов, показываем информацию
	if cfg.HasFileSelection() {
		logger.Info("Using file selection mode",
			"mode", cfg.SelectionMode,
			"files", cfg.GetSelectedFilesCount())
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Для режима выбора файлов, проверяем нужна ли эта директория
			if cfg.HasFileSelection() {
				relPath, _ := filepath.Rel(root, path)

				// Проверяем, есть ли выбранные файлы в этой директории
				hasSelectedFiles := false
				for _, includedPath := range cfg.IncludedPaths {
					if strings.HasPrefix(includedPath, relPath+string(os.PathSeparator)) || includedPath == relPath {
						hasSelectedFiles = true
						break
					}
				}

				// Пропускаем директорию если в ней нет выбранных файлов
				if !hasSelectedFiles && cfg.SelectionMode == "list" {
					return filepath.SkipDir
				}
			}

			// Обычная проверка на пропуск директорий
			if shouldSkipDir(info.Name(), cfg.ExcludedPatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		// Проверка на пропуск файла по имени
		if ShouldSkipFile(info.Name(), cfg.OutputConfigFilename, cfg.OutputFilename, cfg.ExcludedPatterns, cfg.ImportantFiles) {
			return nil
		}

		// Обрабатываем файл с учетом режима выбора
		file := processSingleFileWithSelection(path, root, cfg)
		if file != nil {
			processFile(file)
		}
		return nil
	})

	return err
}

// processSingleFileWithSelection обрабатывает файл с учетом режима выбора
func processSingleFileWithSelection(path, root string, cfg *model.ScanConfig) *model.ProjectFile {
	relPath, _ := filepath.Rel(root, path)

	// Проверяем, включен ли файл в выборку
	if cfg.HasFileSelection() && !cfg.IsFileIncluded(relPath) {
		// Файл не выбран - пропускаем его
		return nil
	}

	// Дальше стандартная обработка
	return processSingleFile(path, root, cfg)
}

func processSingleFile(path, root string, cfg *model.ScanConfig) *model.ProjectFile {
	relPath, _ := filepath.Rel(root, path)
	file := &model.ProjectFile{
		Path: relPath,
		Lang: model.GetFileLanguage(path),
	}

	// Добавим отладочную информацию
	fmt.Printf("Processing file: %s\n", relPath)

	// Применяем все проверки исключений
	if ShouldSkipFile(filepath.Base(path), cfg.OutputConfigFilename, cfg.OutputFilename, cfg.ExcludedPatterns, cfg.ImportantFiles) {
		fmt.Printf("File %s skipped by ShouldSkipFile\n", relPath)
		return nil
	}

	// Проверяем тип файла
	if shouldSkipByType(path, cfg) {
		fmt.Printf("File %s skipped by shouldSkipByType\n", relPath)
		file.Skipped = true
		return file
	}

	// Читаем содержимое файла
	content, err := ReadFile(path, cfg.MaxFileSize)
	if err != nil {
		fmt.Printf("File %s skipped due to read error: %v\n", relPath, err)
		file.Skipped = true
		return file
	}

	file.Content = string(content)
	fmt.Printf("File %s included\n", relPath)
	return file
}

// CountSelectedFiles подсчитывает количество файлов, которые будут включены
func CountSelectedFiles(root string, cfg *model.ScanConfig) (int, error) {
	count := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(root, path)

		// Пропускаем системные файлы
		if ShouldSkipFile(info.Name(), cfg.OutputConfigFilename, cfg.OutputFilename, cfg.ExcludedPatterns, cfg.ImportantFiles) {
			return nil
		}

		// Проверяем включен ли файл
		if !cfg.HasFileSelection() || cfg.IsFileIncluded(relPath) {
			count++
		}

		return nil
	})

	return count, err
}
