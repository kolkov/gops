package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
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
	ExcludedPatterns []string `yaml:"excluded_patterns"`
	ParallelWorkers  int      `yaml:"parallel_workers"`
	Timeout          Duration `yaml:"timeout"`
}

type OutputConfig struct {
	Format          string `yaml:"format"`
	Filename        string `yaml:"filename"`
	AppendTimestamp bool   `yaml:"append_timestamp"` // Новая опция
}

type Duration time.Duration

// Исправленная реализация UnmarshalYAML
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

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Scanner.MaxFileSize == 0 {
		cfg.Scanner.MaxFileSize = 2 * 1024 * 1024 // 2MB
	}
	if cfg.Scanner.ParallelWorkers == 0 {
		cfg.Scanner.ParallelWorkers = 4
	}
	if cfg.Scanner.Timeout == 0 {
		cfg.Scanner.Timeout = Duration(5 * time.Minute)
	}
	if cfg.Output.Format == "" {
		cfg.Output.Format = "markdown"
	}
	if cfg.Output.AppendTimestamp && cfg.Output.Filename == "" {
		cfg.Output.Filename = "project_docs.md"
	}

	// Добавляем разумные исключения по умолчанию
	if len(cfg.Scanner.ExcludedPatterns) == 0 {
		cfg.Scanner.ExcludedPatterns = []string{
			".gitignore",
			".DS_Store",
			"Thumbs.db",
			"*.log",
			"*.tmp",
			"*.bak",
		}
	}

	return &cfg, nil
}
