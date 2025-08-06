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
	log := logger.New(logger.InfoLevel)
	defer log.Sync()

	// Создаем файлы Go проекта, чтобы детектор определил тип проекта
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {}"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "internal"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "internal", "app.go"), []byte("package internal"), 0644))

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
			Filename: "test_output.md",
		},
	}

	scanner := NewProjectScanner(tmpDir, "test_output.md", cfg, log)
	err := scanner.Run(context.Background())
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
	})
}
