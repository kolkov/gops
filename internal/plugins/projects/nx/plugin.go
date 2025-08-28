// Package nx contains the NX Monorepo project type plugin
package nx

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/plugin"
	"github.com/kolkov/gops/pkg/logger"
)

// Plugin - плагин для работы с NX Monorepo проектами
type Plugin struct {
	name        string
	version     string
	description string
	logger      *logger.Logger
	mu          sync.RWMutex
}

// NxConfig структура для nx.json
type NxConfig struct {
	NPMScope        string                 `json:"npmScope"`
	WorkspaceLayout NxWorkspaceLayout      `json:"workspaceLayout"`
	Projects        map[string]interface{} `json:"projects"`
	TargetDefaults  map[string]interface{} `json:"targetDefaults"`
}

type NxWorkspaceLayout struct {
	AppsDir string `json:"appsDir"`
	LibsDir string `json:"libsDir"`
}

// NxProject структура для project.json в NX
type NxProject struct {
	Name       string                 `json:"name"`
	Root       string                 `json:"root"`
	SourceRoot string                 `json:"sourceRoot"`
	ProjectType string                `json:"projectType"`
	Targets    map[string]interface{} `json:"targets"`
	Tags       []string               `json:"tags"`
}

// New создает новый экземпляр плагина
func New() *Plugin {
	return &Plugin{
		name:        "nx",
		version:     "1.0.0",
		description: "NX Monorepo project type detection and analysis plugin",
	}
}

// Plugin interface methods

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Version() string {
	return p.version
}

func (p *Plugin) Description() string {
	return p.description
}

func (p *Plugin) Type() plugin.PluginType {
	return plugin.PluginTypeProject
}

func (p *Plugin) Priority() int {
	return 20 // Higher priority than Angular for monorepos
}

func (p *Plugin) Dependencies() []string {
	return []string{"javascript", "angular"} // Может включать Angular проекты
}

// Lifecycle methods

func (p *Plugin) Init(config map[string]interface{}) error {
	// Инициализация с конфигурацией
	return nil
}

func (p *Plugin) OnLoad(ctx context.Context, logger *logger.Logger) error {
	p.logger = logger
	p.logger.Info("NX Monorepo project plugin loaded")
	return nil
}

func (p *Plugin) OnEnable(ctx context.Context) error {
	p.logger.Info("NX Monorepo project plugin enabled")
	return nil
}

func (p *Plugin) OnDisable(ctx context.Context) error {
	p.logger.Info("NX Monorepo project plugin disabled")
	return nil
}

func (p *Plugin) OnUnload(ctx context.Context) error {
	p.logger.Info("NX Monorepo project plugin unloaded")
	return nil
}

func (p *Plugin) IsEnabled() bool {
	return true
}

func (p *Plugin) GetStatus() plugin.PluginStatus {
	return plugin.StatusActive
}

func (p *Plugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:         p.name,
		Version:      p.version,
		Description:  p.description,
		Author:       "GOPS Team",
		License:      "MIT",
		Homepage:     "https://github.com/kolkov/gops",
		Tags:         []string{"project", "nx", "monorepo", "workspace", "angular", "typescript"},
		Capabilities: []plugin.PluginCapability{plugin.CapabilityProjectDet, plugin.CapabilityDependencies},
	}
}

// ProjectTypePlugin interface methods

