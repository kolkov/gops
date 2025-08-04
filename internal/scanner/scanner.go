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

// ProjectScanner отвечает за сканирование и анализ структуры проекта
type ProjectScanner struct {
	rootDir    string
	outputFile string
	nxMonorepo bool
	nxProjects []utils.NxProject
}

// NewProjectScanner создает новый экземпляр сканера проекта
func NewProjectScanner(rootDir, outputFile string) *ProjectScanner {
	return &ProjectScanner{
		rootDir:    rootDir,
		outputFile: outputFile,
	}
}

// Scan выполняет полное сканирование проекта и генерацию документации
func (s *ProjectScanner) Scan(docGenerator *markdown.DocumentationGenerator) error {
	// Определение типа проекта
	s.detectProjectType()

	// Запись заголовка документации
	projectName := filepath.Base(s.rootDir)
	docGenerator.WriteHeader(projectName, time.Now(), s.nxMonorepo)

	// Сканирование в зависимости от типа проекта
	if s.nxMonorepo {
		return s.scanNxMonorepo(docGenerator)
	}
	return s.scanStandardProject(docGenerator)
}

// detectProjectType определяет тип проекта (Nx Monorepo или стандартный)
func (s *ProjectScanner) detectProjectType() {
	// Проверка на Nx Monorepo
	if utils.IsNxMonorepo(s.rootDir) {
		s.nxMonorepo = true
		s.nxProjects = utils.ParseNxProjects(s.rootDir)
	}
}

// scanNxMonorepo обрабатывает Nx Monorepo проекты
func (s *ProjectScanner) scanNxMonorepo(docGenerator *markdown.DocumentationGenerator) error {
	// Отображение структуры Nx
	docGenerator.WriteNxStructure(s.nxProjects)

	// Выбор проектов
	selectedProjects := s.selectProjects()
	if len(selectedProjects) == 0 {
		return fmt.Errorf("не выбрано ни одного проекта")
	}

	// Обработка выбранных проектов
	for _, project := range selectedProjects {
		if err := s.processNxProject(docGenerator, project); err != nil {
			fmt.Printf("Ошибка обработки проекта %s: %v\n", project.Name, err)
		}
	}
	return nil
}

// scanStandardProject обрабатывает стандартные проекты
func (s *ProjectScanner) scanStandardProject(docGenerator *markdown.DocumentationGenerator) error {
	// Генерация структуры проекта
	docGenerator.WriteStandardProjectTree(s.rootDir)

	// Добавляем заголовок "Основные модули"
	docGenerator.WriteModulesHeader()

	// Сканирование файлов
	return utils.ScanStandardProject(s.rootDir, s.outputFile, func(filePath, lang string, content []byte) {
		docGenerator.WriteFileSection(filePath, content, lang)
	})
}

// selectProjects предоставляет пользователю выбор проектов для обработки
func (s *ProjectScanner) selectProjects() []utils.NxProject {
	fmt.Println("\nНайдены проекты в Nx Monorepo:")
	for i, p := range s.nxProjects {
		fmt.Printf("%2d. [%s] %s (%s)\n", i+1, p.Type, p.Name, p.Root)
	}

	fmt.Println("\nВыберите проекты для документации:")
	fmt.Println("  all  - Все проекты")
	fmt.Println("  1,3,5 - Конкретные проекты (через запятую)")
	fmt.Println("  lib:* - Все библиотеки")
	fmt.Println("  app:* - Все приложения")
	fmt.Print("\nВаш выбор: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	return utils.FilterProjects(s.nxProjects, input)
}

// processNxProject обрабатывает отдельный проект в Nx Monorepo
func (s *ProjectScanner) processNxProject(
	docGenerator *markdown.DocumentationGenerator,
	project utils.NxProject,
) error {
	// Запись заголовка проекта
	docGenerator.WriteProjectHeader(project.Name, project.Type, project.Root)

	// Генерация структуры проекта
	projectRoot := filepath.Join(s.rootDir, project.Root)
	docGenerator.WriteProjectTree(project.SourceDir, projectRoot)

	// Используем новый метод для записи подзаголовка
	docGenerator.WriteSubHeader("Основные модули")

	// Сканирование файлов проекта
	return utils.ScanProjectFiles(
		project.SourceDir,
		s.rootDir,
		s.outputFile,
		func(filePath, lang string, content []byte) {
			docGenerator.WriteFileSection(filePath, content, lang)
		},
	)
}

// validatePaths проверяет корректность путей проекта
func (s *ProjectScanner) validatePaths() error {
	if _, err := os.Stat(s.rootDir); os.IsNotExist(err) {
		return fmt.Errorf("директория проекта не существует: %s", s.rootDir)
	}

	if filepath.IsAbs(s.outputFile) {
		return nil
	}

	// Проверка возможности записи в выходной файл
	testFile := s.outputFile + ".test"
	if f, err := os.Create(testFile); err != nil {
		return fmt.Errorf("невозможно записать в файл %s: %v", s.outputFile, err)
	} else {
		f.Close()
		os.Remove(testFile)
	}

	return nil
}

// initializeScanner выполняет предварительные проверки и инициализацию
func (s *ProjectScanner) initializeScanner() error {
	// Приведение путей к абсолютным
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

	// Валидация путей
	return s.validatePaths()
}
