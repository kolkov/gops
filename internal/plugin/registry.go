// Package plugin содержит реестр и управление плагинами
package plugin

import (
	"fmt"
	"sort"
	"sync"

	"github.com/kolkov/gops/internal/model"
)

// Registry - реестр всех плагинов системы
type Registry struct {
	mu               sync.RWMutex
	languagePlugins  map[string][]LanguagePlugin // extension -> plugins
	projectPlugins   []ProjectTypePlugin
	filterPlugins    []FilterPlugin
	processorPlugins []ProcessorPlugin
	formatterPlugins map[model.OutputFormat]FormatterPlugin
	analyzerPlugins  []AnalyzerPlugin
	extensionPlugins []ExtensionPlugin

	// Индексы для быстрого поиска
	pluginsByName    map[string]Plugin
	pluginPriorities map[string]int

	// Конфигурация
	config *RegistryConfig

	// Статистика
	stats *RegistryStats
}

// RegistryConfig - конфигурация реестра
type RegistryConfig struct {
	// Директории для поиска плагинов
	PluginDirs []string

	// Включенные/отключенные плагины
	EnabledPlugins  []string
	DisabledPlugins []string

	// Автоматическая загрузка
	AutoLoad bool

	// Конфигурации плагинов
	PluginConfigs map[string]map[string]interface{}
}

// RegistryStats - статистика реестра
type RegistryStats struct {
	TotalPlugins   int
	LoadedPlugins  int
	FailedPlugins  int
	ActivePlugins  int
	ProcessedFiles int64
	ProcessingTime int64 // в миллисекундах
}

// Глобальный реестр
var (
	defaultRegistry *Registry
	registryOnce    sync.Once
)

// GetRegistry возвращает глобальный реестр плагинов
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		defaultRegistry = NewRegistry(nil)
	})
	return defaultRegistry
}

// NewRegistry создает новый реестр плагинов
func NewRegistry(config *RegistryConfig) *Registry {
	if config == nil {
		config = &RegistryConfig{
			AutoLoad: true,
		}
	}

	return &Registry{
		languagePlugins:  make(map[string][]LanguagePlugin),
		projectPlugins:   make([]ProjectTypePlugin, 0),
		filterPlugins:    make([]FilterPlugin, 0),
		processorPlugins: make([]ProcessorPlugin, 0),
		formatterPlugins: make(map[model.OutputFormat]FormatterPlugin),
		analyzerPlugins:  make([]AnalyzerPlugin, 0),
		extensionPlugins: make([]ExtensionPlugin, 0),
		pluginsByName:    make(map[string]Plugin),
		pluginPriorities: make(map[string]int),
		config:           config,
		stats:            &RegistryStats{},
	}
}

// Register регистрирует плагин в реестре
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем, не зарегистрирован ли уже
	if _, exists := r.pluginsByName[p.Name()]; exists {
		return fmt.Errorf("plugin %s already registered", p.Name())
	}

	// Проверяем, не отключен ли плагин
	if r.isDisabled(p.Name()) {
		return fmt.Errorf("plugin %s is disabled", p.Name())
	}

	// Инициализируем плагин с конфигурацией
	if config, ok := r.config.PluginConfigs[p.Name()]; ok {
		if err := p.Init(config); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", p.Name(), err)
		}
	} else {
		// Инициализируем с пустой конфигурацией
		if err := p.Init(nil); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", p.Name(), err)
		}
	}

	// Регистрируем по типу
	switch plugin := p.(type) {
	case LanguagePlugin:
		r.registerLanguagePlugin(plugin)
	case ProjectTypePlugin:
		r.registerProjectPlugin(plugin)
	case FilterPlugin:
		r.registerFilterPlugin(plugin)
	case ProcessorPlugin:
		r.registerProcessorPlugin(plugin)
	case FormatterPlugin:
		r.registerFormatterPlugin(plugin)
	case AnalyzerPlugin:
		r.registerAnalyzerPlugin(plugin)
	case ExtensionPlugin:
		r.registerExtensionPlugin(plugin)
	case CompositePlugin:
		r.registerCompositePlugin(plugin)
	default:
		return fmt.Errorf("unknown plugin type: %T", p)
	}

	// Добавляем в общий индекс
	r.pluginsByName[p.Name()] = p
	r.pluginPriorities[p.Name()] = p.Priority()

	// Обновляем статистику
	r.stats.TotalPlugins++
	r.stats.LoadedPlugins++

	// Вызываем lifecycle hook если есть
	if lifecycle, ok := p.(PluginLifecycle); ok {
		if err := lifecycle.OnLoad(); err != nil {
			return fmt.Errorf("plugin %s OnLoad failed: %w", p.Name(), err)
		}

		if err := lifecycle.OnEnable(); err != nil {
			return fmt.Errorf("plugin %s OnEnable failed: %w", p.Name(), err)
		}
	}

	return nil
}

