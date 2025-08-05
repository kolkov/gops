package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"project_scanner/internal/markdown"
	"project_scanner/internal/projecttype"
	"project_scanner/internal/utils"
)

type ProjectScanner struct {
	rootDir            string
	outputFile         string
	projectType        string
	nxMonorepo         bool
	nxProjects         []utils.NxProject
	IncludeStyles      bool
	IncludeMarkup      bool
	IncludeConfigs     bool
	IncludeTests       bool
	IncludeRootPackage bool
	hasRootPackage     bool
}

func NewProjectScanner(rootDir, outputFile string) *ProjectScanner {
	return &ProjectScanner{
		rootDir:    rootDir,
		outputFile: outputFile,
	}
}

func (s *ProjectScanner) AskContentSettings() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nНастройки документации:")
	fmt.Print("1. Включать файлы стилей (CSS, SCSS)? [y/N]: ")
	s.IncludeStyles = readYesNo(reader, false)

	fmt.Print("2. Включать файлы разметки (HTML)? [y/N]: ")
	s.IncludeMarkup = readYesNo(reader, false)

	fmt.Print("3. Включать конфигурационные файлы? [y/N]: ")
	s.IncludeConfigs = readYesNo(reader, false)

	fmt.Print("4. Включать тестовые файлы? [y/N]: ")
	s.IncludeTests = readYesNo(reader, false)

	if s.hasRootPackage {
		fmt.Print("5. Включать корневой package.json? [Y/n]: ")
		s.IncludeRootPackage = readYesNo(reader, true)
	}
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
	// Двойная проверка на конфликт Go и JS
	if s.hasConflictingFiles() {
		return fmt.Errorf("обнаружены оба файла: go.mod и package.json. Сканирование невозможно")
	}

	var subType string

	switch s.projectType {
	case projecttype.NxMonorepo:
		subType = "NX Monorepo"
		return s.scanNxMonorepo(docGenerator)
	case projecttype.JS, projecttype.Angular, projecttype.BrowserExtension:
		subType = strings.ToUpper(s.projectType[:1]) + s.projectType[1:]
		return s.scanJSProject(docGenerator, subType)
	case projecttype.Go:
		subType = "Go"
		return s.scanStandardProject(docGenerator)
	default:
		if utils.ContainsJSFiles(s.rootDir) {
			s.projectType = projecttype.JS
			subType = "JavaScript (автоопределение)"
			return s.scanJSProject(docGenerator, subType)
		}
		return fmt.Errorf("не удалось определить тип проекта")
	}
}

func (s *ProjectScanner) hasConflictingFiles() bool {
	_, goModExists := os.Stat(filepath.Join(s.rootDir, "go.mod"))
	_, pkgJsonExists := os.Stat(filepath.Join(s.rootDir, "package.json"))
	return goModExists == nil && pkgJsonExists == nil
}

func (s *ProjectScanner) InitializeScanner() error {
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

	if err := s.detectProjectType(); err != nil {
		return err
	}

	if s.projectType == projecttype.JS ||
		s.projectType == projecttype.Angular ||
		s.projectType == projecttype.BrowserExtension ||
		s.projectType == projecttype.NxMonorepo {
		rootPkgPath := filepath.Join(s.rootDir, "package.json")
		if _, err := os.Stat(rootPkgPath); err == nil {
			s.hasRootPackage = true
		}
	}

	return s.validatePaths()
}

