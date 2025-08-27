package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFileLanguage(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"main.go", "go"},
		{"script.js", "javascript"},
		{"component.tsx", "tsx"},
		{"styles.css", "css"},
		{"styles.scss", "scss"},
		{"index.html", "html"},
		{"config.json", "json"},
		{"README.md", "markdown"},
		{"main.py", "python"},
		{"App.java", "java"},
		{"main.rs", "rust"},
		{"Dockerfile", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetFileLanguage(tt.file))
		})
	}
}

func TestScanConfig_IsFileIncluded(t *testing.T) {
	t.Run("ModeAll", func(t *testing.T) {
		cfg := &ScanConfig{
			SelectionMode: "all",
		}

		assert.True(t, cfg.IsFileIncluded("any/file.go"))
		assert.True(t, cfg.IsFileIncluded("other/file.js"))
	})

	t.Run("ModeList", func(t *testing.T) {
		cfg := &ScanConfig{
			SelectionMode: "list",
			IncludedPaths: []string{
				"src/main.go",
				"src/utils",
				"README.md",
			},
		}

		// Точное совпадение
		assert.True(t, cfg.IsFileIncluded("src/main.go"))
		assert.True(t, cfg.IsFileIncluded("README.md"))

		// Файл внутри выбранной папки
		assert.True(t, cfg.IsFileIncluded("src/utils/helper.go"))
		assert.True(t, cfg.IsFileIncluded("src/utils/deep/nested.go"))

		// Не включенные файлы
		assert.False(t, cfg.IsFileIncluded("src/test.go"))
		assert.False(t, cfg.IsFileIncluded("docs/api.md"))
	})

	t.Run("ModePatterns", func(t *testing.T) {
		cfg := &ScanConfig{
			SelectionMode: "patterns",
			IncludePatterns: []string{
				"*.md",
				"src/*.go",
				"**/*.service.ts",
			},
		}

		// Соответствие паттернам
		assert.True(t, cfg.IsFileIncluded("README.md"))
		assert.True(t, cfg.IsFileIncluded("src/main.go"))
		assert.True(t, cfg.IsFileIncluded("src/services/user.service.ts"))
		assert.True(t, cfg.IsFileIncluded("deep/nested/auth.service.ts"))

		// Не соответствует паттернам
		assert.False(t, cfg.IsFileIncluded("src/utils/helper.go"))
		assert.False(t, cfg.IsFileIncluded("test.txt"))
	})

	t.Run("EmptyMode", func(t *testing.T) {
		cfg := &ScanConfig{}

		// По умолчанию включаем все
		assert.True(t, cfg.IsFileIncluded("any/file.go"))
	})

	t.Run("UnknownMode", func(t *testing.T) {
		cfg := &ScanConfig{
			SelectionMode: "unknown",
		}

		// Неизвестный режим - включаем все
		assert.True(t, cfg.IsFileIncluded("any/file.go"))
	})
}

func TestScanConfig_HasFileSelection(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"all", false},
		{"list", true},
		{"patterns", true},
		{"interactive", true},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			cfg := &ScanConfig{
				SelectionMode: tt.mode,
			}
			assert.Equal(t, tt.expected, cfg.HasFileSelection())
		})
	}
}

func TestScanConfig_GetSelectedFilesCount(t *testing.T) {
	t.Run("ListMode", func(t *testing.T) {
		cfg := &ScanConfig{
			SelectionMode: "list",
			IncludedPaths: []string{"file1.go", "file2.go", "file3.go"},
		}
		assert.Equal(t, 3, cfg.GetSelectedFilesCount())
	})

	t.Run("PatternsMode", func(t *testing.T) {
		cfg := &ScanConfig{
			SelectionMode:   "patterns",
			IncludePatterns: []string{"*.go", "*.md"},
		}
		assert.Equal(t, 2, cfg.GetSelectedFilesCount())
	})

	t.Run("AllMode", func(t *testing.T) {
		cfg := &ScanConfig{
			SelectionMode: "all",
		}
		assert.Equal(t, 0, cfg.GetSelectedFilesCount())
	})
}

func TestScanConfig_PathNormalization(t *testing.T) {
	cfg := &ScanConfig{
		SelectionMode: "list",
		IncludedPaths: []string{
			"src\\windows\\path",
			"src/unix/path",
		},
	}

	// Проверка нормализации путей Windows
	assert.True(t, cfg.IsFileIncluded("src/windows/path"))
	assert.True(t, cfg.IsFileIncluded("src\\windows\\path"))

	// Проверка нормализации путей Unix
	assert.True(t, cfg.IsFileIncluded("src/unix/path"))
	assert.True(t, cfg.IsFileIncluded("src\\unix\\path"))
}

