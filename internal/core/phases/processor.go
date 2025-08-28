// Package phases содержит фазовый процессор для обработки проектов
package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/plugin"
	"github.com/kolkov/gops/pkg/logger"
)

// PhaseProcessor - координатор всего процесса обработки
type PhaseProcessor struct {
	rootDir string
	config  *model.ProjectConfig
	registry *plugin.Registry
	logger  *logger.Logger

	// Фазы обработки
	metadataPhase   *MetadataPhase
	detectionPhase  *DetectionPhase
	analysisPhase   *AnalysisPhase
	selectionPhase  *SelectionPhase
	loadingPhase    *LoadingPhase
	generationPhase *GenerationPhase

	// Результаты фаз
	metadata      *model.ProjectMetadata
	projectInfo   *model.ProjectInfo
	selectedFiles []string
	loadedProject *model.Project

	// Статистика
	stats *ProcessingStats
}

// ProcessingStats - статистика обработки
type ProcessingStats struct {
	StartTime      time.Time
	EndTime        time.Time
	PhaseTimes     map[string]time.Duration
	FilesScanned   int
	FilesProcessed int
	FilesSkipped   int
	TotalSize      int64
	MemoryUsed     int64
	Errors         []error
}

// PhaseResult - результат выполнения фазы
type PhaseResult struct {
	Phase    string
	Success  bool
	Duration time.Duration
	Error    error
	Data     interface{}
}

// NewPhaseProcessor создает новый процессор
func NewPhaseProcessor(rootDir string, config *model.ProjectConfig, logger *logger.Logger) *PhaseProcessor {
	registry := plugin.GetRegistry()

	return &PhaseProcessor{
		rootDir:  rootDir,
		config:   config,
		registry: registry,
		logger:   logger,
		stats: &ProcessingStats{
			PhaseTimes: make(map[string]time.Duration),
			Errors:     make([]error, 0),
		},
	}
}

// Process запускает полный цикл обработки проекта
func (p *PhaseProcessor) Process(ctx context.Context) (*model.Project, error) {
	p.stats.StartTime = time.Now()
	defer func() {
		p.stats.EndTime = time.Now()
	}()

	p.logger.Info("Starting project processing", "root", p.rootDir)

	// Фаза 1: Сбор метаданных
	if err := p.runPhase(ctx, "metadata", p.collectMetadata); err != nil {
		return nil, fmt.Errorf("metadata phase failed: %w", err)
	}

	// Фаза 2: Определение типа проекта
	if err := p.runPhase(ctx, "detection", p.detectProjectType); err != nil {
		return nil, fmt.Errorf("detection phase failed: %w", err)
	}

	// Фаза 3: Анализ проекта
	if err := p.runPhase(ctx, "analysis", p.analyzeProject); err != nil {
		return nil, fmt.Errorf("analysis phase failed: %w", err)
	}

	// Фаза 4: Выбор файлов
	if err := p.runPhase(ctx, "selection", p.selectFiles); err != nil {
		return nil, fmt.Errorf("selection phase failed: %w", err)
	}

	// Фаза 5: Загрузка контента
	if err := p.runPhase(ctx, "loading", p.loadContent); err != nil {
		return nil, fmt.Errorf("loading phase failed: %w", err)
	}

	// Фаза 6: Генерация документации
	if err := p.runPhase(ctx, "generation", p.generateDocumentation); err != nil {
		return nil, fmt.Errorf("generation phase failed: %w", err)
	}

	p.logger.Info("Project processing completed",
		"duration", time.Since(p.stats.StartTime),
		"files", p.stats.FilesProcessed,
		"size", p.stats.TotalSize)

	return p.loadedProject, nil
}

