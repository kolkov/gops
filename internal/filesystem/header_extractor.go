package filesystem

import (
	"bufio"
	"bytes"
	"fmt"
	"strings" // Удаляем ненужный импорт regexp

	"github.com/kolkov/gops/internal/model"
)

// ExtractFileHeader определяет язык и извлекает заголовочную информацию
func ExtractFileHeader(path string, content []byte, lang string) *model.FileHeader {
	switch lang {
	case "go":
		return ExtractGoHeader(content)
	case "javascript", "typescript":
		return ExtractJSHeader(content)
	case "python":
		return ExtractPythonHeader(content)
	case "java":
		return ExtractJavaHeader(content)
	default:
		return ExtractGenericHeader(content, lang)
	}
}

// ExtractGoHeader извлекает заголовочную информацию из Go-файла
func ExtractGoHeader(content []byte) *model.FileHeader {
	header := &model.FileHeader{
		ExportedFunctions: []string{},
		ExportedTypes:     []string{},
		ExportedConstants: []string{},
		Dependencies:      []string{},
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	inImports := false
	inMultiLineImport := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Обрабатываем импорты
		if strings.HasPrefix(line, "import") {
			inImports = true
			if strings.Contains(line, "(") {
				inMultiLineImport = true
			}
			continue
		}

		if inImports {
			if inMultiLineImport && line == ")" {
				inImports = false
				inMultiLineImport = false
				continue
			}
			if !inMultiLineImport && !strings.HasPrefix(line, "import") {
				inImports = false
			}

			// Извлекаем пути импортов
			if importPath := extractImportPath(line); importPath != "" {
				header.Dependencies = append(header.Dependencies, importPath)
			}
			continue
		}

		// Ищем экспортированные функции
		if strings.HasPrefix(line, "func") && isExported(line[5:]) {
			if fnName := extractFunctionName(line); fnName != "" {
				header.ExportedFunctions = append(header.ExportedFunctions, fnName)
			}
		}

		// Ищем экспортированные типы
		if strings.HasPrefix(line, "type") && isExported(line[5:]) {
			if typeName := extractTypeName(line); typeName != "" {
				header.ExportedTypes = append(header.ExportedTypes, typeName)
			}
		}

		// Ищем экспортированные константы
		if strings.HasPrefix(line, "const") && isExported(line[6:]) {
			if constName := extractConstantName(line); constName != "" {
				header.ExportedConstants = append(header.ExportedConstants, constName)
			}
		}
	}

	// Генерируем краткое описание на основе содержимого
	header.Summary = generateGoSummary(header)

	return header
}

// extractImportPath извлекает путь импорта из строки
func extractImportPath(line string) string {
	// Убираем кавычки и лишние пробелы
	line = strings.Trim(line, `"`)
	line = strings.TrimSpace(line)

	// Убираем алиасы импортов (например, `alias "path"`)
	if parts := strings.Fields(line); len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return line
}

// extractFunctionName извлекает имя функции
func extractFunctionName(line string) string {
	// Убираем ключевое слово "func"
	line = strings.TrimPrefix(line, "func")
	line = strings.TrimSpace(line)

	// Извлекаем имя до первой скобки
	if idx := strings.Index(line, "("); idx > 0 {
		return strings.TrimSpace(line[:idx])
	}

	return line
}

// extractTypeName извлекает имя типа
func extractTypeName(line string) string {
	// Убираем ключевое слово "type"
	line = strings.TrimPrefix(line, "type")
	line = strings.TrimSpace(line)

	// Извлекаем имя до пробела или знака равенства
	if idx := strings.IndexAny(line, " ="); idx > 0 {
		return strings.TrimSpace(line[:idx])
	}

	return line
}

// extractConstantName извлекает имя константы
func extractConstantName(line string) string {
	// Убираем ключевое слово "const"
	line = strings.TrimPrefix(line, "const")
	line = strings.TrimSpace(line)

	// Извлекаем имя до пробела или знака равенства
	if idx := strings.IndexAny(line, " ="); idx > 0 {
		return strings.TrimSpace(line[:idx])
	}

	return line
}

// isExported проверяет, является ли идентификатор экспортированным
func isExported(s string) bool {
	if len(s) == 0 {
		return false
	}
	firstChar := s[0]
	return firstChar >= 'A' && firstChar <= 'Z'
}

// generateGoSummary создает краткое описание файла
func generateGoSummary(header *model.FileHeader) string {
	parts := []string{}

	if len(header.ExportedTypes) > 0 {
		parts = append(parts, fmt.Sprintf("%d types", len(header.ExportedTypes)))
	}

	if len(header.ExportedFunctions) > 0 {
		parts = append(parts, fmt.Sprintf("%d functions", len(header.ExportedFunctions)))
	}

	if len(header.ExportedConstants) > 0 {
		parts = append(parts, fmt.Sprintf("%d constants", len(header.ExportedConstants)))
	}

	if len(parts) > 0 {
		return "Go file with " + strings.Join(parts, ", ")
	}

	return "Go source file"
}

// ExtractJSHeader заглушка для JavaScript/TypeScript
func ExtractJSHeader(content []byte) *model.FileHeader {
	return &model.FileHeader{
		Summary: "JavaScript/TypeScript file - header extraction not yet implemented",
	}
}

// ExtractPythonHeader заглушка для Python
func ExtractPythonHeader(content []byte) *model.FileHeader {
	return &model.FileHeader{
		Summary: "Python file - header extraction not yet implemented",
	}
}

// ExtractJavaHeader заглушка для Java
func ExtractJavaHeader(content []byte) *model.FileHeader {
	return &model.FileHeader{
		Summary: "Java file - header extraction not yet implemented",
	}
}

// ExtractGenericHeader заглушка для других языков
func ExtractGenericHeader(content []byte, lang string) *model.FileHeader {
	return &model.FileHeader{
		Summary: fmt.Sprintf("%s file - header extraction not yet implemented", lang),
	}
}