// Unregister удаляет плагин из реестра
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.pluginsByName[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Вызываем lifecycle hooks
	if lifecycle, ok := plugin.(PluginLifecycle); ok {
		lifecycle.OnDisable()
		lifecycle.OnUnload()
	}

	// Удаляем из специфичных реестров
	switch p := plugin.(type) {
	case LanguagePlugin:
		r.unregisterLanguagePlugin(p)
	case ProjectTypePlugin:
		r.unregisterProjectPlugin(p)
	case FilterPlugin:
		r.unregisterFilterPlugin(p)
		// ... и так далее для других типов
	}

	// Удаляем из общего индекса
	delete(r.pluginsByName, name)
	delete(r.pluginPriorities, name)

	r.stats.LoadedPlugins--

	return nil
}

// GetLanguagePlugin возвращает языковой плагин для расширения файла
func (r *Registry) GetLanguagePlugin(extension string) (LanguagePlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins, exists := r.languagePlugins[extension]
	if !exists || len(plugins) == 0 {
		return nil, false
	}

	// Возвращаем плагин с наивысшим приоритетом
	return plugins[0], true
}

// GetProjectPlugin определяет тип проекта и возвращает соответствующий плагин
func (r *Registry) GetProjectPlugin(metadata *model.ProjectMetadata) (ProjectTypePlugin, float32, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bestPlugin ProjectTypePlugin
	var bestScore float32

	for _, plugin := range r.projectPlugins {
		score, err := plugin.Detect(metadata)
		if err != nil {
			continue
		}

		if score > bestScore {
			bestScore = score
			bestPlugin = plugin
		}
	}

	if bestPlugin == nil {
		return nil, 0, fmt.Errorf("no suitable project plugin found")
	}

	return bestPlugin, bestScore, nil
}

// GetFilterPlugins возвращает все активные плагины фильтрации
func (r *Registry) GetFilterPlugins() []FilterPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Возвращаем копию среза
	result := make([]FilterPlugin, len(r.filterPlugins))
	copy(result, r.filterPlugins)
	return result
}

// GetFormatterPlugin возвращает плагин форматирования для указанного формата
func (r *Registry) GetFormatterPlugin(format model.OutputFormat) (FormatterPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.formatterPlugins[format]
	return plugin, exists
}

// GetAllPlugins возвращает все зарегистрированные плагины
func (r *Registry) GetAllPlugins() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Plugin, 0, len(r.pluginsByName))
	for _, plugin := range r.pluginsByName {
		result = append(result, plugin)
	}

	// Сортируем по приоритету
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority() > result[j].Priority()
	})

	return result
}

// GetPlugin возвращает плагин по имени
func (r *Registry) GetPlugin(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.pluginsByName[name]
	return plugin, exists
}

// GetStats возвращает статистику реестра
func (r *Registry) GetStats() *RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Возвращаем копию
	return &RegistryStats{
		TotalPlugins:   r.stats.TotalPlugins,
		LoadedPlugins:  r.stats.LoadedPlugins,
		FailedPlugins:  r.stats.FailedPlugins,
		ActivePlugins:  r.stats.ActivePlugins,
		ProcessedFiles: r.stats.ProcessedFiles,
		ProcessingTime: r.stats.ProcessingTime,
	}
}

// Private methods

func (r *Registry) registerLanguagePlugin(p LanguagePlugin) {
	for _, ext := range p.SupportedExtensions() {
		plugins := r.languagePlugins[ext]
		plugins = append(plugins, p)

		// Сортируем по приоритету
		sort.Slice(plugins, func(i, j int) bool {
			return plugins[i].Priority() > plugins[j].Priority()
		})

		r.languagePlugins[ext] = plugins
	}
}

func (r *Registry) registerProjectPlugin(p ProjectTypePlugin) {
	r.projectPlugins = append(r.projectPlugins, p)

	// Сортируем по приоритету
	sort.Slice(r.projectPlugins, func(i, j int) bool {
		return r.projectPlugins[i].Priority() > r.projectPlugins[j].Priority()
	})
}

func (r *Registry) registerFilterPlugin(p FilterPlugin) {
	r.filterPlugins = append(r.filterPlugins, p)

	// Сортируем по приоритету фильтра
	sort.Slice(r.filterPlugins, func(i, j int) bool {
		return r.filterPlugins[i].GetFilterPriority() > r.filterPlugins[j].GetFilterPriority()
	})
}

func (r *Registry) registerProcessorPlugin(p ProcessorPlugin) {
	r.processorPlugins = append(r.processorPlugins, p)

	sort.Slice(r.processorPlugins, func(i, j int) bool {
		return r.processorPlugins[i].Priority() > r.processorPlugins[j].Priority()
	})
}

