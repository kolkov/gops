package markdown

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kolkov/gops/internal/model"
)

func (g *Generator) WriteNxStructure(projects []*model.NxProject, rootFiles []string) {
	g.file.WriteString("## Общая структура Nx Monorepo\n\n")
	g.file.WriteString(g.generateNxStructure(projects, rootFiles))
	g.file.WriteString("\n")
}

func (g *Generator) generateNxStructure(projects []*model.NxProject, rootFiles []string) string {
	var builder strings.Builder
	builder.WriteString("```\nnx-monorepo/\n")

	projectsByType := make(map[string][]*model.NxProject)
	for _, p := range projects {
		projectsByType[p.Type] = append(projectsByType[p.Type], p)
	}

	var types []string
	for t := range projectsByType {
		types = append(types, t)
	}
	sort.Strings(types)

	for i, t := range types {
		isLastType := i == len(types)-1 && len(rootFiles) == 0
		prefix := "├── "
		if isLastType {
			prefix = "└── "
		}
		builder.WriteString(fmt.Sprintf("%s%s/\n", prefix, t))

		projects := projectsByType[t]
		sort.Slice(projects, func(i, j int) bool {
			return projects[i].Name < projects[j].Name
		})

		for j, p := range projects {
			isLastProject := j == len(projects)-1
			projectPrefix := "│   "
			if isLastType {
				projectPrefix = "    "
			}
			if isLastProject {
				projectPrefix += "└── "
			} else {
				projectPrefix += "├── "
			}
			builder.WriteString(fmt.Sprintf("%s%s\n", projectPrefix, p.Name))
		}
	}

	if len(rootFiles) > 0 {
		if len(types) > 0 {
			builder.WriteString("│\n")
		}

		sort.Strings(rootFiles)
		for i, file := range rootFiles {
			isLast := i == len(rootFiles)-1
			prefix := "├── "
			if isLast {
				prefix = "└── "
			}
			builder.WriteString(fmt.Sprintf("%s%s\n", prefix, file))
		}
	}

	builder.WriteString("```\n")
	return builder.String()
}
