// Package javascript contains the JavaScript/TypeScript language plugin
package javascript

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/plugin"
	"github.com/kolkov/gops/pkg/logger"
)

// Plugin - плагин для работы с JavaScript/TypeScript кодом
type Plugin struct {
	name        string
	version     string
	description string
	logger      *logger.Logger
	mu          sync.RWMutex
	
	// Регулярные выражения для парсинга
	functionRegex   *regexp.Regexp
	classRegex      *regexp.Regexp
	interfaceRegex  *regexp.Regexp
	importRegex     *regexp.Regexp
	exportRegex     *regexp.Regexp
	commentRegex    *regexp.Regexp
	todoRegex       *regexp.Regexp
}

// New создает новый экземпляр плагина
func New() *Plugin {
	return &Plugin{
		name:        "javascript",
		version:     "1.0.0",
		description: "JavaScript/TypeScript language support plugin",
		
		// Компилируем регулярные выражения один раз
		functionRegex:   regexp.MustCompile(`(?m)^[\s]*(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(`),
		classRegex:      regexp.MustCompile(`(?m)^[\s]*(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`),
		interfaceRegex:  regexp.MustCompile(`(?m)^[\s]*(?:export\s+)?interface\s+(\w+)`),
		importRegex:     regexp.MustCompile(`(?m)^[\s]*import\s+.*?from\s+['"](.*?)['"]`),
		exportRegex:     regexp.MustCompile(`(?m)^[\s]*export\s+(?:default\s+)?(?:const\s+|let\s+|var\s+|function\s+|class\s+|interface\s+)?(\w+)`),
		commentRegex:    regexp.MustCompile(`(?m)^\s*//(.*)$|/\*[\s\S]*?\*/`),
		todoRegex:       regexp.MustCompile(`(?i)(?://|/\*|\*).*?(TODO|FIXME|HACK|XXX).*?(?:\*/|$)`),
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
	return plugin.PluginTypeLanguage
}

func (p *Plugin) Priority() int {
	return 10
}

func (p *Plugin) Dependencies() []string {
	return []string{}
}

// Lifecycle methods

func (p *Plugin) OnLoad(ctx context.Context, logger *logger.Logger) error {
	p.logger = logger
	p.logger.Info("JavaScript/TypeScript language plugin loaded")
	return nil
}

func (p *Plugin) OnEnable(ctx context.Context) error {
	p.logger.Info("JavaScript/TypeScript language plugin enabled")
	return nil
}

func (p *Plugin) OnDisable(ctx context.Context) error {
	p.logger.Info("JavaScript/TypeScript language plugin disabled")
	return nil
}

func (p *Plugin) OnUnload(ctx context.Context) error {
	p.logger.Info("JavaScript/TypeScript language plugin unloaded")
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
		Tags:         []string{"language", "javascript", "typescript", "js", "ts"},
		Capabilities: []string{"parse", "analyze", "extract"},
	}
}

// LanguagePlugin interface methods

func (p *Plugin) GetLanguage() string {
	return "javascript"
}

func (p *Plugin) GetFileExtensions() []string {
	return []string{".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"}
}

func (p *Plugin) CanHandle(filepath string) bool {
	ext := strings.ToLower(filepath)
	for _, validExt := range p.GetFileExtensions() {
		if strings.HasSuffix(ext, validExt) {
			return true
		}
	}
	return false
}

