// Package phases содержит фазу анализа проекта
package phases

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/plugin"
	"github.com/kolkov/gops/pkg/logger"
)

// AnalysisPhase - фаза анализа структуры и компонентов проекта
type AnalysisPhase struct {
	registry *plugin.Registry
	logger   *logger.Logger
}

// NewAnalysisPhase создает новую фазу анализа
func NewAnalysisPhase(registry *plugin.Registry, logger *logger.Logger) *AnalysisPhase {
	return &AnalysisPhase{
		registry: registry,
		logger:   logger,
	}
}

// AnalyzeComponents анализирует компоненты проекта
func (a *AnalysisPhase) AnalyzeComponents(ctx context.Context, metadata *model.ProjectMetadata, info *model.ProjectInfo) ([]model.Component, error) {
	a.logger.Info("Starting component analysis")

	components := make([]model.Component, 0)

	// Анализируем основную структуру
	mainComponents := a.analyzeMainStructure(metadata)
	components = append(components, mainComponents...)

	// Анализируем модульную структуру
	modules := a.analyzeModules(metadata)
	for _, module := range modules {
		components = append(components, model.Component{
			Name:        module.Name,
			Type:        "module",
			Path:        module.Path,
			Description: module.Description,
			Important:   module.Public,
		})
	}

	// Анализируем микросервисы
	services := a.analyzeMicroservices(metadata)
	components = append(components, services...)

	// Анализируем библиотеки
	libraries := a.analyzeLibraries(metadata)
	components = append(components, libraries...)

	// Определяем важность компонентов
	a.rankComponentImportance(components, metadata)

	a.logger.Info("Component analysis completed", "count", len(components))

	return components, nil
}

// AnalyzeMetrics анализирует метрики проекта
func (a *AnalysisPhase) AnalyzeMetrics(ctx context.Context, metadata *model.ProjectMetadata) (*model.ProjectMetrics, error) {
	a.logger.Info("Starting metrics analysis")

	metrics := &model.ProjectMetrics{
		TotalFiles:    metadata.TotalFiles,
		TotalPackages: 0,
	}

	// Подсчитываем LOC по расширениям
	for ext, count := range metadata.FilesByExt {
		if a.isSourceCodeExtension(ext) {
			// Приблизительная оценка LOC
			avgLinesPerFile := a.estimateAverageLOC(ext)
			metrics.TotalLOC += count * avgLinesPerFile
			metrics.TotalSLOC += count * (avgLinesPerFile * 7 / 10) // ~70% SLOC
		}
	}

	// Подсчитываем пакеты/модули
	packages := a.countPackages(metadata)
	metrics.TotalPackages = packages

	// Оцениваем количество функций и классов
	metrics.TotalFunctions = a.estimateFunctions(metadata)
	metrics.TotalClasses = a.estimateClasses(metadata)

	// Рассчитываем среднюю сложность (упрощенная оценка)
	if metrics.TotalFunctions > 0 {
		metrics.AverageComplexity = float64(metrics.TotalLOC) / float64(metrics.TotalFunctions)
	}

	a.logger.Info("Metrics analysis completed",
		"loc", metrics.TotalLOC,
		"files", metrics.TotalFiles,
		"packages", metrics.TotalPackages)

	return metrics, nil
}

// AnalyzeDependencies анализирует зависимости проекта
func (a *AnalysisPhase) AnalyzeDependencies(ctx context.Context, metadata *model.ProjectMetadata) ([]model.Dependency, error) {
	a.logger.Info("Starting dependency analysis")

	dependencies := make([]model.Dependency, 0)

	// Анализируем package.json для Node.js
	if _, exists := metadata.Files["package.json"]; exists {
		// TODO: Парсить package.json и извлекать зависимости
		dependencies = append(dependencies, model.Dependency{
			Name:    "placeholder-npm-deps",
			Version: "*",
			Type:    "npm",
		})
	}

	// Анализируем go.mod для Go
	if _, exists := metadata.Files["go.mod"]; exists {
		// TODO: Парсить go.mod и извлекать зависимости
		dependencies = append(dependencies, model.Dependency{
			Name:    "placeholder-go-deps",
			Version: "*",
			Type:    "go",
		})
	}

	// Анализируем requirements.txt для Python
	if _, exists := metadata.Files["requirements.txt"]; exists {
		// TODO: Парсить requirements.txt
		dependencies = append(dependencies, model.Dependency{
			Name:    "placeholder-python-deps",
			Version: "*",
			Type:    "python",
		})
	}

	a.logger.Info("Dependency analysis completed", "count", len(dependencies))

	return dependencies, nil
}

// Private methods

