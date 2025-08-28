// Package phases содержит фазу выбора файлов для документации
package phases

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/plugin"
	"github.com/kolkov/gops/pkg/logger"
)

// SelectionPhase - фаза выбора файлов для включения в документацию
type SelectionPhase struct {
	config   *model.ProjectConfig
	registry *plugin.Registry
	logger   *logger.Logger

	// Статистика выбора
	stats *SelectionStats
}

// SelectionStats - статистика выбора файлов
type SelectionStats struct {
	TotalFiles      int
	SelectedFiles   int
	SkippedFiles    int
	FilteredByRules int
	FilteredBySize  int
	FilteredByType  int
}

// NewSelectionPhase создает новую фазу выбора
func NewSelectionPhase(config *model.ProjectConfig, registry *plugin.Registry, logger *logger.Logger) *SelectionPhase {
	return &SelectionPhase{
		config:   config,
		registry: registry,
		logger:   logger,
		stats:    &SelectionStats{},
	}
}

// Select выбирает файлы для обработки на основе конфигурации и правил
func (s *SelectionPhase) Select(ctx context.Context, metadata *model.ProjectMetadata, info *model.ProjectInfo) ([]string, error) {
	s.logger.Info("Starting file selection",
		"mode", s.config.ScanConfig.SelectionMode,
		"total_files", metadata.TotalFiles)

	s.stats.TotalFiles = metadata.TotalFiles

	// Выбираем файлы в зависимости от режима
	var selectedFiles []string
	var err error

	switch s.config.ScanConfig.SelectionMode {
	case "all":
		selectedFiles, err = s.selectAll(ctx, metadata)

	case "list":
		selectedFiles, err = s.selectFromList(ctx, metadata)

	case "patterns":
		selectedFiles, err = s.selectByPatterns(ctx, metadata)

	case "interactive":
		// Interactive режим должен быть обработан до этого момента
		selectedFiles, err = s.selectFromList(ctx, metadata)

	case "smart":
		selectedFiles, err = s.selectSmart(ctx, metadata, info)

	default:
		// По умолчанию выбираем умный режим
		selectedFiles, err = s.selectSmart(ctx, metadata, info)
	}

	if err != nil {
		return nil, fmt.Errorf("selection failed: %w", err)
	}

	// Применяем фильтры плагинов
	selectedFiles = s.applyPluginFilters(selectedFiles, metadata)

	// Применяем правила загрузки
	selectedFiles = s.applyLoadingRules(selectedFiles, metadata)

	// Применяем лимиты
	selectedFiles = s.applyLimits(selectedFiles, metadata)

	// Сортируем для стабильного вывода
	sort.Strings(selectedFiles)

	s.stats.SelectedFiles = len(selectedFiles)
	s.stats.SkippedFiles = s.stats.TotalFiles - s.stats.SelectedFiles

	s.logger.Info("File selection completed",
		"selected", s.stats.SelectedFiles,
		"skipped", s.stats.SkippedFiles,
		"filtered_by_rules", s.stats.FilteredByRules,
		"filtered_by_size", s.stats.FilteredBySize)

	return selectedFiles, nil
}

// selectAll выбирает все файлы с учетом исключений
func (s *SelectionPhase) selectAll(ctx context.Context, metadata *model.ProjectMetadata) ([]string, error) {
	selected := make([]string, 0, metadata.TotalFiles)

	for path, fileMeta := range metadata.Files {
		// Проверяем исключения
		if s.shouldExclude(path, fileMeta) {
			s.stats.FilteredByRules++
			continue
		}

		// Проверяем размер
		if fileMeta.Size > s.config.ScanConfig.MaxFileSize {
			s.stats.FilteredBySize++
			continue
		}

		selected = append(selected, path)
	}

	return selected, nil
}

// selectFromList выбирает файлы из заданного списка
func (s *SelectionPhase) selectFromList(ctx context.Context, metadata *model.ProjectMetadata) ([]string, error) {
	selected := make([]string, 0)

	// Создаем map для быстрого поиска
	includedMap := make(map[string]bool)
	for _, path := range s.config.ScanConfig.IncludedPaths {
		includedMap[path] = true
	}

	for path, fileMeta := range metadata.Files {
		// Проверяем, включен ли файл в список
		included := false

		// Точное совпадение
		if includedMap[path] {
			included = true
		}

		// Проверяем, находится ли файл в включенной директории
		if !included {
			for _, includedPath := range s.config.ScanConfig.IncludedPaths {
				// Если includedPath - директория
				if strings.HasPrefix(path, includedPath+"/") {
					included = true
					break
				}
			}
		}

		if !included {
			continue
		}

		// Дополнительные проверки
		if s.shouldExclude(path, fileMeta) {
			s.stats.FilteredByRules++
			continue
		}

		if fileMeta.Size > s.config.ScanConfig.MaxFileSize {
			s.stats.FilteredBySize++
			continue
		}

		selected = append(selected, path)
	}

	return selected, nil
}