func (p *Plugin) ExtractMetadata(ctx context.Context, file *model.ProjectFile) (*model.FileMetadata, error) {
	if !p.CanHandle(file.Path) {
		return nil, fmt.Errorf("cannot handle file: %s", file.Path)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	language := p.detectLanguage(file.Path, file.Content)

	metadata := &model.FileMetadata{
		Language:    language,
		LOC:         p.countLines(file.Content),
		SLOC:        p.countSourceLines(file.Content),
		Complexity:  p.calculateComplexity(file.Content),
		Functions:   p.extractFunctions(file.Content),
		Classes:     p.extractClasses(file.Content),
		Imports:     p.extractImports(file.Content),
		Exports:     p.extractExports(file.Content),
		Comments:    p.extractComments(file.Content),
		TODOs:       p.extractTODOs(file.Content),
		PackageName: p.extractPackageName(file.Path),
	}

	return metadata, nil
}

func (p *Plugin) ParseFile(ctx context.Context, filePath, content string) (*model.ParsedFile, error) {
	if !p.CanHandle(filePath) {
		return nil, fmt.Errorf("cannot handle file: %s", filePath)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	language := p.detectLanguage(filePath, content)

	parsedFile := &model.ParsedFile{
		Path:        filePath,
		Language:    language,
		PackageName: p.extractPackageName(filePath),
		Functions:   p.extractFunctions(content),
		Types:       p.extractClasses(content),
		Imports:     p.extractImports(content),
		Exports:     p.extractExports(content),
		Comments:    p.extractComments(content),
	}

	return parsedFile, nil
}

func (p *Plugin) ExtractDependencies(ctx context.Context, file *model.ProjectFile) ([]string, error) {
	if !p.CanHandle(file.Path) {
		return nil, fmt.Errorf("cannot handle file: %s", file.Path)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	dependencies := make([]string, 0)
	imports := p.extractImports(file.Content)

	for _, imp := range imports {
		dependencies = append(dependencies, imp.Path)
	}

	return dependencies, nil
}

func (p *Plugin) ValidateCode(ctx context.Context, content string) []model.ValidationError {
	errors := make([]model.ValidationError, 0)

	// Простые проверки синтаксиса
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Проверка на незакрытые скобки (упрощенно)
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")
		if openBraces != closeBraces && !strings.HasSuffix(line, "{") && !strings.HasPrefix(line, "}") {
			errors = append(errors, model.ValidationError{
				Message: "Possible unmatched braces",
				Line:    i + 1,
				Column:  0,
				Type:    "syntax",
			})
		}
	}

	return errors
}

// Private helper methods

func (p *Plugin) detectLanguage(filePath, content string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".ts", ".tsx":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	default:
		// Определяем по содержимому
		if strings.Contains(content, "interface ") || 
		   strings.Contains(content, ": string") ||
		   strings.Contains(content, ": number") {
			return "typescript"
		}
		if strings.Contains(content, "React.") || 
		   strings.Contains(content, "jsx") ||
		   strings.Contains(content, "<div") {
			return "jsx"
		}
		return "javascript"
	}
}

func (p *Plugin) countLines(content string) int {
	if content == "" {
		return 0
	}
	return len(strings.Split(content, "\n"))
}

func (p *Plugin) countSourceLines(content string) int {
	if content == "" {
		return 0
	}

	lines := strings.Split(content, "\n")
	sloc := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Пропускаем пустые строки и комментарии
		if trimmed != "" && 
		   !strings.HasPrefix(trimmed, "//") && 
		   !strings.HasPrefix(trimmed, "/*") &&
		   !strings.HasPrefix(trimmed, "*") {
			sloc++
		}
	}

	return sloc
}

func (p *Plugin) calculateComplexity(content string) int {
	complexity := 1 // базовая сложность

	// Подсчитываем ключевые слова, увеличивающие сложность
	complexityKeywords := []string{
		"if", "else", "while", "for", "switch", "case", "catch", "&&", "||"
	}

	for _, keyword := range complexityKeywords {
		complexity += strings.Count(content, keyword)
	}

	return complexity
}

