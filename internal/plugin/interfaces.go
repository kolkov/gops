// Package plugin определяет базовые интерфейсы для системы плагинов
package plugin

import (
	"context"
	"github.com/kolkov/gops/internal/model"
)

// PluginType определяет тип плагина
type PluginType int

const (
	PluginTypeLanguage PluginType = iota
	PluginTypeProject
	PluginTypeFilter
	PluginTypeGenerator
)

// PluginStatus определяет статус плагина
type PluginStatus int

const (
	StatusInactive PluginStatus = iota
	StatusActive
	StatusError
	StatusDeprecated
)

// PluginCapability определяет возможности плагина
type PluginCapability string

const (
	CapabilityASTParsing   PluginCapability = "ast_parsing"
	CapabilitySyntaxCheck  PluginCapability = "syntax_check"
	CapabilityComplexity   PluginCapability = "complexity"
	CapabilityProjectDet   PluginCapability = "project_detection"
	CapabilityDependencies PluginCapability = "dependencies"
)

// Plugin - базовый интерфейс для всех плагинов
type Plugin interface {
	// Name возвращает имя плагина
	Name() string

	// Version возвращает версию плагина
	Version() string

	// Priority возвращает приоритет плагина (больше = выше приоритет)
	Priority() int

	// Init инициализирует плагин с конфигурацией
	Init(config map[string]interface{}) error
}

// LanguagePlugin - плагин для поддержки языка программирования
type LanguagePlugin interface {
	Plugin

	// SupportedExtensions возвращает список поддерживаемых расширений файлов
	SupportedExtensions() []string

	// DetectLanguage определяет язык файла и возвращает confidence score (0.0-1.0)
	DetectLanguage(filepath string, content []byte) (string, float32)

	// ExtractHeaders извлекает сигнатуры функций, типов и т.д. без тел функций
	ExtractHeaders(content []byte, options *model.ExtractOptions) (*model.FileHeader, error)

	// ParseAST парсит файл в AST (Abstract Syntax Tree)
	ParseAST(content []byte) (interface{}, error)

	// FormatHeader форматирует заголовки для вывода в документацию
	FormatHeader(header *model.FileHeader, format model.OutputFormat) (string, error)

	// GetComplexity вычисляет метрики сложности кода
	GetComplexity(content []byte) (*model.ComplexityMetrics, error)
	
	// ExtractMetadata извлекает метаданные файла
	ExtractMetadata(ctx context.Context, file *model.ProjectFile) (*model.FileMetadata, error)
	
	// GetLanguage определяет язык файла
	GetLanguage(filepath string, content []byte) string
}

// ProjectTypePlugin - плагин для определения и анализа типа проекта
type ProjectTypePlugin interface {
	Plugin

	// Detect определяет, соответствует ли проект этому типу
	// Возвращает confidence score (0.0-1.0)
	Detect(metadata *model.ProjectMetadata) (float32, error)

	// MarkerFiles возвращает список файлов-маркеров для этого типа проекта
	MarkerFiles() []string

	// MarkerPatterns возвращает паттерны путей, характерные для проекта
	MarkerPatterns() []string

	// AnalyzeStructure анализирует структуру проекта и возвращает детальную информацию
	AnalyzeStructure(ctx context.Context, metadata *model.ProjectMetadata) (*model.ProjectInfo, error)

	// GetLoadingRules возвращает правила загрузки контента для этого типа проекта
	GetLoadingRules() []model.LoadingRule

	// GetExclusionRules возвращает правила исключения файлов
	GetExclusionRules() []model.ExclusionRule

	// GenerateDocumentation генерирует специфичную для проекта документацию
	GenerateDocumentation(ctx context.Context, project *model.Project, writer model.DocWriter) error

	// GetProjectComponents возвращает список компонентов проекта (модули, пакеты и т.д.)
	GetProjectComponents(metadata *model.ProjectMetadata) ([]model.Component, error)
}

// FilterPlugin - плагин для фильтрации файлов
type FilterPlugin interface {
	Plugin

	// ShouldExclude определяет, должен ли файл быть исключен
	ShouldExclude(path string, metadata *model.FileMetadata) (bool, string)

	// ShouldInclude определяет, должен ли файл быть принудительно включен
	ShouldInclude(path string, metadata *model.FileMetadata) (bool, string)

	// GetLoadingMode определяет режим загрузки для файла
	GetLoadingMode(path string, metadata *model.FileMetadata) model.ContentMode

	// GetPriority возвращает приоритет правила (для разрешения конфликтов)
	GetFilterPriority() int
}

