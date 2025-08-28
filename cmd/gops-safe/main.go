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

// SafeProcessor - безопасный процессор без параллелизма
type SafeProcessor struct {
	rootDir string
	config  *ProcessorConfig
	logger  *logger.Logger
}

// ProcessorConfig - упрощенная конфигурация для безопасной обработки
type ProcessorConfig struct {
	RootDir           string
	OutputFile        string
	MaxFileSize       int64
	IncludeTests      bool
	IncludeConfigs    bool
	IncludeMarkup     bool
	IncludeStyles     bool
	ExcludedPatterns  []string
	DocumentationMode string
	Verbose          bool
}

// ProcessResult - результат обработки
type ProcessResult struct {
	Files         []FileInfo
	ProjectType   string
	ProjectTree   string
	TotalFiles    int
	ProcessedFiles int
	SkippedFiles  int
	TotalSize     int64
	Duration      time.Duration
	Errors        []string
}

// FileInfo - информация о файле
type FileInfo struct {
	Path        string
	Language    string
	Size        int64
	LineCount   int
	Content     string
	Skipped     bool
	SkipReason  string
	ProcessTime time.Duration
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

	zapLogger.Info("🛡️ Starting GOPS Safe (Sequential Processing)")

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
		IncludeTests:      cfg.Scanner.IncludeTests,
		IncludeConfigs:    cfg.Scanner.IncludeConfigs,
		IncludeMarkup:     cfg.Scanner.IncludeMarkup,
		IncludeStyles:     cfg.Scanner.IncludeStyles,
		ExcludedPatterns:  cfg.Scanner.ExcludedPatterns,
		DocumentationMode: cfg.Scanner.DocumentationMode,
		Verbose:          *verbose,
	}

	// Создаем процессор
	processor := &SafeProcessor{
		rootDir: absRoot,
		config:  processorConfig,
		logger:  zapLogger,
	}

	zapLogger.Info("Starting sequential file processing",
		"rootDir", absRoot,
		"outputFile", processorConfig.OutputFile,
	)

	result, err := processor.Process()
	if err != nil {
		zapLogger.Fatal("Processing failed", err)
	}

	// Генерируем документацию
	if err := generateDocumentation(result, processorConfig.OutputFile); err != nil {
		zapLogger.Fatal("Documentation generation failed", err)
	}

	// Проверяем .gitignore
	checkGitignore(absRoot, processorConfig.OutputFile, zapLogger)

	// Выводим результаты
	zapLogger.Info("✅ Sequential processing completed successfully!",
		"duration", result.Duration,
		"total_files", result.TotalFiles,
		"processed", result.ProcessedFiles,
		"skipped", result.SkippedFiles,
	)

	fmt.Printf("\n🛡️ GOPS Safe - Sequential Processing Complete!\n")
	fmt.Printf("📂 Project: %s\n", absRoot)
	fmt.Printf("⏱️  Duration: %v\n", result.Duration)
	fmt.Printf("📄 Total files found: %d\n", result.TotalFiles)
	fmt.Printf("✅ Files processed: %d\n", result.ProcessedFiles)
	fmt.Printf("⏭️  Files skipped: %d\n", result.SkippedFiles)
	fmt.Printf("📊 Total size: %.2f MB\n", float64(result.TotalSize)/1024/1024)
	fmt.Printf("🏗️  Project type: %s\n", result.ProjectType)
	fmt.Printf("💾 Output saved to: %s\n", processorConfig.OutputFile)

	if len(result.Errors) > 0 {
		fmt.Printf("\n⚠️ Errors encountered: %d\n", len(result.Errors))
		for i, err := range result.Errors {
			if i < 5 { // Показываем только первые 5
				fmt.Printf("  - %s\n", err)
			} else {
				fmt.Printf("  ... and %d more\n", len(result.Errors)-5)
				break
			}
		}
	}
}

