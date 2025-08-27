package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kolkov/gops/internal/app"
	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/internal/scanner"
	"github.com/kolkov/gops/pkg/logger"
)

func main() {
	cfgPath := flag.String("c", "./gops_config.yaml", "Path to config file")
	verbose := flag.Bool("v", false, "Verbose output")
	rootDir := flag.String("d", ".", "Project root directory")
	flag.Parse()

	ctx := context.Background()

	// Настройка логгера
	logLevel := logger.InfoLevel
	if *verbose {
		logLevel = logger.DebugLevel
	}
	zapLogger := logger.New(logLevel)
	defer zapLogger.Sync()

	// Получаем абсолютный путь к проекту
	absRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		zapLogger.Fatal("Failed to get absolute path", err)
	}

	// Загружаем конфигурацию или создаем интерактивно
	cfg, err := loadOrCreateConfig(*cfgPath, absRoot, zapLogger)
	if err != nil {
		zapLogger.Fatal("Configuration error", err)
	}

	// Генерируем имя выходного файла
	outputFile := app.GenerateOutputFilename(cfg)

	// Создаем и запускаем сканер
	projectScanner := app.NewProjectScanner(
		absRoot,
		outputFile,
		cfg,
		zapLogger,
	)

	if err := projectScanner.Run(ctx); err != nil {
		zapLogger.Fatal("Scan failed", err)
	}

	zapLogger.Info("Documentation generated successfully",
		"file", outputFile,
		"project", absRoot,
	)

	// Выводим информацию для пользователя
	fmt.Printf("\n✅ Documentation generated successfully!\n")
	fmt.Printf("📄 Output file: %s\n", outputFile)
}

func loadOrCreateConfig(cfgPath, projectDir string, logger *logger.Logger) (*config.Config, error) {
	// Проверяем существование файла конфигурации
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// Файл не существует - предлагаем создать интерактивно
		fmt.Printf("\n⚠️  Configuration file '%s' not found.\n", cfgPath)
		fmt.Println("Let's create one for your project!")

		// Определяем тип проекта
		fmt.Println("🔍 Detecting project type...")
		detector := scanner.NewProjectDetector(projectDir, logger)
		projectType, err := detector.Detect()
		if err != nil {
			logger.Warn("Could not detect project type, will use generic settings", "error", err)
			projectType = "unknown"
		}

		// Создаем конфигурацию интерактивно
		builder := config.NewConfigBuilder(projectType)
		cfg, err := builder.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %w", err)
		}

		// Сохраняем конфигурацию
		fmt.Printf("\n💾 Saving configuration to %s...\n", cfgPath)
		if err := config.Save(cfg, cfgPath); err != nil {
			logger.Error("Failed to save config file", err)
			// Продолжаем с созданной конфигурацией даже если не удалось сохранить
			fmt.Println("⚠️  Could not save config file, but will continue with selected settings.")
		} else {
			fmt.Println("✅ Configuration saved successfully!")
		}

		fmt.Println("🚀 Starting project scan with your configuration...")
		return cfg, nil

	} else if err != nil {
		// Другая ошибка при проверке файла
		return nil, fmt.Errorf("error checking config file: %w", err)
	}

	// Файл существует - загружаем его
	cfg, err := config.Load(cfgPath)
	if err != nil {
		// Файл существует, но не может быть загружен
		logger.Error("Failed to load config file", err, "path", cfgPath)

		// Предлагаем пересоздать файл
		fmt.Printf("\n⚠️  Configuration file '%s' is corrupted or invalid.\n", cfgPath)
		fmt.Println("Would you like to create a new configuration?")

		builder := config.NewConfigBuilder("unknown")
		if builder.AskYesNo("Create new configuration?", true) {
			// Определяем тип проекта
			fmt.Println("\n🔍 Detecting project type...")
			detector := scanner.NewProjectDetector(projectDir, logger)
			projectType, _ := detector.Detect()

			// Создаем новую конфигурацию
			builder = config.NewConfigBuilder(projectType)
			cfg, err = builder.Build()
			if err != nil {
				return nil, fmt.Errorf("failed to build config: %w", err)
			}

			// Сохраняем
			if err := config.Save(cfg, cfgPath); err != nil {
				logger.Error("Failed to save config file", err)
			}

			return cfg, nil
		}

		// Пользователь отказался - используем дефолтную конфигурацию
		fmt.Println("Using default configuration...")
		return config.DefaultConfig(), nil
	}

	logger.Debug("Configuration loaded successfully", "path", cfgPath)
	return cfg, nil
}
