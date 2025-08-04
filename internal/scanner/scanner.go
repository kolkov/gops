package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"project_scanner/internal/markdown"
	"project_scanner/internal/utils"
)

type ProjectScanner struct {
	rootDir        string
	outputFile     string
	nxMonorepo     bool
	nxProjects     []utils.NxProject
	IncludeStyles  bool
	IncludeMarkup  bool
	IncludeConfigs bool
	IncludeTests   bool
}

func NewProjectScanner(rootDir, outputFile string) *ProjectScanner {
	return &ProjectScanner{
		rootDir:    rootDir,
		outputFile: outputFile,
	}
}

func (s *ProjectScanner) AskContentSettings() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nНастройки документации для ВСЕХ проектов:")
	fmt.Print("1. Включать файлы стилей (CSS, SCSS)? [y/N]: ")
	s.IncludeStyles = readYesNo(reader, false)

	fmt.Print("2. Включать файлы разметки (HTML)? [y/N]: ")
	s.IncludeMarkup = readYesNo(reader, false)

	fmt.Print("3. Включать конфигурационные файлы? [y/N]: ")
	s.IncludeConfigs = readYesNo(reader, false)

	fmt.Print("4. Включать тестовые файлы? [y/N]: ")
	s.IncludeTests = readYesNo(reader, false)
}

func readYesNo(reader *bufio.Reader, defaultVal bool) bool {
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "y" || text == "yes" {
		return true
	} else if text == "n" || text == "no" {
		return false
	}
	return defaultVal
}

func (s *ProjectScanner) Scan(docGenerator *markdown.DocumentationGenerator) error {
	if err := s.initializeScanner(); err != nil {
		return err
	}

	s.detectProjectType()

	projectName := filepath.Base(s.rootDir)
	docGenerator.WriteHeader(projectName, time.Now(), s.nxMonorepo)

	if s.nxMonorepo {
		return s.scanNxMonorepo(docGenerator)
	}
	return s.scanStandardProject(docGenerator)
}

func (s *ProjectScanner) detectProjectType() {
	if utils.IsNxMonorepo(s.rootDir) {
		s.nxMonorepo = true
		s.nxProjects = utils.ParseNxProjects(s.rootDir)
	}
}

func (s *ProjectScanner) scanNxMonorepo(docGenerator *markdown.DocumentationGenerator) error {
	docGenerator.WriteNxStructure(s.nxProjects)

	selectedProjects := s.selectProjects()
	if len(selectedProjects) == 0 {
		return fmt.Errorf("не выбрано ни одного проекта")
	}

	for _, project := range selectedProjects {
		if err := s.processNxProject(docGenerator, project); err != nil {
			fmt.Printf("Ошибка обработки проекта %s: %v\n", project.Name, err)
		}
	}

	if !s.IncludeStyles || !s.IncludeMarkup || !s.IncludeConfigs || !s.IncludeTests {
		docGenerator.WriteSkippedFilesNote()
	}

	return nil
}

func (s *ProjectScanner) scanStandardProject(docGenerator *markdown.DocumentationGenerator) error {
	docGenerator.WriteStandardProjectTree(s.rootDir)
	docGenerator.WriteModulesHeader()

	return utils.ScanStandardProject(
		s.rootDir,
		s.outputFile,
		func(filePath, lang string, content []byte) {
			skip := s.shouldSkipFileContent(filePath)
			docGenerator.WriteFileSection(filePath, content, lang, skip)
		},
		s.IncludeStyles,
		s.IncludeMarkup,
		s.IncludeConfigs,
		s.IncludeTests,
	)
}

func (s *ProjectScanner) selectProjects() []utils.NxProject {
	fmt.Println("\nНайдены проекты в Nx Monorepo:")
	for i, p := range s.nxProjects {
		fmt.Printf("%2d. [%s] %s (%s)\n", i+1, p.Type, p.Name, p.Root)
	}

	fmt.Println("\nВыберите проекты для документации:")
	fmt.Println("  all     - Все проекты")
	fmt.Println("  app:*   - Все приложения")
	fmt.Println("  lib:*   - Все библиотеки")
	fmt.Println("  1,3,5   - Конкретные проекты (через запятую)")
	fmt.Print("\nВаш выбор [по умолчанию app:*]: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" {
		input = "app:*"
	}

	return utils.FilterProjects(s.nxProjects, input)
}

func (s *ProjectScanner) processNxProject(
	docGenerator *markdown.DocumentationGenerator,
	project utils.NxProject,
) error {
	docGenerator.WriteProjectHeader(project.Name, project.Type, project.Root)

	projectRoot := filepath.Join(s.rootDir, project.Root)
	projectBasePath := s.rootDir

	// Нормализация путей для Windows
	projectRoot = filepath.ToSlash(projectRoot)
	projectBasePath = filepath.ToSlash(projectBasePath)

	docGenerator.WriteProjectTree(projectRoot, projectBasePath)
	docGenerator.WriteSubHeader("Основные модули")

	return utils.ScanProjectFiles(
		project.SourceDir,
		s.rootDir,
		s.outputFile,
		func(filePath, lang string, content []byte) {
			skip := s.shouldSkipFileContent(filePath)
			docGenerator.WriteFileSection(filePath, content, lang, skip)
		},
		s.IncludeStyles,
		s.IncludeMarkup,
		s.IncludeConfigs,
		s.IncludeTests,
	)
}

func (s *ProjectScanner) shouldSkipFileContent(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	fileName := strings.ToLower(filepath.Base(filePath))

	// Проверка стилей
	if !s.IncludeStyles && (ext == ".css" || ext == ".scss" || ext == ".less") {
		return true
	}

	// Проверка разметки
	if !s.IncludeMarkup && (ext == ".html" || ext == ".htm") {
		return true
	}

	// Проверка конфигов
	if !s.IncludeConfigs && (strings.Contains(fileName, "config") ||
		ext == ".json" || ext == ".yaml" || ext == ".yml") {
		return true
	}

	// Проверка тестов
	if !s.IncludeTests && (strings.Contains(fileName, ".spec.") ||
		strings.Contains(fileName, ".test.") ||
		strings.HasSuffix(fileName, "_test.go")) {
		return true
	}

	return false
}

func (s *ProjectScanner) validatePaths() error {
	if _, err := os.Stat(s.rootDir); os.IsNotExist(err) {
		return fmt.Errorf("директория проекта не существует: %s", s.rootDir)
	}

	if filepath.IsAbs(s.outputFile) {
		return nil
	}

	testFile := s.outputFile + ".test"
	if f, err := os.Create(testFile); err != nil {
		return fmt.Errorf("невозможно записать в файл %s: %v", s.outputFile, err)
	} else {
		f.Close()
		os.Remove(testFile)
	}

	return nil
}

func (s *ProjectScanner) initializeScanner() error {
	if absRoot, err := filepath.Abs(s.rootDir); err == nil {
		s.rootDir = absRoot
	} else {
		return fmt.Errorf("ошибка получения абсолютного пути: %v", err)
	}

	if absOutput, err := filepath.Abs(s.outputFile); err == nil {
		s.outputFile = absOutput
	} else {
		return fmt.Errorf("ошибка получения абсолютного пути: %v", err)
	}

	return s.validatePaths()
}
