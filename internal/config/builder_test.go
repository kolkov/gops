package config

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigBuilder_Build(t *testing.T) {
	// Тест сложно полностью автоматизировать из-за интерактивного ввода
	// Но мы можем проверить базовую функциональность
	t.Run("DefaultsForGoProject", func(t *testing.T) {
		cb := NewConfigBuilder("Go")
		assert.NotNil(t, cb)
		assert.Equal(t, "Go", cb.projectType)
	})

	t.Run("DefaultsForJSProject", func(t *testing.T) {
		cb := NewConfigBuilder("JavaScript")
		assert.NotNil(t, cb)
		assert.Equal(t, "JavaScript", cb.projectType)
	})
}

func TestConfigBuilder_askYesNo(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		question   string
		defaultYes bool
		expected   bool
	}{
		{
			name:       "YesResponse",
			input:      "y\n",
			question:   "Test?",
			defaultYes: false,
			expected:   true,
		},
		{
			name:       "NoResponse",
			input:      "n\n",
			question:   "Test?",
			defaultYes: true,
			expected:   false,
		},
		{
			name:       "EmptyWithDefaultYes",
			input:      "\n",
			question:   "Test?",
			defaultYes: true,
			expected:   true,
		},
		{
			name:       "EmptyWithDefaultNo",
			input:      "\n",
			question:   "Test?",
			defaultYes: false,
			expected:   false,
		},
		{
			name:       "FullYes",
			input:      "yes\n",
			question:   "Test?",
			defaultYes: false,
			expected:   true,
		},
		{
			name:       "FullNo",
			input:      "no\n",
			question:   "Test?",
			defaultYes: true,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &ConfigBuilder{
				reader: bufio.NewReader(strings.NewReader(tt.input)),
			}
			
			result := cb.askYesNo(tt.question, tt.defaultYes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigBuilder_askChoice(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		prompt        string
		min           int
		max           int
		defaultChoice int
		expected      int
	}{
		{
			name:          "ValidChoice",
			input:         "2\n",
			prompt:        "Choose",
			min:           1,
			max:           3,
			defaultChoice: 1,
			expected:      2,
		},
		{
			name:          "DefaultChoice",
			input:         "\n",
			prompt:        "Choose",
			min:           1,
			max:           5,
			defaultChoice: 3,
			expected:      3,
		},
		{
			name:          "MinChoice",
			input:         "1\n",
			prompt:        "Choose",
			min:           1,
			max:           10,
			defaultChoice: 5,
			expected:      1,
		},
		{
			name:          "MaxChoice",
			input:         "10\n",
			prompt:        "Choose",
			min:           1,
			max:           10,
			defaultChoice: 5,
			expected:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &ConfigBuilder{
				reader: bufio.NewReader(strings.NewReader(tt.input)),
			}
			
			result := cb.askChoice(tt.prompt, tt.min, tt.max, tt.defaultChoice)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigBuilder_configureExclusions_GoProject(t *testing.T) {
	// Симулируем ответ "yes" на добавление предложенных паттернов
	// и "no" на добавление custom паттернов
	input := "y\nn\n"
	cb := &ConfigBuilder{
		reader:      bufio.NewReader(strings.NewReader(input)),
		projectType: "Go",
	}
	
	cfg := DefaultConfig()
	initialPatterns := len(cfg.Scanner.ExcludedPatterns)
	
	cb.configureExclusions(cfg)
	
	// Должны добавиться Go-specific паттерны
	assert.Greater(t, len(cfg.Scanner.ExcludedPatterns), initialPatterns)
	assert.Contains(t, cfg.Scanner.ExcludedPatterns, "vendor/**")
	assert.Contains(t, cfg.Scanner.ExcludedPatterns, "*.exe")
}

func TestConfigBuilder_configureExclusions_JSProject(t *testing.T) {
	// Симулируем ответ "yes" на добавление предложенных паттернов
	input := "y\nn\n"
	cb := &ConfigBuilder{
		reader:      bufio.NewReader(strings.NewReader(input)),
		projectType: "JavaScript",
	}
	
	cfg := DefaultConfig()
	initialPatterns := len(cfg.Scanner.ExcludedPatterns)
	
	cb.configureExclusions(cfg)
	
	// Должны добавиться JS-specific паттерны
	assert.Greater(t, len(cfg.Scanner.ExcludedPatterns), initialPatterns)
	assert.Contains(t, cfg.Scanner.ExcludedPatterns, "dist/**")
	assert.Contains(t, cfg.Scanner.ExcludedPatterns, "build/**")
}

func TestConfigBuilder_configureOutput(t *testing.T) {
	t.Run("MarkdownFormat", func(t *testing.T) {
		// Выбираем markdown (1), пустое имя файла (используем дефолт), timestamp = yes
		input := "1\n\ny\n"
		cb := &ConfigBuilder{
			reader: bufio.NewReader(strings.NewReader(input)),
		}
		
		cfg := DefaultConfig()
		cb.configureOutput(cfg)
		
		assert.Equal(t, "markdown", cfg.Output.Format)
		assert.Equal(t, "project_docs.md", cfg.Output.Filename)
		assert.True(t, cfg.Output.AppendTimestamp)
	})
	
	t.Run("CustomFilename", func(t *testing.T) {
		// Выбираем markdown, custom filename, no timestamp
		input := "1\ncustom_docs.md\nn\n"
		cb := &ConfigBuilder{
			reader: bufio.NewReader(strings.NewReader(input)),
		}
		
		cfg := DefaultConfig()
		cb.configureOutput(cfg)
		
		assert.Equal(t, "markdown", cfg.Output.Format)
		assert.Equal(t, "custom_docs.md", cfg.Output.Filename)
		assert.False(t, cfg.Output.AppendTimestamp)
	})
}

func TestConfigBuilder_configureAdvanced(t *testing.T) {
	t.Run("CustomSettings", func(t *testing.T) {
		// 5MB файлы, 8 воркеров
		input := "5\n8\n"
		cb := &ConfigBuilder{
			reader: bufio.NewReader(strings.NewReader(input)),
		}
		
		cfg := DefaultConfig()
		cb.configureAdvanced(cfg)
		
		assert.Equal(t, int64(5*1024*1024), cfg.Scanner.MaxFileSize)
		assert.Equal(t, 8, cfg.Scanner.ParallelWorkers)
	})
	
	t.Run("DefaultSettings", func(t *testing.T) {
		// Пустой ввод - используем дефолты
		input := "\n\n"
		cb := &ConfigBuilder{
			reader: bufio.NewReader(strings.NewReader(input)),
		}
		
		cfg := DefaultConfig()
		originalSize := cfg.Scanner.MaxFileSize
		originalWorkers := cfg.Scanner.ParallelWorkers
		
		cb.configureAdvanced(cfg)
		
		assert.Equal(t, originalSize, cfg.Scanner.MaxFileSize)
		assert.Equal(t, originalWorkers, cfg.Scanner.ParallelWorkers)
	})
}