// selectByPatterns выбирает файлы по паттернам
func (s *SelectionPhase) selectByPatterns(ctx context.Context, metadata *model.ProjectMetadata) ([]string, error) {
	selected := make([]string, 0)

	for path, fileMeta := range metadata.Files {
		// Проверяем соответствие паттернам включения
		matched := false
		for _, pattern := range s.config.ScanConfig.IncludePatterns {
			if s.matchPattern(path, pattern) {
				matched = true
				break
			}
		}

		if !matched && len(s.config.ScanConfig.IncludePatterns) > 0 {
			continue
		}

		// Проверяем исключения
		if s.shouldExclude(path, fileMeta) {
			s.stats.FilteredByRules++
			continue
		}

		if fileMeta.Size > s.config.ScanConfig.MaxFileSize {
			s.stats.FilteredBySize++
			continue
		}

		selected = append(selected, path)
	}

	return selected, nil
}

// selectSmart выбирает файлы на основе анализа проекта
func (s *SelectionPhase) selectSmart(ctx context.Context, metadata *model.ProjectMetadata, info *model.ProjectInfo) ([]string, error) {
	selected := make([]string, 0)

	// Приоритеты файлов
	priorities := s.calculateFilePriorities(metadata, info)

	// Сортируем файлы по приоритету
	type filePriority struct {
		path     string
		priority float32
	}

	sortedFiles := make([]filePriority, 0, len(priorities))
	for path, priority := range priorities {
		sortedFiles = append(sortedFiles, filePriority{path, priority})
	}

	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].priority > sortedFiles[j].priority
	})

	// Выбираем файлы с учетом лимитов
	totalSize := int64(0)
	maxTotalSize := s.config.ScanConfig.MaxTotalSize
	if maxTotalSize == 0 {
		maxTotalSize = 100 * 1024 * 1024 // 100MB по умолчанию
	}

	for _, fp := range sortedFiles {
		fileMeta := metadata.Files[fp.path]

		// Проверяем размер
		if totalSize+fileMeta.Size > maxTotalSize {
			s.logger.Debug("Reached total size limit", "current", totalSize, "limit", maxTotalSize)
			break
		}

		// Проверяем индивидуальный размер файла
		if fileMeta.Size > s.config.ScanConfig.MaxFileSize {
			s.stats.FilteredBySize++
			continue
		}

		// Проверяем исключения
		if s.shouldExclude(fp.path, fileMeta) {
			s.stats.FilteredByRules++
			continue
		}

		selected = append(selected, fp.path)
		totalSize += fileMeta.Size

		// Проверяем лимит по количеству файлов
		if s.config.ScanConfig.MaxFiles > 0 && len(selected) >= s.config.ScanConfig.MaxFiles {
			s.logger.Debug("Reached max files limit", "limit", s.config.ScanConfig.MaxFiles)
			break
		}
	}

	return selected, nil
}

// calculateFilePriorities рассчитывает приоритеты файлов
func (s *SelectionPhase) calculateFilePriorities(metadata *model.ProjectMetadata, info *model.ProjectInfo) map[string]float32 {
	priorities := make(map[string]float32)

	for path, fileMeta := range metadata.Files {
		priority := float32(0.5) // Базовый приоритет

		// Важные файлы
		if s.isImportantFile(path, fileMeta) {
			priority += 0.3
		}

		// Файлы в важных компонентах
		for _, component := range info.Components {
			if component.Important && strings.HasPrefix(path, component.Path) {
				priority += 0.2
				break
			}
		}

		// Entry points
		for _, entryPoint := range info.EntryPoints {
			if path == entryPoint {
				priority += 0.4
				break
			}
		}

		// Конфигурационные файлы
		if s.isConfigFile(path, fileMeta) {
			priority += 0.2
		}

		// Основной исходный код
		if s.isMainSourceFile(path, fileMeta) {
			priority += 0.3
		}

		// Тесты (низкий приоритет по умолчанию)
		if s.isTestFile(path, fileMeta) {
			priority -= 0.3
		}

		// Сгенерированные файлы (очень низкий приоритет)
		if s.isGeneratedFile(path, fileMeta) {
			priority -= 0.5
		}

		// Документация
		if s.isDocFile(path, fileMeta) {
			if s.config.ScanConfig.IncludeDocs {
				priority += 0.1
			} else {
				priority -= 0.4
			}
		}

		// Размер файла (предпочитаем меньшие файлы)
		if fileMeta.Size < 10*1024 { // < 10KB
			priority += 0.1
		} else if fileMeta.Size > 100*1024 { // > 100KB
			priority -= 0.2
		}

		priorities[path] = priority
	}

	return priorities
}

