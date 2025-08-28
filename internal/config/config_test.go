package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("DefaultValues", func(t *testing.T) {
		configContent := []byte(`
scanner:
  include_tests: true
output:
  format: html
`)
		configPath := filepath.Join(tmpDir, "config_default.yaml")
		require.NoError(t, os.WriteFile(configPath, configContent, 0644))

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, int64(2*1024*1024), cfg.Scanner.MaxFileSize)
		assert.Equal(t, 4, cfg.Scanner.ParallelWorkers)
		assert.Equal(t, 5*time.Minute, time.Duration(cfg.Scanner.Timeout))
		assert.True(t, cfg.Scanner.IncludeTests)
		assert.Equal(t, "html", cfg.Output.Format)
	})

	t.Run("FullConfig", func(t *testing.T) {
		configContent := []byte(`
scanner:
  max_file_size: 5242880
  include_tests: false
  include_configs: true
  include_markup: false
  include_styles: true
  include_docs: true
  excluded_patterns: 
    - "*.tmp"
    - "logs/*"
  parallel_workers: 8
  timeout: 10m
output:
  format: markdown
  filename: project_docs.md
  append_timestamp: true
`)
		configPath := filepath.Join(tmpDir, "config_full.yaml")
		require.NoError(t, os.WriteFile(configPath, configContent, 0644))

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, int64(5242880), cfg.Scanner.MaxFileSize)
		assert.False(t, cfg.Scanner.IncludeTests)
		assert.True(t, cfg.Scanner.IncludeConfigs)
		assert.False(t, cfg.Scanner.IncludeMarkup)
		assert.True(t, cfg.Scanner.IncludeStyles)
		assert.True(t, cfg.Scanner.IncludeDocs)
		assert.Contains(t, cfg.Scanner.ExcludedPatterns, "*.tmp")
		assert.Equal(t, 8, cfg.Scanner.ParallelWorkers)
		assert.Equal(t, 10*time.Minute, time.Duration(cfg.Scanner.Timeout))
		assert.Equal(t, "markdown", cfg.Output.Format)
		assert.Equal(t, "project_docs.md", cfg.Output.Filename)
		assert.True(t, cfg.Output.AppendTimestamp)
	})

	t.Run("PartialConfigWithDefaults", func(t *testing.T) {
		configContent := []byte(`
scanner:
  include_tests: false
  include_configs: false
output:
  format: html
`)
		configPath := filepath.Join(tmpDir, "config_partial.yaml")
		require.NoError(t, os.WriteFile(configPath, configContent, 0644))

		cfg, err := Load(configPath)
		require.NoError(t, err)

		// Проверяем, что явно установленные значения сохраняются
		assert.False(t, cfg.Scanner.IncludeTests)
		assert.False(t, cfg.Scanner.IncludeConfigs)
		assert.Equal(t, "html", cfg.Output.Format)

		// Проверяем, что применяются значения по умолчанию для не указанных полей
		assert.Equal(t, int64(2*1024*1024), cfg.Scanner.MaxFileSize)
		assert.Equal(t, 4, cfg.Scanner.ParallelWorkers)
		assert.Equal(t, 5*time.Minute, time.Duration(cfg.Scanner.Timeout))
		assert.True(t, cfg.Scanner.IncludeMarkup)
		assert.True(t, cfg.Scanner.IncludeStyles)
		assert.False(t, cfg.Scanner.IncludeDocs)
		assert.Equal(t, "project_docs.md", cfg.Output.Filename)
		assert.True(t, cfg.Output.AppendTimestamp)
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, int64(2*1024*1024), cfg.Scanner.MaxFileSize)
	assert.Equal(t, 4, cfg.Scanner.ParallelWorkers)
	assert.Equal(t, 5*time.Minute, time.Duration(cfg.Scanner.Timeout))
	assert.False(t, cfg.Scanner.IncludeTests)
	assert.True(t, cfg.Scanner.IncludeConfigs)
	assert.True(t, cfg.Scanner.IncludeMarkup)
	assert.True(t, cfg.Scanner.IncludeStyles)
	assert.False(t, cfg.Scanner.IncludeDocs)
	assert.Len(t, cfg.Scanner.ExcludedPatterns, 6)
	assert.Equal(t, "markdown", cfg.Output.Format)
	assert.Equal(t, "project_docs.md", cfg.Output.Filename)
	assert.True(t, cfg.Output.AppendTimestamp)
}

func TestLoadOrDefault(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("FileExists", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "existing.yaml")
		configContent := []byte(`
scanner:
  include_tests: true
  max_file_size: 1048576
`)
		require.NoError(t, os.WriteFile(configPath, configContent, 0644))

		cfg, exists, err := LoadOrDefault(configPath)
		require.NoError(t, err)
		assert.True(t, exists, "Config file should exist")
		assert.NotNil(t, cfg)
		assert.True(t, cfg.Scanner.IncludeTests)
		assert.Equal(t, int64(1048576), cfg.Scanner.MaxFileSize)
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "nonexistent.yaml")

		cfg, exists, err := LoadOrDefault(configPath)
		require.NoError(t, err)
		assert.False(t, exists, "Config file should not exist")
		assert.NotNil(t, cfg)

		// Should return default config
		assert.Equal(t, int64(2*1024*1024), cfg.Scanner.MaxFileSize)
		assert.False(t, cfg.Scanner.IncludeTests)
		assert.Equal(t, "markdown", cfg.Output.Format)
	})

	t.Run("InvalidFile", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("invalid: yaml: content"), 0644))

		cfg, exists, err := LoadOrDefault(configPath)
		assert.Error(t, err)
		assert.True(t, exists, "File exists but is invalid")
		assert.Nil(t, cfg)
	})
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save_test.yaml")

	cfg := DefaultConfig()
	cfg.Scanner.IncludeTests = true
	cfg.Output.Filename = "custom_output.md"

	err := Save(cfg, configPath)
	require.NoError(t, err)

	// Load the saved config
	loadedCfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, cfg.Scanner.IncludeTests, loadedCfg.Scanner.IncludeTests)
	assert.Equal(t, cfg.Output.Filename, loadedCfg.Output.Filename)
}

func TestCreateSampleConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sample.yaml")

	err := CreateSampleConfig(configPath)
	require.NoError(t, err)

	// Check file exists
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.True(t, info.Size() > 0, "Sample config file should not be empty")

	// Try to load the sample config
	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestDurationMarshalYAML(t *testing.T) {
	d := Duration(5 * time.Minute)
	result, err := d.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, "5m0s", result)
}
