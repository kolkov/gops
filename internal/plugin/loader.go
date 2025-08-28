// Package plugin содержит загрузчик плагинов
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader - загрузчик плагинов
type Loader struct {
	registry    *Registry
	config      *LoaderConfig
	loadedPaths map[string]bool
}

// LoaderConfig - конфигурация загрузчика
type LoaderConfig struct {
	// Директории для поиска плагинов
	PluginDirs []string `yaml:"plugin_dirs"`

	// Встроенные плагины
	BuiltinOnly bool `yaml:"builtin_only"`

	// Внешние плагины (.so файлы)
	EnableExternal bool     `yaml:"enable_external"`
	ExternalPaths  []string `yaml:"external_paths"`

	// Конфигурационный файл плагинов
	ConfigFile string `yaml:"config_file"`

	// Автозагрузка
	AutoDiscover bool `yaml:"auto_discover"`

	// Фильтры
	IncludePatterns []string `yaml:"include_patterns"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

// PluginConfig - конфигурация отдельного плагина
type PluginConfig struct {
	Name     string                 `yaml:"name"`
	Enabled  bool                   `yaml:"enabled"`
	Priority int                    `yaml:"priority"`
	Config   map[string]interface{} `yaml:"config"`
}

// PluginsConfig - конфигурация всех плагинов
type PluginsConfig struct {
	Version      string                    `yaml:"version"`
	Plugins      map[string]*PluginConfig  `yaml:"plugins"`
	GlobalConfig map[string]interface{}    `yaml:"global_config"`
}

// NewLoader создает новый загрузчик плагинов
func NewLoader(registry *Registry, config *LoaderConfig) *Loader {
	if config == nil {
		config = &LoaderConfig{
			PluginDirs: []string{
				"./plugins",
				"~/.gops/plugins",
				"/usr/local/lib/gops/plugins",
			},
			AutoDiscover: true,
		}
	}

	return &Loader{
		registry:    registry,
		config:      config,
		loadedPaths: make(map[string]bool),
	}
}

// LoadAll загружает все доступные плагины
func (l *Loader) LoadAll() error {
	var errors []error

	// Загружаем конфигурацию плагинов
	pluginsConfig, err := l.loadPluginsConfig()
	if err != nil && l.config.ConfigFile != "" {
		errors = append(errors, fmt.Errorf("failed to load plugins config: %w", err))
	}

	// Применяем конфигурацию к реестру
	if pluginsConfig != nil {
		l.applyPluginsConfig(pluginsConfig)
	}

	// Загружаем встроенные плагины
	if err := l.LoadBuiltin(); err != nil {
		errors = append(errors, fmt.Errorf("failed to load builtin plugins: %w", err))
	}

	// Если только встроенные - выходим
	if l.config.BuiltinOnly {
		return l.combineErrors(errors)
	}

	// Загружаем плагины из директорий
	if l.config.AutoDiscover {
		for _, dir := range l.config.PluginDirs {
			expandedDir := l.expandPath(dir)
			if err := l.LoadFromDirectory(expandedDir); err != nil {
				errors = append(errors, fmt.Errorf("failed to load from %s: %w", dir, err))
			}
		}
	}

	// Загружаем внешние плагины
	if l.config.EnableExternal {
		for _, path := range l.config.ExternalPaths {
			expandedPath := l.expandPath(path)
			if err := l.LoadExternal(expandedPath); err != nil {
				errors = append(errors, fmt.Errorf("failed to load external plugin %s: %w", path, err))
			}
		}
	}

	return l.combineErrors(errors)
}

// LoadBuiltin загружает встроенные плагины
func (l *Loader) LoadBuiltin() error {
	// Импортируем встроенные плагины
	// Они автоматически регистрируются через init()

	// Это будет сделано через импорты в отдельном файле builtin.go
	// который импортирует все встроенные плагины

	// Пример:
	// _ "github.com/kolkov/gops/internal/plugins/languages/go"
	// _ "github.com/kolkov/gops/internal/plugins/languages/js"
	// _ "github.com/kolkov/gops/internal/plugins/projects/nx"

	return nil
}

// LoadFromDirectory загружает плагины из директории
func (l *Loader) LoadFromDirectory(dir string) error {
	// Проверяем существование директории
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Директория не существует - не ошибка
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	var errors []error

	// Рекурсивно обходим директорию
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем директории
		if info.IsDir() {
			return nil
		}

		// Проверяем фильтры
		if !l.shouldLoad(path) {
			return nil
		}

		// Определяем тип файла и загружаем
		switch {
		case strings.HasSuffix(path, ".so"):
			// Внешний плагин
			if l.config.EnableExternal {
				if err := l.LoadExternal(path); err != nil {
					errors = append(errors, fmt.Errorf("failed to load %s: %w", path, err))
				}
			}

		case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml"):
			// Конфигурация плагина - пропускаем
			return nil

		default:
			// Неизвестный тип файла
			return nil
		}

		return nil
	})

	if err != nil {
		errors = append(errors, err)
	}

	return l.combineErrors(errors)
}

// LoadExternal загружает внешний плагин (.so файл)
func (l *Loader) LoadExternal(path string) error {
	// Проверяем, не загружен ли уже
	if l.loadedPaths[path] {
		return nil
	}

	// Открываем плагин
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Ищем функцию GetPlugin
	getPluginSymbol, err := p.Lookup("GetPlugin")
	if err != nil {
		return fmt.Errorf("plugin %s missing GetPlugin function: %w", path, err)
	}

	// Проверяем тип функции
	getPlugin, ok := getPluginSymbol.(func() Plugin)
	if !ok {
		// Пробуем альтернативные сигнатуры
		if getPluginMulti, ok := getPluginSymbol.(func() []Plugin); ok {
			// Регистрируем несколько плагинов
			plugins := getPluginMulti()
			for _, plugin := range plugins {
				if err := l.registry.Register(plugin); err != nil {
					return fmt.Errorf("failed to register plugin from %s: %w", path, err)
				}
			}
			l.loadedPaths[path] = true
			return nil
		}

		return fmt.Errorf("invalid GetPlugin function signature in %s", path)
	}

	// Получаем и регистрируем плагин
	plugin := getPlugin()
	if err := l.registry.Register(plugin); err != nil {
		return fmt.Errorf("failed to register plugin from %s: %w", path, err)
	}

	l.loadedPaths[path] = true
	return nil
}

// LoadPlugin загружает конкретный плагин по имени
func (l *Loader) LoadPlugin(name string) error {
	// Сначала проверяем встроенные
	if plugin := l.findBuiltinPlugin(name); plugin != nil {
		return l.registry.Register(plugin)
	}

	// Ищем во внешних
	for _, dir := range l.config.PluginDirs {
		pluginPath := filepath.Join(l.expandPath(dir), name+".so")
		if _, err := os.Stat(pluginPath); err == nil {
			return l.LoadExternal(pluginPath)
		}
	}

	return fmt.Errorf("plugin %s not found", name)
}

// UnloadPlugin выгружает плагин
func (l *Loader) UnloadPlugin(name string) error {
	return l.registry.Unregister(name)
}

// ReloadPlugin перезагружает плагин
func (l *Loader) ReloadPlugin(name string) error {
	if err := l.UnloadPlugin(name); err != nil {
		return fmt.Errorf("failed to unload: %w", err)
	}

	if err := l.LoadPlugin(name); err != nil {
		return fmt.Errorf("failed to load: %w", err)
	}

	return nil
}

// Private methods

func (l *Loader) loadPluginsConfig() (*PluginsConfig, error) {
	if l.config.ConfigFile == "" {
		return nil, nil
	}

	configPath := l.expandPath(l.config.ConfigFile)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config PluginsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (l *Loader) applyPluginsConfig(config *PluginsConfig) {
	if config == nil {
		return
	}

	// Применяем глобальную конфигурацию
	if config.GlobalConfig != nil {
		// TODO: применить к реестру
	}

	// Применяем конфигурации плагинов
	for name, pluginConfig := range config.Plugins {
		if !pluginConfig.Enabled {
			l.registry.config.DisabledPlugins = append(
				l.registry.config.DisabledPlugins,
				name,
			)
		}

		if pluginConfig.Config != nil {
			if l.registry.config.PluginConfigs == nil {
				l.registry.config.PluginConfigs = make(map[string]map[string]interface{})
			}
			l.registry.config.PluginConfigs[name] = pluginConfig.Config
		}
	}
}

func (l *Loader) shouldLoad(path string) bool {
	name := filepath.Base(path)

	// Проверяем include patterns
	if len(l.config.IncludePatterns) > 0 {
		matched := false
		for _, pattern := range l.config.IncludePatterns {
			if match, _ := filepath.Match(pattern, name); match {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Проверяем exclude patterns
	for _, pattern := range l.config.ExcludePatterns {
		if match, _ := filepath.Match(pattern, name); match {
			return false
		}
	}

	return true
}

func (l *Loader) expandPath(path string) string {
	// Раскрываем ~ в домашнюю директорию
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	// Раскрываем переменные окружения
	path = os.ExpandEnv(path)

	return path
}

func (l *Loader) findBuiltinPlugin(name string) Plugin {
	// TODO: реализовать поиск встроенных плагинов по имени
	// Это будет сделано через регистрацию в init()
	return nil
}

func (l *Loader) combineErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	if len(errors) == 1 {
		return errors[0]
	}

	var msgs []string
	for _, err := range errors {
		msgs = append(msgs, err.Error())
	}

	return fmt.Errorf("multiple errors: %s", strings.Join(msgs, "; "))
}

// AutoRegister автоматически находит и регистрирует все доступные плагины
func AutoRegister() error {
	registry := GetRegistry()
	loader := NewLoader(registry, nil)
	return loader.LoadAll()
}

// RegisterBuiltinPlugins регистрирует все встроенные плагины
// Вызывается из init() функций плагинов
func RegisterBuiltinPlugins(plugins ...Plugin) {
	registry := GetRegistry()
	for _, p := range plugins {
		if err := registry.Register(p); err != nil {
			// Логируем ошибку, но не паникуем
			fmt.Fprintf(os.Stderr, "Failed to register builtin plugin %s: %v\n", p.Name(), err)
		}
	}
}