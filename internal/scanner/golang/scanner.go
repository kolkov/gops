package golang

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

var _ scanner.ProjectScanner = (*GoScanner)(nil)

type GoScanner struct {
	rootDir    string
	outputFile string
	cfg        *model.ScanConfig
	logger     *logger.Logger
}

func NewScanner(rootDir, outputFile string, cfg *model.ScanConfig, logger *logger.Logger) *GoScanner {
	return &GoScanner{
		rootDir:    rootDir,
		outputFile: outputFile,
		cfg:        cfg,
		logger:     logger,
	}
}

func (s *GoScanner) Scan(ctx context.Context, docGen docgen.Generator) error {
	// Всегда включаем go.mod
	s.cfg.ImportantFiles = append(s.cfg.ImportantFiles, "go.mod")

	// Всегда исключаем go.sum
	s.cfg.ExcludedPatterns = append(s.cfg.ExcludedPatterns, "go.sum")

	// Запрос на включение других важных файлов
	s.askForImportantFiles()

	meta := &model.ProjectMeta{
		Name:    filepath.Base(s.rootDir),
		Type:    "Go проект",
		RootDir: s.rootDir,
	}

	docGen.WriteHeader(meta)

	treeBuilder := filesystem.NewTreeBuilder(s.rootDir, s.cfg)
	tree, err := treeBuilder.Build()
	if err != nil {
		s.logger.Error("Failed to build project tree", err)
		return err
	}
	docGen.WriteTree(tree)

	docGen.WriteModulesHeader()
	return filesystem.ScanProject(s.rootDir, s.cfg, s.logger, func(file *model.ProjectFile) {
		docGen.WriteFileSection(file)
	})
}

func (s *GoScanner) askForImportantFiles() {
	importantFiles := []string{
		"Makefile",
		"Dockerfile",
		"docker-compose.yml",
		".env",
	}

	fmt.Println("\nВключить другие важные конфигурационные файлы?")
	fmt.Println("1. Makefile (сборка проекта)")
	fmt.Println("2. Dockerfile (контейнеризация проекта)")
	fmt.Println("3. docker-compose.yml (оркестрация контейнеров)")
	fmt.Println("4. .env (переменные окружения)")
	fmt.Println("0. Не включать (по умолчанию)")
	fmt.Print("Выберите файлы через запятую (например, 1,2): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" || input == "0" {
		return
	}

	choices := strings.Split(input, ",")
	for _, choice := range choices {
		choice = strings.TrimSpace(choice)
		if choice == "" {
			continue
		}
		idx, err := strconv.Atoi(choice)
		if err != nil || idx < 1 || idx > len(importantFiles) {
			continue
		}
		s.cfg.ImportantFiles = append(s.cfg.ImportantFiles, importantFiles[idx-1])
	}
}