// runPhase выполняет отдельную фазу обработки
func (p *PhaseProcessor) runPhase(ctx context.Context, name string, fn func(context.Context) error) error {
	startTime := time.Now()

	p.logger.Info(fmt.Sprintf("Phase '%s' started", name))

	// Вызываем хуки расширений
	p.callExtensionHooks(model.ProcessingStage("pre_"+name), nil)

	// Выполняем фазу
	err := fn(ctx)

	duration := time.Since(startTime)
	p.stats.PhaseTimes[name] = duration

	if err != nil {
		p.stats.Errors = append(p.stats.Errors, fmt.Errorf("%s: %w", name, err))
		p.logger.Error(fmt.Sprintf("Phase '%s' failed", name), err, "duration", duration)

		// Вызываем хуки ошибки
		p.callExtensionHooks(model.ProcessingStage("error_"+name), err)

		return err
	}

	p.logger.Info(fmt.Sprintf("Phase '%s' completed", name), "duration", duration)

	// Вызываем хуки после фазы
	p.callExtensionHooks(model.ProcessingStage("post_"+name), nil)

	return nil
}

// Фаза 1: Сбор метаданных
func (p *PhaseProcessor) collectMetadata(ctx context.Context) error {
	p.metadataPhase = NewMetadataPhase(p.rootDir, p.config.ScanConfig, p.logger)

	metadata, err := p.metadataPhase.Collect(ctx)
	if err != nil {
		return err
	}

	p.metadata = metadata
	p.stats.FilesScanned = metadata.TotalFiles
	p.stats.TotalSize = metadata.TotalSize

	p.logger.Info("Metadata collected",
		"files", metadata.TotalFiles,
		"dirs", metadata.TotalDirs,
		"size", metadata.TotalSize)

	return nil
}

// Фаза 2: Определение типа проекта
func (p *PhaseProcessor) detectProjectType(ctx context.Context) error {
	p.detectionPhase = NewDetectionPhase(p.registry, p.logger)

	projectPlugin, confidence, err := p.detectionPhase.Detect(ctx, p.metadata)
	if err != nil {
		// Если не удалось определить тип, используем generic
		p.logger.Warn("Failed to detect project type, using generic", "error", err)
		projectPlugin = nil
		confidence = 0.0
	}

	// Анализируем структуру через плагин проекта
	if projectPlugin != nil {
		info, err := projectPlugin.AnalyzeStructure(ctx, p.metadata)
		if err != nil {
			return fmt.Errorf("project analysis failed: %w", err)
		}

		p.projectInfo = info

		// Применяем правила загрузки от плагина
		if p.config.LoadingRules == nil {
			p.config.LoadingRules = projectPlugin.GetLoadingRules()
		}

		// Применяем правила исключения
		if p.config.ExclusionRules == nil {
			p.config.ExclusionRules = projectPlugin.GetExclusionRules()
		}
	} else {
		// Используем generic project info
		p.projectInfo = &model.ProjectInfo{
			Type:     "generic",
			Language: "unknown",
		}
	}

	p.logger.Info("Project type detected",
		"type", p.projectInfo.Type,
		"confidence", confidence)

	return nil
}

// Фаза 3: Анализ проекта
func (p *PhaseProcessor) analyzeProject(ctx context.Context) error {
	p.analysisPhase = NewAnalysisPhase(p.registry, p.logger)

	// Анализируем компоненты
	components, err := p.analysisPhase.AnalyzeComponents(ctx, p.metadata, p.projectInfo)
	if err != nil {
		p.logger.Warn("Component analysis failed", "error", err)
		// Не критичная ошибка - продолжаем
	} else {
		p.projectInfo.Components = components
	}

	// Анализируем метрики
	metrics, err := p.analysisPhase.AnalyzeMetrics(ctx, p.metadata)
	if err != nil {
		p.logger.Warn("Metrics analysis failed", "error", err)
	} else {
		p.projectInfo.Metrics = metrics
	}

	p.logger.Info("Project analyzed",
		"components", len(p.projectInfo.Components),
		"modules", len(p.projectInfo.Modules))

	return nil
}

// Фаза 4: Выбор файлов
func (p *PhaseProcessor) selectFiles(ctx context.Context) error {
	p.selectionPhase = NewSelectionPhase(p.config, p.registry, p.logger)

	selectedFiles, err := p.selectionPhase.Select(ctx, p.metadata, p.projectInfo)
	if err != nil {
		return err
	}

	p.selectedFiles = selectedFiles
	p.stats.FilesProcessed = len(selectedFiles)

	p.logger.Info("Files selected",
		"total", len(selectedFiles),
		"skipped", p.stats.FilesScanned-len(selectedFiles))

	return nil
}

