// Package phases содержит фазу определения типа проекта
package phases

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/internal/plugin"
	"github.com/kolkov/gops/pkg/logger"
)

// DetectionPhase - фаза определения типа проекта
type DetectionPhase struct {
	registry *plugin.Registry
	logger   *logger.Logger
}

// DetectionResult - результат определения типа проекта
type DetectionResult struct {
	Plugin     plugin.ProjectTypePlugin
	Confidence float32
	Scores     map[string]float32 // Все оценки для отладки
}

// NewDetectionPhase создает новую фазу определения
func NewDetectionPhase(registry *plugin.Registry, logger *logger.Logger) *DetectionPhase {
	return &DetectionPhase{
		registry: registry,
		logger:   logger,
	}
}

// Detect определяет тип проекта
func (d *DetectionPhase) Detect(ctx context.Context, metadata *model.ProjectMetadata) (plugin.ProjectTypePlugin, float32, error) {
	d.logger.Info("Starting project type detection")

	// Получаем проектный плагин из реестра
	projectPlugin, confidence, err := d.registry.GetProjectPlugin(metadata)
	if err != nil {
		// Если не удалось определить, пробуем альтернативные методы
		d.logger.Warn("Primary detection failed, trying alternatives", "error", err)

		result := d.detectWithAllPlugins(ctx, metadata)
		if result.Plugin != nil {
			d.logger.Info("Project type detected via alternative method",
				"type", result.Plugin.Name(),
				"confidence", result.Confidence)
			return result.Plugin, result.Confidence, nil
		}

		return nil, 0, fmt.Errorf("failed to detect project type: %w", err)
	}

	d.logger.Info("Project type detected",
		"type", projectPlugin.Name(),
		"confidence", confidence)

	return projectPlugin, confidence, nil
}

// DetectMultiple определяет несколько возможных типов проекта
func (d *DetectionPhase) DetectMultiple(ctx context.Context, metadata *model.ProjectMetadata) []DetectionResult {
	results := make([]DetectionResult, 0)

	// Получаем все проектные плагины
	plugins := d.registry.GetAllPlugins()

	for _, p := range plugins {
		if projectPlugin, ok := p.(plugin.ProjectTypePlugin); ok {
			confidence, err := projectPlugin.Detect(metadata)
			if err != nil {
				d.logger.Debug("Plugin detection failed",
					"plugin", projectPlugin.Name(),
					"error", err)
				continue
			}

			if confidence > 0 {
				results = append(results, DetectionResult{
					Plugin:     projectPlugin,
					Confidence: confidence,
				})
			}
		}
	}

	// Сортируем по confidence
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	return results
}

// detectWithAllPlugins пробует определить тип проекта всеми доступными плагинами
func (d *DetectionPhase) detectWithAllPlugins(ctx context.Context, metadata *model.ProjectMetadata) *DetectionResult {
	results := d.DetectMultiple(ctx, metadata)

	if len(results) > 0 {
		// Возвращаем результат с наивысшей confidence
		return &results[0]
	}

	// Пробуем эвристическое определение
	return d.detectHeuristic(metadata)
}

// detectHeuristic использует эвристики для определения типа проекта
func (d *DetectionPhase) detectHeuristic(metadata *model.ProjectMetadata) *DetectionResult {
	d.logger.Debug("Using heuristic detection")

	result := &DetectionResult{
		Scores: make(map[string]float32),
	}

	// Анализируем файлы-маркеры
	markerScores := d.analyzeMarkerFiles(metadata)
	for name, score := range markerScores {
		result.Scores[name] = score
	}

	// Анализируем расширения файлов
	extensionScores := d.analyzeFileExtensions(metadata)
	for name, score := range extensionScores {
		if existing, ok := result.Scores[name]; ok {
			result.Scores[name] = existing + score
		} else {
			result.Scores[name] = score
		}
	}

	// Анализируем структуру директорий
	structureScores := d.analyzeDirectoryStructure(metadata)
	for name, score := range structureScores {
		if existing, ok := result.Scores[name]; ok {
			result.Scores[name] = existing + score
		} else {
			result.Scores[name] = score
		}
	}

	// Находим тип с максимальной оценкой
	var bestType string
	var bestScore float32
	for typeStr, score := range result.Scores {
		if score > bestScore {
			bestScore = score
			bestType = typeStr
		}
	}

	// Пытаемся найти соответствующий плагин
	if bestType != "" && bestScore > 0.3 {
		// Ищем плагин по имени типа
		for _, p := range d.registry.GetAllPlugins() {
			if projectPlugin, ok := p.(plugin.ProjectTypePlugin); ok {
				// Простое сравнение имен (можно улучшить)
				if d.matchesProjectType(projectPlugin, bestType) {
					result.Plugin = projectPlugin
					result.Confidence = bestScore
					return result
				}
			}
		}
	}

	return result
}

