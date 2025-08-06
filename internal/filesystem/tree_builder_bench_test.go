package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kolkov/gops/internal/model"
	"github.com/stretchr/testify/require"
)

func BenchmarkTreeBuilder(b *testing.B) {
	testDir := b.TempDir()

	// Создаем большую структуру для бенчмарка
	createLargeStructure(b, testDir, 1000)

	cfg := &model.ScanConfig{}
	builder := NewTreeBuilder(testDir, cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.Build()
		require.NoError(b, err)
	}
}

func BenchmarkTreeBuilderWithExclusions(b *testing.B) {
	testDir := b.TempDir()

	// Создаем структуру с исключениями
	createLargeStructure(b, testDir, 500)

	cfg := &model.ScanConfig{
		ExcludedPatterns: []string{"*.tmp", "node_modules", "__pycache__"},
	}
	builder := NewTreeBuilder(testDir, cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.Build()
		require.NoError(b, err)
	}
}

func createLargeStructure(b *testing.B, baseDir string, fileCount int) {
	for i := 0; i < fileCount; i++ {
		dir := filepath.Join(baseDir, "dir", string(rune('a'+i%26)), string(rune('a'+i%10)))
		require.NoError(b, os.MkdirAll(dir, 0755))

		file := filepath.Join(dir, "file.go")
		require.NoError(b, os.WriteFile(file, []byte("package main\n\nfunc main() {}"), 0644))
	}
}
