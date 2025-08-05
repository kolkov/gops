package golang

import (
	"context"
	"github.com/kolkov/gops/internal/scanner"
	"path/filepath"

	"github.com/kolkov/gops/internal/docgen"
	"github.com/kolkov/gops/internal/filesystem"
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
)

var _ scanner.ProjectScanner = (*GoScanner)(nil)

type GoScanner struct {
	rootDir    string
	outputFile string
	cfg        *model.ScanConfig
	logger     *logger.Logger
}

func NewScanner(rootDir, outputFile string, cfg *model.ScanConfig, logger *logger.Logger) *GoScanner {
	return &GoScanner{
		rootDir:    rootDir,
		outputFile: outputFile,
		cfg:        cfg,
		logger:     logger,
	}
}

func (s *GoScanner) Scan(ctx context.Context, docGen docgen.Generator) error {
	meta := &model.ProjectMeta{
		Name:    filepath.Base(s.rootDir),
		Type:    "Go проект",
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
		docGen.WriteFileSection(file)
	})
}