func (a *AnalysisPhase) analyzeMainStructure(metadata *model.ProjectMetadata) []model.Component {
	components := make([]model.Component, 0)

	// Анализируем корневую структуру
	rootDirs := a.getRootDirectories(metadata)

	for _, dir := range rootDirs {
		componentType := a.determineComponentType(dir)
		if componentType != "" {
			components = append(components, model.Component{
				Name:      dir,
				Type:      componentType,
				Path:      dir,
				Important: a.isImportantDirectory(dir),
			})
		}
	}

	return components
}

func (a *AnalysisPhase) analyzeModules(metadata *model.ProjectMetadata) []model.Module {
	modules := make([]model.Module, 0)

	// Ищем модульные структуры
	for path := range metadata.Files {
		dir := filepath.Dir(path)

		// Go модули (внутри internal/)
		if strings.Contains(dir, "internal/") {
			parts := strings.Split(dir, "/")
			for i, part := range parts {
				if part == "internal" && i+1 < len(parts) {
					moduleName := parts[i+1]
					if !a.moduleExists(modules, moduleName) {
						modules = append(modules, model.Module{
							Name:   moduleName,
							Path:   filepath.Join("internal", moduleName),
							Type:   "internal",
							Public: false,
						})
					}
					break
				}
			}
		}

		// Публичные пакеты (pkg/)
		if strings.Contains(dir, "pkg/") {
			parts := strings.Split(dir, "/")
			for i, part := range parts {
				if part == "pkg" && i+1 < len(parts) {
					moduleName := parts[i+1]
					if !a.moduleExists(modules, moduleName) {
						modules = append(modules, model.Module{
							Name:   moduleName,
							Path:   filepath.Join("pkg", moduleName),
							Type:   "public",
							Public: true,
						})
					}
					break
				}
			}
		}
	}

	return modules
}

func (a *AnalysisPhase) analyzeMicroservices(metadata *model.ProjectMetadata) []model.Component {
	services := make([]model.Component, 0)

	// Паттерны для определения микросервисов
	servicePatterns := []string{
		"services/",
		"microservices/",
		"apps/",     // Для NX monorepo
		"packages/", // Для monorepo
	}

	foundServices := make(map[string]bool)

	for path := range metadata.Files {
		for _, pattern := range servicePatterns {
			if strings.Contains(path, pattern) {
				parts := strings.Split(path, pattern)
				if len(parts) > 1 {
					remainingPath := parts[1]
					serviceParts := strings.Split(remainingPath, "/")
					if len(serviceParts) > 0 && serviceParts[0] != "" {
						serviceName := serviceParts[0]
						if !foundServices[serviceName] {
							foundServices[serviceName] = true
							services = append(services, model.Component{
								Name:        serviceName,
								Type:        "service",
								Path:        filepath.Join(pattern[:len(pattern)-1], serviceName),
								Description: "Microservice",
								Important:   true,
							})
						}
					}
				}
			}
		}
	}

	return services
}

func (a *AnalysisPhase) analyzeLibraries(metadata *model.ProjectMetadata) []model.Component {
	libraries := make([]model.Component, 0)

	// Паттерны для библиотек
	libPatterns := []string{
		"libs/",
		"lib/",
		"libraries/",
		"shared/",
		"common/",
	}

	foundLibs := make(map[string]bool)

	for path := range metadata.Files {
		for _, pattern := range libPatterns {
			if strings.Contains(path, pattern) {
				parts := strings.Split(path, pattern)
				if len(parts) > 1 {
					remainingPath := parts[1]
					libParts := strings.Split(remainingPath, "/")
					if len(libParts) > 0 && libParts[0] != "" {
						libName := libParts[0]
						if !foundLibs[libName] {
							foundLibs[libName] = true
							libraries = append(libraries, model.Component{
								Name:        libName,
								Type:        "library",
								Path:        filepath.Join(pattern[:len(pattern)-1], libName),
								Description: "Shared library",
								Important:   true,
							})
						}
					}
				}
			}
		}
	}

	return libraries
}

func (a *AnalysisPhase) getRootDirectories(metadata *model.ProjectMetadata) []string {
	dirs := make(map[string]bool)

	for path := range metadata.Files {
		parts := strings.Split(path, string(filepath.Separator))
		if len(parts) > 0 && parts[0] != "" {
			dirs[parts[0]] = true
		}
	}

	result := make([]string, 0, len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}

	return result
}

func (a *AnalysisPhase) determineComponentType(dir string) string {
	switch dir {
	case "cmd", "bin":
		return "executable"
	case "pkg", "lib", "libs":
		return "library"
	case "internal":
		return "internal"
	case "api":
		return "api"
	case "web", "frontend", "client":
		return "frontend"
	case "backend", "server":
		return "backend"
	case "docs", "documentation":
		return "documentation"
	case "test", "tests", "spec", "specs":
		return "tests"
	case "scripts", "tools":
		return "tools"
	case "config", "configs":
		return "configuration"
	case "migrations", "db":
		return "database"
	case "assets", "static", "public":
		return "assets"
	case "vendor", "node_modules":
		return "dependencies"
	case "build", "dist", "out":
		return "build"
	case "src":
		return "source"
	case "examples", "samples":
		return "examples"
	case "apps":
		return "applications"
	case "services", "microservices":
		return "services"
	default:
		// Проверяем специфичные для языков/фреймворков
		if strings.HasPrefix(dir, ".") {
			return "" // Скрытые директории
		}
		return "component"
	}
}

