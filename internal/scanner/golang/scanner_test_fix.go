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

func TestGoScanner_Scan_NoInput(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(logger.InfoLevel)

	// Redirect stdin to avoid interactive input
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	os.Stdin = nil // Will cause EOF, simulating no input

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644))

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