func (r *Registry) registerFormatterPlugin(p FormatterPlugin) {
	for _, format := range p.SupportedFormats() {
		// Если уже есть плагин для этого формата, выбираем с большим приоритетом
		if existing, exists := r.formatterPlugins[format]; exists {
			if p.Priority() > existing.Priority() {
				r.formatterPlugins[format] = p
			}
		} else {
			r.formatterPlugins[format] = p
		}
	}
}

func (r *Registry) registerAnalyzerPlugin(p AnalyzerPlugin) {
	r.analyzerPlugins = append(r.analyzerPlugins, p)

	sort.Slice(r.analyzerPlugins, func(i, j int) bool {
		return r.analyzerPlugins[i].Priority() > r.analyzerPlugins[j].Priority()
	})
}

func (r *Registry) registerExtensionPlugin(p ExtensionPlugin) {
	r.extensionPlugins = append(r.extensionPlugins, p)

	sort.Slice(r.extensionPlugins, func(i, j int) bool {
		return r.extensionPlugins[i].Priority() > r.extensionPlugins[j].Priority()
	})
}

func (r *Registry) registerCompositePlugin(p CompositePlugin) {
	// Регистрируем каждую возможность отдельно
	if lang, ok := p.AsLanguagePlugin(); ok {
		r.registerLanguagePlugin(lang)
	}

	if proj, ok := p.AsProjectPlugin(); ok {
		r.registerProjectPlugin(proj)
	}

	if filter, ok := p.AsFilterPlugin(); ok {
		r.registerFilterPlugin(filter)
	}

	if proc, ok := p.AsProcessorPlugin(); ok {
		r.registerProcessorPlugin(proc)
	}
}

func (r *Registry) unregisterLanguagePlugin(p LanguagePlugin) {
	for _, ext := range p.SupportedExtensions() {
		plugins := r.languagePlugins[ext]
		for i, plugin := range plugins {
			if plugin.Name() == p.Name() {
				// Удаляем из среза
				r.languagePlugins[ext] = append(plugins[:i], plugins[i+1:]...)
				break
			}
		}
	}
}

func (r *Registry) unregisterProjectPlugin(p ProjectTypePlugin) {
	for i, plugin := range r.projectPlugins {
		if plugin.Name() == p.Name() {
			r.projectPlugins = append(r.projectPlugins[:i], r.projectPlugins[i+1:]...)
			break
		}
	}
}

func (r *Registry) unregisterFilterPlugin(p FilterPlugin) {
	for i, plugin := range r.filterPlugins {
		if plugin.Name() == p.Name() {
			r.filterPlugins = append(r.filterPlugins[:i], r.filterPlugins[i+1:]...)
			break
		}
	}
}

func (r *Registry) isDisabled(name string) bool {
	for _, disabled := range r.config.DisabledPlugins {
		if disabled == name {
			return true
		}
	}

	// Если есть список enabled и плагина там нет - он отключен
	if len(r.config.EnabledPlugins) > 0 {
		for _, enabled := range r.config.EnabledPlugins {
			if enabled == name {
				return false
			}
		}
		return true // Не в списке enabled
	}

	return false
}

// HealthCheck проверяет состояние всех плагинов
func (r *Registry) HealthCheck() map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	errors := make(map[string]error)

	for name, plugin := range r.pluginsByName {
		if lifecycle, ok := plugin.(PluginLifecycle); ok {
			if err := lifecycle.HealthCheck(); err != nil {
				errors[name] = err
			}
		}
	}

	return errors
}

// Reload перезагружает все плагины
func (r *Registry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Сохраняем текущие плагины
	oldPlugins := make([]Plugin, 0, len(r.pluginsByName))
	for _, p := range r.pluginsByName {
		oldPlugins = append(oldPlugins, p)
	}

	// Очищаем реестры
	r.languagePlugins = make(map[string][]LanguagePlugin)
	r.projectPlugins = nil
	r.filterPlugins = nil
	r.processorPlugins = nil
	r.formatterPlugins = make(map[model.OutputFormat]FormatterPlugin)
	r.analyzerPlugins = nil
	r.extensionPlugins = nil
	r.pluginsByName = make(map[string]Plugin)
	r.pluginPriorities = make(map[string]int)

	// Перерегистрируем плагины
	var errors []error
	for _, p := range oldPlugins {
		// Разблокируем для Register
		r.mu.Unlock()
		err := r.Register(p)
		r.mu.Lock()

		if err != nil {
			errors = append(errors, fmt.Errorf("failed to re-register %s: %w", p.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("reload completed with errors: %v", errors)
	}

	return nil
}