func (s *ProjectScanner) detectProjectType() error {
	// Проверка на конфликт Go и JS
	goModPath := filepath.Join(s.rootDir, "go.mod")
	pkgJsonPath := filepath.Join(s.rootDir, "package.json")

	_, goModExists := os.Stat(goModPath)
	_, pkgJsonExists := os.Stat(pkgJsonPath)

	if goModExists == nil && pkgJsonExists == nil {
		return fmt.Errorf("обнаружены и go.mod, и package.json в корне проекта. Это не поддерживается")
	}

	// Приоритет 1: Nx Monorepo
	if utils.IsNxMonorepo(s.rootDir) {
		s.projectType = projecttype.NxMonorepo
		s.nxMonorepo = true
		s.nxProjects = utils.ParseNxProjects(s.rootDir)
		return nil
	}

	// Приоритет 2: Специфичные типы проектов
	if s.isBrowserExtension() {
		s.projectType = projecttype.BrowserExtension
		return nil
	}

	if s.isAngularProject() {
		s.projectType = projecttype.Angular
		return nil
	}

	// Приоритет 3: Общие типы проектов
	if pkgJsonExists == nil {
		s.projectType = projecttype.JS
		return nil
	}

	if goModExists == nil {
		s.projectType = projecttype.Go
		return nil
	}

	s.projectType = projecttype.Unknown
	return nil
}

func (s *ProjectScanner) isBrowserExtension() bool {
	manifestPath := filepath.Join(s.rootDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return true
	}

	browserFiles := []string{"background.js", "content-script.js", "popup.html"}
	count := 0
	for _, file := range browserFiles {
		filePath := filepath.Join(s.rootDir, file)
		if _, err := os.Stat(filePath); err == nil {
			count++
		}
	}
	return count >= 2
}

func (s *ProjectScanner) isAngularProject() bool {
	angularJsonPath := filepath.Join(s.rootDir, "angular.json")
	if _, err := os.Stat(angularJsonPath); err == nil {
		return true
	}

	angularFiles := []string{"src/main.ts", "src/app/app.module.ts"}
	count := 0
	for _, file := range angularFiles {
		filePath := filepath.Join(s.rootDir, file)
		if _, err := os.Stat(filePath); err == nil {
			count++
		}
	}
	return count >= 2
}

func (s *ProjectScanner) scanJSProject(docGenerator *markdown.DocumentationGenerator, subType string) error {
	docGenerator.WriteHeader(filepath.Base(s.rootDir), time.Now(), false, s.projectType, subType)
	docGenerator.WriteStandardProjectTree(s.rootDir)
	docGenerator.WriteModulesHeader()

	s.processMandatoryJSFiles(docGenerator)

	switch s.projectType {
	case projecttype.BrowserExtension:
		return s.scanBrowserExtension(docGenerator)
	case projecttype.Angular:
		return s.scanAngularProject(docGenerator)
	default:
		return s.scanGenericJSProject(docGenerator)
	}
}

func (s *ProjectScanner) processMandatoryJSFiles(docGenerator *markdown.DocumentationGenerator) {
	// Обрабатываем только package.json
	if s.IncludeRootPackage && s.hasRootPackage {
		filePath := filepath.Join(s.rootDir, "package.json")
		if _, err := os.Stat(filePath); err == nil {
			content, err := os.ReadFile(filePath)
			if err == nil {
				lang := "json"
				docGenerator.WriteFileSection("package.json", content, lang, false)
			}
		}
	}
}

func (s *ProjectScanner) scanBrowserExtension(docGenerator *markdown.DocumentationGenerator) error {
	var extensionFiles []string
	processed := make(map[string]bool)

	for _, file := range extensionFiles {
		filePath := filepath.Join(s.rootDir, file)
		if _, exists := processed[filePath]; !exists {
			if _, err := os.Stat(filePath); err == nil {
				content, err := os.ReadFile(filePath)
				if err == nil {
					lang := "json"
					if strings.HasSuffix(file, ".js") {
						lang = "javascript"
					}
					docGenerator.WriteFileSection(file, content, lang, false)
					processed[filePath] = true
				}
			}
		}
	}

	return s.scanGenericJSProject(docGenerator)
}

func (s *ProjectScanner) scanAngularProject(docGenerator *markdown.DocumentationGenerator) error {
	angularFiles := []string{"angular.json", "src/main.ts", "src/app/app.module.ts"}

	for _, file := range angularFiles {
		filePath := filepath.Join(s.rootDir, file)
		if _, err := os.Stat(filePath); err == nil {
			content, err := os.ReadFile(filePath)
			if err == nil {
				lang := "typescript"
				if strings.HasSuffix(file, ".json") {
					lang = "json"
				}
				docGenerator.WriteFileSection(file, content, lang, false)
			}
		}
	}

	return s.scanGenericJSProject(docGenerator)
}