func (p *Plugin) extractFunctions(content string) []model.FunctionInfo {
	functions := make([]model.FunctionInfo, 0)
	matches := p.functionRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			function := model.FunctionInfo{
				Name:     match[1],
				Line:     p.getLineNumber(content, match[0]),
				IsPublic: p.isExported(content, match[1]),
			}

			// Простая оценка сложности функции
			funcStart := strings.Index(content, match[0])
			if funcStart != -1 {
				// Находим конец функции (упрощенно)
				funcContent := content[funcStart:]
				braceCount := 0
				funcEnd := 0
				
				for i, char := range funcContent {
					if char == '{' {
						braceCount++
					} else if char == '}' {
						braceCount--
						if braceCount == 0 {
							funcEnd = i
							break
						}
					}
				}
				
				if funcEnd > 0 {
					function.Complexity = p.calculateComplexity(funcContent[:funcEnd])
				}
			}

			functions = append(functions, function)
		}
	}

	return functions
}

func (p *Plugin) extractClasses(content string) []model.ClassInfo {
	classes := make([]model.ClassInfo, 0)

	// Классы
	matches := p.classRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			class := model.ClassInfo{
				Name:     match[1],
				Line:     p.getLineNumber(content, match[0]),
				IsPublic: p.isExported(content, match[1]),
				Type:     "class",
			}
			classes = append(classes, class)
		}
	}

	// Интерфейсы (для TypeScript)
	matches = p.interfaceRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			interfaceInfo := model.ClassInfo{
				Name:      match[1],
				StartLine: p.getLineNumber(content, match[0]),
				IsPublic:  p.isExported(content, match[1]),
			}
			classes = append(classes, interfaceInfo)
		}
	}

	return classes
}

func (p *Plugin) extractImports(content string) []model.ImportInfo {
	imports := make([]model.ImportInfo, 0)
	matches := p.importRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			importInfo := model.ImportInfo{
				Path: match[1],
				Line: p.getLineNumber(content, match[0]),
			}
			imports = append(imports, importInfo)
		}
	}

	return imports
}

func (p *Plugin) extractExports(content string) []string {
	exports := make([]string, 0)
	exportedNames := make(map[string]bool)

	matches := p.exportRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			name := match[1]
			if !exportedNames[name] {
				exports = append(exports, name)
				exportedNames[name] = true
			}
		}
	}

	return exports
}

func (p *Plugin) extractComments(content string) []model.CommentInfo {
	comments := make([]model.CommentInfo, 0)
	matches := p.commentRegex.FindAllString(content, -1)

	for _, match := range matches {
		commentType := "line"
		text := strings.TrimSpace(match)
		
		if strings.HasPrefix(text, "/*") {
			commentType = "block"
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimSuffix(text, "*/")
		} else {
			text = strings.TrimPrefix(text, "//")
		}

		comment := model.CommentInfo{
			Text: strings.TrimSpace(text),
			Line: p.getLineNumber(content, match),
			Type: commentType,
		}
		comments = append(comments, comment)
	}

	return comments
}

func (p *Plugin) extractTODOs(content string) []model.TODOInfo {
	todos := make([]model.TODOInfo, 0)
	matches := p.todoRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			todo := model.TODOInfo{
				Text: strings.TrimSpace(match[0]),
				Line: p.getLineNumber(content, match[0]),
				Type: strings.ToLower(match[1]),
			}
			todos = append(todos, todo)
		}
	}

	return todos
}

func (p *Plugin) extractPackageName(filePath string) string {
	// Для JS/TS извлекаем имя из пути
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "" {
		return filepath.Base(filepath.Dir(filePath))
	}
	return filepath.Base(dir)
}

func (p *Plugin) getLineNumber(content, substring string) int {
	index := strings.Index(content, substring)
	if index == -1 {
		return 1
	}
	
	return strings.Count(content[:index], "\n") + 1
}

func (p *Plugin) isExported(content, name string) bool {
	// Проверяем, есть ли export для этого имени
	exportPattern := fmt.Sprintf(`export.*%s`, name)
	matched, _ := regexp.MatchString(exportPattern, content)
	return matched
}

// Register регистрирует плагин в системе
func Register() plugin.Plugin {
	return New()
}