func (p *Plugin) Detect(metadata *model.ProjectMetadata) (float32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	confidence := float32(0.0)

	// Проверяем nx.json - основной маркер NX workspace
	if _, exists := metadata.Files["nx.json"]; exists {
		confidence += 0.6
	}

	// Проверяем workspace.json (старые версии NX)
	if _, exists := metadata.Files["workspace.json"]; exists {
		confidence += 0.5
	}

	// Проверяем характерную структуру NX
	nxDirs := []string{"apps", "libs", "tools"}
	foundDirs := 0
	for _, dir := range nxDirs {
		if p.hasDirectoryInFiles(metadata.Files, dir) {
			foundDirs++
			confidence += 0.1
		}
	}

	// Если найдены и nx.json и характерные директории
	if foundDirs >= 2 && confidence >= 0.6 {
		confidence += 0.2
	}

	// Проверяем package.json на NX зависимости
	if packageFile, exists := metadata.Files["package.json"]; exists {
		if p.hasNxDependencies(packageFile.Path) {
			confidence += 0.2
		}
	}

	// Проверяем наличие project.json файлов в подпроектах
	projectJsonCount := 0
	for path := range metadata.Files {
		if strings.HasSuffix(path, "project.json") && path != "project.json" {
			projectJsonCount++
		}
	}
	if projectJsonCount > 1 {
		confidence += 0.1
	}

	p.logger.Debug("NX detection", "confidence", confidence, "projectJsonCount", projectJsonCount)

	// Возвращаем confidence, ограниченный 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence, nil
}

func (p *Plugin) AnalyzeStructure(ctx context.Context, metadata *model.ProjectMetadata) (*model.ProjectInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	info := &model.ProjectInfo{
		Type:     "NX Monorepo",
		Language: "TypeScript/JavaScript",
		Version:  p.detectNxVersion(metadata),
		Components: make([]model.Component, 0),
		Modules:    make([]model.Module, 0),
	}

	// Анализируем NX workspace структуру
	if err := p.analyzeNxWorkspace(metadata, info); err != nil {
		p.logger.Warn("Failed to analyze NX workspace", "error", err)
	}

	// Анализируем приложения (apps)
	p.analyzeNxApps(metadata, info)

	// Анализируем библиотеки (libs)
	p.analyzeNxLibs(metadata, info)

	// Анализируем инструменты (tools)
	p.analyzeNxTools(metadata, info)

	return info, nil
}

func (p *Plugin) GetLoadingRules() []model.LoadingRule {
	return []model.LoadingRule{
		// NX конфигурации - высший приоритет
		{
			Pattern:     "nx.json",
			ContentMode: model.ContentModeFull,
			Priority:    30,
		},
		{
			Pattern:     "workspace.json",
			ContentMode: model.ContentModeFull,
			Priority:    25,
		},
		{
			Pattern:     "**/project.json",
			ContentMode: model.ContentModeFull,
			Priority:    20,
		},
		// Основные конфигурации проекта
		{
			Pattern:     "package.json",
			ContentMode: model.ContentModeFull,
			Priority:    20,
		},
		{
			Pattern:     "tsconfig.base.json",
			ContentMode: model.ContentModeFull,
			Priority:    15,
		},
		// Angular компоненты в приложениях
		{
			Pattern:     "apps/**/*.component.ts",
			ContentMode: model.ContentModeFull,
			Priority:    15,
		},
		{
			Pattern:     "apps/**/*.service.ts",
			ContentMode: model.ContentModeFull,
			Priority:    12,
		},
		{
			Pattern:     "apps/**/*.module.ts",
			ContentMode: model.ContentModeFull,
			Priority:    15,
		},
		// Библиотеки - полное содержимое
		{
			Pattern:     "libs/**/*.ts",
			ContentMode: model.ContentModeFull,
			Priority:    10,
		},
		{
			Pattern:     "libs/**/index.ts",
			ContentMode: model.ContentModeFull,
			Priority:    18,
		},
		// HTML шаблоны - только заголовки
		{
			Pattern:     "**/*.component.html",
			ContentMode: model.ContentModeHeaders,
			Priority:    8,
		},
		// Стили - сжатые
		{
			Pattern:     "**/*.scss",
			ContentMode: model.ContentModeCompressed,
			Priority:    5,
		},
		{
			Pattern:     "**/*.css",
			ContentMode: model.ContentModeNone,
			Priority:    3,
		},
		// Инструменты
		{
			Pattern:     "tools/**/*.ts",
			ContentMode: model.ContentModeFull,
			Priority:    8,
		},
	}
}

