// Package go contains the Go language plugin
package go

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/plugin"
	"github.com/kolkov/gops/pkg/logger"
)

// Plugin - плагин для работы с Go кодом
type Plugin struct {
	name        string
	version     string
	description string
	logger      *logger.Logger
	mu          sync.RWMutex
	fileSet     *token.FileSet
}

// New создает новый экземпляр плагина
func New() *Plugin {
	return &Plugin{
		name:        "go",
		version:     "1.0.0",
		description: "Go language support plugin",
		fileSet:     token.NewFileSet(),
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
	p.logger.Info("Go language plugin loaded")
	return nil
}

func (p *Plugin) OnEnable(ctx context.Context) error {
	p.logger.Info("Go language plugin enabled")
	return nil
}

func (p *Plugin) OnDisable(ctx context.Context) error {
	p.logger.Info("Go language plugin disabled")
	return nil
}

func (p *Plugin) OnUnload(ctx context.Context) error {
	p.logger.Info("Go language plugin unloaded")
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
		Tags:         []string{"language", "go", "ast"},
		Capabilities: []string{"parse", "analyze", "extract"},
	}
}

// LanguagePlugin interface methods

func (p *Plugin) GetLanguage() string {
	return "go"
}

func (p *Plugin) GetFileExtensions() []string {
	return []string{".go"}
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

	// Парсим Go файл
	parsed, err := parser.ParseFile(p.fileSet, file.Path, file.Content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", file.Path, err)
	}

	metadata := &model.FileMetadata{
		Language:    p.GetLanguage(),
		LOC:         p.countLines(file.Content),
		SLOC:        p.countSourceLines(file.Content),
		Complexity:  p.calculateComplexity(parsed),
		Functions:   p.extractFunctions(parsed),
		Classes:     p.extractStructs(parsed),
		Imports:     p.extractImports(parsed),
		Exports:     p.extractExports(parsed),
		Comments:    p.extractComments(parsed),
		TODOs:       p.extractTODOs(file.Content),
		PackageName: parsed.Name.Name,
	}

	return metadata, nil
}

func (p *Plugin) ParseFile(ctx context.Context, filePath, content string) (*model.ParsedFile, error) {
	if !p.CanHandle(filePath) {
		return nil, fmt.Errorf("cannot handle file: %s", filePath)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Парсим Go файл
	parsed, err := parser.ParseFile(p.fileSet, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", filePath, err)
	}

	parsedFile := &model.ParsedFile{
		Path:        filePath,
		Language:    p.GetLanguage(),
		AST:         parsed,
		PackageName: parsed.Name.Name,
		Functions:   p.extractFunctions(parsed),
		Types:       p.extractStructs(parsed),
		Imports:     p.extractImports(parsed),
		Exports:     p.extractExports(parsed),
		Comments:    p.extractComments(parsed),
	}

	return parsedFile, nil
}

func (p *Plugin) ExtractDependencies(ctx context.Context, file *model.ProjectFile) ([]string, error) {
	if !p.CanHandle(file.Path) {
		return nil, fmt.Errorf("cannot handle file: %s", file.Path)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	parsed, err := parser.ParseFile(p.fileSet, file.Path, file.Content, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", file.Path, err)
	}

	dependencies := make([]string, 0)
	for _, imp := range parsed.Imports {
		if imp.Path != nil {
			// Удаляем кавычки
			path := strings.Trim(imp.Path.Value, "\"")
			dependencies = append(dependencies, path)
		}
	}

	return dependencies, nil
}

func (p *Plugin) ValidateCode(ctx context.Context, content string) []model.ValidationError {
	errors := make([]model.ValidationError, 0)

	// Простая валидация - пытаемся распарсить
	_, err := parser.ParseFile(p.fileSet, "", content, 0)
	if err != nil {
		errors = append(errors, model.ValidationError{
			Message: err.Error(),
			Line:    0,
			Column:  0,
			Type:    "syntax",
		})
	}

	return errors
}

// Private helper methods

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
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			sloc++
		}
	}

	return sloc
}

