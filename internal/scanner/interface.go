package scanner

import (
	"context"

	"github.com/kolkov/gops/internal/docgen"
	"github.com/kolkov/gops/pkg/logger"
)

// ProjectScanner определяет интерфейс для сканирования проектов
type ProjectScanner interface {
	Scan(ctx context.Context, docGen docgen.Generator) error
}

// ProjectDetector определяет интерфейс для определения типа проекта
type ProjectDetector interface {
	Detect() (string, error)
}

// NewProjectDetector создает детектор проекта
func NewProjectDetector(rootDir string, logger *logger.Logger) ProjectDetector {
	return &projectDetector{
		rootDir: rootDir,
		logger:  logger,
	}
}