// ProcessorPlugin - плагин для обработки контента
type ProcessorPlugin interface {
	Plugin

	// CanProcess проверяет, может ли плагин обработать данный файл
	CanProcess(filepath string, metadata *model.FileMetadata) bool

	// Process обрабатывает контент файла
	Process(content []byte, metadata *model.FileMetadata) ([]byte, error)

	// PostProcess выполняет пост-обработку после загрузки всех файлов
	PostProcess(files []*model.ProjectFile) error
}

// FormatterPlugin - плагин для форматирования вывода
type FormatterPlugin interface {
	Plugin

	// SupportedFormats возвращает список поддерживаемых форматов
	SupportedFormats() []model.OutputFormat

	// Format форматирует проект для вывода
	Format(project *model.Project, format model.OutputFormat) ([]byte, error)

	// FormatFile форматирует отдельный файл
	FormatFile(file *model.ProjectFile, format model.OutputFormat) (string, error)

	// GetTemplates возвращает шаблоны для генерации
	GetTemplates() map[string]string
}

// AnalyzerPlugin - плагин для анализа кода
type AnalyzerPlugin interface {
	Plugin

	// Analyze выполняет анализ проекта
	Analyze(ctx context.Context, project *model.Project) (*model.AnalysisResult, error)

	// GetMetrics собирает метрики проекта
	GetMetrics(project *model.Project) (*model.ProjectMetrics, error)

	// FindDependencies находит зависимости
	FindDependencies(project *model.Project) (*model.DependencyGraph, error)

	// DetectPatterns определяет паттерны проектирования
	DetectPatterns(project *model.Project) ([]model.Pattern, error)
}

// ExtensionPlugin - плагин для расширения функциональности
type ExtensionPlugin interface {
	Plugin

	// Hook вызывается на различных этапах обработки
	Hook(stage model.ProcessingStage, data interface{}) error

	// GetCommands возвращает дополнительные команды CLI
	GetCommands() []model.Command

	// GetConfiguration возвращает схему конфигурации
	GetConfiguration() model.ConfigSchema
}

// PluginMetadata - метаданные плагина
type PluginMetadata struct {
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Author       string             `json:"author"`
	Description  string             `json:"description"`
	License      string             `json:"license"`
	Homepage     string             `json:"homepage"`
	Type         PluginType         `json:"type"`
	Status       PluginStatus       `json:"status"`
	Priority     int                `json:"priority"`
	Capabilities []PluginCapability `json:"capabilities"`
	Dependencies []string           `json:"dependencies"`
	Tags         []string           `json:"tags"`
	ConfigSchema map[string]string  `json:"config_schema"`
}

// PluginCapabilities - возможности плагина
type PluginCapabilities struct {
	SupportsAsync      bool
	SupportsStreaming  bool
	RequiresNetwork    bool
	RequiresFileSystem bool
	MaxFileSize        int64
	MaxMemoryUsage     int64
}

// PluginLifecycle - интерфейс жизненного цикла плагина
type PluginLifecycle interface {
	// OnLoad вызывается при загрузке плагина
	OnLoad() error

	// OnUnload вызывается при выгрузке плагина
	OnUnload() error

	// OnEnable вызывается при включении плагина
	OnEnable() error

	// OnDisable вызывается при отключении плагина
	OnDisable() error

	// HealthCheck проверяет состояние плагина
	HealthCheck() error
}

// CompositePlugin - плагин, реализующий несколько интерфейсов
type CompositePlugin interface {
	Plugin

	// GetCapabilities возвращает список реализованных интерфейсов
	GetCapabilities() []string

	// AsLanguagePlugin возвращает плагин как LanguagePlugin, если поддерживается
	AsLanguagePlugin() (LanguagePlugin, bool)

	// AsProjectPlugin возвращает плагин как ProjectTypePlugin, если поддерживается
	AsProjectPlugin() (ProjectTypePlugin, bool)

	// AsFilterPlugin возвращает плагин как FilterPlugin, если поддерживается
	AsFilterPlugin() (FilterPlugin, bool)

	// AsProcessorPlugin возвращает плагин как ProcessorPlugin, если поддерживается
	AsProcessorPlugin() (ProcessorPlugin, bool)
}

