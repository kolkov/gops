package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ConfigBuilder интерактивный построитель конфигурации
type ConfigBuilder struct {
	reader      *bufio.Reader
	projectType string
}

// NewConfigBuilder создает новый построитель конфигурации
func NewConfigBuilder(projectType string) *ConfigBuilder {
	return &ConfigBuilder{
		reader:      bufio.NewReader(os.Stdin),
		projectType: projectType,
	}
}

// Build интерактивно создает конфигурацию
func (cb *ConfigBuilder) Build() (*Config, error) {
	fmt.Println("\n🛠️  Configuration Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Detected project type: %s\n\n", cb.projectType)

	cfg := DefaultConfig()

	// Настройки сканера
	fmt.Println("📂 Scanner Settings")
	fmt.Println("─────────────────────")

	// Тесты - по умолчанию не включаем
	cfg.Scanner.IncludeTests = cb.askYesNo("Include test files?", false)

	// Конфиги - по умолчанию включаем
	cfg.Scanner.IncludeConfigs = cb.askYesNo("Include configuration files (json, yaml, etc)?", true)

	// В зависимости от типа проекта предлагаем разные опции
	switch cb.projectType {
	case "Go", "go":
		// Для Go проектов обычно не нужны стили и разметка
		cfg.Scanner.IncludeMarkup = cb.askYesNo("Include HTML files?", false)
		cfg.Scanner.IncludeStyles = cb.askYesNo("Include CSS/SCSS files?", false)
		cfg.Scanner.IncludeDocs = cb.askYesNo("Include markdown documentation (*.md)?", false)

	case "Angular", "React", "Vue", "JavaScript", "js", "NX", "nx":
		// Для фронтенд проектов обычно нужны стили и разметка
		cfg.Scanner.IncludeMarkup = cb.askYesNo("Include HTML/template files?", true)
		cfg.Scanner.IncludeStyles = cb.askYesNo("Include CSS/SCSS files?", true)
		cfg.Scanner.IncludeDocs = cb.askYesNo("Include markdown documentation (*.md)?", false)

	default:
		// Для неизвестных проектов спрашиваем всё
		cfg.Scanner.IncludeMarkup = cb.askYesNo("Include markup files (HTML, etc)?", true)
		cfg.Scanner.IncludeStyles = cb.askYesNo("Include style files (CSS, SCSS, etc)?", true)
		cfg.Scanner.IncludeDocs = cb.askYesNo("Include markdown documentation (*.md)?", false)
	}

	// Исключения
	fmt.Println("\n🚫 Exclusion Patterns")
	fmt.Println("─────────────────────")
	cb.configureExclusions(cfg)

	// Настройки вывода
	fmt.Println("\n📄 Output Settings")
	fmt.Println("──────────────────")
	cb.configureOutput(cfg)

	// Дополнительные настройки
	if cb.askYesNo("\nConfigure advanced settings?", false) {
		fmt.Println("\n⚙️  Advanced Settings")
		fmt.Println("────────────────────")
		cb.configureAdvanced(cfg)
	}

	return cfg, nil
}

func (cb *ConfigBuilder) configureExclusions(cfg *Config) {
	// Предлагаем стандартные исключения в зависимости от типа проекта
	var suggestions []string

	switch cb.projectType {
	case "Go", "go":
		suggestions = []string{
			"vendor/**",
			"*.exe",
			"*.dll",
			"*.so",
			"*.dylib",
		}
	case "Angular", "React", "Vue", "JavaScript", "js", "NX", "nx":
		suggestions = []string{
			"dist/**",
			"build/**",
			".angular/**",
			".nx/**",
			"coverage/**",
		}
	default:
		suggestions = []string{
			"*.log",
			"*.tmp",
			"*.bak",
		}
	}

	fmt.Println("Current exclusion patterns:")
	for i, pattern := range cfg.Scanner.ExcludedPatterns {
		fmt.Printf("  %d. %s\n", i+1, pattern)
	}

	if len(suggestions) > 0 {
		fmt.Println("\nSuggested additional patterns for your project type:")
		for i, pattern := range suggestions {
			fmt.Printf("  %d. %s\n", i+1, pattern)
		}

		if cb.askYesNo("Add suggested patterns?", true) {
			cfg.Scanner.ExcludedPatterns = append(cfg.Scanner.ExcludedPatterns, suggestions...)
		}
	}

	if cb.askYesNo("Add custom exclusion patterns?", false) {
		fmt.Println("Enter patterns one per line (empty line to finish):")
		for {
			fmt.Print("> ")
			pattern, _ := cb.reader.ReadString('\n')
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				break
			}
			cfg.Scanner.ExcludedPatterns = append(cfg.Scanner.ExcludedPatterns, pattern)
		}
	}
}

func (cb *ConfigBuilder) configureOutput(cfg *Config) {
	// Формат вывода
	fmt.Println("Output format:")
	fmt.Println("  1. Markdown (default)")
	fmt.Println("  2. HTML (coming soon)")

	choice := cb.askChoice("Select format", 1, 2, 1)
	if choice == 1 {
		cfg.Output.Format = "markdown"
	} else {
		cfg.Output.Format = "html"
	}

	// Имя файла
	fmt.Printf("Output filename (default: %s): ", cfg.Output.Filename)
	filename, _ := cb.reader.ReadString('\n')
	filename = strings.TrimSpace(filename)
	if filename != "" {
		cfg.Output.Filename = filename
	}

	// Добавление временной метки
	cfg.Output.AppendTimestamp = cb.askYesNo("Append timestamp to filename?", true)
}

func (cb *ConfigBuilder) configureAdvanced(cfg *Config) {
	// Максимальный размер файла
	fmt.Printf("Maximum file size in MB (default: 2): ")
	sizeStr, _ := cb.reader.ReadString('\n')
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
			cfg.Scanner.MaxFileSize = int64(size * 1024 * 1024)
		}
	}

	// Параллельные воркеры
	fmt.Printf("Number of parallel workers (default: 4): ")
	workersStr, _ := cb.reader.ReadString('\n')
	workersStr = strings.TrimSpace(workersStr)
	if workersStr != "" {
		if workers, err := strconv.Atoi(workersStr); err == nil && workers > 0 {
			cfg.Scanner.ParallelWorkers = workers
		}
	}
}

// AskYesNo экспортированный метод для использования в других пакетах
func (cb *ConfigBuilder) AskYesNo(question string, defaultYes bool) bool {
	return cb.askYesNo(question, defaultYes)
}

func (cb *ConfigBuilder) askYesNo(question string, defaultYes bool) bool {
	defaultStr := "[y/N]"
	if defaultYes {
		defaultStr = "[Y/n]"
	}

	for {
		fmt.Printf("%s %s: ", question, defaultStr)
		response, _ := cb.reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))

		if response == "" {
			return defaultYes
		}

		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please answer 'y' for yes or 'n' for no.")
		}
	}
}

func (cb *ConfigBuilder) askChoice(prompt string, min, max, defaultChoice int) int {
	for {
		fmt.Printf("%s [%d-%d, default: %d]: ", prompt, min, max, defaultChoice)
		response, _ := cb.reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "" {
			return defaultChoice
		}

		choice, err := strconv.Atoi(response)
		if err == nil && choice >= min && choice <= max {
			return choice
		}

		fmt.Printf("Please enter a number between %d and %d.\n", min, max)
	}
}