func (p *Plugin) GetExclusionRules() []model.ExclusionRule {
	return []model.ExclusionRule{
		{
			Pattern: "node_modules/**",
			Reason:  "Dependencies",
		},
		{
			Pattern: "dist/**",
			Reason:  "Build artifacts",
		},
		{
			Pattern: "**/dist/**",
			Reason:  "App/lib build artifacts",
		},
		{
			Pattern: ".nx/**",
			Reason:  "NX cache",
		},
		{
			Pattern: "coverage/**",
			Reason:  "Test coverage",
		},
		{
			Pattern: "**/coverage/**",
			Reason:  "App/lib test coverage",
		},
		{
			Pattern: "**/*.spec.ts",
			Reason:  "Unit test files",
		},
		{
			Pattern: "**/*.e2e-spec.ts",
			Reason:  "E2E test files",
		},
		{
			Pattern: "**/e2e/**",
			Reason:  "E2E test directories",
		},
		{
			Pattern: ".angular/**",
			Reason:  "Angular cache",
		},
		{
			Pattern: "tmp/**",
			Reason:  "Temporary files",
		},
	}
}

// Private helper methods

func (p *Plugin) hasNxDependencies(packagePath string) bool {
	content, err := os.ReadFile(packagePath)
	if err != nil {
		return false
	}

	var packageJson map[string]interface{}
	if err := json.Unmarshal(content, &packageJson); err != nil {
		return false
	}

	// Проверяем dependencies и devDependencies
	depKeys := []string{"dependencies", "devDependencies"}
	nxPackages := []string{"@nrwl/nx", "@nx/nx", "nx"}

	for _, depKey := range depKeys {
		if deps, ok := packageJson[depKey].(map[string]interface{}); ok {
			for _, nxPkg := range nxPackages {
				if _, hasNx := deps[nxPkg]; hasNx {
					return true
				}
			}
		}
	}

	return false
}

func (p *Plugin) hasDirectoryInFiles(files map[string]*model.FileInfo, dirPath string) bool {
	for path := range files {
		if strings.HasPrefix(path, dirPath+"/") {
			return true
		}
	}
	return false
}

func (p *Plugin) detectNxVersion(metadata *model.ProjectMetadata) string {
	packageFile, exists := metadata.Files["package.json"]
	if !exists {
		return "unknown"
	}

	content, err := os.ReadFile(packageFile.Path)
	if err != nil {
		return "unknown"
	}

	var packageJson map[string]interface{}
	if err := json.Unmarshal(content, &packageJson); err != nil {
		return "unknown"
	}

	// Проверяем devDependencies для NX
	if devDeps, ok := packageJson["devDependencies"].(map[string]interface{}); ok {
		if nxVersion, hasNx := devDeps["nx"].(string); hasNx {
			return strings.TrimPrefix(nxVersion, "^")
		}
		if nxVersion, hasNx := devDeps["@nrwl/nx"].(string); hasNx {
			return strings.TrimPrefix(nxVersion, "^")
		}
		if nxVersion, hasNx := devDeps["@nx/nx"].(string); hasNx {
			return strings.TrimPrefix(nxVersion, "^")
		}
	}

	return "unknown"
}

func (p *Plugin) analyzeNxWorkspace(metadata *model.ProjectMetadata, info *model.ProjectInfo) error {
	nxConfigFile, exists := metadata.Files["nx.json"]
	if !exists {
		return fmt.Errorf("nx.json not found")
	}

	content, err := os.ReadFile(nxConfigFile.Path)
	if err != nil {
		return fmt.Errorf("failed to read nx.json: %w", err)
	}

	var nxConfig NxConfig
	if err := json.Unmarshal(content, &nxConfig); err != nil {
		return fmt.Errorf("failed to parse nx.json: %w", err)
	}

	// Добавляем информацию о workspace
	if nxConfig.NPMScope != "" {
		info.Description = fmt.Sprintf("NX Workspace (@%s)", nxConfig.NPMScope)
	}

	return nil
}

