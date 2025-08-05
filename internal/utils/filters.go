package utils

import (
	"path/filepath"
	"strconv"
	"strings"
)

func ShouldSkipDir(name string) bool {
	skipDirs := []string{
		"vendor", "node_modules", ".git", ".vscode", ".idea", "dist",
		"build", "__pycache__", "bin", "obj", "testdata", "coverage",
		".angular", ".nx", "target", "out", "__tests__", "__snapshots__",
		".next", ".nuxt", ".cache", "cypress", "e2e", "coverage",
	}
	for _, dir := range skipDirs {
		if strings.EqualFold(name, dir) {
			return true
		}
	}
	return false
}

func ShouldIncludeFile(name string) bool {
	fileExt := filepath.Ext(name)
	includeExtensions := []string{
		".go", ".ts", ".js", ".jsx", ".tsx", ".html", ".scss", ".css",
		".json", ".yaml", ".yml", ".md", ".mod", ".sum", ".mjs", ".cjs",
	}

	for _, ext := range includeExtensions {
		if strings.EqualFold(fileExt, ext) {
			return true
		}
	}

	includeFiles := []string{
		"go.mod", "go.sum", "angular.json", "nx.json",
		"package.json", "tsconfig.json", "project.json",
	}

	for _, file := range includeFiles {
		if strings.EqualFold(name, file) {
			return true
		}
	}

	return false
}

func IsGeneratedFile(filename, currentOutput string) bool {
	if filename == filepath.Base(currentOutput) {
		return true
	}
	if strings.HasPrefix(filename, "project_documentation") &&
		strings.HasSuffix(filename, ".md") {
		return true
	}
	if filename == "project_structure.txt" {
		return true
	}
	return false
}

func GetFileLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".jsx":
		return "jsx"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".mod", ".sum":
		return "mod"
	case ".html", ".htm":
		return "html"
	case ".scss", ".sass":
		return "scss"
	case ".css":
		return "css"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	default:
		return "go"
	}
}

func FilterProjects(projects []NxProject, input string) []NxProject {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	if strings.EqualFold(input, "all") {
		return projects
	}

	if strings.Contains(input, ":") {
		parts := strings.Split(input, ":")
		if len(parts) == 2 {
			filter := parts[0]
			return filterProjectsByType(projects, filter)
		}
	}

	return selectSpecificProjects(projects, input)
}

func filterProjectsByType(projects []NxProject, filter string) []NxProject {
	var filtered []NxProject
	for _, p := range projects {
		if p.Type == filter {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func selectSpecificProjects(projects []NxProject, input string) []NxProject {
	var selected []NxProject
	selections := strings.Split(input, ",")

	for _, s := range selections {
		s = strings.TrimSpace(s)
		if idx, err := strconv.Atoi(s); err == nil && idx > 0 && idx <= len(projects) {
			selected = append(selected, projects[idx-1])
		}
	}

	return selected
}
