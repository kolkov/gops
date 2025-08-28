// Package model содержит модели данных для системы документации
package model

import (
	"time"
)

// ContentMode определяет режим загрузки контента файла
type ContentMode int

const (
	// ContentModeNone - не загружать контент
	ContentModeNone ContentMode = iota
	// ContentModeHeaders - загружать только заголовки/сигнатуры
	ContentModeHeaders
	// ContentModeFull - загружать полный контент
	ContentModeFull
	// ContentModeCompressed - загружать сжатый контент
	ContentModeCompressed
	// ContentModeReference - только ссылка на файл
	ContentModeReference
)

// OutputFormat определяет формат вывода документации
type OutputFormat string

const (
	OutputFormatMarkdown OutputFormat = "markdown"
	OutputFormatHTML     OutputFormat = "html"
	OutputFormatJSON     OutputFormat = "json"
	OutputFormatXML      OutputFormat = "xml"
	OutputFormatPDF      OutputFormat = "pdf"
	OutputFormatCustom   OutputFormat = "custom"
)

// ProcessingStage определяет этап обработки
type ProcessingStage string

const (
	StagePreScan      ProcessingStage = "pre_scan"
	StageScan         ProcessingStage = "scan"
	StagePostScan     ProcessingStage = "post_scan"
	StagePreAnalyze   ProcessingStage = "pre_analyze"
	StageAnalyze      ProcessingStage = "analyze"
	StagePostAnalyze  ProcessingStage = "post_analyze"
	StagePreGenerate  ProcessingStage = "pre_generate"
	StageGenerate     ProcessingStage = "generate"
	StagePostGenerate ProcessingStage = "post_generate"
)

// FileMetadata - метаданные файла без содержимого
type FileMetadata struct {
	// Базовая информация
	Path      string `json:"path"`
	Name      string `json:"name"`
	Dir       string `json:"dir"`
	Extension string `json:"extension"`
	Size      int64  `json:"size"`
	IsDir     bool   `json:"is_dir"`
	IsSymlink bool   `json:"is_symlink"`
	IsHidden  bool   `json:"is_hidden"`

	// Временные метки
	ModTime    time.Time `json:"mod_time"`
	CreateTime time.Time `json:"create_time,omitempty"`
	AccessTime time.Time `json:"access_time,omitempty"`

	// Права доступа
	Permissions uint32 `json:"permissions"`
	Owner       string `json:"owner,omitempty"`
	Group       string `json:"group,omitempty"`

	// Связи
	SymlinkTarget string          `json:"symlink_target,omitempty"`
	Children      []*FileMetadata `json:"children,omitempty"`
	Parent        *FileMetadata   `json:"-"`

	// Дополнительные атрибуты
	Attributes map[string]interface{} `json:"attributes,omitempty"`

	// Анализ кода - добавляем для совместимости с плагинами
	Language   string   `json:"language,omitempty"`
	LOC        int      `json:"loc,omitempty"`        // Lines of Code
	SLOC       int      `json:"sloc,omitempty"`       // Source Lines of Code  
	Complexity int      `json:"complexity,omitempty"`
	Functions  int      `json:"functions,omitempty"`
	Classes    int      `json:"classes,omitempty"`
	Imports    []string `json:"imports,omitempty"`
	Exports    []string `json:"exports,omitempty"`
	Comments   int      `json:"comments,omitempty"`

	// Статус обработки
	Processed   bool        `json:"processed"`
	Selected    bool        `json:"selected"`
	LoadingMode ContentMode `json:"loading_mode"`
	SkipReason  string      `json:"skip_reason,omitempty"`
}

// ProjectMetadata - метаданные всего проекта
type ProjectMetadata struct {
	// Основная информация
	RootPath    string `json:"root_path"`
	RootDir     string `json:"root_dir"`     // Для совместимости
	Name        string `json:"name"`
	ProjectName string `json:"project_name"` // Для совместимости
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Version     string `json:"version,omitempty"`

	// Структура
	Root  *FileMetadata            `json:"root"`
	Files map[string]*FileMetadata `json:"-"` // Быстрый доступ по пути

	// Статистика
	TotalFiles int            `json:"total_files"`
	TotalDirs  int            `json:"total_dirs"`
	TotalSize  int64          `json:"total_size"`
	FilesByExt map[string]int `json:"files_by_ext"`
	FilesByDir map[string]int `json:"files_by_dir"`

	// Временные метки
	ScanTime     time.Time     `json:"scan_time"`
	ScanDuration time.Duration `json:"scan_duration"`

	// Дополнительные данные
	VCSInfo      *VCSInfo     `json:"vcs_info,omitempty"`
	BuildInfo    *BuildInfo   `json:"build_info,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

// VCSInfo - информация о системе контроля версий
type VCSInfo struct {
	Type         string    `json:"type"` // git, svn, hg и т.д.
	RemoteURL    string    `json:"remote_url,omitempty"`
	Branch       string    `json:"branch,omitempty"`
	Commit       string    `json:"commit,omitempty"`
	Tag          string    `json:"tag,omitempty"`
	IsDirty      bool      `json:"is_dirty"`
	LastCommit   time.Time `json:"last_commit,omitempty"`
	Contributors []string  `json:"contributors,omitempty"`
}

// BuildInfo - информация о сборке проекта
type BuildInfo struct {
	System        string                 `json:"system"` // maven, gradle, make, npm и т.д.
	Targets       []string               `json:"targets,omitempty"`
	Scripts       []string               `json:"scripts,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

// Dependency - зависимость проекта
type Dependency struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Type     string `json:"type"` // compile, runtime, test и т.д.
	Scope    string `json:"scope,omitempty"`
	Optional bool   `json:"optional"`
}

