package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectScanner_Run(t *testing.T) {
	tmpDir := t.TempDir()

	// Очищаем все файлы после теста
	defer func() {
		// Удаляем все .md файлы, созданные во время теста
		files, _ := filepath.Glob(filepath.Join(tmpDir, "*.md"))
		for _, file := range files {
			_ = os.Remove(file)
		}
	}()

	log := logger.New(logger.InfoLevel)
	defer log.Sync()

	// Создаем файлы Go проекта
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {}"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "internal"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "internal", "app.go"), []byte("package internal"), 0644))

	// Используем временный файл для вывода
	outputFile := filepath.Join(tmpDir, "test_output.md")

	cfg := &config.Config{
		Scanner: config.ScannerConfig{
			MaxFileSize:      1024,
			IncludeTests:     true,
			IncludeConfigs:   true,
			IncludeMarkup:    true,
			IncludeStyles:    true,
			ExcludedPatterns: []string{"excluded.*"},
			ParallelWorkers:  4,
		},
		Output: config.OutputConfig{
			Format:   "markdown",
			Filename: outputFile, // Используем временный файл
		},
	}

	// Перенаправляем stdin для избежания интерактивного ввода
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tmpFile, _ := os.CreateTemp("", "stdin")
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.WriteString("\n") // Simulate Enter (no additional files)
	_, _ = tmpFile.Seek(0, 0)
	os.Stdin = tmpFile

	scanner := NewProjectScanner(tmpDir, outputFile, cfg, log)
	err := scanner.Run(context.Background())
	assert.NoError(t, err)

	// Проверяем, что файл создан
	_, err = os.Stat(outputFile)
	assert.NoError(t, err)
}

func TestConvertConfig(t *testing.T) {
	scannerCfg := config.ScannerConfig{
		MaxFileSize:      1024,
		IncludeTests:     true,
		IncludeConfigs:   true,
		IncludeMarkup:    true,
		IncludeStyles:    true,
		ExcludedPatterns: []string{"excluded.*"},
		ParallelWorkers:  4,
	}

	scanCfg := convertConfig(scannerCfg, "test_output.md", "test_config")

	assert.True(t, scanCfg.IncludeTests)
	assert.True(t, scanCfg.IncludeConfigs)
	assert.Contains(t, scanCfg.ExcludedPatterns, "node_modules")
	assert.Contains(t, scanCfg.ExcludedPatterns, "test_config*.md")
	assert.Contains(t, scanCfg.ExcludedPatterns, "excluded.*")
}

func TestGenerateOutputFilename(t *testing.T) {
	t.Run("DefaultFilename", func(t *testing.T) {
		cfg := &config.Config{Output: config.OutputConfig{}}
		filename := GenerateOutputFilename(cfg)
		assert.Contains(t, filename, "project_docs_")
		assert.Contains(t, filename, ".md")
		// Удаляем созданный файл
		defer os.Remove(filename)
	})

	t.Run("CustomFilename", func(t *testing.T) {
		cfg := &config.Config{Output: config.OutputConfig{
			Filename: "custom.md",
		}}
		filename := GenerateOutputFilename(cfg)
		assert.Equal(t, "custom.md", filename)
	})

	t.Run("AppendTimestamp", func(t *testing.T) {
		cfg := &config.Config{Output: config.OutputConfig{
			Filename:        "custom.md",
			AppendTimestamp: true,
		}}
		filename := GenerateOutputFilename(cfg)
		assert.Contains(t, filename, "custom_")
		assert.Contains(t, filename, ".md")
		// Удаляем созданный файл
		defer func() {
			files, _ := filepath.Glob("custom_*.md")
			for _, f := range files {
				_ = os.Remove(f)
			}
		}()
	})
}

func TestConvertConfig_SkipMarkdownByDefault(t *testing.T) {
	scannerCfg := config.ScannerConfig{IncludeDocs: false}
	scanCfg := convertConfig(scannerCfg, "out.md", "config")
	assert.Contains(t, scanCfg.ExcludedPatterns, "*.md")
}

func TestConvertConfig_IncludeMarkdownWhenEnabled(t *testing.T) {
	scannerCfg := config.ScannerConfig{IncludeDocs: true}
	scanCfg := convertConfig(scannerCfg, "out.md", "config")
	assert.NotContains(t, scanCfg.ExcludedPatterns, "*.md")
}
