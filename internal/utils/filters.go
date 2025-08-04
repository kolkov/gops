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
		".angular", ".nx",
	}
	for _, dir := range skipDirs {
		if name == dir {
			return true
		}
	}
	return false
}

func ShouldIncludeFile(name string) bool {
	fileExt := filepath.Ext(name) // Исправлено: переменная переименована
	includeExtensions := []string{
		".go", ".ts", ".html", ".scss", ".css",
		".json", ".yaml", ".yml", ".md",
	}

	for _, ext := range includeExtensions {
		if strings.EqualFold(fileExt, ext) {
			return true
		}
	}

	includeFiles := []string{
		"go.mod", "go.sum", "angular.json",
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
	switch filepath.Ext(path) {
	case ".mod", ".sum":
		return "mod"
	case ".ts", ".tsx":
		return "typescript"
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