func (s *ProjectScanner) scanGenericJSProject(docGenerator *markdown.DocumentationGenerator) error {
	rootPackagePath := filepath.Join(s.rootDir, "package.json")

	return filepath.Walk(s.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Явно пропускаем package-lock.json и дубликаты package.json
		if info.Name() == "package-lock.json" || info.Name() == "yarn.lock" {
			return nil
		}
		if path == rootPackagePath && s.IncludeRootPackage {
			return nil // Уже обработан в mandatory files
		}

		if info.IsDir() {
			if utils.ShouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(s.rootDir, path)
		ext := strings.ToLower(filepath.Ext(path))

		shouldProcess := false
		switch ext {
		case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
			shouldProcess = true
		case ".css", ".scss", ".less":
			shouldProcess = s.IncludeStyles
		case ".html", ".htm":
			shouldProcess = s.IncludeMarkup
		case ".json", ".yaml", ".yml":
			shouldProcess = s.IncludeConfigs
		}

		if shouldProcess && !s.IncludeTests {
			fileName := strings.ToLower(info.Name())
			if strings.Contains(fileName, ".test.") ||
				strings.Contains(fileName, ".spec.") ||
				strings.Contains(filepath.Dir(path), "test") ||
				strings.Contains(filepath.Dir(path), "__tests__") {
				shouldProcess = false
			}
		}

		if shouldProcess {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			lang := utils.GetFileLanguage(path)
			docGenerator.WriteFileSection(relPath, content, lang, false)
		}

		return nil
	})
}

func (s *ProjectScanner) scanNxMonorepo(docGenerator *markdown.DocumentationGenerator) error {
	docGenerator.WriteHeader(filepath.Base(s.rootDir), time.Now(), true, projecttype.NxMonorepo, "NX Monorepo")
	docGenerator.WriteNxStructure(s.nxProjects)

	if s.IncludeRootPackage && s.hasRootPackage {
		rootPkgPath := filepath.Join(s.rootDir, "package.json")
		if content, err := os.ReadFile(rootPkgPath); err == nil {
			docGenerator.WriteSubHeader("Корневой package.json")
			docGenerator.WriteFileSection("package.json", content, "json", false)
		}
	}

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

	projectRoot = filepath.ToSlash(projectRoot)
	projectBasePath = filepath.ToSlash(projectBasePath)

	docGenerator.WriteProjectTree(projectRoot, projectBasePath)
	docGenerator.WriteSubHeader("Основные модули")

	if s.IncludeConfigs {
		projectJsonPath := filepath.Join(projectRoot, "project.json")
		if content, err := os.ReadFile(projectJsonPath); err == nil {
			relPath, _ := filepath.Rel(projectBasePath, projectJsonPath)
			docGenerator.WriteFileSection(relPath, content, "json", false)
		}
	}

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
	fileName := strings.ToLower(filepath.Base(filePath))

	if fileName == "project.json" {
		return false
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	if !s.IncludeStyles && (ext == ".css" || ext == ".scss" || ext == ".less") {
		return true
	}

	if !s.IncludeMarkup && (ext == ".html" || ext == ".htm") {
		return true
	}

	if !s.IncludeConfigs && (strings.Contains(fileName, "config") ||
		ext == ".json" || ext == ".yaml" || ext == ".yml") {
		return true
	}

	if !s.IncludeTests && (strings.Contains(fileName, ".spec.") ||
		strings.Contains(fileName, ".test.") ||
		strings.HasSuffix(fileName, "_test.go") ||
		strings.HasSuffix(fileName, "_test.js")) {
		return true
	}

	return false
}

func (s *ProjectScanner) scanStandardProject(docGenerator *markdown.DocumentationGenerator) error {
	docGenerator.WriteHeader(filepath.Base(s.rootDir), time.Now(), false, projecttype.Go, "Go проект")
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
