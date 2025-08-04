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
	}
	for _, dir := range skipDirs {
		if name == dir {
			return true
		}
	}
	return false
}

func ShouldIncludeFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".go" || name == "go.mod" || name == "go.sum" ||
		ext == ".ts" || ext == ".html" || ext == ".scss" || ext == ".css" ||
		ext == ".json" || strings.HasSuffix(name, "angular.json") ||
		strings.HasSuffix(name, "package.json") || strings.HasSuffix(name, "tsconfig.json")
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
	switch {
	case strings.HasSuffix(path, ".mod"):
		return "mod"
	case strings.HasSuffix(path, ".sum"):
		return "mod"
	case strings.HasSuffix(path, ".ts"):
		return "typescript"
	case strings.HasSuffix(path, ".html"):
		return "html"
	case strings.HasSuffix(path, ".scss"):
		return "scss"
	case strings.HasSuffix(path, ".css"):
		return "css"
	case strings.HasSuffix(path, ".json"):
		return "json"
	default:
		return "go"
	}
}

func FilterProjects(projects []NxProject, input string) []NxProject {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// Специальные случаи
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

	// Выбор конкретных проектов
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