// Фаза 5: Загрузка контента
func (p *PhaseProcessor) loadContent(ctx context.Context) error {
	p.loadingPhase = NewLoadingPhase(p.config, p.registry, p.logger)

	project, err := p.loadingPhase.Load(ctx, p.metadata, p.selectedFiles)
	if err != nil {
		return err
	}

	p.loadedProject = project
	p.loadedProject.Info = p.projectInfo

	// Подсчитываем статистику
	for _, file := range project.Files {
		if file.Skipped {
			p.stats.FilesSkipped++
		}
	}

	p.logger.Info("Content loaded",
		"files", len(project.Files),
		"skipped", p.stats.FilesSkipped)

	return nil
}

// Фаза 6: Генерация документации
func (p *PhaseProcessor) generateDocumentation(ctx context.Context) error {
	p.generationPhase = NewGenerationPhase(p.config.OutputConfig, p.registry, p.logger)

	err := p.generationPhase.Generate(ctx, p.loadedProject)
	if err != nil {
		return err
	}

	p.logger.Info("Documentation generated",
		"format", p.config.OutputConfig.Format,
		"output", p.config.OutputConfig.Filename)

	return nil
}

// callExtensionHooks вызывает хуки расширений
func (p *PhaseProcessor) callExtensionHooks(stage model.ProcessingStage, data interface{}) {
	// Получаем все extension плагины из реестра
	for _, plugin := range p.registry.GetAllPlugins() {
		if ext, ok := plugin.(plugin.ExtensionPlugin); ok {
			if err := ext.Hook(stage, data); err != nil {
				p.logger.Warn("Extension hook failed",
					"plugin", plugin.Name(),
					"stage", stage,
					"error", err)
			}
		}
	}
}

// GetStats возвращает статистику обработки
func (p *PhaseProcessor) GetStats() *ProcessingStats {
	return p.stats
}

// GetMetadata возвращает метаданные проекта
func (p *PhaseProcessor) GetMetadata() *model.ProjectMetadata {
	return p.metadata
}

// GetProjectInfo возвращает информацию о проекте
func (p *PhaseProcessor) GetProjectInfo() *model.ProjectInfo {
	return p.projectInfo
}

// GetProject возвращает загруженный проект
func (p *PhaseProcessor) GetProject() *model.Project {
	return p.loadedProject
}

// LoadingPhase - фаза загрузки контента файлов
type LoadingPhase struct {
	config   *model.ProjectConfig
	registry *plugin.Registry
	logger   *logger.Logger
}

func NewLoadingPhase(config *model.ProjectConfig, registry *plugin.Registry, logger *logger.Logger) *LoadingPhase {
	return &LoadingPhase{
		config:   config,
		registry: registry,
		logger:   logger,
	}
}

