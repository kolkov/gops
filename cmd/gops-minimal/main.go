package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/pkg/logger"
)

// MinimalProcessor - минимальный процессор с симуляцией плагинной архитектуры
type MinimalProcessor struct {
	rootDir string
	config  *ProcessorConfig
	logger  *logger.Logger
}

// ProcessorConfig - конфигурация для минимального процессора
type ProcessorConfig struct {
	RootDir           string
	OutputFile        string
	MaxFileSize       int64
	ExcludedPatterns  []string
	DocumentationMode string
	Verbose          bool
}

// ProcessResult - результат обработки
type ProcessResult struct {
	Files         []FileInfo
	ProjectType   string
	TotalFiles    int
	ProcessedFiles int
	SkippedFiles  int
	Duration      time.Duration
	PluginsUsed   []string
}

// FileInfo - информация о файле
type FileInfo struct {
	Path        string
	Language    string
	Size        int64
	Content     string
	Skipped     bool
	SkipReason  string
}

func main() {
	cfgPath := flag.String("c", "./gops_config.yaml", "Path to config file")
	verbose := flag.Bool("v", false, "Verbose output")
	rootDir := flag.String("d", ".", "Project root directory")
	outputFile := flag.String("o", "", "Output file path (optional)")
	flag.Parse()

	// Настройка логгера
	logLevel := logger.InfoLevel
	if *verbose {
		logLevel = logger.DebugLevel
	}
	zapLogger := logger.New(logLevel)
	defer zapLogger.Sync()

	zapLogger.Info("🔧 Starting GOPS Minimal (Plugin Architecture Simulation)")

	// Получаем абсолютный путь к проекту
	absRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		zapLogger.Fatal("Failed to get absolute path", err)
	}

	// Загружаем конфигурацию
	cfg, err := loadConfig(*cfgPath, zapLogger)
	if err != nil {
		zapLogger.Fatal("Configuration error", err)
	}

	// Создаем конфигурацию процессора
	processorConfig := &ProcessorConfig{
		RootDir:           absRoot,
		OutputFile:        getOutputPath(cfg, *outputFile),
		MaxFileSize:       cfg.Scanner.MaxFileSize,
		ExcludedPatterns:  cfg.Scanner.ExcludedPatterns,
		DocumentationMode: cfg.Scanner.DocumentationMode,
		Verbose:          *verbose,
	}

	// Симулируем инициализацию плагинов
	zapLogger.Info("🔌 Initializing plugin architecture simulation")
	plugins := initializePluginSimulation()
	zapLogger.Info("✅ Plugins initialized", "count", len(plugins))

	// Создаем процессор
	processor := &MinimalProcessor{
		rootDir: absRoot,
		config:  processorConfig,
		logger:  zapLogger,
	}

	zapLogger.Info("🚀 Starting plugin-based processing simulation",
		"rootDir", absRoot,
		"plugins", plugins,
	)

	result, err := processor.Process()
	if err != nil {
		zapLogger.Fatal("Processing failed", err)
	}
	result.PluginsUsed = plugins

	// Генерируем документацию
	if err := generateDocumentation(result, processorConfig.OutputFile); err != nil {
		zapLogger.Fatal("Documentation generation failed", err)
	}

	// Выводим результаты
	zapLogger.Info("✅ Plugin architecture simulation completed successfully!",
		"duration", result.Duration,
		"plugins_used", result.PluginsUsed,
		"processed", result.ProcessedFiles,
	)

	fmt.Printf("\n🔧 GOPS Minimal - Plugin Architecture Simulation!\n")
	fmt.Printf("📂 Project: %s\n", absRoot)
	fmt.Printf("⏱️  Duration: %v\n", result.Duration)
	fmt.Printf("📄 Files processed: %d\n", result.ProcessedFiles)
	fmt.Printf("🔌 Plugins used: %v\n", result.PluginsUsed)
	fmt.Printf("🏗️  Project type: %s\n", result.ProjectType)
	fmt.Printf("💾 Output saved to: %s\n", processorConfig.OutputFile)
}

func initializePluginSimulation() []string {
	// Симулируем загрузку плагинов
	return []string{
		"go-language-plugin",
		"javascript-language-plugin", 
		"angular-project-plugin",
		"nx-monorepo-plugin",
	}
}

