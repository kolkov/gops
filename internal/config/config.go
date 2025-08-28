package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// NullBool представляет nullable boolean значение
type NullBool struct {
	Bool  bool
	Valid bool // Valid is true if Bool has been set
}

// UnmarshalYAML реализует интерфейс yaml.Unmarshaler
func (n *NullBool) UnmarshalYAML(value *yaml.Node) error {
	var b bool
	if err := value.Decode(&b); err != nil {
		return err
	}
	n.Bool = b
	n.Valid = true
	return nil
}

// MarshalYAML реализует интерфейс yaml.Marshaler
func (n NullBool) MarshalYAML() (interface{}, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Bool, nil
}

// IsTrue возвращает true если значение явно установлено и равно true
func (n NullBool) IsTrue() bool {
	return n.Valid && n.Bool
}

// IsFalse возвращает true если значение явно установлено и равно false
func (n NullBool) IsFalse() bool {
	return n.Valid && !n.Bool
}

// Константы для режимов выбора
const (
	SelectionModeAll         = "all"         // Включить все файлы (текущее поведение)
	SelectionModeInteractive = "interactive" // Интерактивный выбор
	SelectionModePatterns    = "patterns"    // По паттернам из конфига
	SelectionModeList        = "list"        // По списку путей из конфига
)

// Config представляет финальную конфигурацию с обычными bool значениями
type Config struct {
	Scanner ScannerConfig `yaml:"scanner"`
	Output  OutputConfig  `yaml:"output"`
}

// ScannerConfig представляет финальную конфигурацию сканера
type ScannerConfig struct {
	MaxFileSize       int64    `yaml:"max_file_size"`
	IncludeTests      bool     `yaml:"include_tests"`
	IncludeConfigs    bool     `yaml:"include_configs"`
	IncludeMarkup     bool     `yaml:"include_markup"`
	IncludeStyles     bool     `yaml:"include_styles"`
	IncludeDocs       bool     `yaml:"include_docs"`
	ExcludedPatterns  []string `yaml:"excluded_patterns"`
	ParallelWorkers   int      `yaml:"parallel_workers"`
	Timeout           Duration `yaml:"timeout"`
	DocumentationMode string   `yaml:"documentation_mode"`
	SelectionMode     string   `yaml:"selection_mode"`
	IncludedPaths     []string `yaml:"included_paths"`
	IncludePatterns   []string `yaml:"include_patterns"`
	ImportantFiles    []string `yaml:"important_files"`
	ShowFullStructure bool     `yaml:"show_full_structure"`
}

// OutputConfig представляет финальную конфигурацию вывода
type OutputConfig struct {
	Format          string `yaml:"format"`
	Filename        string `yaml:"filename"`
	AppendTimestamp bool   `yaml:"append_timestamp"`
}

// rawConfig представляет промежуточную структуру для загрузки с NullBool
type rawConfig struct {
	Scanner struct {
		MaxFileSize       int64    `yaml:"max_file_size"`
		IncludeTests      NullBool `yaml:"include_tests"`
		IncludeConfigs    NullBool `yaml:"include_configs"`
		IncludeMarkup     NullBool `yaml:"include_markup"`
		IncludeStyles     NullBool `yaml:"include_styles"`
		IncludeDocs       NullBool `yaml:"include_docs"`
		ExcludedPatterns  []string `yaml:"excluded_patterns"`
		ParallelWorkers   int      `yaml:"parallel_workers"`
		Timeout           Duration `yaml:"timeout"`
		DocumentationMode string   `yaml:"documentation_mode"`
		SelectionMode     string   `yaml:"selection_mode"`
		IncludedPaths     []string `yaml:"included_paths"`
		IncludePatterns   []string `yaml:"include_patterns"`
		ImportantFiles    []string `yaml:"important_files"`
		ShowFullStructure bool     `yaml:"show_full_structure"`
	} `yaml:"scanner"`
	Output struct {
		Format          string `yaml:"format"`
		Filename        string `yaml:"filename"`
		AppendTimestamp *bool  `yaml:"append_timestamp"`
	} `yaml:"output"`
}

