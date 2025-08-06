package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/internal/docgen"
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/scanner"
	"github.com/kolkov/gops/internal/scanner/golang"
	"github.com/kolkov/gops/internal/scanner/js"
	"github.com/kolkov/gops/internal/scanner/nx"
	"github.com/kolkov/gops/pkg/logger"
)

type ProjectScanner struct {
	rootDir    string
	outputFile string
	cfg        *config.Config
	logger     *logger.Logger
}

func NewProjectScanner(rootDir, outputFile string, cfg *config.Config, logger *logger.Logger) *ProjectScanner {
	// Явное исключение gops_config.yaml
	if strings.EqualFold(filepath.Base(outputFile), "gops_config.yaml") {
		logger.Fatal("Cannot use gops_config.yaml as output file", nil)
	}

	return &ProjectScanner{
		rootDir:    rootDir,
		outputFile: outputFile,
		cfg:        cfg,
		logger:     logger,
	}
}

func (s *ProjectScanner) Run(ctx context.Context) error {
	start := time.Now()
	defer func() {
		s.logger.Info("Scan completed", "duration", time.Since(start))
	}()

	ext := filepath.Ext(s.cfg.Output.Filename)
	base := s.cfg.Output.Filename[:len(s.cfg.Output.Filename)-len(ext)]

	scanCfg := convertConfig(s.cfg.Scanner, s.outputFile, base)

	detector := scanner.NewProjectDetector(s.rootDir, s.logger)
	projectType, err := detector.Detect()
	if err != nil {
		return fmt.Errorf("project detection failed: %w", err)
	}

	s.logger.Debug("Detected project type", "type", projectType)

	var projectScanner scanner.ProjectScanner
	switch projectType {
	case scanner.NxMonorepo:
		projectScanner = nx.NewScanner(s.rootDir, s.outputFile, &scanCfg, s.logger)
	case scanner.Go:
		projectScanner = golang.NewScanner(s.rootDir, s.outputFile, &scanCfg, s.logger)
	default:
		projectScanner = js.NewScanner(s.rootDir, s.outputFile, &scanCfg, s.logger)
	}

	var docGenerator docgen.Generator
	switch s.cfg.Output.Format {
	case "html":
		// Placeholder for HTML generator
	default:
		docGenerator = docgen.NewMarkdownGenerator(s.outputFile, s.logger)
	}
	defer docGenerator.Close()

	if err := projectScanner.Scan(ctx, docGenerator); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	return nil
}

func convertConfig(cfg config.ScannerConfig, outputFile, outConfigFilename string) model.ScanConfig {
	excluded := make([]string, len(cfg.ExcludedPatterns))
	copy(excluded, cfg.ExcludedPatterns)

	baseName := filepath.Base(outConfigFilename)
	if baseName != "" {
		baseWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		docPattern := baseWithoutExt + "*.md"
		excluded = append(excluded, docPattern)
	}

	// Системные исключения (всегда применяются)
	systemExcludes := []string{
		"node_modules",                                     // Исключить всю директорию node_modules
		"package-lock.json", "yarn.lock", "pnpm-lock.yaml", // Lock-файлы пакетных менеджеров
		"go.sum",                   // Go lock-файл
		".git", ".idea", ".vscode", // Системные директории
		"dist", "build", "out", "bin", "obj", // Артефакты сборки
		"__pycache__", ".pytest_cache", // Кэш Python
		"coverage", ".nyc_output", // Отчеты покрытия
		".angular", ".nx", ".next", ".nuxt", // Фреймворк-специфичные
		"__tests__", "__snapshots__", "e2e", // Тестовые директории
		"*.log", "*.tmp", "*.bak", // Временные файлы
		"*.png", "*.jpg", "*.jpeg", "*.gif", "*.ico", "*.svg", "*.bmp", "*.webp", // Изображения
		"project_structure.txt", // Файл структуры проекта (специальное исключение)
	}

	// Добавляем системные исключения, если их еще нет в конфиге
	for _, excl := range systemExcludes {
		found := false
		for _, pattern := range excluded {
			if pattern == excl {
				found = true
				break
			}
		}
		if !found {
			excluded = append(excluded, excl)
		}
	}

	return model.ScanConfig{
		IncludeTests:         cfg.IncludeTests,
		IncludeConfigs:       cfg.IncludeConfigs,
		IncludeMarkup:        cfg.IncludeMarkup,
		IncludeStyles:        cfg.IncludeStyles,
		ExcludedPatterns:     excluded,
		MaxFileSize:          cfg.MaxFileSize,
		ParallelWorkers:      cfg.ParallelWorkers,
		OutputFilename:       outputFile,
		OutputConfigFilename: outConfigFilename,
	}
}

func GenerateOutputFilename(cfg *config.Config) string {
	if cfg.Output.Filename == "" {
		return fmt.Sprintf("project_docs_%s.md", time.Now().Format("20060102_150405"))
	}

	if cfg.Output.AppendTimestamp {
		ext := filepath.Ext(cfg.Output.Filename)
		base := cfg.Output.Filename[:len(cfg.Output.Filename)-len(ext)]
		return fmt.Sprintf("%s_%s%s", base, time.Now().Format("20060102_150405"), ext)
	}

	return cfg.Output.Filename
}