func (l *LoadingPhase) Load(ctx context.Context, metadata *model.ProjectMetadata, selectedFiles []string) (*model.Project, error) {
	l.logger.Info("Loading project content", "files", len(selectedFiles))

	project := &model.Project{
		Metadata:  metadata,
		Files:     make([]*model.ProjectFile, 0, len(selectedFiles)),
		FileMap:   make(map[string]*model.ProjectFile),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Загружаем файлы параллельно
	type fileResult struct {
		file *model.ProjectFile
		err  error
	}

	// Канал для результатов
	resultsChan := make(chan fileResult, len(selectedFiles))
	
	// Ограничиваем количество горутин
	maxWorkers := l.config.ScanConfig.ParallelWorkers
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	
	semaphore := make(chan struct{}, maxWorkers)

	// Запускаем горутины для загрузки файлов
	for _, filePath := range selectedFiles {
		go func(path string) {
			semaphore <- struct{}{} // Захватываем семафор
			defer func() { <-semaphore }() // Освобождаем семафор

			file, err := l.loadSingleFile(ctx, path, metadata.RootDir)
			resultsChan <- fileResult{file: file, err: err}
		}(filePath)
	}

	// Собираем результаты
	filesLoaded := 0
	for i := 0; i < len(selectedFiles); i++ {
		result := <-resultsChan
		if result.err != nil {
			l.logger.Warn("Failed to load file", "error", result.err)
			continue
		}
		
		if result.file != nil {
			project.Files = append(project.Files, result.file)
			project.FileMap[result.file.Path] = result.file
			if !result.file.Skipped {
				filesLoaded++
			}
		}
	}

	l.logger.Info("Content loading completed", "loaded", filesLoaded, "total", len(selectedFiles))
	return project, nil
}

func (l *LoadingPhase) loadSingleFile(ctx context.Context, filePath, rootDir string) (*model.ProjectFile, error) {
	// Проверяем контекст
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := filepath.Join(rootDir, filePath)
	
	// Создаем базовый объект файла
	file := &model.ProjectFile{
		Path:     filePath,
		Language: l.getFileLanguage(filePath),
		Size:     0,
		ModTime:  time.Now(),
	}

	// Получаем информацию о файле
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("stat file %s: %w", filePath, err)
	}

	file.Size = info.Size()
	file.ModTime = info.ModTime()

	// Проверяем размер файла
	if file.Size > l.config.ScanConfig.MaxFileSize {
		l.logger.Warn("File too large, skipping", "file", filePath, "size", file.Size)
		file.Skipped = true
		file.SkipReason = "file too large"
		return file, nil
	}

	// Определяем режим загрузки контента
	contentMode := l.determineContentMode(filePath)
	
	switch contentMode {
	case model.ContentModeNone:
		file.Skipped = true
		file.SkipReason = "excluded by content mode"
		
	case model.ContentModeHeaders:
		// Загружаем только заголовки/метаданные
		if err := l.loadFileHeaders(ctx, file, fullPath); err != nil {
			l.logger.Warn("Failed to load file headers", "file", filePath, "error", err)
		}
		
	case model.ContentModeFull:
		// Загружаем полный контент
		if err := l.loadFileContent(ctx, file, fullPath); err != nil {
			l.logger.Warn("Failed to load file content", "file", filePath, "error", err)
			file.Skipped = true
			file.SkipReason = err.Error()
		}
		
	case model.ContentModeCompressed:
		// Загружаем контент и сжимаем
		if err := l.loadFileContent(ctx, file, fullPath); err != nil {
			l.logger.Warn("Failed to load file content", "file", filePath, "error", err)
			file.Skipped = true
			file.SkipReason = err.Error()
		} else {
			// Применяем сжатие (например, удаляем комментарии)
			l.compressContent(file)
		}
	}

	return file, nil
}

func (l *LoadingPhase) loadFileContent(ctx context.Context, file *model.ProjectFile, fullPath string) error {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	
	file.Content = string(content)
	
	// Если есть языковой плагин, используем его для извлечения метаданных
	if plugin := l.getLanguagePlugin(file.Language); plugin != nil {
		metadata, err := plugin.ExtractMetadata(ctx, file)
		if err != nil {
			l.logger.Debug("Failed to extract metadata with plugin", "file", file.Path, "error", err)
		} else {
			file.Metadata = metadata
		}
	}
	
	return nil
}

func (l *LoadingPhase) loadFileHeaders(ctx context.Context, file *model.ProjectFile, fullPath string) error {
	// Для заголовков читаем только начало файла
	f, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	
	// Читаем первые 1KB для анализа заголовков
	buffer := make([]byte, 1024)
	n, err := f.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("read file headers: %w", err)
	}
	
	file.Content = string(buffer[:n])
	
	// Используем языковой плагин для извлечения заголовков
	if plugin := l.getLanguagePlugin(file.Language); plugin != nil {
		metadata, err := plugin.ExtractMetadata(ctx, file)
		if err == nil {
			file.Metadata = metadata
		}
	}
	
	// Очищаем контент, оставляем только метаданные
	file.Content = ""
	
	return nil
}

func (l *LoadingPhase) compressContent(file *model.ProjectFile) {
	// Простая компрессия - удаляем лишние пробелы и пустые строки
	lines := strings.Split(file.Content, "\n")
	compressed := make([]string, 0, len(lines))
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			compressed = append(compressed, trimmed)
		}
	}
	
	file.Content = strings.Join(compressed, "\n")
}