func (p *Plugin) analyzeNxApps(metadata *model.ProjectMetadata, info *model.ProjectInfo) {
	appsFound := make(map[string]bool)

	// Ищем приложения в директории apps
	for path := range metadata.Files {
		if strings.HasPrefix(path, "apps/") {
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				appName := parts[1]
				if !appsFound[appName] {
					appsFound[appName] = true

					// Определяем тип приложения
					appType := p.detectAppType(metadata, "apps/"+appName)

					component := model.Component{
						Name:        appName,
						Type:        "nx-app",
						Path:        "apps/" + appName,
						Description: fmt.Sprintf("NX Application (%s)", appType),
						Important:   true, // Все приложения важны
					}

					info.Components = append(info.Components, component)
				}
			}
		}
	}
}

func (p *Plugin) analyzeNxLibs(metadata *model.ProjectMetadata, info *model.ProjectInfo) {
	libsFound := make(map[string]bool)

	// Ищем библиотеки в директории libs
	for path := range metadata.Files {
		if strings.HasPrefix(path, "libs/") {
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				libName := parts[1]
				if !libsFound[libName] {
					libsFound[libName] = true

					// Определяем тип библиотеки
					libType := p.detectLibType(metadata, "libs/"+libName)

					module := model.Module{
						Name:        libName,
						Path:        "libs/" + libName,
						Type:        "nx-lib",
						Public:      true, // Библиотеки обычно публичные
						Description: fmt.Sprintf("NX Library (%s)", libType),
					}

					info.Modules = append(info.Modules, module)
				}
			}
		}
	}
}

func (p *Plugin) analyzeNxTools(metadata *model.ProjectMetadata, info *model.ProjectInfo) {
	toolsFound := make(map[string]bool)

	// Ищем инструменты в директории tools
	for path := range metadata.Files {
		if strings.HasPrefix(path, "tools/") {
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				toolName := parts[1]
				if !toolsFound[toolName] {
					toolsFound[toolName] = true

					component := model.Component{
						Name:        toolName,
						Type:        "nx-tool",
						Path:        "tools/" + toolName,
						Description: "NX Development Tool",
						Important:   false,
					}

					info.Components = append(info.Components, component)
				}
			}
		}
	}
}

func (p *Plugin) detectAppType(metadata *model.ProjectMetadata, appPath string) string {
	// Проверяем на Angular
	if p.hasFileInPath(metadata, appPath, "*.component.ts") {
		return "Angular"
	}

	// Проверяем на React
	if p.hasFileInPath(metadata, appPath, "*.jsx") || p.hasFileInPath(metadata, appPath, "*.tsx") {
		return "React"
	}

	// Проверяем на Node.js
	if p.hasFileInPath(metadata, appPath, "main.ts") || p.hasFileInPath(metadata, appPath, "main.js") {
		return "Node.js"
	}

	return "TypeScript"
}

func (p *Plugin) detectLibType(metadata *model.ProjectMetadata, libPath string) string {
	// Проверяем на UI библиотеку
	if p.hasFileInPath(metadata, libPath, "*.component.ts") {
		return "UI"
	}

	// Проверяем на утилитную библиотеку
	if strings.Contains(libPath, "util") || strings.Contains(libPath, "shared") {
		return "Utility"
	}

	// Проверяем на data библиотеку
	if strings.Contains(libPath, "data") || p.hasFileInPath(metadata, libPath, "*.service.ts") {
		return "Data"
	}

	return "Feature"
}

func (p *Plugin) hasFileInPath(metadata *model.ProjectMetadata, basePath, pattern string) bool {
	for path := range metadata.Files {
		if strings.HasPrefix(path, basePath+"/") {
			fileName := filepath.Base(path)
			if matched, _ := filepath.Match(pattern, fileName); matched {
				return true
			}
		}
	}
	return false
}

// Register регистрирует плагин в системе
func Register() plugin.Plugin {
	return New()
}