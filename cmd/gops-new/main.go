package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/internal/core/phases"
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/plugin"
	"github.com/kolkov/gops/pkg/logger"

	// Регистрация плагинов
	goPlugin "github.com/kolkov/gops/internal/plugins/languages/go"
	jsPlugin "github.com/kolkov/gops/internal/plugins/languages/javascript"
	angularPlugin "github.com/kolkov/gops/internal/plugins/projects/angular"
	nxPlugin "github.com/kolkov/gops/internal/plugins/projects/nx"
)

func main() {
	// Парсинг флагов командной строки
	cfgPath := flag.String("c", "./gops_config.yaml", "Path to config file")
	verbose := flag.Bool("v", false, "Verbose output")
	rootDir := flag.String("d", ".", "Project root directory")
	outputFile := flag.String("o", "", "Output file path (optional)")
	format := flag.String("f", "markdown", "Output format (markdown, json)")
	flag.Parse()

	ctx := context.Background()

	// Настройка логгера
	logLevel := logger.InfoLevel
	if *verbose {
		logLevel = logger.DebugLevel
	}
	zapLogger := logger.New(logLevel)
	defer zapLogger.Sync()

	zapLogger.Info("🚀 Starting GOPS with new plugin architecture!")

	// Получаем абсолютный путь к проекту
	absRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		zapLogger.Fatal("Failed to get absolute path", err)
	}

	// Загружаем или создаем конфигурацию
	cfg, err := loadOrCreateConfig(*cfgPath, absRoot, zapLogger)
	if err != nil {
		zapLogger.Fatal("Configuration error", err)
	}

	// Инициализируем систему плагинов
	if err := initializePlugins(ctx, zapLogger); err != nil {
		zapLogger.Fatal("Failed to initialize plugins", err)
	}

	// Создаем конфигурацию для новой архитектуры
	processorConfig := &model.ProcessorConfig{
		RootDir:           absRoot,
		OutputDir:         filepath.Dir(getOutputPath(cfg, *outputFile)),
		MaxWorkers:        cfg.Scanner.ParallelWorkers,
		MaxFileSize:       cfg.Scanner.MaxFileSize,
		IncludeTests:      cfg.Scanner.IncludeTests,
		IncludeConfigs:    cfg.Scanner.IncludeConfigs,
		IncludeMarkup:     cfg.Scanner.IncludeMarkup,
		IncludeStyles:     cfg.Scanner.IncludeStyles,
		ExcludedPatterns:  cfg.Scanner.ExcludedPatterns,
		DocumentationMode: cfg.Scanner.DocumentationMode,
		OutputFormat:      model.OutputFormat(*format),
		Verbose:          *verbose,
	}

	// Создаем и запускаем фазовый процессор
	processor := phases.NewProcessor(processorConfig, zapLogger)

	zapLogger.Info("Starting project processing with plugin architecture",
		"rootDir", absRoot,
		"workers", processorConfig.MaxWorkers,
		"format", *format,
	)

	start := time.Now()
	result, err := processor.Process(ctx)
	duration := time.Since(start)

	if err != nil {
		zapLogger.Fatal("Processing failed", err)
	}

	// Выводим результаты
	zapLogger.Info("✅ Processing completed successfully!",
		"duration", duration,
		"files", len(result.Files),
		"plugins_used", len(result.PluginsUsed),
	)

	fmt.Printf("\n🎉 GOPS New Architecture - Processing Complete!\n")
	fmt.Printf("📂 Project: %s\n", absRoot)
	fmt.Printf("⏱️  Duration: %v\n", duration)
	fmt.Printf("📄 Files processed: %d\n", len(result.Files))
	fmt.Printf("🔌 Plugins used: %v\n", result.PluginsUsed)
	
	if result.ProjectInfo != nil {
		fmt.Printf("🏗️  Project type: %s (%s)\n", result.ProjectInfo.Type, result.ProjectInfo.Language)
		if result.ProjectInfo.Version != "" {
			fmt.Printf("📋 Version: %s\n", result.ProjectInfo.Version)
		}
		fmt.Printf("📦 Components: %d\n", len(result.ProjectInfo.Components))
		fmt.Printf("📚 Modules: %d\n", len(result.ProjectInfo.Modules))
	}

	// Сохраняем результат
	outputPath := getOutputPath(cfg, *outputFile)
	if err := saveResult(result, outputPath, model.OutputFormat(*format)); err != nil {
		zapLogger.Fatal("Failed to save result", err)
	}

	fmt.Printf("💾 Output saved to: %s\n", outputPath)
}

func initializePlugins(ctx context.Context, logger *logger.Logger) error {
	registry := plugin.GetRegistry()
	
	// Регистрируем языковые плагины
	plugins := []plugin.Plugin{
		goPlugin.Register(),
		jsPlugin.Register(),
	}
	
	// Регистрируем проектные плагины
	projectPlugins := []plugin.Plugin{
		angularPlugin.Register(),
		nxPlugin.Register(),
	}
	
	plugins = append(plugins, projectPlugins...)
	
	// Загружаем все плагины
	for _, p := range plugins {
		if err := registry.Register(p); err != nil {
			return fmt.Errorf("failed to register plugin %s: %w", p.Name(), err)
		}
		
		if err := p.OnLoad(ctx, logger); err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", p.Name(), err)
		}
		
		if err := p.OnEnable(ctx); err != nil {
			return fmt.Errorf("failed to enable plugin %s: %w", p.Name(), err)
		}
	}
	
	logger.Info("✅ All plugins initialized successfully", "count", len(plugins))
	return nil
}

