package nx

import (
	"bufio"
	"context"
	"fmt"
	"github.com/kolkov/gops/internal/scanner"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kolkov/gops/internal/docgen"
	"github.com/kolkov/gops/internal/filesystem"
	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
)

var _ scanner.ProjectScanner = (*NxScanner)(nil)

type NxScanner struct {
	rootDir    string
	outputFile string
	cfg        *model.ScanConfig
	logger     *logger.Logger
	projects   []*model.NxProject
}

func NewScanner(rootDir, outputFile string, cfg *model.ScanConfig, logger *logger.Logger) *NxScanner {
	return &NxScanner{
		rootDir:    rootDir,
		outputFile: outputFile,
		cfg:        cfg,
		logger:     logger,
	}
}

func (s *NxScanner) Scan(ctx context.Context, docGen docgen.Generator) error {
	if err := s.loadProjects(); err != nil {
		return fmt.Errorf("failed to load Nx projects: %w", err)
	}

	selectedProjects := s.selectProjects()
	if len(selectedProjects) == 0 {
		return fmt.Errorf("no projects selected for scanning")
	}

	// Запрос на включение важных файлов
	s.askForImportantFiles()

	rootFiles, err := s.getRootFiles()
	if err != nil {
		s.logger.Error("Failed to get root files", err)
		rootFiles = []string{}
	}

	meta := &model.ProjectMeta{
		Name:    filepath.Base(s.rootDir),
		Type:    "NX Monorepo",
		RootDir: s.rootDir,
	}
	docGen.WriteHeader(meta)
	docGen.WriteNxStructure(selectedProjects, rootFiles)

	for _, project := range selectedProjects {
		if err := s.scanProject(ctx, docGen, project); err != nil {
			s.logger.Error("Project scan failed", err)
			return fmt.Errorf("project %s: %w", project.Name, err)
		}
	}

	return nil
}

func (s *NxScanner) askForImportantFiles() {
	importantFiles := []string{
		"package.json",
		"nx.json",
		"project.json",
	}

	fmt.Println("\nВключить важные конфигурационные файлы?")
	fmt.Println("1. package.json (общая конфигурация проекта)")
	fmt.Println("2. nx.json (конфигурация Nx Monorepo)")
	fmt.Println("3. project.json (конфигурация приложения)")
	fmt.Println("0. Не включать (по умолчанию)")
	fmt.Print("Выберите файлы через запятую (например, 1,2,3): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" {
		return
	}

	choices := strings.Split(input, ",")
	for _, choice := range choices {
		idx, err := strconv.Atoi(strings.TrimSpace(choice))
		if err != nil || idx < 1 || idx > len(importantFiles) {
			continue
		}
		s.cfg.ImportantFiles = append(s.cfg.ImportantFiles, importantFiles[idx-1])
	}
}

func (s *NxScanner) getRootFiles() ([]string, error) {
	files, err := os.ReadDir(s.rootDir)
	if err != nil {
		return nil, err
	}

	var rootFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()

		if filesystem.ShouldSkipFile(name, s.cfg.OutputConfigFilename, s.cfg.OutputFilename, s.cfg.ExcludedPatterns, s.cfg.ImportantFiles) {
			continue
		}

		skip := false
		for _, pattern := range s.cfg.ExcludedPatterns {
			if matched, _ := filepath.Match(pattern, name); matched {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		rootFiles = append(rootFiles, name)
	}
	return rootFiles, nil
}

func (s *NxScanner) selectProjects() []*model.NxProject {
	fmt.Println("\nAvailable Nx projects:")
	for i, p := range s.projects {
		fmt.Printf("%2d. %s (%s)\n", i+1, p.Name, p.Type)
	}

	fmt.Println("\nSelect projects to scan:")
	fmt.Println("  all     - All projects")
	fmt.Println("  1,3,5   - Specific projects (comma-separated indices)")
	fmt.Print("\nYour choice [default: all]: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" {
		input = "all"
	}

	return s.filterProjects(input)
}

func (s *NxScanner) filterProjects(input string) []*model.NxProject {
	if strings.EqualFold(input, "all") {
		return s.projects
	}

	var selected []*model.NxProject
	indices := strings.Split(input, ",")

	for _, idxStr := range indices {
		idx, err := strconv.Atoi(strings.TrimSpace(idxStr))
		if err != nil || idx < 1 || idx > len(s.projects) {
			continue
		}
		selected = append(selected, s.projects[idx-1])
	}

	return selected
}

func (s *NxScanner) loadProjects() error {
	projectDirs := []string{"apps", "libs", "tools", "packages"}
	for _, dir := range projectDirs {
		fullPath := filepath.Join(s.rootDir, dir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				return nil
			}

			projectFile := filepath.Join(path, "project.json")
			if _, err := os.Stat(projectFile); err != nil {
				return nil
			}

			relPath, _ := filepath.Rel(s.rootDir, path)
			s.projects = append(s.projects, &model.NxProject{
				Name:      info.Name(),
				Type:      dir,
				Root:      relPath,
				SourceDir: path,
			})

			return filepath.SkipDir
		})

		if err != nil {
			return err
		}
	}
	return nil
}

func (s *NxScanner) scanProject(ctx context.Context, docGen docgen.Generator, project *model.NxProject) error {
	docGen.WriteProjectHeader(project.Name, project.Type, project.Root)

	treeBuilder := filesystem.NewTreeBuilder(project.SourceDir, s.cfg)
	tree, err := treeBuilder.Build()
	if err != nil {
		return err
	}
	docGen.WriteProjectTree(tree)

	docGen.WriteModulesHeader()
	return filesystem.ScanProject(project.SourceDir, s.cfg, s.logger, func(file *model.ProjectFile) {
		docGen.WriteFileSection(file)
	})
}
