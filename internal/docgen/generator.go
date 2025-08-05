package docgen

import (
	"github.com/kolkov/gops/internal/docgen/markdown"
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
)

type Generator interface {
	WriteHeader(meta *model.ProjectMeta)
	WriteTree(structure string)
	WriteFileSection(file *model.ProjectFile)
	WriteNxStructure(projects []*model.NxProject, rootFiles []string)
	WriteProjectHeader(name, ptype, root string)
	WriteProjectTree(structure string)
	WriteModulesHeader()
	Close() error
}

func NewMarkdownGenerator(outputFile string, logger *logger.Logger) Generator {
	return markdown.NewGenerator(outputFile, logger)
}
