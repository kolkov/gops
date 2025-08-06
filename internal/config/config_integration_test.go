package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigIntegrationSuite struct {
	suite.Suite
	testDir string
}

func (s *ConfigIntegrationSuite) SetupSuite() {
	s.testDir = s.T().TempDir()
}

func (s *ConfigIntegrationSuite) TestComplexConfigScenarios() {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		validate    func(*Config)
	}{
		{
			name: "Zero values with defaults",
			configYAML: `scanner:
  include_tests: true
output:
  format: markdown`,
			expectError: false,
			validate: func(cfg *Config) {
				assert.Equal(s.T(), int64(2*1024*1024), cfg.Scanner.MaxFileSize)
				assert.Equal(s.T(), 4, cfg.Scanner.ParallelWorkers)
			},
		},
		{
			name: "All values specified",
			configYAML: `scanner:
  max_file_size: 5242880
  include_tests: false
  include_configs: true
  include_markup: true
  include_styles: false
  excluded_patterns:
    - "*.tmp"
    - "__cache__/*"
  parallel_workers: 8
  timeout: 15m
output:
  format: html
  filename: custom.md
  append_timestamp: true`,
			expectError: false,
			validate: func(cfg *Config) {
				assert.Equal(s.T(), int64(5242880), cfg.Scanner.MaxFileSize)
				assert.Equal(s.T(), 8, cfg.Scanner.ParallelWorkers)
			},
		},
		{
			name: "Invalid duration format",
			configYAML: `scanner:
  timeout: invalid_duration`,
			expectError: true,
			validate:    nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			safeName := strings.ReplaceAll(tt.name, " ", "_")
			configPath := filepath.Join(s.testDir, safeName+".yaml")

			require.NoError(s.T(), os.WriteFile(configPath, []byte(tt.configYAML), 0644))

			cfg, err := Load(configPath)

			if tt.expectError {
				assert.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				if tt.validate != nil {
					tt.validate(cfg)
				}
			}
		})
	}
}

func TestConfigIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ConfigIntegrationSuite))
}