func (p *Plugin) calculateComplexity(file *ast.File) int {
	complexity := 1 // базовая сложность

	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.RangeStmt, *ast.ForStmt, *ast.TypeSwitchStmt, *ast.SwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

func (p *Plugin) extractFunctions(file *ast.File) []model.FunctionInfo {
	functions := make([]model.FunctionInfo, 0)

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			function := model.FunctionInfo{
				Name: fn.Name.Name,
				Line: p.fileSet.Position(fn.Pos()).Line,
			}

			// Извлекаем параметры
			if fn.Type.Params != nil {
				for _, param := range fn.Type.Params.List {
					for _, name := range param.Names {
						function.Parameters = append(function.Parameters, name.Name)
					}
				}
			}

			// Проверяем, является ли функция публичной
			function.IsPublic = ast.IsExported(fn.Name.Name)

			// Вычисляем сложность функции
			complexity := 1
			ast.Inspect(fn, func(node ast.Node) bool {
				switch node.(type) {
				case *ast.IfStmt, *ast.RangeStmt, *ast.ForStmt, *ast.TypeSwitchStmt, *ast.SwitchStmt:
					complexity++
				}
				return true
			})
			function.Complexity = complexity

			functions = append(functions, function)
		}
		return true
	})

	return functions
}

func (p *Plugin) extractStructs(file *ast.File) []model.ClassInfo {
	structs := make([]model.ClassInfo, 0)

	ast.Inspect(file, func(n ast.Node) bool {
		if gen, ok := n.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
			for _, spec := range gen.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if _, ok := ts.Type.(*ast.StructType); ok {
						class := model.ClassInfo{
							Name:     ts.Name.Name,
							Line:     p.fileSet.Position(ts.Pos()).Line,
							IsPublic: ast.IsExported(ts.Name.Name),
							Type:     "struct",
						}
						structs = append(structs, class)
					} else if _, ok := ts.Type.(*ast.InterfaceType); ok {
						class := model.ClassInfo{
							Name:     ts.Name.Name,
							Line:     p.fileSet.Position(ts.Pos()).Line,
							IsPublic: ast.IsExported(ts.Name.Name),
							Type:     "interface",
						}
						structs = append(structs, class)
					}
				}
			}
		}
		return true
	})

	return structs
}

func (p *Plugin) extractImports(file *ast.File) []model.ImportInfo {
	imports := make([]model.ImportInfo, 0)

	for _, imp := range file.Imports {
		importInfo := model.ImportInfo{
			Path: strings.Trim(imp.Path.Value, "\""),
			Line: p.fileSet.Position(imp.Pos()).Line,
		}

		if imp.Name != nil {
			importInfo.Alias = imp.Name.Name
		}

		imports = append(imports, importInfo)
	}

	return imports
}

func (p *Plugin) extractExports(file *ast.File) []string {
	exports := make([]string, 0)
	exportedNames := make(map[string]bool)

	// Функции
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.IsExported() {
				name := fn.Name.Name
				if !exportedNames[name] {
					exports = append(exports, name)
					exportedNames[name] = true
				}
			}
		}
	}

	// Типы, константы, переменные
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						name := s.Name.Name
						if !exportedNames[name] {
							exports = append(exports, name)
							exportedNames[name] = true
						}
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							n := name.Name
							if !exportedNames[n] {
								exports = append(exports, n)
								exportedNames[n] = true
							}
						}
					}
				}
			}
		}
		return true
	})

	return exports
}

func (p *Plugin) extractComments(file *ast.File) []model.CommentInfo {
	comments := make([]model.CommentInfo, 0)

	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			commentInfo := model.CommentInfo{
				Text: strings.TrimPrefix(strings.TrimPrefix(comment.Text, "//"), "/*"),
				Line: p.fileSet.Position(comment.Pos()).Line,
			}

			if strings.HasPrefix(comment.Text, "//") {
				commentInfo.Type = "line"
			} else {
				commentInfo.Type = "block"
			}

			comments = append(comments, commentInfo)
		}
	}

	return comments
}

func (p *Plugin) extractTODOs(content string) []model.TODOInfo {
	todos := make([]model.TODOInfo, 0)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Ищем TODO, FIXME, HACK, XXX в комментариях
		if strings.Contains(trimmed, "//") {
			comment := strings.TrimSpace(strings.SplitN(trimmed, "//", 2)[1])
			for _, marker := range []string{"TODO", "FIXME", "HACK", "XXX"} {
				if strings.Contains(strings.ToUpper(comment), marker) {
					todos = append(todos, model.TODOInfo{
						Text:   comment,
						Line:   i + 1,
						Type:   strings.ToLower(marker),
						Author: "", // Можно попытаться извлечь из комментария
					})
					break
				}
			}
		}
	}

	return todos
}

// Register регистрирует плагин в системе
func Register() plugin.Plugin {
	return New()
}