type Duration time.Duration

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(dur)
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Scanner: ScannerConfig{
			MaxFileSize:    2 * 1024 * 1024, // 2MB
			IncludeTests:   false,
			IncludeConfigs: true,
			IncludeMarkup:  true,
			IncludeStyles:  true,
			IncludeDocs:    false,
			ExcludedPatterns: []string{
				".gitignore",
				".DS_Store",
				"Thumbs.db",
				"*.log",
				"*.tmp",
				"*.bak",
			},
			ParallelWorkers:   4,
			Timeout:           Duration(5 * time.Minute),
			SelectionMode:     SelectionModeAll,
			IncludedPaths:     []string{},
			IncludePatterns:   []string{},
			ImportantFiles:    []string{},
			ShowFullStructure: false,
		},
		Output: OutputConfig{
			Format:          "markdown",
			Filename:        "project_docs.md",
			AppendTimestamp: true,
		},
	}
}

// LoadOrDefault загружает конфигурацию из файла или возвращает дефолтную
func LoadOrDefault(path string) (*Config, bool, error) {
	// Проверяем существование файла
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Файл не существует - возвращаем дефолтную конфигурацию
		return DefaultConfig(), false, nil
	}

	// Файл существует - загружаем его
	cfg, err := Load(path)
	if err != nil {
		return nil, true, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, true, nil
}

// Load загружает конфигурацию из файла
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Сначала загружаем во временную структуру с NullBool
	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Затем преобразуем в финальную структуру, применяя дефолты
	cfg := rawToConfig(&raw)
	applyDefaults(cfg)

	return cfg, nil
}

// rawToConfig преобразует промежуточную структуру в основную, применяя дефолты для неустановленных значений
func rawToConfig(raw *rawConfig) *Config {
	defaults := DefaultConfig()
	cfg := &Config{}

	// Scanner конфигурация
	cfg.Scanner.MaxFileSize = raw.Scanner.MaxFileSize
	if raw.Scanner.MaxFileSize == 0 {
		cfg.Scanner.MaxFileSize = defaults.Scanner.MaxFileSize
	}

	// Для NullBool полей применяем значения только если они установлены
	if raw.Scanner.IncludeTests.Valid {
		cfg.Scanner.IncludeTests = raw.Scanner.IncludeTests.Bool
	} else {
		cfg.Scanner.IncludeTests = defaults.Scanner.IncludeTests
	}

	if raw.Scanner.IncludeConfigs.Valid {
		cfg.Scanner.IncludeConfigs = raw.Scanner.IncludeConfigs.Bool
	} else {
		cfg.Scanner.IncludeConfigs = defaults.Scanner.IncludeConfigs
	}

	if raw.Scanner.IncludeMarkup.Valid {
		cfg.Scanner.IncludeMarkup = raw.Scanner.IncludeMarkup.Bool
	} else {
		cfg.Scanner.IncludeMarkup = defaults.Scanner.IncludeMarkup
	}

	if raw.Scanner.IncludeStyles.Valid {
		cfg.Scanner.IncludeStyles = raw.Scanner.IncludeStyles.Bool
	} else {
		cfg.Scanner.IncludeStyles = defaults.Scanner.IncludeStyles
	}

	if raw.Scanner.IncludeDocs.Valid {
		cfg.Scanner.IncludeDocs = raw.Scanner.IncludeDocs.Bool
	} else {
		cfg.Scanner.IncludeDocs = defaults.Scanner.IncludeDocs
	}

	cfg.Scanner.ExcludedPatterns = raw.Scanner.ExcludedPatterns
	if len(cfg.Scanner.ExcludedPatterns) == 0 {
		cfg.Scanner.ExcludedPatterns = defaults.Scanner.ExcludedPatterns
	}

	cfg.Scanner.ParallelWorkers = raw.Scanner.ParallelWorkers
	if cfg.Scanner.ParallelWorkers == 0 {
		cfg.Scanner.ParallelWorkers = defaults.Scanner.ParallelWorkers
	}

	cfg.Scanner.Timeout = raw.Scanner.Timeout
	if cfg.Scanner.Timeout == 0 {
		cfg.Scanner.Timeout = defaults.Scanner.Timeout
	}

	cfg.Scanner.DocumentationMode = raw.Scanner.DocumentationMode
	if cfg.Scanner.DocumentationMode == "" {
		cfg.Scanner.DocumentationMode = defaults.Scanner.DocumentationMode
	}

	cfg.Scanner.SelectionMode = raw.Scanner.SelectionMode
	if cfg.Scanner.SelectionMode == "" {
		cfg.Scanner.SelectionMode = defaults.Scanner.SelectionMode
	}

	cfg.Scanner.IncludedPaths = raw.Scanner.IncludedPaths
	if cfg.Scanner.IncludedPaths == nil {
		cfg.Scanner.IncludedPaths = defaults.Scanner.IncludedPaths
	}

	cfg.Scanner.IncludePatterns = raw.Scanner.IncludePatterns
	if cfg.Scanner.IncludePatterns == nil {
		cfg.Scanner.IncludePatterns = defaults.Scanner.IncludePatterns
	}

	cfg.Scanner.ImportantFiles = raw.Scanner.ImportantFiles
	if cfg.Scanner.ImportantFiles == nil {
		cfg.Scanner.ImportantFiles = defaults.Scanner.ImportantFiles
	}

	cfg.Scanner.ShowFullStructure = raw.Scanner.ShowFullStructure

	// Output конфигурация
	cfg.Output.Format = raw.Output.Format
	if cfg.Output.Format == "" {
		cfg.Output.Format = defaults.Output.Format
	}

	cfg.Output.Filename = raw.Output.Filename
	if cfg.Output.Filename == "" {
		cfg.Output.Filename = defaults.Output.Filename
	}

	if raw.Output.AppendTimestamp != nil {
		cfg.Output.AppendTimestamp = *raw.Output.AppendTimestamp
	} else {
		cfg.Output.AppendTimestamp = defaults.Output.AppendTimestamp
	}

	return cfg
}

