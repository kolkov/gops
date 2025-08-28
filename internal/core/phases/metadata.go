// Package phases содержит фазу сбора метаданных
package phases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
)

// MetadataPhase - фаза сбора метаданных без чтения контента файлов
type MetadataPhase struct {
	rootDir string
	config  model.ScanConfig
	logger  *logger.Logger

	// Для параллельной обработки
	workerPool chan struct{}
	mu         sync.Mutex
	wg         sync.WaitGroup
}

// NewMetadataPhase создает новую фазу сбора метаданных
func NewMetadataPhase(rootDir string, config model.ScanConfig, logger *logger.Logger) *MetadataPhase {
	workers := config.ParallelWorkers
	if workers <= 0 {
		workers = 4
	}

	return &MetadataPhase{
		rootDir:    rootDir,
		config:     config,
		logger:     logger,
		workerPool: make(chan struct{}, workers),
	}
}

// Collect собирает метаданные проекта
func (m *MetadataPhase) Collect(ctx context.Context) (*model.ProjectMetadata, error) {
	startTime := time.Now()

	// Проверяем существование корневой директории
	rootInfo, err := os.Stat(m.rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat root directory: %w", err)
	}

	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", m.rootDir)
	}

	// Создаем корневую метадату
	rootMeta := &model.FileMetadata{
		Path:     m.rootDir,
		Name:     filepath.Base(m.rootDir),
		Dir:      filepath.Dir(m.rootDir),
		IsDir:    true,
		Size:     rootInfo.Size(),
		ModTime:  rootInfo.ModTime(),
		Children: make([]*model.FileMetadata, 0),
	}

	// Создаем метаданные проекта
	metadata := &model.ProjectMetadata{
		RootPath:   m.rootDir,
		Name:       filepath.Base(m.rootDir),
		Root:       rootMeta,
		Files:      make(map[string]*model.FileMetadata),
		FilesByExt: make(map[string]int),
		FilesByDir: make(map[string]int),
		ScanTime:   startTime,
	}

	// Собираем метаданные рекурсивно
	if err := m.collectRecursive(ctx, rootMeta, metadata); err != nil {
		return nil, fmt.Errorf("failed to collect metadata: %w", err)
	}

	// Определяем VCS информацию
	metadata.VCSInfo = m.detectVCS()

	// Определяем build систему
	metadata.BuildInfo = m.detectBuildSystem(metadata)

	// Завершаем сбор
	metadata.ScanDuration = time.Since(startTime)

	m.logger.Info("Metadata collection completed",
		"files", metadata.TotalFiles,
		"dirs", metadata.TotalDirs,
		"size", metadata.TotalSize,
		"duration", metadata.ScanDuration)

	return metadata, nil
}

// collectRecursive рекурсивно собирает метаданные
func (m *MetadataPhase) collectRecursive(ctx context.Context, parent *model.FileMetadata, projectMeta *model.ProjectMetadata) error {
	// Проверяем отмену контекста
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Читаем содержимое директории
	entries, err := os.ReadDir(parent.Path)
	if err != nil {
		// Логируем ошибку, но продолжаем
		m.logger.Warn("Failed to read directory", "path", parent.Path, "error", err)
		return nil
	}

	// Обрабатываем записи
	for _, entry := range entries {
		// Проверяем отмену контекста
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fullPath := filepath.Join(parent.Path, entry.Name())
		relPath, _ := filepath.Rel(m.rootDir, fullPath)

		// Проверяем исключения
		if m.shouldSkip(entry.Name(), relPath, entry.IsDir()) {
			continue
		}

		// Получаем информацию о файле
		info, err := entry.Info()
		if err != nil {
			m.logger.Warn("Failed to get file info", "path", fullPath, "error", err)
			continue
		}

		// Создаем метаданные
		meta := &model.FileMetadata{
			Path:      fullPath,
			Name:      entry.Name(),
			Dir:       parent.Path,
			Extension: filepath.Ext(entry.Name()),
			Size:      info.Size(),
			IsDir:     entry.IsDir(),
			IsHidden:  strings.HasPrefix(entry.Name(), "."),
			ModTime:   info.ModTime(),
			Parent:    parent,
			Processed: false,
		}

		// Определяем режим загрузки по умолчанию
		meta.LoadingMode = m.determineLoadingMode(meta)

		// Добавляем к родителю
		parent.Children = append(parent.Children, meta)

		// Обновляем статистику
		m.mu.Lock()
		if meta.IsDir {
			projectMeta.TotalDirs++
			meta.Children = make([]*model.FileMetadata, 0)

			// Рекурсивно обрабатываем поддиректорию
			m.mu.Unlock()
			if err := m.collectRecursive(ctx, meta, projectMeta); err != nil {
				return err
			}
		} else {
			projectMeta.TotalFiles++
			projectMeta.TotalSize += meta.Size

			// Обновляем статистику по расширениям
			if meta.Extension != "" {
				projectMeta.FilesByExt[meta.Extension]++
			} else {
				projectMeta.FilesByExt["[no-ext]"]++
			}

			// Обновляем статистику по директориям
			dirName := filepath.Base(meta.Dir)
			projectMeta.FilesByDir[dirName]++

			// Добавляем в индекс файлов
			projectMeta.Files[relPath] = meta

			m.mu.Unlock()
		}
	}

	return nil
}