// applyPluginFilters применяет фильтры из плагинов
func (s *SelectionPhase) applyPluginFilters(files []string, metadata *model.ProjectMetadata) []string {
	filterPlugins := s.registry.GetFilterPlugins()
	if len(filterPlugins) == 0 {
		return files
	}

	filtered := make([]string, 0, len(files))

	for _, path := range files {
		fileMeta := metadata.Files[path]

		shouldInclude := true
		var excludeReason string

		// Проверяем все фильтр-плагины
		for _, filterPlugin := range filterPlugins {
			// Проверяем исключение
			if exclude, reason := filterPlugin.ShouldExclude(path, fileMeta); exclude {
				shouldInclude = false
				excludeReason = reason
				break
			}

			// Проверяем принудительное включение
			if include, _ := filterPlugin.ShouldInclude(path, fileMeta); include {
				shouldInclude = true
				break
			}
		}

		if shouldInclude {
			filtered = append(filtered, path)
		} else {
			s.logger.Debug("File filtered by plugin", "path", path, "reason", excludeReason)
			s.stats.FilteredByRules++
		}
	}

	return filtered
}

// applyLoadingRules применяет правила загрузки
func (s *SelectionPhase) applyLoadingRules(files []string, metadata *model.ProjectMetadata) []string {
	if len(s.config.LoadingRules) == 0 {
		return files
	}

	// Сортируем правила по приоритету
	sort.Slice(s.config.LoadingRules, func(i, j int) bool {
		return s.config.LoadingRules[i].Priority > s.config.LoadingRules[j].Priority
	})

	filtered := make([]string, 0, len(files))

	for _, path := range files {
		fileMeta := metadata.Files[path]

		// Проверяем правила
		include := true
		for _, rule := range s.config.LoadingRules {
			// Проверяем паттерн
			matched := false
			if rule.Pattern != "" {
				matched = s.matchPattern(path, rule.Pattern)
			}

			for _, pattern := range rule.Patterns {
				if s.matchPattern(path, pattern) {
					matched = true
					break
				}
			}

			if matched {
				// Применяем правило
				switch rule.Mode {
				case model.ContentModeNone:
					include = false
				case model.ContentModeReference:
					// Только ссылка - пропускаем из выборки
					include = false
				default:
					// Включаем файл
					include = true
				}

				// Проверяем размер
				if rule.MaxSize > 0 && fileMeta.Size > rule.MaxSize {
					include = false
				}

				break // Применяем первое подходящее правило
			}
		}

		if include {
			filtered = append(filtered, path)
		} else {
			s.stats.FilteredByRules++
		}
	}

	return filtered
}

// applyLimits применяет ограничения на количество и размер файлов
func (s *SelectionPhase) applyLimits(files []string, metadata *model.ProjectMetadata) []string {
	if s.config.ScanConfig.MaxFiles <= 0 && s.config.ScanConfig.MaxTotalSize <= 0 {
		return files
	}

	limited := make([]string, 0, len(files))
	totalSize := int64(0)

	for _, path := range files {
		fileMeta := metadata.Files[path]

		// Проверяем лимит по количеству
		if s.config.ScanConfig.MaxFiles > 0 && len(limited) >= s.config.ScanConfig.MaxFiles {
			s.logger.Info("Reached max files limit", "limit", s.config.ScanConfig.MaxFiles)
			break
		}

		// Проверяем лимит по общему размеру
		if s.config.ScanConfig.MaxTotalSize > 0 && totalSize+fileMeta.Size > s.config.ScanConfig.MaxTotalSize {
			s.logger.Info("Reached total size limit", "limit", s.config.ScanConfig.MaxTotalSize)
			break
		}

		limited = append(limited, path)
		totalSize += fileMeta.Size
	}

	return limited
}

// Helper methods

