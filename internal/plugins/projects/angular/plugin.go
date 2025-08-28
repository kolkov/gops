// Package angular contains the Angular project type plugin
package angular

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

// Plugin - плагин для работы с Angular проектами
type Plugin struct {
	name        string
	version     string
	description string
	logger      *logger.Logger
	mu          sync.RWMutex
}

// AngularConfig структура для angular.json
type AngularConfig struct {
	Projects map[string]AngularProject `json:"projects"`
	Version  int                       `json:"version"`
}

type AngularProject struct {
	ProjectType string `json:"projectType"`
	Root        string `json:"root"`
	SourceRoot  string `json:"sourceRoot"`
	Architect   map[string]interface{} `json:"architect"`
}

// New создает новый экземпляр плагина
func New() *Plugin {
	return &Plugin{
		name:        "angular",
		version:     "1.0.0",
		description: "Angular project type detection and analysis plugin",
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
	return 15 // Higher priority than generic
}

func (p *Plugin) Dependencies() []string {
	return []string{"javascript"} // Зависим от JS плагина
}

// Lifecycle methods

func (p *Plugin) OnLoad(ctx context.Context, logger *logger.Logger) error {
	p.logger = logger
	p.logger.Info("Angular project plugin loaded")
	return nil
}

func (p *Plugin) OnEnable(ctx context.Context) error {
	p.logger.Info("Angular project plugin enabled")
	return nil
}

func (p *Plugin) OnDisable(ctx context.Context) error {
	p.logger.Info("Angular project plugin disabled")
	return nil
}

func (p *Plugin) OnUnload(ctx context.Context) error {
	p.logger.Info("Angular project plugin unloaded")
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
		Tags:         []string{"project", "angular", "typescript", "frontend"},
		Capabilities: []string{"detect", "analyze", "structure"},
	}
}

// ProjectTypePlugin interface methods

func (p *Plugin) Detect(metadata *model.ProjectMetadata) (float32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	confidence := float32(0.0)

	// Проверяем angular.json - основной маркер Angular проекта
	if _, exists := metadata.Files["angular.json"]; exists {
		confidence += 0.5
	}

	// Проверяем package.json на Angular зависимости
	if packageFile, exists := metadata.Files["package.json"]; exists {
		if p.hasAngularDependencies(packageFile.Path) {
			confidence += 0.3
		}
	}

	// Проверяем структуру Angular проекта
	angularDirs := []string{"src/app", "src/assets", "src/environments"}
	foundDirs := 0
	for _, dir := range angularDirs {
		if p.hasDirectoryInFiles(metadata.Files, dir) {
			foundDirs++
		}
	}
	if foundDirs >= 2 {
		confidence += 0.2
	}

	// Проверяем наличие Angular файлов
	for path := range metadata.Files {
		if strings.HasSuffix(path, ".component.ts") ||
		   strings.HasSuffix(path, ".service.ts") ||
		   strings.HasSuffix(path, ".module.ts") ||
		   strings.HasSuffix(path, ".component.html") {
			confidence += 0.05
			if confidence >= 1.0 {
				break
			}
		}
	}

	p.logger.Debug("Angular detection", "confidence", confidence)

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
		Type:     "Angular",
		Language: "TypeScript",
		Version:  p.detectAngularVersion(metadata),
		Components: make([]model.Component, 0),
		Modules:    make([]model.Module, 0),
	}

	// Анализируем Angular структуру
	p.analyzeAngularComponents(metadata, info)
	p.analyzeAngularModules(metadata, info)
	p.analyzeAngularServices(metadata, info)

	return info, nil
}

func (p *Plugin) GetLoadingRules() []model.LoadingRule {
	return []model.LoadingRule{
		{
			Pattern:     "**/*.component.html",
			ContentMode: model.ContentModeHeaders,
			Priority:    10,
		},
		{
			Pattern:     "**/*.component.css",
			ContentMode: model.ContentModeNone,
			Priority:    5,
		},
		{
			Pattern:     "**/*.component.scss",
			ContentMode: model.ContentModeHeaders,
			Priority:    8,
		},
		{
			Pattern:     "**/*.component.ts",
			ContentMode: model.ContentModeFull,
			Priority:    20,
		},
		{
			Pattern:     "**/*.service.ts",
			ContentMode: model.ContentModeFull,
			Priority:    15,
		},
		{
			Pattern:     "**/*.module.ts",
			ContentMode: model.ContentModeFull,
			Priority:    20,
		},
		{
			Pattern:     "angular.json",
			ContentMode: model.ContentModeFull,
			Priority:    25,
		},
		{
			Pattern:     "package.json",
			ContentMode: model.ContentModeFull,
			Priority:    20,
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
			Pattern: ".angular/**",
			Reason:  "Angular cache",
		},
		{
			Pattern: "coverage/**",
			Reason:  "Test coverage",
		},
		{
			Pattern: "**/*.spec.ts",
			Reason:  "Test files",
		},
		{
			Pattern: "**/*.e2e-spec.ts",
			Reason:  "E2E test files",
		},
	}
}