// shouldSkip определяет, нужно ли пропустить файл/директорию
func (m *MetadataPhase) shouldSkip(name, relPath string, isDir bool) bool {
	// Всегда пропускаем файлы конфигурации gops
	if strings.Contains(strings.ToLower(name), "gops_config") {
		return true
	}

	// Системные исключения для директорий
	if isDir {
		systemDirExcludes := []string{
			".git", ".svn", ".hg", ".bzr",
			"node_modules", "vendor", "bower_components",
			"dist", "build", "out", "target",
			"__pycache__", ".pytest_cache",
			".idea", ".vscode", ".vs",
			"bin", "obj",
		}

		for _, exclude := range systemDirExcludes {
			if name == exclude {
				return true
			}
		}
	}

	// Проверяем пользовательские паттерны исключения
	for _, pattern := range m.config.ExcludePatterns {
		// Поддержка glob паттернов
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}

		// Поддержка паттернов с путями
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
	}

	// Проверяем максимальную глубину
	if m.config.MaxDepth > 0 {
		depth := strings.Count(relPath, string(os.PathSeparator))
		if depth > m.config.MaxDepth {
			return true
		}
	}

	return false
}

// determineLoadingMode определяет режим загрузки для файла
func (m *MetadataPhase) determineLoadingMode(meta *model.FileMetadata) model.ContentMode {
	// Большие файлы - только ссылка
	if meta.Size > m.config.MaxFileSize {
		return model.ContentModeReference
	}

	// Бинарные файлы - пропускаем
	if m.isBinary(meta.Extension) {
		return model.ContentModeNone
	}

	// Генерированные файлы
	if m.isGenerated(meta.Name, meta.Path) {
		if m.config.IncludeGenerated {
			return model.ContentModeHeaders
		}
		return model.ContentModeNone
	}

	// Тестовые файлы
	if m.isTest(meta.Name, meta.Path) {
		if m.config.IncludeTests {
			return model.ContentModeHeaders
		}
		return model.ContentModeNone
	}

	// Конфигурационные файлы
	if m.isConfig(meta.Name, meta.Extension) {
		if m.config.IncludeConfigs {
			return model.ContentModeFull
		}
		return model.ContentModeNone
	}

	// Документация
	if m.isDoc(meta.Extension) {
		if m.config.IncludeDocs {
			return model.ContentModeFull
		}
		return model.ContentModeNone
	}

	// Стили
	if m.isStyle(meta.Extension) {
		if m.config.IncludeStyles {
			return model.ContentModeHeaders
		}
		return model.ContentModeNone
	}

	// Разметка
	if m.isMarkup(meta.Extension) {
		if m.config.IncludeMarkup {
			return model.ContentModeHeaders
		}
		return model.ContentModeNone
	}

	// Исходный код - по умолчанию headers
	if m.isSourceCode(meta.Extension) {
		return model.ContentModeHeaders
	}

	// Все остальное - полный контент
	return model.ContentModeFull
}

// Вспомогательные методы для определения типов файлов

func (m *MetadataPhase) isBinary(ext string) bool {
	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib", ".a", ".o",
		".class", ".jar", ".war", ".ear",
		".pyc", ".pyo",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".svg",
		".mp3", ".mp4", ".avi", ".mov", ".wav",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".ttf", ".otf", ".woff", ".woff2", ".eot",
	}

	ext = strings.ToLower(ext)
	for _, binExt := range binaryExts {
		if ext == binExt {
			return true
		}
	}
	return false
}

func (m *MetadataPhase) isGenerated(name, path string) bool {
	// Паттерны генерированных файлов
	patterns := []string{
		"*.generated.*",
		"*.gen.*",
		"*.pb.go",
		"*.pb.cc",
		"*.pb.h",
		"*_gen.go",
		"*_generated.go",
	}

	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	// Проверяем путь на генерированные директории
	if strings.Contains(path, "/generated/") || strings.Contains(path, "/gen/") {
		return true
	}

	return false
}

func (m *MetadataPhase) isTest(name, path string) bool {
	// Тестовые паттерны
	patterns := []string{
		"*_test.go",
		"*.test.js",
		"*.test.ts",
		"*.spec.js",
		"*.spec.ts",
		"test_*.py",
		"*Test.java",
		"*.test.cpp",
		"*.test.c",
	}

	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	// Проверяем тестовые директории
	testDirs := []string{"/test/", "/tests/", "/__tests__/", "/spec/", "/specs/"}
	for _, dir := range testDirs {
		if strings.Contains(path, dir) {
			return true
		}
	}

	return false
}