func (s *SelectionPhase) shouldExclude(path string, meta *model.FileMetadata) bool {
	// Проверяем правила исключения
	for _, rule := range s.config.ExclusionRules {
		if rule.Pattern != "" && s.matchPattern(path, rule.Pattern) {
			return true
		}

		for _, pattern := range rule.Patterns {
			if s.matchPattern(path, pattern) {
				return true
			}
		}
	}

	// Проверяем паттерны исключения из конфига
	for _, pattern := range s.config.ScanConfig.ExcludePatterns {
		if s.matchPattern(path, pattern) {
			return true
		}
	}

	// Проверяем типы файлов
	if !s.config.ScanConfig.IncludeTests && s.isTestFile(path, meta) {
		s.stats.FilteredByType++
		return true
	}

	if !s.config.ScanConfig.IncludeConfigs && s.isConfigFile(path, meta) {
		s.stats.FilteredByType++
		return true
	}

	if !s.config.ScanConfig.IncludeDocs && s.isDocFile(path, meta) {
		s.stats.FilteredByType++
		return true
	}

	if !s.config.ScanConfig.IncludeGenerated && s.isGeneratedFile(path, meta) {
		s.stats.FilteredByType++
		return true
	}

	return false
}

func (s *SelectionPhase) matchPattern(path, pattern string) bool {
	// Поддержка ** для рекурсивного поиска
	if strings.Contains(pattern, "**") {
		// Простая реализация - можно улучшить
		pattern = strings.ReplaceAll(pattern, "**", "*")
	}

	matched, _ := filepath.Match(pattern, path)
	return matched
}

func (s *SelectionPhase) isImportantFile(path string, meta *model.FileMetadata) bool {
	// Проверяем список важных файлов
	for _, important := range s.config.ScanConfig.ImportantFiles {
		if path == important || filepath.Base(path) == important {
			return true
		}
	}

	// Эвристики для определения важности
	importantNames := []string{
		"main", "index", "app", "server", "client",
		"core", "base", "root", "entry", "bootstrap",
	}

	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, name := range importantNames {
		if strings.Contains(strings.ToLower(baseName), name) {
			return true
		}
	}

	return false
}

func (s *SelectionPhase) isConfigFile(path string, meta *model.FileMetadata) bool {
	configExts := []string{
		".json", ".yaml", ".yml", ".toml", ".ini",
		".cfg", ".conf", ".config", ".properties",
	}

	ext := strings.ToLower(meta.Extension)
	for _, configExt := range configExts {
		if ext == configExt {
			return true
		}
	}

	configNames := []string{
		"config", "settings", "env", ".env",
		"dockerfile", "docker-compose",
		"makefile", "package.json", "tsconfig.json",
	}

	baseName := strings.ToLower(filepath.Base(path))
	for _, name := range configNames {
		if strings.Contains(baseName, name) {
			return true
		}
	}

	return false
}

func (s *SelectionPhase) isMainSourceFile(path string, meta *model.FileMetadata) bool {
	// Проверяем, является ли файл основным исходным кодом
	sourceExts := []string{
		".go", ".js", ".ts", ".jsx", ".tsx",
		".py", ".java", ".c", ".cpp", ".cs",
		".rs", ".rb", ".php", ".swift", ".kt",
	}

	ext := strings.ToLower(meta.Extension)
	for _, srcExt := range sourceExts {
		if ext == srcExt {
			// Исключаем тесты и сгенерированные файлы
			if !s.isTestFile(path, meta) && !s.isGeneratedFile(path, meta) {
				return true
			}
		}
	}

	return false
}

func (s *SelectionPhase) isTestFile(path string, meta *model.FileMetadata) bool {
	testPatterns := []string{
		"_test.go", ".test.", ".spec.", "_spec.",
		"test_", "spec_", "/test/", "/tests/",
		"/__tests__/", "/spec/", "/specs/",
	}

	pathLower := strings.ToLower(path)
	for _, pattern := range testPatterns {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}

	return false
}

func (s *SelectionPhase) isGeneratedFile(path string, meta *model.FileMetadata) bool {
	generatedPatterns := []string{
		".generated.", ".gen.", "_gen.",
		".pb.go", ".pb.cc", ".pb.h",
		"_generated", "/generated/", "/gen/",
		".min.js", ".min.css",
	}

	pathLower := strings.ToLower(path)
	for _, pattern := range generatedPatterns {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}

	return false
}

func (s *SelectionPhase) isDocFile(path string, meta *model.FileMetadata) bool {
	docExts := []string{
		".md", ".markdown", ".rst", ".txt",
		".adoc", ".asciidoc", ".org",
	}

	ext := strings.ToLower(meta.Extension)
	for _, docExt := range docExts {
		if ext == docExt {
			return true
		}
	}

	// Проверяем директории документации
	docDirs := []string{
		"/docs/", "/documentation/", "/doc/",
		"README", "CHANGELOG", "LICENSE",
	}

	pathUpper := strings.ToUpper(path)
	for _, pattern := range docDirs {
		if strings.Contains(pathUpper, pattern) {
			return true
		}
	}

	return false
}

// GetStats возвращает статистику выбора
func (s *SelectionPhase) GetStats() *SelectionStats {
	return s.stats
}