func (l *LoadingPhase) getFileLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Сопоставляем с нашими новыми языковыми константами
	switch ext {
	case ".go":
		return "go"
	case ".js", ".mjs", ".cjs":
		return "javascript"  
	case ".jsx":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx": 
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	default:
		return "text"
	}
}

func (l *LoadingPhase) determineContentMode(filePath string) model.ContentMode {
	// По умолчанию загружаем полный контент
	mode := model.ContentModeFull
	
	// Проверяем правила загрузки из конфигурации
	if l.config.LoadingRules != nil {
		for _, rule := range l.config.LoadingRules {
			if rule.Matches(filePath) {
				mode = rule.ContentMode
				break
			}
		}
	}
	
	return mode
}

func (l *LoadingPhase) getLanguagePlugin(language string) plugin.LanguagePlugin {
	// Получаем языковой плагин из реестра
	for _, p := range l.registry.GetAllPlugins() {
		if langPlugin, ok := p.(plugin.LanguagePlugin); ok {
			if langPlugin.GetLanguage() == language {
				return langPlugin
			}
		}
	}
	return nil
}

// GenerationPhase - фаза генерации документации
type GenerationPhase struct {
	config   model.OutputConfig
	registry *plugin.Registry
	logger   *logger.Logger
}

func NewGenerationPhase(config model.OutputConfig, registry *plugin.Registry, logger *logger.Logger) *GenerationPhase {
	return &GenerationPhase{
		config:   config,
		registry: registry,
		logger:   logger,
	}
}

func (g *GenerationPhase) Generate(ctx context.Context, project *model.Project) error {
	g.logger.Info("Starting documentation generation", "format", g.config.Format, "output", g.config.Filename)

	// Создаем генератор в зависимости от формата
	var generator DocumentGenerator
	var err error

	switch g.config.Format {
	case model.OutputFormatMarkdown:
		generator, err = g.createMarkdownGenerator()
	case model.OutputFormatHTML:
		generator, err = g.createHTMLGenerator()
	case model.OutputFormatJSON:
		generator, err = g.createJSONGenerator()
	default:
		return fmt.Errorf("unsupported output format: %s", g.config.Format)
	}

	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}
	defer generator.Close()

	// Генерируем документацию
	if err := g.generateDocumentation(ctx, generator, project); err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	g.logger.Info("Documentation generation completed", "output", g.config.Filename)
	return nil
}

func (g *GenerationPhase) createMarkdownGenerator() (DocumentGenerator, error) {
	return &MarkdownDocumentGenerator{
		outputPath: g.config.Filename,
		config:     g.config,
		logger:     g.logger,
	}, nil
}

func (g *GenerationPhase) createHTMLGenerator() (DocumentGenerator, error) {
	// TODO: Реализовать HTML генератор
	return nil, fmt.Errorf("HTML generator not implemented yet")
}

func (g *GenerationPhase) createJSONGenerator() (DocumentGenerator, error) {
	// TODO: Реализовать JSON генератор  
	return nil, fmt.Errorf("JSON generator not implemented yet")
}