func (m *MetadataPhase) isConfig(name, ext string) bool {
	// Конфигурационные расширения
	configExts := []string{".json", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf", ".config", ".properties"}
	ext = strings.ToLower(ext)
	for _, configExt := range configExts {
		if ext == configExt {
			return true
		}
	}

	// Конфигурационные имена
	configNames := []string{
		"package.json", "tsconfig.json", "webpack.config.js",
		"Dockerfile", "docker-compose.yml", "docker-compose.yaml",
		"Makefile", "makefile", "GNUmakefile",
		".env", ".env.local", ".env.production",
		"pom.xml", "build.gradle", "settings.gradle",
		"Cargo.toml", "go.mod", "go.sum",
		"requirements.txt", "setup.py", "setup.cfg",
		"Gemfile", "Gemfile.lock",
	}

	nameLower := strings.ToLower(name)
	for _, configName := range configNames {
		if nameLower == strings.ToLower(configName) {
			return true
		}
	}

	return false
}

func (m *MetadataPhase) isDoc(ext string) bool {
	docExts := []string{".md", ".markdown", ".rst", ".txt", ".adoc", ".asciidoc"}
	ext = strings.ToLower(ext)
	for _, docExt := range docExts {
		if ext == docExt {
			return true
		}
	}
	return false
}

func (m *MetadataPhase) isStyle(ext string) bool {
	styleExts := []string{".css", ".scss", ".sass", ".less", ".styl"}
	ext = strings.ToLower(ext)
	for _, styleExt := range styleExts {
		if ext == styleExt {
			return true
		}
	}
	return false
}

func (m *MetadataPhase) isMarkup(ext string) bool {
	markupExts := []string{".html", ".htm", ".xhtml", ".xml", ".svg"}
	ext = strings.ToLower(ext)
	for _, markupExt := range markupExts {
		if ext == markupExt {
			return true
		}
	}
	return false
}

func (m *MetadataPhase) isSourceCode(ext string) bool {
	sourceExts := []string{
		// Go
		".go",
		// JavaScript/TypeScript
		".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs",
		// Python
		".py", ".pyw", ".pyx",
		// Java/Kotlin
		".java", ".kt", ".kts",
		// C/C++
		".c", ".h", ".cc", ".cpp", ".cxx", ".hpp", ".hxx",
		// C#
		".cs", ".csx",
		// Rust
		".rs",
		// Ruby
		".rb",
		// PHP
		".php",
		// Swift
		".swift",
		// Objective-C
		".m", ".mm",
		// Shell
		".sh", ".bash", ".zsh", ".fish",
		// Others
		".scala", ".clj", ".ex", ".exs", ".erl", ".hrl",
		".lua", ".pl", ".pm", ".r", ".R", ".dart",
	}

	ext = strings.ToLower(ext)
	for _, sourceExt := range sourceExts {
		if ext == sourceExt {
			return true
		}
	}
	return false
}

// detectVCS определяет систему контроля версий
func (m *MetadataPhase) detectVCS() *model.VCSInfo {
	// Проверяем Git
	gitDir := filepath.Join(m.rootDir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return &model.VCSInfo{
			Type: "git",
			// TODO: читать branch, commit и т.д. из .git
		}
	}

	// Проверяем SVN
	svnDir := filepath.Join(m.rootDir, ".svn")
	if info, err := os.Stat(svnDir); err == nil && info.IsDir() {
		return &model.VCSInfo{
			Type: "svn",
		}
	}

	// Проверяем Mercurial
	hgDir := filepath.Join(m.rootDir, ".hg")
	if info, err := os.Stat(hgDir); err == nil && info.IsDir() {
		return &model.VCSInfo{
			Type: "hg",
		}
	}

	return nil
}

// detectBuildSystem определяет систему сборки
func (m *MetadataPhase) detectBuildSystem(metadata *model.ProjectMetadata) *model.BuildInfo {
	// Проверяем различные системы сборки по файлам-маркерам

	// Maven
	if _, exists := metadata.Files["pom.xml"]; exists {
		return &model.BuildInfo{System: "maven"}
	}

	// Gradle
	if _, exists := metadata.Files["build.gradle"]; exists {
		return &model.BuildInfo{System: "gradle"}
	}
	if _, exists := metadata.Files["build.gradle.kts"]; exists {
		return &model.BuildInfo{System: "gradle-kotlin"}
	}

	// NPM/Yarn
	if _, exists := metadata.Files["package.json"]; exists {
		if _, yarnExists := metadata.Files["yarn.lock"]; yarnExists {
			return &model.BuildInfo{System: "yarn"}
		}
		return &model.BuildInfo{System: "npm"}
	}

	// Go modules
	if _, exists := metadata.Files["go.mod"]; exists {
		return &model.BuildInfo{System: "go-modules"}
	}

	// Cargo (Rust)
	if _, exists := metadata.Files["Cargo.toml"]; exists {
		return &model.BuildInfo{System: "cargo"}
	}

	// Make
	for _, name := range []string{"Makefile", "makefile", "GNUmakefile"} {
		if _, exists := metadata.Files[name]; exists {
			return &model.BuildInfo{System: "make"}
		}
	}

	// CMake
	if _, exists := metadata.Files["CMakeLists.txt"]; exists {
		return &model.BuildInfo{System: "cmake"}
	}

	return nil
}