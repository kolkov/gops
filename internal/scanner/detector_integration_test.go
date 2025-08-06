package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DetectorIntegrationSuite struct {
	suite.Suite
	testDir string
	logger  *logger.Logger
}

func (s *DetectorIntegrationSuite) SetupSuite() {
	s.logger = logger.New(logger.InfoLevel)
	s.testDir = s.T().TempDir()
}

func (s *DetectorIntegrationSuite) TestRealisticProjectDetection() {
	tests := []struct {
		name         string
		setupFunc    func() string
		expectedType string
		expectError  bool
	}{
		{
			name: "Complete Go Project",
			setupFunc: func() string {
				dir := filepath.Join(s.testDir, "go-project")
				require.NoError(s.T(), os.MkdirAll(dir, 0755))

				// Стандартная структура Go проекта
				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`
module github.com/example/project
go 1.24
require github.com/stretchr/testify v1.8.4
				`), 0644))

				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "main.go"), []byte(`
package main
func main() {}
				`), 0644))

				require.NoError(s.T(), os.MkdirAll(filepath.Join(dir, "internal", "service"), 0755))
				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "internal", "service", "service.go"), []byte(`
package service
func Process() {}
				`), 0644))

				return dir
			},
			expectedType: Go,
			expectError:  false,
		},
		{
			name: "NX Monorepo with Apps and Libs",
			setupFunc: func() string {
				dir := filepath.Join(s.testDir, "nx-workspace")
				require.NoError(s.T(), os.MkdirAll(dir, 0755))

				// NX специфичные файлы
				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "nx.json"), []byte(`{"npmScope": "test"}`), 0644))
				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "workspace.json"), []byte(`{"version": 2}`), 0644))

				// Структура приложений
				require.NoError(s.T(), os.MkdirAll(filepath.Join(dir, "apps", "web-app", "src"), 0755))
				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "apps", "web-app", "project.json"), []byte(`{"name": "web-app"}`), 0644))

				require.NoError(s.T(), os.MkdirAll(filepath.Join(dir, "libs", "ui", "src"), 0755))
				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "libs", "ui", "project.json"), []byte(`{"name": "ui"}`), 0645))

				return dir
			},
			expectedType: NxMonorepo,
			expectError:  false,
		},
		{
			name: "React Project with TS",
			setupFunc: func() string {
				dir := filepath.Join(s.testDir, "react-app")
				require.NoError(s.T(), os.MkdirAll(dir, 0755))

				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
					"name": "react-app",
					"dependencies": {"react": "^18.0.0", "react-dom": "^18.0.0"}
				}`), 0644))

				require.NoError(s.T(), os.MkdirAll(filepath.Join(dir, "src"), 0755))
				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "src", "App.tsx"), []byte(`
import React from 'react';
const App: React.FC = () => <div>Hello</div>;
export default App;
				`), 0644))

				return dir
			},
			expectedType: JS,
			expectError:  false,
		},
		{
			name: "Browser Extension",
			setupFunc: func() string {
				dir := filepath.Join(s.testDir, "extension")
				require.NoError(s.T(), os.MkdirAll(dir, 0755))

				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{
					"manifest_version": 3,
					"name": "Test Extension",
					"version": "1.0.0"
				}`), 0644))

				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "background.js"), []byte(`
chrome.runtime.onInstalled.addListener(() => {});
				`), 0644))

				require.NoError(s.T(), os.WriteFile(filepath.Join(dir, "popup.html"), []byte(`
<!DOCTYPE html>
<html><body><h1>Extension</h1></body></html>
				`), 0644))

				return dir
			},
			expectedType: BrowserExtension,
			expectError:  false,
		},
		{
			name: "Empty Directory",
			setupFunc: func() string {
				dir := filepath.Join(s.testDir, "empty")
				require.NoError(s.T(), os.MkdirAll(dir, 0755))
				return dir
			},
			expectedType: Unknown,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			projectDir := tt.setupFunc()

			detector := NewProjectDetector(projectDir, s.logger)
			projectType, err := detector.Detect()

			if tt.expectError {
				assert.Error(s.T(), err)
			} else {
				assert.NoError(s.T(), err)
				assert.Equal(s.T(), tt.expectedType, projectType)
			}
		})
	}
}

func (s *DetectorIntegrationSuite) TestConflictingProjectDetection() {
	// Проверка конфликтующих файлов
	testDir := filepath.Join(s.testDir, "conflict")
	require.NoError(s.T(), os.MkdirAll(testDir, 0755))

	require.NoError(s.T(), os.WriteFile(filepath.Join(testDir, "go.mod"), []byte("module test"), 0644))
	require.NoError(s.T(), os.WriteFile(filepath.Join(testDir, "package.json"), []byte(`{"name": "test"}`), 0644))

	detector := NewProjectDetector(testDir, s.logger)
	_, err := detector.Detect()

	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, ErrConflictingFiles)
}

func TestDetectorIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DetectorIntegrationSuite))
}