// analyzeMarkerFiles анализирует файлы-маркеры
func (d *DetectionPhase) analyzeMarkerFiles(metadata *model.ProjectMetadata) map[string]float32 {
	scores := make(map[string]float32)

	// Маркеры для разных типов проектов
	markers := map[string][]struct {
		file   string
		weight float32
	}{
		"go": {
			{"go.mod", 0.5},
			{"go.sum", 0.2},
			{"main.go", 0.1},
		},
		"nodejs": {
			{"package.json", 0.4},
			{"node_modules", 0.1},
			{"package-lock.json", 0.1},
			{"yarn.lock", 0.1},
		},
		"python": {
			{"requirements.txt", 0.3},
			{"setup.py", 0.3},
			{"setup.cfg", 0.2},
			{"pyproject.toml", 0.3},
			{"Pipfile", 0.2},
		},
		"java": {
			{"pom.xml", 0.4},
			{"build.gradle", 0.4},
			{"settings.gradle", 0.2},
		},
		"rust": {
			{"Cargo.toml", 0.5},
			{"Cargo.lock", 0.2},
		},
		"c": {
			{"Makefile", 0.2},
			{"makefile", 0.2},
			{"CMakeLists.txt", 0.3},
			{"configure", 0.2},
		},
		"kernel": {
			{"Kconfig", 0.4},
			{"Kbuild", 0.3},
			{"MAINTAINERS", 0.3},
			{"arch", 0.2},
			{"drivers", 0.2},
			{"kernel", 0.2},
		},
		"nx": {
			{"nx.json", 0.5},
			{"workspace.json", 0.3},
			{"apps", 0.2},
			{"libs", 0.2},
		},
		"angular": {
			{"angular.json", 0.5},
			{".angular", 0.2},
			{"src/app", 0.1},
		},
		"react": {
			{"src/App.jsx", 0.2},
			{"src/App.tsx", 0.2},
			{"public/index.html", 0.1},
		},
		"vue": {
			{"vue.config.js", 0.3},
			{"src/App.vue", 0.2},
		},
	}

	// Проверяем наличие маркерных файлов
	for projectType, markerList := range markers {
		var score float32
		for _, marker := range markerList {
			// Проверяем в индексе файлов
			for path := range metadata.Files {
				if path == marker.file || filepath.Base(path) == marker.file {
					score += marker.weight
					break
				}
			}
		}
		if score > 0 {
			scores[projectType] = score
		}
	}

	return scores
}

// analyzeFileExtensions анализирует расширения файлов
func (d *DetectionPhase) analyzeFileExtensions(metadata *model.ProjectMetadata) map[string]float32 {
	scores := make(map[string]float32)

	totalFiles := float32(metadata.TotalFiles)
	if totalFiles == 0 {
		return scores
	}

	// Анализируем преобладающие расширения
	for ext, count := range metadata.FilesByExt {
		ratio := float32(count) / totalFiles

		switch ext {
		case ".go":
			scores["go"] += ratio * 0.5
		case ".js", ".jsx", ".mjs", ".cjs":
			scores["nodejs"] += ratio * 0.3
		case ".ts", ".tsx":
			scores["typescript"] += ratio * 0.3
			scores["nodejs"] += ratio * 0.2
		case ".py", ".pyw":
			scores["python"] += ratio * 0.4
		case ".java":
			scores["java"] += ratio * 0.4
		case ".rs":
			scores["rust"] += ratio * 0.5
		case ".c", ".h":
			scores["c"] += ratio * 0.3
			// Может быть и ядро
			if metadata.TotalFiles > 1000 {
				scores["kernel"] += ratio * 0.1
			}
		case ".cpp", ".hpp", ".cc", ".cxx":
			scores["cpp"] += ratio * 0.3
		case ".cs":
			scores["dotnet"] += ratio * 0.4
		case ".rb":
			scores["ruby"] += ratio * 0.4
		case ".php":
			scores["php"] += ratio * 0.4
		}
	}

	return scores
}