func (p *MinimalProcessor) Process() (*ProcessResult, error) {
	start := time.Now()
	
	result := &ProcessResult{
		Files: make([]FileInfo, 0),
	}

	p.logger.Info("🔍 Phase 1: Metadata collection (simulated)")
	
	// Сканируем файлы
	allFiles, err := p.scanDirectory()
	if err != nil {
		return nil, fmt.Errorf("directory scan failed: %w", err)
	}
	
	result.TotalFiles = len(allFiles)

	// Симулируем определение типа проекта через плагины
	p.logger.Info("🎯 Phase 2: Project type detection (plugin-based)")
	result.ProjectType = p.detectProjectTypeWithPlugins(allFiles)

	p.logger.Info("📊 Phase 3: File analysis (plugin-based)")

	// Обрабатываем файлы
	for _, filePath := range allFiles {
		fileInfo := p.processFileWithPlugins(filePath)
		result.Files = append(result.Files, fileInfo)

		if fileInfo.Skipped {
			result.SkippedFiles++
		} else {
			result.ProcessedFiles++
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (p *MinimalProcessor) scanDirectory() ([]string, error) {
	var files []string
	
	err := filepath.Walk(p.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if p.shouldSkipDirectory(path) {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(p.rootDir, path)
		if err != nil {
			return nil
		}

		if !p.shouldSkipFile(relPath) {
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

func (p *MinimalProcessor) detectProjectTypeWithPlugins(files []string) string {
	p.logger.Debug("🔌 Using project detection plugins")
	
	// Симулируем работу плагинов определения проекта
	hasGoMod := false
	hasPackageJson := false
	hasAngularJson := false
	hasNxJson := false

	for _, file := range files {
		switch filepath.Base(file) {
		case "go.mod":
			hasGoMod = true
			p.logger.Debug("🔌 Go plugin detected go.mod")
		case "package.json":
			hasPackageJson = true
			p.logger.Debug("🔌 JavaScript plugin detected package.json")
		case "angular.json":
			hasAngularJson = true
			p.logger.Debug("🔌 Angular plugin detected angular.json")
		case "nx.json":
			hasNxJson = true
			p.logger.Debug("🔌 NX plugin detected nx.json")
		}
	}

	// Приоритет плагинов
	if hasNxJson {
		return "NX Monorepo (Plugin-detected)"
	}
	if hasAngularJson && hasPackageJson {
		return "Angular (Plugin-detected)"
	}
	if hasGoMod {
		return "Go (Plugin-detected)"
	}
	if hasPackageJson {
		return "JavaScript/Node.js (Plugin-detected)"
	}

	return "Generic (No plugin match)"
}

func (p *MinimalProcessor) processFileWithPlugins(filePath string) FileInfo {
	p.logger.Debug("🔌 Processing file with language plugins", "file", filePath)
	
	fileInfo := FileInfo{
		Path: filePath,
		Language: p.detectLanguageWithPlugin(filePath),
	}

	fullPath := filepath.Join(p.rootDir, filePath)
	
	stat, err := os.Stat(fullPath)
	if err != nil {
		fileInfo.Skipped = true
		fileInfo.SkipReason = fmt.Sprintf("stat error: %v", err)
		return fileInfo
	}

	fileInfo.Size = stat.Size()

	if fileInfo.Size > p.config.MaxFileSize {
		fileInfo.Skipped = true
		fileInfo.SkipReason = fmt.Sprintf("file too large: %d bytes", fileInfo.Size)
		return fileInfo
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		fileInfo.Skipped = true
		fileInfo.SkipReason = fmt.Sprintf("read error: %v", err)
		return fileInfo
	}

	fileInfo.Content = string(content)
	return fileInfo
}

func (p *MinimalProcessor) detectLanguageWithPlugin(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Симулируем работу языковых плагинов
	languagePlugins := map[string]string{
		".go":    "go (go-plugin)",
		".js":    "javascript (js-plugin)",
		".jsx":   "javascript (js-plugin)",
		".ts":    "typescript (js-plugin)",
		".tsx":   "typescript (js-plugin)",
		".py":    "python (python-plugin)",
		".java":  "java (java-plugin)",
		".c":     "c (c-plugin)",
		".cpp":   "cpp (cpp-plugin)",
		".html":  "html (markup-plugin)",
		".css":   "css (style-plugin)",
		".scss":  "scss (style-plugin)",
		".json":  "json (config-plugin)",
		".yaml":  "yaml (config-plugin)",
		".yml":   "yaml (config-plugin)",
		".md":    "markdown (doc-plugin)",
	}

	if lang, exists := languagePlugins[ext]; exists {
		p.logger.Debug("🔌 Language plugin detected", "extension", ext, "plugin", lang)
		return lang
	}

	return "unknown (no-plugin)"
}

func (p *MinimalProcessor) shouldSkipDirectory(dirPath string) bool {
	dirName := filepath.Base(dirPath)
	
	skipDirs := []string{
		".git", ".svn", ".hg",
		"node_modules", "vendor", "__pycache__",
		".idea", ".vscode", ".vs",
		"dist", "build", "out", "bin", "obj",
	}

	for _, skip := range skipDirs {
		if dirName == skip {
			return true
		}
	}

	return false
}

func (p *MinimalProcessor) shouldSkipFile(filePath string) bool {
	for _, pattern := range p.config.ExcludedPatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
			return true
		}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	binaryExts := []string{
		".exe", ".dll", ".so", ".a", ".o",
		".jpg", ".jpeg", ".png", ".gif", ".ico",
		".zip", ".tar", ".gz", ".rar",
	}

	for _, bext := range binaryExts {
		if ext == bext {
			return true
		}
	}

	return false
}

func loadConfig(cfgPath string, logger *logger.Logger) (*config.Config, error) {
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		logger.Info("Configuration file not found, using defaults", "path", cfgPath)
		return getDefaultConfig(), nil
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		logger.Warn("Failed to load config, using defaults", "error", err)
		return getDefaultConfig(), nil
	}

	logger.Debug("Configuration loaded successfully", "path", cfgPath)
	return cfg, nil
}

func getDefaultConfig() *config.Config {
	return &config.Config{
		Scanner: config.ScannerConfig{
			MaxFileSize:       1024 * 1024, // 1MB
			DocumentationMode: "full",
			ExcludedPatterns: []string{
				"node_modules", ".git", ".idea", ".vscode",
				"dist", "build", "*.exe", "*.dll", "*.log", "*.tmp",
			},
		},
		Output: config.OutputConfig{
			Filename:         "project_docs_minimal.md",
			Format:          "markdown",
			AppendTimestamp: true,
		},
	}
}

func getOutputPath(cfg *config.Config, outputFlag string) string {
	if outputFlag != "" {
		return outputFlag
	}

	filename := cfg.Output.Filename
	if filename == "" {
		filename = "project_docs_minimal.md"
	}

	if cfg.Output.AppendTimestamp {
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		filename = fmt.Sprintf("%s_%s%s", base, time.Now().Format("20060102_150405"), ext)
	}

	return filename
}

func generateDocumentation(result *ProcessResult, outputPath string) error {
	content := fmt.Sprintf(`# Project Documentation (Plugin Architecture Simulation)

**Generated:** %s  
**Processing Mode:** Plugin-Based Architecture (Simulated)  
**Duration:** %v  

## 🔌 Plugin System Summary

- **Plugins Used:** %v
- **Project Detection:** Plugin-based type detection
- **Language Analysis:** Plugin-based language detection
- **File Processing:** Plugin-coordinated processing

## Project Summary

- **Project Type:** %s
- **Total Files:** %d
- **Files Processed:** %d  
- **Files Skipped:** %d

## Plugin Activity Log

`, 
		time.Now().Format("2006-01-02 15:04:05"),
		result.Duration,
		result.PluginsUsed,
		result.ProjectType,
		result.TotalFiles,
		result.ProcessedFiles,
		result.SkippedFiles,
	)

	// Группируем файлы по языкам (показываем работу плагинов)
	langStats := make(map[string]int)
	for _, file := range result.Files {
		if !file.Skipped {
			langStats[file.Language]++
		}
	}

	content += "### Language Plugin Statistics\n\n"
	for lang, count := range langStats {
		content += fmt.Sprintf("- **%s:** %d files\n", lang, count)
	}

	content += "\n## Sample Files (Plugin-Processed)\n\n"

	// Показываем первые 20 файлов с указанием плагина
	processedCount := 0
	for _, file := range result.Files {
		if file.Skipped || processedCount >= 20 {
			continue
		}

		content += fmt.Sprintf("### %s\n\n", file.Path)
		content += fmt.Sprintf("**Language Plugin:** %s  \n", file.Language)
		content += fmt.Sprintf("**Size:** %d bytes  \n\n", file.Size)

		if len(file.Content) > 0 && len(file.Content) < 2000 {
			// Убираем "(plugin-name)" из языка для подсветки синтаксиса
			syntaxLang := strings.Split(file.Language, " ")[0]
			content += fmt.Sprintf("```%s\n%s\n```\n\n", syntaxLang, file.Content)
		}

		processedCount++
	}

	if result.ProcessedFiles > 20 {
		content += fmt.Sprintf("... and %d more files processed by plugins\n", result.ProcessedFiles-20)
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}