func (a *AnalysisPhase) isImportantDirectory(dir string) bool {
	important := []string{
		"src", "cmd", "pkg", "internal", "api",
		"core", "main", "app", "apps",
		"services", "components", "modules",
	}

	dir = strings.ToLower(dir)
	for _, imp := range important {
		if dir == imp {
			return true
		}
	}

	return false
}

func (a *AnalysisPhase) moduleExists(modules []model.Module, name string) bool {
	for _, m := range modules {
		if m.Name == name {
			return true
		}
	}
	return false
}

func (a *AnalysisPhase) rankComponentImportance(components []model.Component, metadata *model.ProjectMetadata) {
	for i := range components {
		// Подсчитываем количество файлов в компоненте
		fileCount := 0
		for path := range metadata.Files {
			if strings.HasPrefix(path, components[i].Path) {
				fileCount++
			}
		}

		// Определяем важность по количеству файлов и типу
		if fileCount > 10 || components[i].Type == "core" || components[i].Type == "api" {
			components[i].Important = true
		}

		// Добавляем описание если его нет
		if components[i].Description == "" {
			components[i].Description = a.generateComponentDescription(components[i], fileCount)
		}
	}
}

func (a *AnalysisPhase) generateComponentDescription(component model.Component, fileCount int) string {
	typeDescriptions := map[string]string{
		"executable":    "Command-line application",
		"library":       "Shared library",
		"internal":      "Internal packages",
		"api":          "API definitions and handlers",
		"frontend":      "Frontend application",
		"backend":       "Backend services",
		"documentation": "Project documentation",
		"tests":        "Test suites",
		"tools":        "Development tools and scripts",
		"configuration": "Configuration files",
		"database":     "Database schemas and migrations",
		"assets":       "Static assets",
		"dependencies": "External dependencies",
		"build":        "Build artifacts",
		"source":       "Source code",
		"examples":     "Example code and demos",
		"applications": "Applications",
		"services":     "Microservices",
		"component":    "Project component",
	}

	if desc, ok := typeDescriptions[component.Type]; ok {
		return desc
	}

	return "Project component"
}

func (a *AnalysisPhase) isSourceCodeExtension(ext string) bool {
	sourceExts := []string{
		".go", ".js", ".ts", ".jsx", ".tsx",
		".py", ".java", ".c", ".h", ".cpp", ".hpp",
		".cs", ".rs", ".rb", ".php", ".swift",
		".kt", ".scala", ".r", ".lua", ".pl",
	}

	ext = strings.ToLower(ext)
	for _, srcExt := range sourceExts {
		if ext == srcExt {
			return true
		}
	}

	return false
}

func (a *AnalysisPhase) estimateAverageLOC(ext string) int {
	// Приблизительные средние LOC для разных типов файлов
	estimates := map[string]int{
		".go":   150,
		".js":   120,
		".ts":   130,
		".py":   100,
		".java": 200,
		".c":    180,
		".cpp":  200,
		".cs":   150,
		".rs":   140,
		".rb":   80,
		".php":  150,
	}

	if loc, ok := estimates[strings.ToLower(ext)]; ok {
		return loc
	}

	return 100 // По умолчанию
}

func (a *AnalysisPhase) countPackages(metadata *model.ProjectMetadata) int {
	packages := make(map[string]bool)

	for path := range metadata.Files {
		dir := filepath.Dir(path)
		// Считаем уникальные директории как пакеты
		packages[dir] = true
	}

	return len(packages)
}

func (a *AnalysisPhase) estimateFunctions(metadata *model.ProjectMetadata) int {
	// Очень грубая оценка: ~1 функция на 20 строк кода
	if metadata.TotalFiles > 0 {
		avgFunctionsPerFile := 5
		sourceFiles := 0

		for ext, count := range metadata.FilesByExt {
			if a.isSourceCodeExtension(ext) {
				sourceFiles += count
			}
		}

		return sourceFiles * avgFunctionsPerFile
	}

	return 0
}

func (a *AnalysisPhase) estimateClasses(metadata *model.ProjectMetadata) int {
	// Оценка по объектно-ориентированным языкам
	classBasedFiles := 0

	classExts := []string{".java", ".cs", ".cpp", ".py", ".rb", ".php", ".ts", ".kt"}
	for _, ext := range classExts {
		if count, ok := metadata.FilesByExt[ext]; ok {
			classBasedFiles += count
		}
	}

	// Примерно 1-2 класса на файл в ООП языках
	return classBasedFiles * 1
}