// analyzeDirectoryStructure анализирует структуру директорий
func (d *DetectionPhase) analyzeDirectoryStructure(metadata *model.ProjectMetadata) map[string]float32 {
	scores := make(map[string]float32)

	// Проверяем характерные директории
	dirs := make(map[string]bool)
	for path := range metadata.Files {
		dir := filepath.Dir(path)
		parts := strings.Split(dir, string(filepath.Separator))
		for _, part := range parts {
			dirs[part] = true
		}
	}

	// Паттерны директорий для разных типов проектов
	patterns := map[string][]string{
		"go":     {"cmd", "pkg", "internal", "vendor"},
		"nodejs": {"node_modules", "src", "dist", "public"},
		"java":   {"src/main/java", "src/test/java", "target"},
		"python": {"tests", "docs", "scripts", "__pycache__"},
		"kernel": {"arch", "drivers", "fs", "kernel", "mm", "net"},
		"nx":     {"apps", "libs", "tools", "dist/apps", "dist/libs"},
	}

	for projectType, dirList := range patterns {
		matches := 0
		for _, dir := range dirList {
			if dirs[dir] {
				matches++
			}
		}
		if matches > 0 {
			scores[projectType] = float32(matches) / float32(len(dirList)) * 0.3
		}
	}

	return scores
}

// matchesProjectType проверяет соответствие плагина типу проекта
func (d *DetectionPhase) matchesProjectType(plugin plugin.ProjectTypePlugin, projectType string) bool {
	pluginName := strings.ToLower(plugin.Name())
	projectType = strings.ToLower(projectType)

	// Простое сравнение и частичное совпадение
	if pluginName == projectType {
		return true
	}

	// Проверяем, содержит ли имя плагина тип проекта
	if strings.Contains(pluginName, projectType) {
		return true
	}

	// Специальные случаи
	switch projectType {
	case "nodejs":
		return strings.Contains(pluginName, "node") ||
			strings.Contains(pluginName, "javascript") ||
			strings.Contains(pluginName, "js")
	case "kernel":
		return strings.Contains(pluginName, "linux") ||
			strings.Contains(pluginName, "kernel")
	case "cpp":
		return strings.Contains(pluginName, "c++") ||
			strings.Contains(pluginName, "cpp")
	}

	return false
}

// GetProjectLanguages определяет используемые языки в проекте
func (d *DetectionPhase) GetProjectLanguages(metadata *model.ProjectMetadata) []string {
	languages := make(map[string]int)

	// Подсчитываем файлы по языкам
	for ext, count := range metadata.FilesByExt {
		lang := d.extensionToLanguage(ext)
		if lang != "" {
			languages[lang] += count
		}
	}

	// Сортируем языки по количеству файлов
	type langCount struct {
		lang  string
		count int
	}

	sorted := make([]langCount, 0, len(languages))
	for lang, count := range languages {
		sorted = append(sorted, langCount{lang, count})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// Возвращаем список языков
	result := make([]string, 0, len(sorted))
	for _, lc := range sorted {
		result = append(result, lc.lang)
	}

	return result
}

// extensionToLanguage конвертирует расширение файла в язык
func (d *DetectionPhase) extensionToLanguage(ext string) string {
	switch strings.ToLower(ext) {
	case ".go":
		return "Go"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "JavaScript"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".py", ".pyw":
		return "Python"
	case ".java":
		return "Java"
	case ".rs":
		return "Rust"
	case ".c", ".h":
		return "C"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hxx":
		return "C++"
	case ".cs":
		return "C#"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".swift":
		return "Swift"
	case ".kt", ".kts":
		return "Kotlin"
	case ".m", ".mm":
		return "Objective-C"
	case ".scala":
		return "Scala"
	case ".r", ".R":
		return "R"
	case ".lua":
		return "Lua"
	case ".pl", ".pm":
		return "Perl"
	case ".sh", ".bash", ".zsh":
		return "Shell"
	case ".sql":
		return "SQL"
	case ".html", ".htm":
		return "HTML"
	case ".css", ".scss", ".sass":
		return "CSS"
	case ".xml":
		return "XML"
	case ".yaml", ".yml":
		return "YAML"
	case ".json":
		return "JSON"
	case ".md", ".markdown":
		return "Markdown"
	default:
		return ""
	}
}