func TestScanConfig_ComplexPatterns(t *testing.T) {
	cfg := &ScanConfig{
		SelectionMode: "patterns",
		IncludePatterns: []string{
			"**/*.go", // Изменено с src/**/*.go
			"test/**/*_test.go",
			"docs/*.md",
		},
	}

	tests := []struct {
		path     string
		included bool
	}{
		{"src/main.go", true}, // Теперь соответствует **/*.go
		{"src/utils/helper.go", true},
		{"src/deep/nested/file.go", true},
		{"test/unit_test.go", true}, // Теперь соответствует test/**/*_test.go
		{"test/deep/integration_test.go", true},
		{"docs/README.md", true},
		{"docs/api/swagger.md", false}, // не соответствует docs/*.md (только в корне docs)
		{"main.go", true},              // Теперь соответствует **/*.go
		{"src/file.js", false},         // не .go файл
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.included, cfg.IsFileIncluded(tt.path),
				"Path %s should be included=%v", tt.path, tt.included)
		})
	}
}

func TestScanConfig_FolderSelection(t *testing.T) {
	cfg := &ScanConfig{
		SelectionMode: "list",
		IncludedPaths: []string{
			"src/features/auth",
			"src/features/user",
			"docs",
		},
	}

	tests := []struct {
		path     string
		included bool
	}{
		// Файлы в выбранных папках
		{"src/features/auth/login.ts", true},
		{"src/features/auth/logout.ts", true},
		{"src/features/auth/deep/nested.ts", true},
		{"src/features/user/profile.ts", true},
		{"docs/README.md", true},
		{"docs/api/swagger.yaml", true},

		// Файлы вне выбранных папок
		{"src/features/admin/panel.ts", false},
		{"src/main.ts", false},
		{"README.md", false},
		{"tests/auth.test.ts", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.included, cfg.IsFileIncluded(tt.path),
				"Path %s should be included=%v", tt.path, tt.included)
		})
	}
}

func TestMatchDoublestar(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
		desc     string
	}{
		// Паттерны с ** в начале
		{"**/*.go", "main.go", true, "go файл в корне"},
		{"**/*.go", "src/main.go", true, "go файл в подпапке"},
		{"**/*.go", "deep/nested/file.go", true, "go файл глубоко"},
		{"**/*.go", "main.js", false, "не go файл"},

		// Паттерны с ** в середине
		{"src/**/*.go", "src/main.go", true, "файл в корне src"},
		{"src/**/*.go", "src/utils/helper.go", true, "файл в подпапке src"},
		{"src/**/*.go", "src/deep/nested/file.go", true, "файл глубоко в src"},
		{"src/**/*.go", "main.go", false, "файл не в src"},
		{"src/**/*.go", "src/file.js", false, "не .go файл"},

		// Паттерны с ** в конце
		{"src/**", "src/main.go", true, "любой файл в src"},
		{"src/**", "src/utils/helper.go", true, "файл в подпапке src"},
		{"src/**", "main.go", false, "файл не в src"},

		// Паттерны с конкретным суффиксом
		{"test/**/*_test.go", "test/unit_test.go", true, "тестовый файл в test"},
		{"test/**/*_test.go", "test/integration/db_test.go", true, "тестовый файл в подпапке"},
		{"test/**/*_test.go", "test/helper.go", false, "не тестовый файл"},

		// Простые паттерны без ** (должны обрабатываться matchSimplePattern)
		{"*.md", "README.md", true, "md файл в корне"},
		{"*.md", "docs/README.md", false, "md файл не в корне"},
		{"docs/*.md", "docs/README.md", true, "md файл в docs"},
		{"docs/*.md", "docs/api/swagger.md", false, "md файл в подпапке docs"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := matchDoublestar(tt.pattern, tt.path)
			assert.Equal(t, tt.expected, result, "Pattern '%s' with path '%s'", tt.pattern, tt.path)
		})
	}
}

func TestPatternEdgeCases(t *testing.T) {
	cfg := &ScanConfig{
		SelectionMode: "patterns",
		IncludePatterns: []string{
			"**/*.go",
			"src/**/test/*.go",
			"**/vendor/**",
		},
	}

	tests := []struct {
		path     string
		included bool
		desc     string
	}{
		{"main.go", true, "go файл в корне"},
		{"deep/nested/file.go", true, "go файл глубоко вложенный"},
		{"src/module/test/unit.go", true, "соответствует специфичному паттерну"},
		{"src/test/unit.go", true, "соответствует паттерну с нулем директорий"}, // Изменено с false на true
		{"vendor/package/file.go", true, "файл в vendor"},
		{"src/vendor/lib/code.js", true, "vendor в подпапке"},
		{"main.js", false, "не go файл"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.included, cfg.IsFileIncluded(tt.path), tt.desc)
		})
	}
}