func (g *GenerationPhase) generateDocumentation(ctx context.Context, generator DocumentGenerator, project *model.Project) error {
	// Инициализируем генератор
	if err := generator.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Генерируем заголовок документа
	if err := generator.GenerateHeader(ctx, project); err != nil {
		return fmt.Errorf("failed to generate header: %w", err)
	}

	// Генерируем структуру проекта
	if err := generator.GenerateProjectStructure(ctx, project); err != nil {
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	// Генерируем информацию о файлах
	if err := generator.GenerateFiles(ctx, project); err != nil {
		return fmt.Errorf("failed to generate files section: %w", err)
	}

	// Генерируем заключение
	if err := generator.GenerateFooter(ctx, project); err != nil {
		return fmt.Errorf("failed to generate footer: %w", err)
	}

	return nil
}

// DocumentGenerator - интерфейс для генераторов документации
type DocumentGenerator interface {
	Initialize(ctx context.Context) error
	GenerateHeader(ctx context.Context, project *model.Project) error
	GenerateProjectStructure(ctx context.Context, project *model.Project) error
	GenerateFiles(ctx context.Context, project *model.Project) error
	GenerateFooter(ctx context.Context, project *model.Project) error
	Close() error
}

// MarkdownDocumentGenerator - генератор Markdown документации
type MarkdownDocumentGenerator struct {
	outputPath string
	config     model.OutputConfig
	logger     *logger.Logger
	file       *os.File
}

func (m *MarkdownDocumentGenerator) Initialize(ctx context.Context) error {
	var err error
	m.file, err = os.Create(m.outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	return nil
}

func (m *MarkdownDocumentGenerator) GenerateHeader(ctx context.Context, project *model.Project) error {
	fmt.Fprintf(m.file, "# Проект: %s\n\n", project.Metadata.ProjectName)
	fmt.Fprintf(m.file, "**Дата генерации:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(m.file, "**Корневая директория:** %s\n\n", project.Metadata.RootDir)
	fmt.Fprintf(m.file, "**Общее количество файлов:** %d\n\n", len(project.Files))
	
	// Информация о проекте
	if project.Info != nil {
		fmt.Fprintf(m.file, "**Тип проекта:** %s\n\n", project.Info.Type)
		fmt.Fprintf(m.file, "**Язык:** %s\n\n", project.Info.Language)
		
		if len(project.Info.Components) > 0 {
			fmt.Fprintf(m.file, "**Компоненты:** %d\n\n", len(project.Info.Components))
		}
	}
	
	// Содержание
	fmt.Fprintf(m.file, "## Содержание\n")
	fmt.Fprintf(m.file, "- [Структура проекта](#структура-проекта)\n")
	fmt.Fprintf(m.file, "- [Файлы проекта](#файлы-проекта)\n")
	fmt.Fprintf(m.file, "\n")
	
	return nil
}

func (m *MarkdownDocumentGenerator) GenerateProjectStructure(ctx context.Context, project *model.Project) error {
	fmt.Fprintf(m.file, "## Структура проекта\n\n")
	
	// Простое дерево файлов
	fmt.Fprintf(m.file, "```\n")
	for _, file := range project.Files {
		if !file.Skipped {
			fmt.Fprintf(m.file, "%s\n", file.Path)
		}
	}
	fmt.Fprintf(m.file, "```\n\n")
	
	return nil
}

func (m *MarkdownDocumentGenerator) GenerateFiles(ctx context.Context, project *model.Project) error {
	fmt.Fprintf(m.file, "## Файлы проекта\n\n")
	
	for _, file := range project.Files {
		if file.Skipped {
			continue
		}
		
		fmt.Fprintf(m.file, "### %s\n\n", file.Path)
		
		if file.Metadata != nil {
			fmt.Fprintf(m.file, "**Язык:** %s  \n", file.Language)
			fmt.Fprintf(m.file, "**Размер:** %d bytes  \n", file.Size)
			if file.Metadata.LOC > 0 {
				fmt.Fprintf(m.file, "**Строки кода:** %d  \n", file.Metadata.LOC)
			}
			fmt.Fprintf(m.file, "\n")
		}
		
		if file.Content != "" {
			fmt.Fprintf(m.file, "```%s\n", file.Language)
			fmt.Fprintf(m.file, "%s\n", file.Content)
			fmt.Fprintf(m.file, "```\n\n")
		}
		
		fmt.Fprintf(m.file, "---\n\n")
	}
	
	return nil
}

func (m *MarkdownDocumentGenerator) GenerateFooter(ctx context.Context, project *model.Project) error {
	fmt.Fprintf(m.file, "## Сводка\n\n")
	
	totalFiles := len(project.Files)
	skippedFiles := 0
	for _, file := range project.Files {
		if file.Skipped {
			skippedFiles++
		}
	}
	
	fmt.Fprintf(m.file, "- **Всего файлов:** %d\n", totalFiles)
	fmt.Fprintf(m.file, "- **Обработано:** %d\n", totalFiles-skippedFiles)
	fmt.Fprintf(m.file, "- **Пропущено:** %d\n", skippedFiles)
	fmt.Fprintf(m.file, "- **Дата генерации:** %s\n", time.Now().Format("2006-01-02 15:04:05"))
	
	return nil
}

func (m *MarkdownDocumentGenerator) Close() error {
	if m.file != nil {
		return m.file.Close()
	}
	return nil
}