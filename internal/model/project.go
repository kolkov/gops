package model

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type Duration time.Duration

const (
	MaxFileSize = 2 * 1024 * 1024 // 2MB
)

var (
	ErrFileTooLarge = errors.New("file size exceeds maximum limit")
)

type ProjectMeta struct {
	Name    string
	Type    string
	RootDir string
}

type ProjectFile struct {
	Path    string
	Content []byte
	Lang    string
	Skipped bool
}

type NxProject struct {
	Name      string
	Type      string
	Root      string
	SourceDir string
}

type ScanConfig struct {
	// Основные настройки
	IncludeTests         bool
	IncludeConfigs       bool
	IncludeMarkup        bool
	IncludeStyles        bool
	ExcludedPatterns     []string
	MaxFileSize          int64
	ParallelWorkers      int
	OutputFilename       string
	OutputConfigFilename string
	ImportantFiles       []string

	// Настройки выбора файлов
	SelectionMode   string   // "all", "interactive", "patterns", "list"
	IncludedPaths   []string // Список конкретных путей для включения
	IncludePatterns []string // Паттерны для включения файлов
}

// IsFileIncluded проверяет, должен ли файл быть включен в документацию
func (sc *ScanConfig) IsFileIncluded(relPath string) bool {
	// Если режим "all" или не задан - включаем все файлы
	if sc.SelectionMode == "" || sc.SelectionMode == "all" {
		return true
	}

	// Для режима "list" проверяем точное совпадение путей
	if sc.SelectionMode == "list" && len(sc.IncludedPaths) > 0 {
		normalizedPath := filepath.ToSlash(relPath)
		for _, includedPath := range sc.IncludedPaths {
			normalizedIncluded := filepath.ToSlash(includedPath)
			if normalizedPath == normalizedIncluded {
				return true
			}
			// Проверяем, если это файл внутри выбранной папки
			if strings.HasPrefix(normalizedPath, normalizedIncluded+"/") {
				return true
			}
		}
		return false
	}

	// Для режима "patterns" проверяем соответствие паттернам
	if sc.SelectionMode == "patterns" && len(sc.IncludePatterns) > 0 {
		normalizedPath := filepath.ToSlash(relPath)
		for _, pattern := range sc.IncludePatterns {
			pattern = filepath.ToSlash(pattern)

			// Если паттерн содержит **, используем специальную логику
			if strings.Contains(pattern, "**") {
				if matchDoublestar(pattern, normalizedPath) {
					return true
				}
			} else {
				// Для простых паттернов нужна специальная обработка
				// чтобы правильно учитывать структуру директорий
				if matchSimplePattern(pattern, normalizedPath) {
					return true
				}
			}
		}
		return false
	}

	return true
}

// HasFileSelection проверяет, активен ли режим выбора файлов
func (sc *ScanConfig) HasFileSelection() bool {
	return sc.SelectionMode == "list" || sc.SelectionMode == "patterns" || sc.SelectionMode == "interactive"
}

// GetSelectedFilesCount возвращает количество выбранных файлов
func (sc *ScanConfig) GetSelectedFilesCount() int {
	switch sc.SelectionMode {
	case "list":
		return len(sc.IncludedPaths)
	case "patterns":
		return len(sc.IncludePatterns)
	default:
		return 0
	}
}

func GetFileLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".jsx":
		return "jsx"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".html", ".htm":
		return "html"
	case ".scss", ".sass":
		return "scss"
	case ".css":
		return "css"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".sh", ".bash":
		return "bash"
	case ".xml":
		return "xml"
	case ".sql":
		return "sql"
	default:
		return "text"
	}
}

// matchDoublestar проверяет соответствие пути паттерну с поддержкой **
// ** означает "ноль или более директорий"
// ВАЖНО: эта функция должна вызываться ТОЛЬКО для паттернов содержащих **
// matchDoublestar проверяет соответствие пути паттерну с поддержкой **
// ** означает "ноль или более директорий"
func matchDoublestar(pattern, path string) bool {
	// Нормализуем пути
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Если паттерн не содержит **, используем matchSimplePattern
	if !strings.Contains(pattern, "**") {
		return matchSimplePattern(pattern, path)
	}

	// Обработка паттернов вида **/*.ext
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		parts := strings.Split(path, "/")

		// Проверяем каждый возможный суффикс пути
		for i := 0; i < len(parts); i++ {
			subpath := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(suffix, subpath); matched {
				return true
			}
		}
		return false
	}

	// Обработка паттернов вида path/**
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}

	// Обработка паттернов вида prefix/**/suffix
	if idx := strings.Index(pattern, "/**/"); idx >= 0 {
		prefix := pattern[:idx]
		suffix := pattern[idx+4:]

		// Проверяем префикс
		if !strings.HasPrefix(path, prefix+"/") && path != prefix {
			return false
		}

		// Остаток пути после prefix/
		remainder := path
		if path != prefix {
			remainder = path[len(prefix)+1:]
		}

		// Для паттерна вида src/**/test/*.go
		// ** в середине означает "ноль или более директорий"
		parts := strings.Split(remainder, "/")

		// Проверяем соответствие суффиксу
		for i := 0; i <= len(parts); i++ {
			subpath := strings.Join(parts[i:], "/")
			if subpath != "" {
				if matched, _ := filepath.Match(suffix, subpath); matched {
					return true
				}
			}
		}

		// Специальная обработка для паттернов с путями в суффиксе
		if strings.Contains(suffix, "/") {
			suffixParts := strings.Split(suffix, "/")
			for i := 0; i <= len(parts)-len(suffixParts); i++ {
				match := true
				for j, suffixPart := range suffixParts {
					if i+j >= len(parts) {
						match = false
						break
					}
					if strings.Contains(suffixPart, "*") || strings.Contains(suffixPart, "?") {
						if matched, _ := filepath.Match(suffixPart, parts[i+j]); !matched {
							match = false
							break
						}
					} else {
						if suffixPart != parts[i+j] {
							match = false
							break
						}
					}
				}
				if match {
					return true
				}
			}
		}
	}

	return false
}

// matchSimplePattern проверяет соответствие пути простому паттерну (без **)
// Учитывает структуру директорий: * не может соответствовать /
func matchSimplePattern(pattern, path string) bool {
	// Нормализуем пути
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Разбиваем паттерн и путь на сегменты
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	// Количество сегментов должно совпадать
	if len(patternParts) != len(pathParts) {
		return false
	}

	// Проверяем каждый сегмент
	for i := 0; i < len(patternParts); i++ {
		// Для каждого сегмента используем filepath.Match
		matched, err := filepath.Match(patternParts[i], pathParts[i])
		if err != nil || !matched {
			return false
		}
	}

	return true
}
