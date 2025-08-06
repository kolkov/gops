package app

import (
	"context"
	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectScanner_Run(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(logger.InfoLevel)
	defer log.Sync()

	// Create mock project files to avoid "unsupported project type" error
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644))

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
		filename := GenerateOutputFilename(cfg) // Исправлено на экспортируемую функцию
		assert.Contains(t, filename, "project_docs_")
		assert.Contains(t, filename, ".md")
	})

	t.Run("CustomFilename", func(t *testing.T) {
		cfg := &config.Config{Output: config.OutputConfig{
			Filename: "custom.md",
		}}
		filename := GenerateOutputFilename(cfg) // Исправлено на экспортируемую функцию
		assert.Equal(t, "custom.md", filename)
	})

	t.Run("AppendTimestamp", func(t *testing.T) {
		cfg := &config.Config{Output: config.OutputConfig{
			Filename:        "custom.md",
			AppendTimestamp: true,
		}}
		filename := GenerateOutputFilename(cfg) // Исправлено на экспортируемую функцию
		assert.Contains(t, filename, "custom_")
		assert.Contains(t, filename, ".md")
	})
}