func loadOrCreateConfig(cfgPath, projectDir string, logger *logger.Logger) (*config.Config, error) {
	// Проверяем существование файла конфигурации
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		logger.Info("Configuration file not found, creating default config", "path", cfgPath)
		
		// Создаем дефолтную конфигурацию для новой архитектуры
		cfg := &config.Config{
			Scanner: config.ScannerConfig{
				MaxFileSize:       1024 * 1024, // 1MB
				ParallelWorkers:   4,
				IncludeTests:      false,
				IncludeConfigs:    true,
				IncludeMarkup:     false,
				IncludeStyles:     false,
				IncludeDocs:       false,
				DocumentationMode: "full",
				SelectionMode:     "all",
				ExcludedPatterns: []string{
					"node_modules",
					".git",
					".idea",
					".vscode",
					"dist",
					"build",
					"*.log",
					"*.tmp",
				},
			},
			Output: config.OutputConfig{
				Filename:         "project_docs_new.md",
				Format:          "markdown",
				AppendTimestamp: true,
			},
		}
		
		return cfg, nil
	} else if err != nil {
		return nil, fmt.Errorf("error checking config file: %w", err)
	}
	
	// Загружаем существующую конфигурацию
	cfg, err := config.Load(cfgPath)
	if err != nil {
		logger.Warn("Failed to load config, using defaults", "error", err)
		return loadOrCreateConfig("", projectDir, logger) // Создаем дефолтную
	}
	
	logger.Debug("Configuration loaded successfully", "path", cfgPath)
	return cfg, nil
}

func getOutputPath(cfg *config.Config, outputFlag string) string {
	if outputFlag != "" {
		return outputFlag
	}
	
	filename := cfg.Output.Filename
	if filename == "" {
		filename = "project_docs_new.md"
	}
	
	if cfg.Output.AppendTimestamp {
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		filename = fmt.Sprintf("%s_%s%s", base, time.Now().Format("20060102_150405"), ext)
	}
	
	return filename
}

func saveResult(result *model.ProcessingResult, outputPath string, format model.OutputFormat) error {
	switch format {
	case model.OutputFormatJSON:
		return saveJSON(result, outputPath)
	default:
		return saveMarkdown(result, outputPath)
	}
}

func saveJSON(result *model.ProcessingResult, outputPath string) error {
	// Простая JSON реализация - можно расширить
	content := fmt.Sprintf(`{
  "project_info": {
    "type": "%s",
    "language": "%s",
    "components_count": %d,
    "modules_count": %d
  },
  "files_count": %d,
  "plugins_used": %v,
  "processing_time": "%s"
}`, 
		getStringValue(result.ProjectInfo, "Type"),
		getStringValue(result.ProjectInfo, "Language"), 
		getComponentsCount(result.ProjectInfo),
		getModulesCount(result.ProjectInfo),
		len(result.Files),
		result.PluginsUsed,
		time.Now().Format(time.RFC3339),
	)
	
	return os.WriteFile(outputPath, []byte(content), 0644)
}

func saveMarkdown(result *model.ProcessingResult, outputPath string) error {
	// Простая Markdown реализация
	content := fmt.Sprintf(`# Project Documentation (Generated by GOPS New Architecture)

**Generated:** %s

## Project Information

`, time.Now().Format("2006-01-02 15:04:05"))

	if result.ProjectInfo != nil {
		content += fmt.Sprintf(`- **Type:** %s
- **Language:** %s
- **Components:** %d
- **Modules:** %d

`, result.ProjectInfo.Type, result.ProjectInfo.Language, len(result.ProjectInfo.Components), len(result.ProjectInfo.Modules))
	}

	content += fmt.Sprintf(`## Processing Summary

- **Files processed:** %d
- **Plugins used:** %s

## Files

`, len(result.Files), fmt.Sprintf("%v", result.PluginsUsed))

	for _, file := range result.Files {
		content += fmt.Sprintf(`### %s

**Language:** %s  
**Size:** %d bytes  

`, file.Path, file.Language, len(file.Content))

		if file.Content != "" {
			content += fmt.Sprintf("```%s\n%s\n```\n\n", file.Language, file.Content)
		}
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// Helper functions
func getStringValue(info *model.ProjectInfo, field string) string {
	if info == nil {
		return ""
	}
	switch field {
	case "Type":
		return info.Type
	case "Language":
		return info.Language
	}
	return ""
}

func getComponentsCount(info *model.ProjectInfo) int {
	if info == nil {
		return 0
	}
	return len(info.Components)
}

func getModulesCount(info *model.ProjectInfo) int {
	if info == nil {
		return 0
	}
	return len(info.Modules)
}