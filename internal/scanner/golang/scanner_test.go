package golang

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

func TestGoScanner_Scan(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(logger.InfoLevel)

	// Create Go project structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644))

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

func TestGoScanner_ImportantFiles(t *testing.T) {
	scanner := &GoScanner{
		cfg: &model.ScanConfig{},
	}

	// Simulate user input
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	tmpFile, _ := os.CreateTemp("", "stdin")
	defer os.Remove(tmpFile.Name())

	_, _ = tmpFile.WriteString("1,3\n")
	_, _ = tmpFile.Seek(0, 0)
	os.Stdin = tmpFile

	scanner.askForImportantFiles()

	assert.Contains(t, scanner.cfg.ImportantFiles, "Makefile")
	assert.Contains(t, scanner.cfg.ImportantFiles, "docker-compose.yml")
}