func (p *SafeProcessor) Process() (*ProcessResult, error) {
	start := time.Now()
	
	result := &ProcessResult{
		Files:     make([]FileInfo, 0),
		Errors:    make([]string, 0),
	}

	p.logger.Info("🔍 Phase 1: Scanning directory structure")
	
	// Сканируем все файлы
	allFiles, err := p.scanDirectory()
	if err != nil {
		return nil, fmt.Errorf("directory scan failed: %w", err)
	}
	
	result.TotalFiles = len(allFiles)
	p.logger.Info("Directory scan complete", "files", result.TotalFiles)

	// Определяем тип проекта (простое определение)
	result.ProjectType = p.detectProjectType(allFiles)
	p.logger.Info("Project type detected", "type", result.ProjectType)
	
	// Генерируем дерево структуры проекта
	p.logger.Info("🌳 Phase 2.5: Generating project tree structure")
	result.ProjectTree = p.generateProjectTree(allFiles)

	p.logger.Info("📄 Phase 2: Processing files sequentially")

	// Обрабатываем файлы ПОСЛЕДОВАТЕЛЬНО (безопасно!)
	for i, filePath := range allFiles {
		if p.config.Verbose && i%100 == 0 {
			p.logger.Debug("Processing progress", "file", i+1, "total", len(allFiles))
		}

		fileInfo := p.processFile(filePath)
		result.Files = append(result.Files, fileInfo)

		if fileInfo.Skipped {
			result.SkippedFiles++
		} else {
			result.ProcessedFiles++
			result.TotalSize += fileInfo.Size
		}
	}

	result.Duration = time.Since(start)
	p.logger.Info("✅ Sequential processing completed",
		"processed", result.ProcessedFiles,
		"skipped", result.SkippedFiles,
		"duration", result.Duration,
	)

	return result, nil
}

