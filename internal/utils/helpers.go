package utils

import (
	"os"
	"path/filepath"
)

type NxProject struct {
	Name      string
	Type      string
	Root      string
	SourceDir string
}

// Добавляем проверку на корневой project.json
func IsNxMonorepo(rootDir string) bool {
	nxFiles := []string{"nx.json", "workspace.json"}
	for _, file := range nxFiles {
		if _, err := os.Stat(filepath.Join(rootDir, file)); err == nil {
			return true
		}
	}
	return false
}

func ParseNxProjects(rootDir string) []NxProject {
	projects := []NxProject{}

	scanDir := func(dir, ptype string) {
		dirPath := filepath.Join(rootDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return
		}

		filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}

			projectConfig := filepath.Join(path, "project.json")
			if _, err := os.Stat(projectConfig); err == nil {
				relPath, _ := filepath.Rel(rootDir, path)
				projectName := filepath.Base(path)

				sourceDir := filepath.Join(path, "src")
				if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
					sourceDir = path
				}

				projects = append(projects, NxProject{
					Name:      projectName,
					Type:      ptype,
					Root:      relPath,
					SourceDir: sourceDir,
				})
				return filepath.SkipDir
			}
			return nil
		})
	}

	scanDir("apps", "app")
	scanDir("libs", "lib")
	scanDir("tools", "tool")

	return projects
}

func ScanProjectFiles(
	sourceDir,
	rootDir,
	outputFile string,
	processFile func(string, string, []byte),
	includeStyles,
	includeMarkup,
	includeConfigs,
	includeTests bool,
) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if ShouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !ShouldIncludeFile(info.Name()) || IsGeneratedFile(info.Name(), outputFile) {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lang := GetFileLanguage(path)
		processFile(relPath, lang, content)
		return nil
	})
}

func ScanStandardProject(
	rootDir,
	outputFile string,
	processFile func(string, string, []byte),
	includeStyles,
	includeMarkup,
	includeConfigs,
	includeTests bool,
) error {
	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if ShouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !ShouldIncludeFile(info.Name()) || IsGeneratedFile(info.Name(), outputFile) {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lang := GetFileLanguage(path)
		processFile(relPath, lang, content)
		return nil
	})
}
