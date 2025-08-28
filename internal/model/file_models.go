// Package model содержит расширенные модели для работы с файлами
package model

import (
	"path/filepath" 
	"strings"
	"time"
)

// Project - полная модель проекта
type Project struct {
	// Метаданные
	Metadata *ProjectMetadata `json:"metadata"`
	Info     *ProjectInfo     `json:"info"`

	// Файлы
	Files   []*ProjectFile          `json:"files"`
	FileMap map[string]*ProjectFile `json:"-"` // Быстрый доступ

	// Анализ
	Analysis *AnalysisResult `json:"analysis,omitempty"`

	// Конфигурация
	Config *ProjectConfig `json:"config,omitempty"`

	// Время создания
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProjectFile - файл проекта с контентом
type ProjectFile struct {
	// Метаданные
	Metadata *FileMetadata `json:"metadata"`

	// Контент
	Content     string `json:"-"`                 // Текстовый контент (для совместимости)
	ContentText string `json:"content,omitempty"` // Текстовый контент
	ContentSize int64  `json:"content_size"`

	// Заголовки (для режима headers-only)
	Header *FileHeader `json:"header,omitempty"`

	// Язык и тип - добавляем alias для совместимости
	Language string `json:"language"`
	Lang     string `json:"lang"` // Alias для совместимости
	FileType string `json:"file_type"`
	MimeType string `json:"mime_type,omitempty"`

	// Статус
	LoadingMode ContentMode `json:"loading_mode"`
	Skipped     bool        `json:"skipped"`
	SkipReason  string      `json:"skip_reason,omitempty"`
	Error       string      `json:"error,omitempty"`

	// Метрики
	Metrics *FileMetrics `json:"metrics,omitempty"`

	// Физические атрибуты - добавляем для совместимости
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`

	// Связи
	Imports      []string `json:"imports,omitempty"`
	Exports      []string `json:"exports,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// FileHeader - заголовочная информация файла (сигнатуры без реализации)
type FileHeader struct {
	// Язык
	Language string `json:"language"`

	// Пакет/модуль/namespace
	Package   string `json:"package,omitempty"`
	Module    string `json:"module,omitempty"`
	Namespace string `json:"namespace,omitempty"`

	// Для совместимости со старой моделью
	ExportedFunctions []string `json:"exported_functions,omitempty"`
	ExportedTypes     []string `json:"exported_types,omitempty"`
	ExportedConstants []string `json:"exported_constants,omitempty"`
	Dependencies      []string `json:"dependencies,omitempty"`
	Summary           string   `json:"summary,omitempty"`

	// Импорты/включения
	Imports  []ImportInfo `json:"imports,omitempty"`
	Includes []string     `json:"includes,omitempty"` // Для C/C++

	// Функции и методы
	Functions []FunctionSignature `json:"functions,omitempty"`
	Methods   []MethodSignature   `json:"methods,omitempty"`

	// Типы данных
	Types      []TypeDefinition      `json:"types,omitempty"`
	Classes    []ClassDefinition     `json:"classes,omitempty"`
	Interfaces []InterfaceDefinition `json:"interfaces,omitempty"`
	Structs    []StructDefinition    `json:"structs,omitempty"`
	Enums      []EnumDefinition      `json:"enums,omitempty"`

	// Переменные и константы
	Variables []VariableDeclaration `json:"variables,omitempty"`
	Constants []ConstantDeclaration `json:"constants,omitempty"`

	// Макросы (для C/C++)
	Macros []MacroDefinition `json:"macros,omitempty"`

	// Аннотации/декораторы
	Annotations []Annotation `json:"annotations,omitempty"`

	// Комментарии документации
	DocComments []DocComment `json:"doc_comments,omitempty"`

	// Краткое описание (дублирует Summary выше, убрано для избежания конфликта)
	Description string `json:"description,omitempty"`
}

// ImportInfo - информация об импорте
type ImportInfo struct {
	Path       string   `json:"path"`
	Alias      string   `json:"alias,omitempty"`
	Names      []string `json:"names,omitempty"` // Для selective imports
	IsWildcard bool     `json:"is_wildcard"`
}

// FunctionSignature - сигнатура функции
type FunctionSignature struct {
	Name        string      `json:"name"`
	Parameters  []Parameter `json:"parameters,omitempty"`
	Returns     []Return    `json:"returns,omitempty"`
	ReturnType  string      `json:"return_type,omitempty"`
	Generics    []Generic   `json:"generics,omitempty"`
	Modifiers   []string    `json:"modifiers,omitempty"` // public, static, async и т.д.
	Annotations []string    `json:"annotations,omitempty"`
	DocComment  string      `json:"doc_comment,omitempty"`
	Complexity  int         `json:"complexity,omitempty"`
	StartLine   int         `json:"start_line,omitempty"`
	EndLine     int         `json:"end_line,omitempty"`
	IsExported  bool        `json:"is_exported"`
	IsAsync     bool        `json:"is_async"`
	IsGenerator bool        `json:"is_generator"`
}

// MethodSignature - сигнатура метода
type MethodSignature struct {
	FunctionSignature
	Receiver   string `json:"receiver,omitempty"` // Для Go
	Class      string `json:"class,omitempty"`
	IsStatic   bool   `json:"is_static"`
	IsVirtual  bool   `json:"is_virtual"`
	IsAbstract bool   `json:"is_abstract"`
	IsOverride bool   `json:"is_override"`
	Visibility string `json:"visibility"` // public, private, protected
}

// Parameter - параметр функции/метода
type Parameter struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	DefaultValue string   `json:"default_value,omitempty"`
	IsOptional   bool     `json:"is_optional"`
	IsVariadic   bool     `json:"is_variadic"`
	IsReference  bool     `json:"is_reference"`
	IsPointer    bool     `json:"is_pointer"`
	Annotations  []string `json:"annotations,omitempty"`
}

// Return - возвращаемое значение
type Return struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// Generic - обобщенный тип (generic/template)
type Generic struct {
	Name       string `json:"name"`
	Constraint string `json:"constraint,omitempty"`
	Default    string `json:"default,omitempty"`
}

// TypeDefinition - определение типа
type TypeDefinition struct {
	Name       string    `json:"name"`
	Kind       string    `json:"kind"` // alias, typedef, newtype и т.д.
	BaseType   string    `json:"base_type,omitempty"`
	Generics   []Generic `json:"generics,omitempty"`
	DocComment string    `json:"doc_comment,omitempty"`
	IsExported bool      `json:"is_exported"`
}

// ClassDefinition - определение класса
type ClassDefinition struct {
	Name         string               `json:"name"`
	BaseClasses  []string             `json:"base_classes,omitempty"`
	Interfaces   []string             `json:"interfaces,omitempty"`
	Fields       []FieldDefinition    `json:"fields,omitempty"`
	Methods      []MethodSignature    `json:"methods,omitempty"`
	Properties   []PropertyDefinition `json:"properties,omitempty"`
	Constructors []FunctionSignature  `json:"constructors,omitempty"`
	Destructor   *FunctionSignature   `json:"destructor,omitempty"`
	Generics     []Generic            `json:"generics,omitempty"`
	Annotations  []string             `json:"annotations,omitempty"`
	DocComment   string               `json:"doc_comment,omitempty"`
	IsAbstract   bool                 `json:"is_abstract"`
	IsFinal      bool                 `json:"is_final"`
	IsExported   bool                 `json:"is_exported"`
}

// InterfaceDefinition - определение интерфейса
type InterfaceDefinition struct {
	Name       string               `json:"name"`
	Extends    []string             `json:"extends,omitempty"`
	Methods    []MethodSignature    `json:"methods"`
	Properties []PropertyDefinition `json:"properties,omitempty"`
	Generics   []Generic            `json:"generics,omitempty"`
	DocComment string               `json:"doc_comment,omitempty"`
	IsExported bool                 `json:"is_exported"`
}

// StructDefinition - определение структуры
type StructDefinition struct {
	Name       string            `json:"name"`
	Fields     []FieldDefinition `json:"fields"`
	Methods    []MethodSignature `json:"methods,omitempty"`
	Embeds     []string          `json:"embeds,omitempty"` // Для Go
	Generics   []Generic         `json:"generics,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"` // Для Go struct tags
	DocComment string            `json:"doc_comment,omitempty"`
	IsExported bool              `json:"is_exported"`
	IsPacked   bool              `json:"is_packed"` // Для C/C++
}

// FieldDefinition - определение поля
type FieldDefinition struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	DefaultValue string            `json:"default_value,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	Annotations  []string          `json:"annotations,omitempty"`
	DocComment   string            `json:"doc_comment,omitempty"`
	Visibility   string            `json:"visibility"`
	IsStatic     bool              `json:"is_static"`
	IsReadonly   bool              `json:"is_readonly"`
	IsVolatile   bool              `json:"is_volatile"`
}

// PropertyDefinition - определение свойства (для языков с properties)
type PropertyDefinition struct {
	Name         string             `json:"name"`
	Type         string             `json:"type"`
	Getter       *FunctionSignature `json:"getter,omitempty"`
	Setter       *FunctionSignature `json:"setter,omitempty"`
	DefaultValue string             `json:"default_value,omitempty"`
	Annotations  []string           `json:"annotations,omitempty"`
	DocComment   string             `json:"doc_comment,omitempty"`
	Visibility   string             `json:"visibility"`
	IsStatic     bool               `json:"is_static"`
	IsReadonly   bool               `json:"is_readonly"`
}

// EnumDefinition - определение перечисления
type EnumDefinition struct {
	Name       string      `json:"name"`
	BaseType   string      `json:"base_type,omitempty"`
	Values     []EnumValue `json:"values"`
	DocComment string      `json:"doc_comment,omitempty"`
	IsExported bool        `json:"is_exported"`
}

// EnumValue - значение перечисления
type EnumValue struct {
	Name       string `json:"name"`
	Value      string `json:"value,omitempty"`
	DocComment string `json:"doc_comment,omitempty"`
}

// VariableDeclaration - объявление переменной
type VariableDeclaration struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Value      string   `json:"value,omitempty"`
	Modifiers  []string `json:"modifiers,omitempty"`
	DocComment string   `json:"doc_comment,omitempty"`
	IsExported bool     `json:"is_exported"`
	IsConst    bool     `json:"is_const"`
	IsStatic   bool     `json:"is_static"`
}

// ConstantDeclaration - объявление константы
type ConstantDeclaration struct {
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Value      string `json:"value"`
	DocComment string `json:"doc_comment,omitempty"`
	IsExported bool   `json:"is_exported"`
}

// MacroDefinition - определение макроса (для C/C++)
type MacroDefinition struct {
	Name       string   `json:"name"`
	Parameters []string `json:"parameters,omitempty"`
	Value      string   `json:"value"`
	DocComment string   `json:"doc_comment,omitempty"`
}

// Annotation - аннотация/декоратор
type Annotation struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// DocComment - комментарий документации
type DocComment struct {
	Type        string      `json:"type"`   // function, class, variable и т.д.
	Target      string      `json:"target"` // Имя элемента
	Summary     string      `json:"summary"`
	Description string      `json:"description,omitempty"`
	Parameters  []ParamDoc  `json:"parameters,omitempty"`
	Returns     []ReturnDoc `json:"returns,omitempty"`
	Throws      []string    `json:"throws,omitempty"`
	Examples    []string    `json:"examples,omitempty"`
	SeeAlso     []string    `json:"see_also,omitempty"`
	Deprecated  string      `json:"deprecated,omitempty"`
	Since       string      `json:"since,omitempty"`
	Author      string      `json:"author,omitempty"`
}

// ParamDoc - документация параметра
type ParamDoc struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description"`
}