func (p *SafeProcessor) scanDirectory() ([]string, error) {
	var files []string
	
	err := filepath.Walk(p.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			p.logger.Warn("Error accessing path", "path", path, "error", err)
			return nil // Продолжаем обработку
		}

		// Пропускаем директории
		if info.IsDir() {
			// Проверяем, нужно ли пропустить эту директорию
			if p.shouldSkipDirectory(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Получаем относительный путь
		relPath, err := filepath.Rel(p.rootDir, path)
		if err != nil {
			p.logger.Warn("Failed to get relative path", "path", path, "error", err)
			return nil
		}

		// Проверяем фильтры
		if !p.shouldSkipFile(relPath) {
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

func (p *SafeProcessor) processFile(filePath string) FileInfo {
	start := time.Now()
	
	fileInfo := FileInfo{
		Path: filePath,
		Language: p.detectLanguage(filePath),
	}

	fullPath := filepath.Join(p.rootDir, filePath)
	
	// Получаем информацию о файле
	stat, err := os.Stat(fullPath)
	if err != nil {
		fileInfo.Skipped = true
		fileInfo.SkipReason = fmt.Sprintf("stat error: %v", err)
		return fileInfo
	}

	fileInfo.Size = stat.Size()

	// Проверяем размер файла
	if fileInfo.Size > p.config.MaxFileSize {
		fileInfo.Skipped = true
		fileInfo.SkipReason = fmt.Sprintf("file too large: %d bytes", fileInfo.Size)
		return fileInfo
	}

	// Читаем содержимое файла
	content, err := os.ReadFile(fullPath)
	if err != nil {
		fileInfo.Skipped = true
		fileInfo.SkipReason = fmt.Sprintf("read error: %v", err)
		return fileInfo
	}

	fileInfo.Content = string(content)
	fileInfo.LineCount = strings.Count(fileInfo.Content, "\n") + 1
	fileInfo.ProcessTime = time.Since(start)

	return fileInfo
}


func (p *SafeProcessor) shouldSkipDirectory(dirPath string) bool {
	dirName := filepath.Base(dirPath)
	
	// Служебные директории  
	skipDirs := []string{
		".gops", ".claude", ".git", ".svn", ".hg",
		"node_modules", "vendor", "__pycache__",
		".idea", ".vscode", ".vs", ".angular", ".nx",
		"dist", "build", "out", "bin", "obj", "coverage", ".nyc_output",
	}
	
	for _, skip := range skipDirs {
		if dirName == skip {
			return true
		}
	}
	
	// Проверяем паттерны из конфигурации
	for _, pattern := range p.config.ExcludedPatterns {
		if matched, _ := filepath.Match(pattern, dirName); matched {
			return true
		}
	}
	
	return false
}

func (p *SafeProcessor) shouldSkipFile(filePath string) bool {
	// Проверяем по паттернам исключений
	for _, pattern := range p.config.ExcludedPatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
			return true
		}
		// Также проверяем полный путь
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	base := filepath.Base(filePath)

	// Пропускаем системные файлы
	if strings.HasPrefix(base, ".") && base != ".gitignore" && base != ".env" {
		return true
	}
	
	// Исключаем сгенерированные файлы документации
	if strings.HasPrefix(base, "project_docs_") && strings.HasSuffix(base, ".md") {
		return true
	}

	// Пропускаем бинарные файлы
	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib", ".a", ".o", ".lib",
		".jpg", ".jpeg", ".png", ".gif", ".ico", ".bmp", ".webp", ".svg",
		".mp4", ".avi", ".mov", ".wmv", ".flv", ".mp3", ".wav",
		".zip", ".tar", ".gz", ".rar", ".7z",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
	}

	for _, bext := range binaryExts {
		if ext == bext {
			return true
		}
	}

	// Логика включения по типам файлов
	if !p.config.IncludeTests && (strings.Contains(filePath, "_test.") || strings.Contains(filePath, ".test.") || strings.Contains(filePath, "/test/")) {
		return true
	}

	return false
}

func (p *SafeProcessor) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	languageMap := map[string]string{
		".go":    "go",
		".js":    "javascript",
		".jsx":   "javascript",
		".ts":    "typescript",
		".tsx":   "typescript",
		".py":    "python",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".cs":    "csharp",
		".php":   "php",
		".rb":    "ruby",
		".rs":    "rust",
		".kt":    "kotlin",
		".swift": "swift",
		".html":  "html",
		".css":   "css",
		".scss":  "scss",
		".less":  "less",
		".xml":   "xml",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".toml":  "toml",
		".md":    "markdown",
		".txt":   "text",
		".sh":    "bash",
		".sql":   "sql",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "unknown"
}

func (p *SafeProcessor) generateProjectTree(files []string) string {
	// Создаем структуру директорий, исключая служебные
	dirs := make(map[string][]string)
	
	for _, file := range files {
		// Пропускаем служебные файлы и директории
		if p.shouldSkipFromTree(file) {
			continue
		}
		
		dir := filepath.Dir(file)
		if dir == "." {
			dirs[""] = append(dirs[""], filepath.Base(file))
		} else {
			// Разбиваем путь на части
			parts := strings.Split(dir, string(filepath.Separator))
			
			// Проверяем, что ни одна из частей пути не является служебной
			skipDir := false
			for _, part := range parts {
				if p.shouldSkipFromTree(part) {
					skipDir = true
					break
				}
			}
			if skipDir {
				continue
			}
			
			for i := range parts {
				parentPath := strings.Join(parts[:i], "/")
				
				if parentPath == "" {
					if !contains(dirs[""], parts[i]+"/") {
						dirs[""] = append(dirs[""], parts[i]+"/")
					}
				} else {
					if !contains(dirs[parentPath], parts[i]+"/") {
						dirs[parentPath] = append(dirs[parentPath], parts[i]+"/")
					}
				}
			}
			
			// Добавляем файл в соответствующую директорию
			dirs[strings.ReplaceAll(dir, string(filepath.Separator), "/")] = 
				append(dirs[strings.ReplaceAll(dir, string(filepath.Separator), "/")], filepath.Base(file))
		}
	}
	
	// Генерируем дерево
	var tree strings.Builder
	tree.WriteString(".\n")
	
	// Рекурсивно строим дерево
	p.buildTreeLevel(&tree, dirs, "", 0)
	
	return tree.String()
}

func (p *SafeProcessor) buildTreeLevel(tree *strings.Builder, dirs map[string][]string, currentPath string, level int) {
	items := dirs[currentPath]
	
	// Сортируем: сначала директории, потом файлы
	var directories, files []string
	for _, item := range items {
		if strings.HasSuffix(item, "/") {
			directories = append(directories, item)
		} else {
			files = append(files, item)
		}
	}
	
	allItems := append(directories, files...)
	
	for i, item := range allItems {
		// Определяем префикс для дерева
		prefix := strings.Repeat("│   ", level)
		if i == len(allItems)-1 {
			prefix += "└── "
		} else {
			prefix += "├── "
		}
		
		tree.WriteString(prefix + item + "\n")
		
		// Если это директория, рекурсивно обрабатываем её
		if strings.HasSuffix(item, "/") {
			var nextPath string
			if currentPath == "" {
				nextPath = strings.TrimSuffix(item, "/")
			} else {
				nextPath = currentPath + "/" + strings.TrimSuffix(item, "/")
			}
			p.buildTreeLevel(tree, dirs, nextPath, level+1)
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (p *SafeProcessor) shouldSkipFromTree(path string) bool {
	// Проверяем базовое имя файла/директории
	baseName := filepath.Base(path)
	
	// Служебные директории и файлы
	skipPaths := []string{
		".gops", ".claude", ".git", ".svn", ".hg",
		"node_modules", "vendor", "__pycache__",
		".idea", ".vscode", ".vs", ".angular", ".nx",
		"dist", "build", "out", "bin", "obj", "coverage", ".nyc_output",
		"go.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
	}
	
	for _, skip := range skipPaths {
		if baseName == skip {
			return true
		}
	}
	
	// Исключаем сгенерированные файлы документации
	if strings.HasPrefix(baseName, "project_docs_") && strings.HasSuffix(baseName, ".md") {
		return true
	}
	
	// Проверяем паттерны из конфигурации
	for _, pattern := range p.config.ExcludedPatterns {
		if matched, _ := filepath.Match(pattern, baseName); matched {
			return true
		}
	}
	
	return false
}

func (p *SafeProcessor) detectProjectType(files []string) string {
	// Простая логика определения типа проекта
	hasGoMod := false
	hasPackageJson := false
	hasAngularJson := false
	hasNxJson := false

	for _, file := range files {
		switch filepath.Base(file) {
		case "go.mod":
			hasGoMod = true
		case "package.json":
			hasPackageJson = true
		case "angular.json":
			hasAngularJson = true
		case "nx.json":
			hasNxJson = true
		}
	}

	if hasNxJson {
		return "NX Monorepo"
	}
	if hasAngularJson && hasPackageJson {
		return "Angular"
	}
	if hasGoMod {
		return "Go"
	}
	if hasPackageJson {
		return "JavaScript/Node.js"
	}

	return "Generic"
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
			ParallelWorkers:   1, // Последовательная обработка
			IncludeTests:      false,
			IncludeConfigs:    true,
			IncludeMarkup:     false,
			IncludeStyles:     false,
			IncludeDocs:       false,
			DocumentationMode: "full",
			SelectionMode:     "all",
			ExcludedPatterns: []string{
				"node_modules", ".git", ".idea", ".vscode",
				"dist", "build", "*.exe", "*.dll", "*.log", "*.tmp",
			},
		},
		Output: config.OutputConfig{
			Filename:         "project_docs_safe.md",
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
		filename = "project_docs_safe.md"
	}

	if cfg.Output.AppendTimestamp {
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		filename = fmt.Sprintf("%s_%s%s", base, time.Now().Format("20060102_150405"), ext)
	}

	return filename
}

func generateDocumentation(result *ProcessResult, outputPath string) error {
	content := fmt.Sprintf(`# Project Documentation (Safe Sequential Processing)

**Generated:** %s  
**Processing Mode:** Sequential (Thread-Safe)  
**Duration:** %v  

## Project Summary

- **Project Type:** %s
- **Total Files Found:** %d
- **Files Processed:** %d  
- **Files Skipped:** %d
- **Total Size:** %.2f MB

## Project Structure

` + "```" + `
%s
` + "```" + `

## File Statistics by Language

`, 
		time.Now().Format("2006-01-02 15:04:05"),
		result.Duration,
		result.ProjectType,
		result.TotalFiles,
		result.ProcessedFiles,
		result.SkippedFiles,
		float64(result.TotalSize)/1024/1024,
		result.ProjectTree,
	)

	// Группируем файлы по языкам
	langStats := make(map[string]int)
	for _, file := range result.Files {
		if !file.Skipped {
			langStats[file.Language]++
		}
	}

	for lang, count := range langStats {
		content += fmt.Sprintf("- **%s:** %d files\n", lang, count)
	}

	content += "\n## Files\n\n"

	// Добавляем информацию о первых 50 обработанных файлах
	processedCount := 0
	for _, file := range result.Files {
		if file.Skipped {
			continue
		}
		
		if processedCount >= 50 {
			content += fmt.Sprintf("... and %d more processed files\n\n", result.ProcessedFiles-50)
			break
		}

		content += fmt.Sprintf("### %s\n\n", file.Path)
		content += fmt.Sprintf("**Language:** %s  \n", file.Language)
		content += fmt.Sprintf("**Size:** %d bytes (%d lines)  \n", file.Size, file.LineCount)
		content += fmt.Sprintf("**Process Time:** %v  \n\n", file.ProcessTime)

		if len(file.Content) > 0 && len(file.Content) < 5000 {
			content += fmt.Sprintf("```%s\n%s\n```\n\n", file.Language, file.Content)
		} else if len(file.Content) >= 5000 {
			// Показываем только первые 20 строк для больших файлов
			lines := strings.Split(file.Content, "\n")
			if len(lines) > 20 {
				preview := strings.Join(lines[:20], "\n")
				content += fmt.Sprintf("```%s\n%s\n... (%d more lines)\n```\n\n", 
					file.Language, preview, len(lines)-20)
			} else {
				content += fmt.Sprintf("```%s\n%s\n```\n\n", file.Language, file.Content)
			}
		}

		processedCount++
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// checkGitignore проверяет .gitignore и выдает предупреждения о необходимых исключениях
func checkGitignore(rootDir, outputFile string, logger *logger.Logger) {
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	
	// Проверяем, существует ли .gitignore
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		logger.Warn("⚠️  .gitignore file not found. Consider creating one to exclude GOPS service files.")
		printGitignoreRecommendations(outputFile)
		return
	}
	
	// Читаем содержимое .gitignore
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		logger.Warn("Failed to read .gitignore", "error", err)
		return
	}
	
	gitignoreContent := string(content)
	
	// Определяем какие паттерны должны быть в .gitignore
	requiredPatterns := []string{
		".gops/",           // Папка GOPS
		".claude/",         // Папка Claude Code
		"project_docs_*.md", // Сгенерированная документация
		"gops_config_*.yaml", // Старые конфиги GOPS
	}
	
	// Рекомендуемые паттерны (не обязательные)
	recommendedPatterns := []string{
		"*.exe",           // Исполняемые файлы Windows
		"*.log",           // Лог файлы
		"*.tmp",           // Временные файлы
		"node_modules/",   // Node.js зависимости
		".vscode/",        // VS Code настройки
		".idea/",          // JetBrains IDE настройки
	}
	
	var missingRequired []string
	var missingRecommended []string
	
	// Проверяем обязательные паттерны
	for _, pattern := range requiredPatterns {
		if !strings.Contains(gitignoreContent, pattern) && !strings.Contains(gitignoreContent, strings.TrimSuffix(pattern, "/")) {
			missingRequired = append(missingRequired, pattern)
		}
	}
	
	// Проверяем рекомендуемые паттерны
	for _, pattern := range recommendedPatterns {
		if !strings.Contains(gitignoreContent, pattern) && !strings.Contains(gitignoreContent, strings.TrimSuffix(pattern, "/")) {
			missingRecommended = append(missingRecommended, pattern)
		}
	}
	
	// Выводим предупреждения
	if len(missingRequired) > 0 {
		logger.Warn("🚨 IMPORTANT: Missing required patterns in .gitignore to exclude GOPS service files:")
		for _, pattern := range missingRequired {
			logger.Warn("  - " + pattern)
		}
		fmt.Printf("\n🚨 GITIGNORE WARNING: Add these patterns to .gitignore:\n")
		for _, pattern := range missingRequired {
			fmt.Printf("   %s\n", pattern)
		}
		fmt.Printf("\nRun: echo -e '\\n# GOPS service files\\n%s' >> .gitignore\n", strings.Join(missingRequired, "\\n"))
	}
	
	if len(missingRecommended) > 0 && len(missingRecommended) >= 3 {
		logger.Info("💡 Recommended: Consider adding these patterns to .gitignore:")
		for _, pattern := range missingRecommended[:3] { // Показываем только первые 3
			logger.Info("  - " + pattern)
		}
		if len(missingRecommended) > 3 {
			logger.Info(fmt.Sprintf("  ... and %d more", len(missingRecommended)-3))
		}
	}
	
	if len(missingRequired) == 0 {
		logger.Info("✅ .gitignore correctly excludes GOPS service files")
	}
}

func printGitignoreRecommendations(outputFile string) {
	fmt.Printf("\n💡 RECOMMENDATION: Create .gitignore with these patterns:\n")
	fmt.Printf(`
# GOPS service files and generated documentation  
.gops/
.claude/
project_docs_*.md
gops_config_*.yaml

# Build artifacts
*.exe
*.dll
*.so
*.dylib

# Logs and temporary files
*.log
*.tmp
*.bak

# IDE and editor files
.vscode/
.idea/
.vs/

# Dependencies
node_modules/
vendor/

# OS generated files
.DS_Store
Thumbs.db
`)
	fmt.Printf("\nCreate: echo 'See above' > .gitignore\n")
}