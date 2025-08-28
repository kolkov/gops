package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/pkg/logger"
)

// Простая тестовая версия без плагинов - просто проверим что базовая структура работает
func main() {
	cfgPath := flag.String("c", "./gops_config.yaml", "Path to config file")
	verbose := flag.Bool("v", false, "Verbose output")
	rootDir := flag.String("d", ".", "Project root directory")
	flag.Parse()

	// Настройка логгера
	logLevel := logger.InfoLevel
	if *verbose {
		logLevel = logger.DebugLevel
	}
	zapLogger := logger.New(logLevel)
	defer zapLogger.Sync()

	zapLogger.Info("🧪 Testing GOPS structure without plugins")

	// Получаем абсолютный путь к проекту
	absRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		zapLogger.Fatal("Failed to get absolute path", err)
	}

	zapLogger.Info("Project path resolved", "path", absRoot)

	// Простая проверка что директория существует
	if _, err := os.Stat(absRoot); os.IsNotExist(err) {
		zapLogger.Fatal("Project directory does not exist", err)
	}

	// Пытаемся загрузить конфигурацию (минимальную)
	cfg, err := loadSimpleConfig(*cfgPath)
	if err != nil {
		zapLogger.Warn("Could not load config, using defaults", "error", err)
		cfg = getDefaultConfig()
	}

	zapLogger.Info("Configuration loaded", "config", fmt.Sprintf("%+v", cfg.Scanner))

	// Простое сканирование структуры директории
	files, err := scanDirectory(absRoot)
	if err != nil {
		zapLogger.Fatal("Directory scan failed", err)
	}

	zapLogger.Info("✅ Directory scan completed", 
		"files_found", len(files),
		"project", absRoot,
	)

	fmt.Printf("\n🎉 GOPS Test - Basic Structure Works!\n")
	fmt.Printf("📂 Project: %s\n", absRoot)
	fmt.Printf("📄 Files found: %d\n", len(files))
	fmt.Printf("⏱️  Test completed in: %v\n", time.Since(time.Now()))

	// Выводим первые 10 файлов для проверки
	fmt.Println("\n📋 Sample files:")
	for i, file := range files {
		if i >= 10 {
			fmt.Printf("... and %d more\n", len(files)-10)
			break
		}
		fmt.Printf("  %s\n", file)
	}
}

func loadSimpleConfig(cfgPath string) (*config.Config, error) {
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", cfgPath)
	}
	
	return config.Load(cfgPath)
}

func getDefaultConfig() *config.Config {
	return &config.Config{
		Scanner: config.ScannerConfig{
			MaxFileSize:      1024 * 1024, // 1MB
			ParallelWorkers:  4,
			IncludeTests:     false,
			IncludeConfigs:   true,
			DocumentationMode: "full",
			SelectionMode:    "all",
			ExcludedPatterns: []string{
				"node_modules",
				".git",
				"*.exe",
				"*.dll",
				"*.so",
			},
		},
		Output: config.OutputConfig{
			Filename: "project_test.md",
			Format:   "markdown",
		},
	}
}

func scanDirectory(root string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			// Простая фильтрация
			if !shouldSkipFile(path) {
				relPath, _ := filepath.Rel(root, path)
				files = append(files, relPath)
			}
		}
		
		return nil
	})
	
	return files, err
}

func shouldSkipFile(path string) bool {
	name := filepath.Base(path)
	ext := filepath.Ext(path)
	
	// Пропускаем системные файлы
	if name[0] == '.' {
		return true
	}
	
	// Пропускаем бинарные файлы
	binaryExts := []string{".exe", ".dll", ".so", ".dylib", ".a", ".o"}
	for _, bext := range binaryExts {
		if ext == bext {
			return true
		}
	}
	
	return false
}