// ReturnDoc - документация возвращаемого значения
type ReturnDoc struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description"`
}

// FileMetrics - метрики файла
type FileMetrics struct {
	Lines      int `json:"lines"`
	SLOC       int `json:"sloc"` // Source lines of code
	Comments   int `json:"comments"`
	Blanks     int `json:"blanks"`
	Complexity int `json:"complexity"`
	Functions  int `json:"functions"`
	Classes    int `json:"classes"`
	Imports    int `json:"imports"`
}

// ProjectConfig - конфигурация проекта
type ProjectConfig struct {
	// Настройки сканирования
	ScanConfig ScanConfig `json:"scan_config"`

	// Настройки загрузки
	LoadingRules []LoadingRule `json:"loading_rules"`

	// Настройки исключения
	ExclusionRules []ExclusionRule `json:"exclusion_rules"`

	// Настройки вывода
	OutputConfig OutputConfig `json:"output_config"`

	// Настройки плагинов
	PluginConfigs map[string]interface{} `json:"plugin_configs,omitempty"`
}

// ScanConfig - конфигурация сканирования
type ScanConfig struct {
	// Режимы
	SelectionMode     string `json:"selection_mode"`
	DocumentationMode string `json:"documentation_mode"`

	// Пути
	IncludedPaths  []string `json:"included_paths,omitempty"`
	ExcludedPaths  []string `json:"excluded_paths,omitempty"`
	ImportantFiles []string `json:"important_files,omitempty"`

	// Паттерны
	IncludePatterns  []string `json:"include_patterns,omitempty"`
	ExcludePatterns  []string `json:"exclude_patterns,omitempty"`
	ExcludedPatterns []string `json:"excluded_patterns,omitempty"` // Alias для совместимости

	// Типы файлов
	IncludeTests     bool `json:"include_tests"`
	IncludeConfigs   bool `json:"include_configs"`
	IncludeMarkup    bool `json:"include_markup"`
	IncludeStyles    bool `json:"include_styles"`
	IncludeDocs      bool `json:"include_docs"`
	IncludeGenerated bool `json:"include_generated"`

	// Ограничения
	MaxFileSize  int64 `json:"max_file_size"`
	MaxTotalSize int64 `json:"max_total_size"`
	MaxFiles     int   `json:"max_files"`
	MaxDepth     int   `json:"max_depth"`

	// Производительность
	ParallelWorkers int    `json:"parallel_workers"`
	BufferSize      int    `json:"buffer_size"`
	UseCache        bool   `json:"use_cache"`
	CachePath       string `json:"cache_path,omitempty"`

	// Для совместимости со старой моделью
	OutputFilename       string `json:"output_filename,omitempty"`
	OutputConfigFilename string `json:"output_config_filename,omitempty"`
	ShowFullStructure    bool   `json:"show_full_structure"`
	Timeout              int64  `json:"timeout"`
}

