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
		assert.Contains(t, cfg.Scanner.ExcludedPatterns, "*.tmp")
		assert.Equal(t, 8, cfg.Scanner.ParallelWorkers)
		assert.Equal(t, 10*time.Minute, time.Duration(cfg.Scanner.Timeout))
		assert.Equal(t, "markdown", cfg.Output.Format)
		assert.Equal(t, "project_docs.md", cfg.Output.Filename)
		assert.True(t, cfg.Output.AppendTimestamp)
	})

	t.Run("InvalidDuration", func(t *testing.T) {
		configContent := []byte(`
scanner:
  timeout: invalid
`)
		configPath := filepath.Join(tmpDir, "config_invalid.yaml")
		require.NoError(t, os.WriteFile(configPath, configContent, 0644))

		_, err := Load(configPath)
		assert.Error(t, err)
	})
}