// ProjectInfo - детальная информация о проекте после анализа
type ProjectInfo struct {
	Type        string   `json:"type"`
	Version     string   `json:"version,omitempty"` // Для совместимости с плагинами
	Framework   string   `json:"framework,omitempty"`
	Language    string   `json:"language"`
	Languages   []string `json:"languages"` // Для многоязычных проектов
	Description string   `json:"description,omitempty"` // Добавляем для плагинов

	// Компоненты проекта
	Components []Component `json:"components"`
	Modules    []Module    `json:"modules"`
	Packages   []Package   `json:"packages"`

	// Структурная информация
	EntryPoints []string `json:"entry_points,omitempty"`
	ConfigFiles []string `json:"config_files,omitempty"`
	TestDirs    []string `json:"test_dirs,omitempty"`
	DocsDirs    []string `json:"docs_dirs,omitempty"`

	// Метрики
	Metrics *ProjectMetrics `json:"metrics,omitempty"`

	// Дополнительные данные
	CustomData map[string]interface{} `json:"custom_data,omitempty"`
}

// Component - компонент проекта (подсистема, модуль, сервис)
type Component struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Path         string   `json:"path"`
	Description  string   `json:"description,omitempty"`
	Language     string   `json:"language,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Important    bool     `json:"important"`
	EntryPoint   string   `json:"entry_point,omitempty"`
	ConfigFiles  []string `json:"config_files,omitempty"`
}

// Module - модуль проекта
type Module struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Type         string   `json:"type"`
	Version      string   `json:"version,omitempty"`
	Description  string   `json:"description,omitempty"` // Добавляем для плагинов
	Dependencies []string `json:"dependencies,omitempty"`
	Exports      []string `json:"exports,omitempty"`
	Public       bool     `json:"public"`
}

// Package - пакет проекта
type Package struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	License     string `json:"license,omitempty"`
}

// LoadingRule - правило загрузки контента
type LoadingRule struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Priority    int         `json:"priority"`
	Pattern     string      `json:"pattern,omitempty"`
	Patterns    []string    `json:"patterns,omitempty"`
	Mode        ContentMode `json:"mode"`
	ContentMode ContentMode `json:"content_mode"` // Дублируем для совместимости
	MaxSize     int64       `json:"max_size,omitempty"`
	Condition   string      `json:"condition,omitempty"` // Выражение для оценки
}

// Matches проверяет, соответствует ли файл правилу
func (lr *LoadingRule) Matches(filepath string) bool {
	// Простая реализация - можно улучшить
	return true
}


// ParsedFile представляет распарсенный файл
type ParsedFile struct {
	Path         string          `json:"path"`
	Language     string          `json:"language"`
	Functions    []FunctionInfo  `json:"functions"`
	Classes      []ClassInfo     `json:"classes"`
	Imports      []string        `json:"imports"`
	Exports      []string        `json:"exports"`
	Comments     []CommentInfo   `json:"comments"`
	TODOs        []TODOInfo      `json:"todos"`
	Dependencies []string        `json:"dependencies"`
	Complexity   int             `json:"complexity"`
	LineCount    int             `json:"line_count"`
}

// ValidationError представляет ошибку валидации
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

// FunctionInfo содержит информацию о функции
type FunctionInfo struct {
	Name       string   `json:"name"`
	Signature  string   `json:"signature"`
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	Parameters []string `json:"parameters"`
	ReturnType string   `json:"return_type"`
	IsPublic   bool     `json:"is_public"`
	Comment    string   `json:"comment"`
	Complexity int      `json:"complexity"`
}

// ClassInfo содержит информацию о классе/структуре  
type ClassInfo struct {
	Name      string         `json:"name"`
	StartLine int            `json:"start_line"`
	EndLine   int            `json:"end_line"`
	Methods   []FunctionInfo `json:"methods"`
	Fields    []FieldInfo    `json:"fields"`
	IsPublic  bool           `json:"is_public"`
	Comment   string         `json:"comment"`
}

// FieldInfo содержит информацию о поле
type FieldInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	IsPublic bool   `json:"is_public"`
	Comment  string `json:"comment"`
}

// CommentInfo содержит информацию о комментарии
type CommentInfo struct {
	Text      string `json:"text"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Type      string `json:"type"` // single, multi, doc
}