// applyDefaults применяет дефолтные значения для оставшихся незаполненных полей
func applyDefaults(cfg *Config) {
	defaults := DefaultConfig()

	// Убеждаемся, что все slice-поля не nil
	if cfg.Scanner.ExcludedPatterns == nil {
		cfg.Scanner.ExcludedPatterns = defaults.Scanner.ExcludedPatterns
	}
	if cfg.Scanner.IncludedPaths == nil {
		cfg.Scanner.IncludedPaths = defaults.Scanner.IncludedPaths
	}
	if cfg.Scanner.IncludePatterns == nil {
		cfg.Scanner.IncludePatterns = defaults.Scanner.IncludePatterns
	}
	if cfg.Scanner.ImportantFiles == nil {
		cfg.Scanner.ImportantFiles = defaults.Scanner.ImportantFiles
	}

	// Устанавливаем режим документации по умолчанию, если не задан
	if cfg.Scanner.DocumentationMode == "" {
		cfg.Scanner.DocumentationMode = "full"
	}
}

// Save сохраняет конфигурацию в файл
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateSampleConfig создает пример конфигурационного файла
func CreateSampleConfig(path string) error {
	sampleConfig := `# GOPS Configuration File
# Documentation generator for projects

scanner:
  # Maximum file size to process (in bytes)
  max_file_size: 2097152  # 2MB
  
  # Include test files in documentation
  include_tests: false
  
  # Include configuration files
  include_configs: true
  
  # Include markup files (HTML, etc.)
  include_markup: true
  
  # Include style files (CSS, SCSS, etc.)
  include_styles: true
  
  # Include documentation files (*.md)
  include_docs: false
  
  # Patterns to exclude from scanning
  excluded_patterns:
    - ".gitignore"
    - ".DS_Store"
    - "Thumbs.db"
    - "*.log"
    - "*.tmp"
    - "*.bak"
  
  # Number of parallel workers for scanning
  parallel_workers: 4
  
  # Timeout for scanning operation
  timeout: 5m
  
  # File selection mode: all, interactive, patterns, list
  selection_mode: all
  
  # Specific paths to include (when selection_mode is "list")
  # included_paths:
  #   - "src/core"
  #   - "src/features"
  
  # Patterns to include (when selection_mode is "patterns")
  # include_patterns:
  #   - "src/**/*.service.ts"
  #   - "src/**/*.component.ts"

output:
  # Output format (markdown, html)
  format: markdown
  
  # Base filename for output
  filename: project_docs.md
  
  # Append timestamp to filename
  append_timestamp: true
`

	return os.WriteFile(path, []byte(sampleConfig), 0644)
}

// ValidateSelectionMode проверяет корректность режима выбора
func ValidateSelectionMode(mode string) bool {
	switch mode {
	case SelectionModeAll, SelectionModeInteractive, SelectionModePatterns, SelectionModeList:
		return true
	default:
		return false
	}
}