// OutputConfig - конфигурация вывода
type OutputConfig struct {
	Format           OutputFormat `json:"format"`
	Filename         string       `json:"filename"`
	Directory        string       `json:"directory,omitempty"`
	AppendTimestamp  bool         `json:"append_timestamp"`
	Template         string       `json:"template,omitempty"`
	IncludeTOC       bool         `json:"include_toc"`
	IncludeIndex     bool         `json:"include_index"`
	IncludeMetrics   bool         `json:"include_metrics"`
	SplitByComponent bool         `json:"split_by_component"`
}

// DocWriter - интерфейс для записи документации
type DocWriter interface {
	// WriteHeader пишет заголовок документации
	WriteHeader(meta *ProjectMeta) error

	// WriteTree пишет дерево проекта
	WriteTree(structure string) error

	// WriteFileSection пишет секцию файла
	WriteFileSection(file *ProjectFile) error

	// WriteComponent пишет компонент
	WriteComponent(component Component) error

	// WriteMetrics пишет метрики
	WriteMetrics(metrics *ProjectMetrics) error

	// WriteTOC пишет оглавление
	WriteTOC(sections []string) error

	// Close закрывает writer
	Close() error
}

// Методы для совместимости ScanConfig

// HasFileSelection проверяет, активен ли режим выбора файлов  
func (sc *ScanConfig) HasFileSelection() bool {
	return sc.SelectionMode == "list" || sc.SelectionMode == "patterns" || sc.SelectionMode == "interactive"
}

