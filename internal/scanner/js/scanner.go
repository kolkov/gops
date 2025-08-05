package js

import (
	"context"
	"github.com/kolkov/gops/internal/scanner"
	"path/filepath"
	"strings"

	"github.com/kolkov/gops/internal/docgen"
	"github.com/kolkov/gops/internal/filesystem"
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
)

var _ scanner.ProjectScanner = (*JSScanner)(nil)

type JSScanner struct {
	rootDir    string
	outputFile string
	cfg        *model.ScanConfig
	logger     *logger.Logger
}

func NewScanner(rootDir, outputFile string, cfg *model.ScanConfig, logger *logger.Logger) *JSScanner {
	return &JSScanner{
		rootDir:    rootDir,
		outputFile: outputFile,
		cfg:        cfg,
		logger:     logger,
	}
}

func (s *JSScanner) Scan(ctx context.Context, docGen docgen.Generator) error {
	projectType := scanner.DetectJSFramework(s.rootDir)
	meta := &model.ProjectMeta{
		Name:    filepath.Base(s.rootDir),
		Type:    projectType + " проект",
		RootDir: s.rootDir,
	}

	docGen.WriteHeader(meta)

	treeBuilder := filesystem.NewTreeBuilder(s.rootDir, s.cfg)
	tree, err := treeBuilder.Build()
	if err != nil {
		s.logger.Error("Failed to build project tree", err)
		return err
	}
	docGen.WriteTree(tree)

	docGen.WriteModulesHeader()
	return filesystem.ScanProject(s.rootDir, s.cfg, s.logger, func(file *model.ProjectFile) {
		if s.shouldSkip(file.Path) {
			file.Skipped = true
		}
		docGen.WriteFileSection(file)
	})
}

func (s *JSScanner) shouldSkip(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	fileName := strings.ToLower(filepath.Base(path))

	switch {
	case !s.cfg.IncludeStyles && (ext == ".css" || ext == ".scss" || ext == ".sass" || ext == ".less"):
		return true
	case !s.cfg.IncludeMarkup && (ext == ".html" || ext == ".htm"):
		return true
	case !s.cfg.IncludeConfigs && (ext == ".json" || ext == ".yaml" || ext == ".yml" || strings.Contains(fileName, "config")):
		return true
	case !s.cfg.IncludeTests && (strings.Contains(fileName, ".spec.") || strings.Contains(fileName, ".test.")):
		return true
	default:
		return false
	}
}