// TODOInfo содержит информацию о TODO/FIXME
type TODOInfo struct {
	Text     string `json:"text"`
	Line     int    `json:"line"`
	Type     string `json:"type"` // TODO, FIXME, NOTE, etc.
	Priority string `json:"priority"`
}

// FileInfo содержит общую информацию о файле
type FileInfo struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	ModTime   string `json:"mod_time"`
	Language  string `json:"language"`
	Type      string `json:"type"`
	Encoding  string `json:"encoding"`
	LineCount int    `json:"line_count"`
}


// ExclusionRule - правило исключения файлов
type ExclusionRule struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority"`
	Pattern     string   `json:"pattern,omitempty"`
	Patterns    []string `json:"patterns,omitempty"`
	Reason      string   `json:"reason"`
	Override    bool     `json:"override"` // Может ли быть переопределено
}

// ExtractOptions - опции извлечения заголовков
type ExtractOptions struct {
	IncludePrivate   bool `json:"include_private"`
	IncludeComments  bool `json:"include_comments"`
	IncludeTests     bool `json:"include_tests"`
	IncludeGenerated bool `json:"include_generated"`
	MaxDepth         int  `json:"max_depth"`
	SimplifyTypes    bool `json:"simplify_types"`
}

// ComplexityMetrics - метрики сложности кода
type ComplexityMetrics struct {
	Cyclomatic      int     `json:"cyclomatic"`
	Cognitive       int     `json:"cognitive"`
	Halstead        float64 `json:"halstead"`
	Maintainability float64 `json:"maintainability"`
	LinesOfCode     int     `json:"lines_of_code"`
	CommentRatio    float64 `json:"comment_ratio"`
}

// ProjectMetrics - метрики проекта
type ProjectMetrics struct {
	TotalLOC          int     `json:"total_loc"`
	TotalSLOC         int     `json:"total_sloc"` // Source lines of code
	TotalComments     int     `json:"total_comments"`
	TotalFiles        int     `json:"total_files"`
	TotalFunctions    int     `json:"total_functions"`
	TotalClasses      int     `json:"total_classes"`
	TotalPackages     int     `json:"total_packages"`
	AverageComplexity float64 `json:"average_complexity"`
	TestCoverage      float64 `json:"test_coverage,omitempty"`
	TechDebt          float64 `json:"tech_debt,omitempty"`
}

// AnalysisResult - результат анализа проекта
type AnalysisResult struct {
	Timestamp    time.Time        `json:"timestamp"`
	Duration     time.Duration    `json:"duration"`
	Metrics      *ProjectMetrics  `json:"metrics"`
	Dependencies *DependencyGraph `json:"dependencies,omitempty"`
	Issues       []Issue          `json:"issues,omitempty"`
	Patterns     []Pattern        `json:"patterns,omitempty"`
	Suggestions  []string         `json:"suggestions,omitempty"`
}

// DependencyGraph - граф зависимостей
type DependencyGraph struct {
	Nodes  []DependencyNode `json:"nodes"`
	Edges  []DependencyEdge `json:"edges"`
	Cycles [][]string       `json:"cycles,omitempty"`
}

// DependencyNode - узел в графе зависимостей
type DependencyNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
	External bool   `json:"external"`
}

// DependencyEdge - ребро в графе зависимостей
type DependencyEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"` // import, include, require и т.д.
	Weight int    `json:"weight,omitempty"`
}

// Issue - проблема, найденная при анализе
type Issue struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"` // error, warning, info
	Message    string `json:"message"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Rule       string `json:"rule,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Pattern - паттерн проектирования
type Pattern struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Confidence  float32  `json:"confidence"`
	Files       []string `json:"files"`
	Description string   `json:"description,omitempty"`
}

// Command - команда CLI от плагина
type Command struct {
	Name        string
	Description string
	Usage       string
	Flags       []Flag
	Action      func(args []string) error
}

// Flag - флаг команды
type Flag struct {
	Name        string
	Short       string
	Description string
	Type        string
	Default     interface{}
	Required    bool
}

// ConfigSchema - схема конфигурации плагина
type ConfigSchema struct {
	Version    string              `json:"version"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property - свойство конфигурации
type Property struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
}
