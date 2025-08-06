package filesystem

import (
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"os"
	"path/filepath"
)

type SecurityTestSuite struct {
	suite.Suite
	testDir string
	logger  *logger.Logger
}

func (s *SecurityTestSuite) SetupSuite() {
	s.logger = logger.New(logger.InfoLevel)
	s.testDir = s.T().TempDir()
}

func (s *SecurityTestSuite) TestSystemFilesExclusion() {
	// Создаем только 2 нормальных файла (точно 2)
	normalFiles := []string{
		"main.go",
		"app.js",
	}

	for _, file := range normalFiles {
		fullPath := filepath.Join(s.testDir, file)
		require.NoError(s.T(), os.WriteFile(fullPath, []byte("content"), 0644))
	}

	// Создаем системные файлы, которые должны быть исключены
	systemItems := []string{
		"package-lock.json",
		"yarn.lock",
		"go.sum",
		".git/config",
		"node_modules/module/index.js",
		"__pycache__/cache.py",
		"dist/app.js",
		"build/output.js",
	}

	for _, item := range systemItems {
		fullPath := filepath.Join(s.testDir, item)
		require.NoError(s.T(), os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(s.T(), os.WriteFile(fullPath, []byte("content"), 0644))
	}

	// Настраиваем конфиг для включения всех типов файлов
	cfg := &model.ScanConfig{
		MaxFileSize:    1024,
		IncludeConfigs: true, // Включаем JSON конфиги
		IncludeMarkup:  true, // Включаем README
		IncludeStyles:  true, // Включаем CSS
		IncludeTests:   true, // Включаем тесты
	}

	processedFiles := 0
	processedPaths := []string{}

	err := ScanProject(s.testDir, cfg, s.logger, func(file *model.ProjectFile) {
		if !file.Skipped {
			processedFiles++
			processedPaths = append(processedPaths, file.Path)
		}
	})

	assert.NoError(s.T(), err)

	// Проверяем точное количество
	assert.Equal(s.T(), 2, processedFiles, "Should process exactly 2 files")

	// Проверяем конкретные файлы
	assert.Contains(s.T(), processedPaths, "main.go")
	assert.Contains(s.T(), processedPaths, "app.js")

	// Проверяем, что NO системные файлы обработаны
	for _, systemFile := range systemItems {
		assert.NotContains(s.T(), processedPaths, systemFile)
	}
}

func (s *SecurityTestSuite) TestLargeFileProtection() {
	largeFile := filepath.Join(s.testDir, "large.bin")
	largeContent := make([]byte, 10*1024*1024) // 10MB
	require.NoError(s.T(), os.WriteFile(largeFile, largeContent, 0644))

	cfg := &model.ScanConfig{
		MaxFileSize: 1024, // 1KB
	}

	skipped := false
	err := ScanProject(s.testDir, cfg, s.logger, func(file *model.ProjectFile) {
		if file.Path == "large.bin" {
			skipped = file.Skipped
		}
	})

	assert.NoError(s.T(), err)
	assert.True(s.T(), skipped, "Large files should be skipped")
}

func (s *SecurityTestSuite) TestSimpleExclusion() {
	// Упрощенный тест, который проверяет основную логику исключений
	normalFile := filepath.Join(s.testDir, "normal.txt")
	require.NoError(s.T(), os.WriteFile(normalFile, []byte("content"), 0644))

	systemFile := filepath.Join(s.testDir, "package-lock.json")
	require.NoError(s.T(), os.WriteFile(systemFile, []byte("content"), 0644))

	cfg := &model.ScanConfig{
		MaxFileSize: 1024,
	}

	processedFiles := 0
	err := ScanProject(s.testDir, cfg, s.logger, func(file *model.ProjectFile) {
		if !file.Skipped {
			processedFiles++
			assert.Equal(s.T(), "normal.txt", file.Path)
		}
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, processedFiles, "Should process only normal.txt")
}
