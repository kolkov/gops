package filesystem

import (
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("NormalFile", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "test.txt")
		content := []byte("Hello, World!")
		err := os.WriteFile(filePath, content, 0644)
		require.NoError(t, err)

		result, err := ReadFile(filePath, 1024)
		require.NoError(t, err)
		assert.Equal(t, content, result)
	})

	t.Run("FileTooLarge", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "large.txt")
		content := []byte(strings.Repeat("a", 1025))
		err := os.WriteFile(filePath, content, 0644)
		require.NoError(t, err)

		_, err = ReadFile(filePath, 1024)
		assert.ErrorIs(t, err, model.ErrFileTooLarge)
	})

	t.Run("FileNotExist", func(t *testing.T) {
		_, err := ReadFile(filepath.Join(tmpDir, "nonexistent.txt"), 1024)
		assert.Error(t, err)
	})

	t.Run("GopsConfig", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "gops_config.yaml")
		err := os.WriteFile(filePath, []byte("config"), 0644)
		require.NoError(t, err)

		_, err = ReadFile(filePath, 1024)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gops config file should be skipped")
	})
}

func TestShouldSkipFile(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *model.ScanConfig
		expected bool
		desc     string
	}{
		{
			name:     "gops_config.yaml",
			expected: true,
			desc:     "Should skip gops config file",
		},
		{
			name:     "go.sum",
			expected: true,
			desc:     "Should skip go.sum",
		},
		{
			name:     "output.md",
			cfg:      &model.ScanConfig{OutputFilename: "output.md"},
			expected: true,
			desc:     "Should skip output file",
		},
		{
			name:     "project_docs_2023.md",
			cfg:      &model.ScanConfig{OutputConfigFilename: "project_docs.md"},
			expected: true,
			desc:     "Should skip documentation files",
		},
		{
			name:     "package-lock.json",
			expected: true,
			desc:     "Should skip lock files",
		},
		{
			name:     "~$temp.docx",
			expected: true,
			desc:     "Should skip temporary files",
		},
		{
			name:     "error.log",
			cfg:      &model.ScanConfig{ExcludedPatterns: []string{"*.log"}},
			expected: true,
			desc:     "Should match excluded pattern",
		},
		{
			name:     "important.config",
			cfg:      &model.ScanConfig{ImportantFiles: []string{"important.config"}},
			expected: false,
			desc:     "Should not skip important files",
		},
		{
			name:     "normal.txt",
			expected: false,
			desc:     "Should not skip normal files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cfg := tt.cfg
			if cfg == nil {
				cfg = &model.ScanConfig{}
			}

			result := ShouldSkipFile(
				tt.name,
				cfg.OutputConfigFilename,
				cfg.OutputFilename,
				cfg.ExcludedPatterns,
				cfg.ImportantFiles,
			)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name     string
		excluded []string
		expected bool
		desc     string
	}{
		{
			name:     ".git",
			expected: true,
			desc:     "Should skip .git directory",
		},
		{
			name:     "node_modules",
			expected: true,
			desc:     "Should skip node_modules",
		},
		{
			name:     "dist",
			expected: true,
			desc:     "Should skip dist directory",
		},
		{
			name:     "__pycache__",
			expected: true,
			desc:     "Should skip __pycache__",
		},
		{
			name:     "excluded_dir",
			excluded: []string{"excluded_*"},
			expected: true,
			desc:     "Should match excluded pattern",
		},
		{
			name:     "src",
			expected: false,
			desc:     "Should not skip normal directory",
		},
		{
			name:     "app",
			expected: false,
			desc:     "Should not skip normal directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := shouldSkipDir(tt.name, tt.excluded)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipByType(t *testing.T) {
	tests := []struct {
		path     string
		cfg      *model.ScanConfig
		expected bool
		desc     string
	}{
		{
			path:     "styles.css",
			cfg:      &model.ScanConfig{IncludeStyles: false},
			expected: true,
			desc:     "Should skip CSS when not included",
		},
		{
			path:     "styles.scss",
			cfg:      &model.ScanConfig{IncludeStyles: false},
			expected: true,
			desc:     "Should skip SCSS when not included",
		},
		{
			path:     "index.html",
			cfg:      &model.ScanConfig{IncludeMarkup: false},
			expected: true,
			desc:     "Should skip HTML when not included",
		},
		{
			path:     "config.json",
			cfg:      &model.ScanConfig{IncludeConfigs: false},
			expected: true,
			desc:     "Should skip config files when not included",
		},
		{
			path:     "app.spec.js",
			cfg:      &model.ScanConfig{IncludeTests: false},
			expected: true,
			desc:     "Should skip test files when not included",
		},
		{
			path:     "important.config",
			cfg:      &model.ScanConfig{ImportantFiles: []string{"important.config"}, IncludeConfigs: false},
			expected: false,
			desc:     "Should not skip important files",
		},
		{
			path:     "app.js",
			expected: false,
			desc:     "Should not skip normal files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cfg := tt.cfg
			if cfg == nil {
				cfg = &model.ScanConfig{}
			}

			result := shouldSkipByType(tt.path, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем структуру проекта
	dirs := []string{
		"src",
		"src/components",
		"src/utils",
		"test",
		"dist",
		"node_modules",
	}

	for _, dir := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, dir), 0755))
	}

	files := []string{
		"src/main.js",
		"src/components/button.js",
		"src/utils/helpers.js",
		"test/app.spec.js",
		"dist/app.js",
		"node_modules/react/index.js",
		"config.json",
		"README.md",
		"important.config",
	}

	for _, file := range files {
		filePath := filepath.Join(tmpDir, file)
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
		require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))
	}

	cfg := &model.ScanConfig{
		ExcludedPatterns:     []string{"*.spec.js"},
		ImportantFiles:       []string{"important.config"},
		IncludeTests:         false,
		IncludeConfigs:       false,
		IncludeDocs:          true, // Включаем документацию
		OutputFilename:       "output.md",
		OutputConfigFilename: "project_docs.md",
		MaxFileSize:          1024,
	}

	log := logger.New(logger.InfoLevel)
	processed := make(map[string]bool)
	processedPaths := []string{}

	processFile := func(file *model.ProjectFile) {
		if !file.Skipped {
			processed[file.Path] = true
			processedPaths = append(processedPaths, file.Path)
		}
	}

	t.Run("FullScan", func(t *testing.T) {
		err := ScanProject(tmpDir, cfg, log, processFile)
		require.NoError(t, err)

		// Проверяем, что ожидаемые файлы были обработаны
		assert.True(t, processed[filepath.Join("src", "main.js")])
		assert.True(t, processed[filepath.Join("src", "components", "button.js")])
		assert.True(t, processed[filepath.Join("src", "utils", "helpers.js")])
		assert.True(t, processed["README.md"]) // Теперь должен быть включен
		assert.True(t, processed["important.config"])

		// Проверяем, что исключенные файлы не были обработаны
		assert.False(t, processed["test/app.spec.js"])
		assert.False(t, processed["dist/app.js"])
		assert.False(t, processed["node_modules/react/index.js"])
		assert.False(t, processed["config.json"])
	})

	t.Run("IncludeTests", func(t *testing.T) {
		cfgCopy := *cfg
		cfgCopy.IncludeTests = true
		cfgCopy.ExcludedPatterns = []string{} // Очищаем исключения
		processed = make(map[string]bool)

		err := ScanProject(tmpDir, &cfgCopy, log, processFile)
		require.NoError(t, err)
		assert.True(t, processed[filepath.Join("test", "app.spec.js")])
	})

	t.Run("IncludeConfigs", func(t *testing.T) {
		cfgCopy := *cfg
		cfgCopy.IncludeConfigs = true
		processed = make(map[string]bool)

		err := ScanProject(tmpDir, &cfgCopy, log, processFile)
		require.NoError(t, err)
		assert.True(t, processed["config.json"])
	})

	t.Run("FileTooLarge", func(t *testing.T) {
		// Создаем большой файл
		largeFilePath := filepath.Join(tmpDir, "large.bin")
		largeContent := make([]byte, 2048)
		require.NoError(t, os.WriteFile(largeFilePath, largeContent, 0644))

		cfgCopy := *cfg
		cfgCopy.MaxFileSize = 1024
		processed = make(map[string]bool)

		err := ScanProject(tmpDir, &cfgCopy, log, processFile)
		require.NoError(t, err)
		assert.False(t, processed["large.bin"]) // Файл должен быть пропущен из-за размера
	})

	t.Run("SkipDirectories", func(t *testing.T) {
		processed = make(map[string]bool)
		err := ScanProject(tmpDir, cfg, log, processFile)
		require.NoError(t, err)

		// Проверяем, что директории не обрабатываются как файлы
		for dir := range processed {
			assert.False(t, strings.HasSuffix(dir, "/"))
		}
	})
}

func TestProcessSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, World!")
	require.NoError(t, os.WriteFile(filePath, content, 0644))

	cfg := &model.ScanConfig{
		MaxFileSize:          1024,
		ImportantFiles:       []string{"important.config"},
		IncludeConfigs:       false,
		IncludeTests:         false,
		OutputFilename:       "output.md",
		OutputConfigFilename: "project_docs.md",
	}

	t.Run("NormalFile", func(t *testing.T) {
		file := processSingleFile(filePath, tmpDir, cfg)
		assert.Equal(t, "test.txt", file.Path)
		assert.Equal(t, "text", file.Lang)
		assert.Equal(t, content, file.Content)
		assert.False(t, file.Skipped)
	})

	t.Run("ImportantFile", func(t *testing.T) {
		importantPath := filepath.Join(tmpDir, "important.config")
		require.NoError(t, os.WriteFile(importantPath, content, 0644))

		file := processSingleFile(importantPath, tmpDir, cfg)
		assert.Equal(t, "important.config", file.Path)
		assert.False(t, file.Skipped)
	})

	t.Run("SkippedFile", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config.json")
		require.NoError(t, os.WriteFile(configPath, content, 0644))

		file := processSingleFile(configPath, tmpDir, cfg)
		assert.True(t, file.Skipped)
	})

	t.Run("FileTooLarge", func(t *testing.T) {
		largePath := filepath.Join(tmpDir, "large.bin")
		largeContent := make([]byte, 2048)
		require.NoError(t, os.WriteFile(largePath, largeContent, 0644))

		cfgCopy := *cfg
		cfgCopy.MaxFileSize = 1024

		file := processSingleFile(largePath, tmpDir, &cfgCopy)
		assert.True(t, file.Skipped)
	})

	t.Run("ErrorReadingFile", func(t *testing.T) {
		// Несуществующий файл
		file := processSingleFile(filepath.Join(tmpDir, "nonexistent.txt"), tmpDir, cfg)
		assert.True(t, file.Skipped)
	})
}

func TestShouldSkipFile_ExcludesGopsConfig(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"gops_config.yaml", true},
		{"gops_config_old.yaml", true},
		{"custom_gops_config.yml", true},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipFile(
				tt.name,
				"project_docs.md", // outputConfigFilename
				"output.md",       // outputFilename
				[]string{},        // excludedPatterns
				[]string{},        // importantFiles
			)
			assert.Equal(t, tt.expected, result, "File %s should be skipped=%v", tt.name, tt.expected)
		})
	}
}
