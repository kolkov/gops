package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Константы для режимов выбора
const (
	SelectionModeAll         = "all"         // Включить все файлы (текущее поведение)
	SelectionModeInteractive = "interactive" // Интерактивный выбор
	SelectionModePatterns    = "patterns"    // По паттернам из конфига
	SelectionModeList        = "list"        // По списку путей из конфига
)

type Config struct {
	Scanner ScannerConfig `yaml:"scanner"`
	Output  OutputConfig  `yaml:"output"`
}

type ScannerConfig struct {
	MaxFileSize      int64    `yaml:"max_file_size"`
	IncludeTests     bool     `yaml:"include_tests"`
	IncludeConfigs   bool     `yaml:"include_configs"`
	IncludeMarkup    bool     `yaml:"include_markup"`
	IncludeStyles    bool     `yaml:"include_styles"`
	IncludeDocs      bool     `yaml:"include_docs"`
	ExcludedPatterns []string `yaml:"excluded_patterns"`
	ParallelWorkers  int      `yaml:"parallel_workers"`
	Timeout          Duration `yaml:"timeout"`

	// Поля для выборочного включения файлов
	SelectionMode   string   `yaml:"selection_mode"`   // Режим выбора файлов
	IncludedPaths   []string `yaml:"included_paths"`   // Конкретные пути для включения
	IncludePatterns []string `yaml:"include_patterns"` // Паттерны для включения
}

type OutputConfig struct {
	Format          string `yaml:"format"`
	Filename        string `yaml:"filename"`
	AppendTimestamp bool   `yaml:"append_timestamp"`
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
			ParallelWorkers: 4,
			Timeout:         Duration(5 * time.Minute),
			SelectionMode:   SelectionModeAll, // По умолчанию включаем все файлы
			IncludedPaths:   []string{},
			IncludePatterns: []string{},
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

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Применяем дефолтные значения для незаполненных полей
	applyDefaults(&cfg)

	return &cfg, nil
}

// applyDefaults применяет дефолтные значения для незаполненных полей
func applyDefaults(cfg *Config) {
	defaults := DefaultConfig()

	if cfg.Scanner.MaxFileSize == 0 {
		cfg.Scanner.MaxFileSize = defaults.Scanner.MaxFileSize
	}
	if cfg.Scanner.ParallelWorkers == 0 {
		cfg.Scanner.ParallelWorkers = defaults.Scanner.ParallelWorkers
	}
	if cfg.Scanner.Timeout == 0 {
		cfg.Scanner.Timeout = defaults.Scanner.Timeout
	}
	if cfg.Output.Format == "" {
		cfg.Output.Format = defaults.Output.Format
	}
	if cfg.Output.AppendTimestamp && cfg.Output.Filename == "" {
		cfg.Output.Filename = defaults.Output.Filename
	}
	if cfg.Scanner.SelectionMode == "" {
		cfg.Scanner.SelectionMode = defaults.Scanner.SelectionMode
	}

	// Добавляем разумные исключения по умолчанию если их нет
	if len(cfg.Scanner.ExcludedPatterns) == 0 {
		cfg.Scanner.ExcludedPatterns = defaults.Scanner.ExcludedPatterns
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