// Private helper methods

func (p *Plugin) hasAngularDependencies(packagePath string) bool {
	content, err := os.ReadFile(packagePath)
	if err != nil {
		return false
	}

	var packageJson map[string]interface{}
	if err := json.Unmarshal(content, &packageJson); err != nil {
		return false
	}

	// Проверяем dependencies
	if deps, ok := packageJson["dependencies"].(map[string]interface{}); ok {
		if _, hasCore := deps["@angular/core"]; hasCore {
			return true
		}
	}

	// Проверяем devDependencies
	if devDeps, ok := packageJson["devDependencies"].(map[string]interface{}); ok {
		if _, hasCore := devDeps["@angular/core"]; hasCore {
			return true
		}
		if _, hasCLI := devDeps["@angular/cli"]; hasCLI {
			return true
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

func (p *Plugin) detectAngularVersion(metadata *model.ProjectMetadata) string {
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

	// Проверяем dependencies
	if deps, ok := packageJson["dependencies"].(map[string]interface{}); ok {
		if coreVersion, hasCore := deps["@angular/core"].(string); hasCore {
			return strings.TrimPrefix(coreVersion, "^")
		}
	}

	return "unknown"
}

func (p *Plugin) analyzeAngularComponents(metadata *model.ProjectMetadata, info *model.ProjectInfo) {
	componentsFound := make(map[string]bool)

	for path := range metadata.Files {
		if strings.HasSuffix(path, ".component.ts") {
			componentName := p.extractComponentName(path)
			if componentName != "" && !componentsFound[componentName] {
				componentsFound[componentName] = true
				
				component := model.Component{
					Name:        componentName,
					Type:        "angular-component",
					Path:        filepath.Dir(path),
					Description: "Angular component",
					Important:   p.isImportantComponent(componentName),
				}
				
				info.Components = append(info.Components, component)
			}
		}
	}
}

func (p *Plugin) analyzeAngularModules(metadata *model.ProjectMetadata, info *model.ProjectInfo) {
	for path := range metadata.Files {
		if strings.HasSuffix(path, ".module.ts") {
			moduleName := p.extractModuleName(path)
			if moduleName != "" {
				module := model.Module{
					Name:        moduleName,
					Path:        path,
					Type:        "angular-module",
					Public:      !strings.Contains(path, "src/app/"),
					Description: "Angular module",
				}
				
				info.Modules = append(info.Modules, module)
			}
		}
	}
}

func (p *Plugin) analyzeAngularServices(metadata *model.ProjectMetadata, info *model.ProjectInfo) {
	for path := range metadata.Files {
		if strings.HasSuffix(path, ".service.ts") {
			serviceName := p.extractServiceName(path)
			if serviceName != "" {
				service := model.Component{
					Name:        serviceName,
					Type:        "angular-service", 
					Path:        filepath.Dir(path),
					Description: "Angular service",
					Important:   true, // Services are usually important
				}
				
				info.Components = append(info.Components, service)
			}
		}
	}
}

func (p *Plugin) extractComponentName(path string) string {
	fileName := filepath.Base(path)
	// app.component.ts -> AppComponent
	name := strings.TrimSuffix(fileName, ".component.ts")
	return p.toPascalCase(name)
}

func (p *Plugin) extractModuleName(path string) string {
	fileName := filepath.Base(path)
	// app.module.ts -> AppModule
	name := strings.TrimSuffix(fileName, ".module.ts")
	return p.toPascalCase(name)
}

func (p *Plugin) extractServiceName(path string) string {
	fileName := filepath.Base(path)
	// user.service.ts -> UserService
	name := strings.TrimSuffix(fileName, ".service.ts")
	return p.toPascalCase(name)
}

func (p *Plugin) toPascalCase(input string) string {
	parts := strings.Split(input, "-")
	result := ""
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return result
}

func (p *Plugin) isImportantComponent(componentName string) bool {
	importantComponents := []string{
		"AppComponent",
		"HomeComponent", 
		"MainComponent",
		"HeaderComponent",
		"FooterComponent",
		"NavigationComponent",
	}
	
	for _, important := range importantComponents {
		if componentName == important {
			return true
		}
	}
	
	return false
}

// Register регистрирует плагин в системе
func Register() plugin.Plugin {
	return New()
}