// GetSelectedFilesCount возвращает количество выбранных файлов
func (sc *ScanConfig) GetSelectedFilesCount() int {
	switch sc.SelectionMode {
	case "list":
		return len(sc.IncludedPaths)
	case "patterns":
		return len(sc.IncludePatterns)
	default:
		return 0
	}
}

// IsFileIncluded проверяет, должен ли файл быть включен в документацию
func (sc *ScanConfig) IsFileIncluded(relPath string) bool {
	// Если режим "all" или не задан - включаем все файлы
	if sc.SelectionMode == "" || sc.SelectionMode == "all" {
		return true
	}

	// Для режима "list" проверяем точное совпадение путей
	if sc.SelectionMode == "list" && len(sc.IncludedPaths) > 0 {
		for _, includedPath := range sc.IncludedPaths {
			if relPath == includedPath {
				return true
			}
			// Проверяем, если это файл внутри выбранной папки
			if strings.HasPrefix(relPath, includedPath+"/") {
				return true
			}
		}
		return false
	}

	// Для режима "patterns" проверяем соответствие паттернам
	if sc.SelectionMode == "patterns" && len(sc.IncludePatterns) > 0 {
		for _, pattern := range sc.IncludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return true
			}
		}
		return false
	}

	return true
}

// ProjectMeta - метаинформация для генерации документации
type ProjectMeta struct {
	Name        string
	Type        string
	RootDir     string
	Version     string
	Description string
	Generated   time.Time
}