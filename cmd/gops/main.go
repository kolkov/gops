package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/kolkov/gops/internal/model"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kolkov/gops/internal/app"
	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/internal/scanner"
	"github.com/kolkov/gops/internal/selector"
	"github.com/kolkov/gops/pkg/logger"
)

func main() {
	cfgPath := flag.String("c", "./gops_config.yaml", "Path to config file")
	verbose := flag.Bool("v", false, "Verbose output")
	rootDir := flag.String("d", ".", "Project root directory")
	interactive := flag.Bool("i", false, "Interactive file selection mode")
	useLastSelection := flag.Bool("l", false, "Use last saved selection")
	selectionName := flag.String("s", "", "Load saved selection by name")
	includePaths := flag.String("p", "", "Comma-separated paths to include")
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

	// Обрабатываем режимы выбора файлов
	if err := handleFileSelection(cfg, absRoot, zapLogger,
		*interactive, *useLastSelection, *selectionName, *includePaths); err != nil {
		zapLogger.Fatal("File selection error", err)
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

	if len(cfg.Scanner.IncludedPaths) > 0 {
		fmt.Printf("📁 Included %d selected files\n", len(cfg.Scanner.IncludedPaths))
	}
}

func handleFileSelection(cfg *config.Config, rootDir string, logger *logger.Logger,
	interactive, useLastSelection bool, selectionName, includePaths string) error {

	selectionMgr := selector.NewSelectionManager(rootDir)

	// Приоритет опций командной строки над конфигурацией
	if interactive {
		// Интерактивный режим выбора файлов
		return handleInteractiveSelection(cfg, rootDir, logger, selectionMgr)
	}

	if useLastSelection {
		// Использовать последнюю сохраненную выборку
		if selection, err := selectionMgr.LoadSelection("last"); err == nil {
			cfg.Scanner.SelectionMode = "list"
			cfg.Scanner.IncludedPaths = selection.Paths
			fmt.Printf("📋 Using last saved selection: %d files\n", len(selection.Paths))
		} else {
			logger.Warn("No last selection found, using all files")
			cfg.Scanner.SelectionMode = "all"
		}
		return nil
	}

	if selectionName != "" {
		// Загрузить именованную выборку
		if selection, err := selectionMgr.LoadSelection(selectionName); err == nil {
			cfg.Scanner.SelectionMode = "list"
			cfg.Scanner.IncludedPaths = selection.Paths
			fmt.Printf("📋 Using saved selection '%s': %d files\n", selectionName, len(selection.Paths))
		} else {
			return fmt.Errorf("failed to load selection '%s': %w", selectionName, err)
		}
		return nil
	}

	if includePaths != "" {
		// Использовать пути из командной строки
		paths := strings.Split(includePaths, ",")
		for i, path := range paths {
			paths[i] = strings.TrimSpace(path)
		}
		cfg.Scanner.SelectionMode = "list"
		cfg.Scanner.IncludedPaths = paths
		fmt.Printf("📁 Including %d specified paths\n", len(paths))
		return nil
	}

	// Проверяем настройки из конфигурации
	switch cfg.Scanner.SelectionMode {
	case "interactive":
		return handleInteractiveSelection(cfg, rootDir, logger, selectionMgr)
	case "list", "patterns":
		// Уже настроено в конфигурации
		if len(cfg.Scanner.IncludedPaths) > 0 {
			fmt.Printf("📁 Using %d paths from configuration\n", len(cfg.Scanner.IncludedPaths))
		}
		return nil
	default:
		// По умолчанию - все файлы
		cfg.Scanner.SelectionMode = "all"
		return nil
	}
}

func handleInteractiveSelection(cfg *config.Config, rootDir string, logger *logger.Logger,
	selectionMgr *selector.SelectionManager) error {

	// Проверяем есть ли сохраненные выборки
	selections, _ := selectionMgr.ListSelections()
	if len(selections) > 0 {
		fmt.Println("\n📋 Found saved selections. What would you like to do?")
		fmt.Println("  1. Create new selection")
		fmt.Println("  2. Use saved selection")
		fmt.Print("Choice [1-2]: ")

		var choice int
		fmt.Scanln(&choice)

		if choice == 2 {
			if selection, err := selectionMgr.InteractiveLoad(); err == nil {
				cfg.Scanner.SelectionMode = "list"
				cfg.Scanner.IncludedPaths = selection.Paths
				fmt.Printf("✅ Loaded selection: %s (%d files)\n", selection.Name, len(selection.Paths))
				return nil
			}
		}
	}

	// Создаем новую выборку
	scanCfg := &model.ScanConfig{
		MaxFileSize:      cfg.Scanner.MaxFileSize,
		ExcludedPatterns: cfg.Scanner.ExcludedPatterns,
	}

	fileSelector := selector.NewFileSelector(rootDir, scanCfg, logger)
	selectedPaths, err := fileSelector.SelectFiles()
	if err != nil {
		return fmt.Errorf("file selection failed: %w", err)
	}

	if len(selectedPaths) == 0 {
		return fmt.Errorf("no files selected")
	}

	// Сохраняем выборку для будущего использования
	fmt.Print("\nSave this selection for future use? [Y/n]: ")
	var saveChoice string
	fmt.Scanln(&saveChoice)

	if saveChoice != "n" && saveChoice != "N" {
		fmt.Print("Selection name: ")
		var name string
		fmt.Scanln(&name)
		if name == "" {
			name = fmt.Sprintf("selection_%s", time.Now().Format("20060102_150405"))
		}

		fmt.Print("Description (optional): ")
		var description string
		fmt.Scanln(&description)

		if err := selectionMgr.SaveSelection(name, description, selectedPaths); err != nil {
			logger.Error("Failed to save selection", err)
		} else {
			fmt.Printf("✅ Selection saved as '%s'\n", name)
		}
	}

	cfg.Scanner.SelectionMode = "list"
	cfg.Scanner.IncludedPaths = selectedPaths
	fmt.Printf("✅ Selected %d files for documentation\n", len(selectedPaths))

	return nil
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
