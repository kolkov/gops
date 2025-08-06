package js

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kolkov/gops/internal/docgen"
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSScanner_Scan(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(logger.InfoLevel)

	// Create JS project structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "index.js"), []byte("console.log('test')"), 0644))

	cfg := &model.ScanConfig{
		OutputFilename:       "output.md",
		OutputConfigFilename: "config.yaml",
	}

	scanner := NewScanner(tmpDir, "output.md", cfg, log)

	// Mock generator
	mockGen := new(docgen.MockGenerator)

	err := scanner.Scan(context.Background(), mockGen)
	assert.NoError(t, err)
}

func TestJSScanner_ImportantFiles(t *testing.T) {
	scanner := &JSScanner{
		cfg: &model.ScanConfig{},
	}

	// Simulate user input
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	tmpFile, _ := os.CreateTemp("", "stdin")
	defer os.Remove(tmpFile.Name())

	_, _ = tmpFile.WriteString("1,2\n")
	_, _ = tmpFile.Seek(0, 0)
	os.Stdin = tmpFile

	scanner.askForImportantFiles()

	assert.Contains(t, scanner.cfg.ImportantFiles, "package.json")
	assert.Contains(t, scanner.cfg.ImportantFiles, "tsconfig.json")
}
