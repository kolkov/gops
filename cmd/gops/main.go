package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Ошибка получения текущей директории: %v\n", err)
		return
	}

	// Генерируем имя файла с датой и временем
	currentTime := time.Now()
	timeStamp := currentTime.Format("20060102_150405")
	outputFile := fmt.Sprintf("project_documentation_%s.md", timeStamp)

	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Ошибка создания файла: %v\n", err)
		return
	}
	defer file.Close()

	projectName := filepath.Base(rootDir)

	// Заголовок проекта
	file.WriteString(fmt.Sprintf("# Проект: %s\n\n", projectName))

	// Дата и время генерации с пояснением
	file.WriteString(fmt.Sprintf("**Дата генерации:** %s\n", currentTime.Format("2006-01-02 15:04:05")))
	file.WriteString("**Полнота:** Полная документация проекта\n\n")

	// Собираем структуру проекта
	file.WriteString("## Полная структура проекта\n\n")
	err = printProjectTree(file, rootDir, outputFile)
	if err != nil {
		fmt.Printf("Ошибка построения структуры проекта: %v\n", err)
	}
	file.WriteString("\n")

	var projectFiles []string
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем ненужные директории
		if info.IsDir() {
			base := filepath.Base(path)
			if shouldSkipDir(base) {
				return filepath.SkipDir
			}
			return nil
		}

		// Пропускаем сгенерированные файлы документации
		if isGeneratedFile(info.Name(), outputFile) {
			return nil
		}

		// Включаем нужные файлы
		if shouldIncludeFile(info.Name()) {
			relPath, _ := filepath.Rel(rootDir, path)
			projectFiles = append(projectFiles, relPath)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Ошибка сканирования директорий: %v\n", err)
		return
	}

	// Сортируем файлы по алфавиту
	sort.Strings(projectFiles)

	for _, filePath := range projectFiles {
		content, err := os.ReadFile(filepath.Join(rootDir, filePath))
		if err != nil {
			fmt.Printf("Ошибка чтения файла %s: %v\n", filePath, err)
			continue
		}

		// Определяем язык для подсветки
		lang := getFileLanguage(filePath)

		file.WriteString(fmt.Sprintf("## %s\n\n", filePath))
		file.WriteString(fmt.Sprintf("```%s\n", lang))
		file.Write(content)
		if len(content) > 0 && content[len(content)-1] != '\n' {
			file.WriteString("\n")
		}
		file.WriteString("```\n\n")
	}

	fmt.Printf("Документация сохранена в %s\n", outputFile)
}

// Функция для определения сгенерированных файлов
func isGeneratedFile(filename, currentOutput string) bool {
	// Исключаем текущий выходной файл
	if filename == filepath.Base(currentOutput) {
		return true
	}

	// Исключаем файлы документации по паттерну
	if strings.HasPrefix(filename, "project_documentation") &&
		strings.HasSuffix(filename, ".md") {
		return true
	}

	// Исключаем файлы структуры проекта
	if filename == "project_structure.txt" {
		return true
	}

	return false
}

// Функция для вывода структуры проекта в виде дерева
func printProjectTree(out *os.File, root, currentOutput string) error {
	// Собираем все элементы проекта
	type node struct {
		path     string
		children []*node
		isDir    bool
		level    int
	}

	rootNode := &node{path: ".", isDir: true, level: 0}
	nodeMap := map[string]*node{".": rootNode}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(root, path)
		if relPath == "." {
			return nil
		}

		// Пропускаем скрытые файлы/папки
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Пропускаем исключенные директории
		if info.IsDir() && shouldSkipDir(info.Name()) {
			return filepath.SkipDir
		}

		// Пропускаем сгенерированные файлы
		if !info.IsDir() && isGeneratedFile(info.Name(), currentOutput) {
			return nil
		}

		// Создаем узел
		parentPath := filepath.Dir(relPath)
		parent := nodeMap[parentPath]
		if parent == nil {
			return nil
		}

		n := &node{
			path:  relPath,
			isDir: info.IsDir(),
			level: parent.level + 1,
		}
		parent.children = append(parent.children, n)
		nodeMap[relPath] = n

		return nil
	})

	if err != nil {
		return err
	}

	// Рекурсивная функция для вывода дерева
	var printNode func(n *node, last bool)
	printNode = func(n *node, last bool) {
		// Выводим отступы для текущего уровня
		for i := 0; i < n.level-1; i++ {
			out.WriteString("│   ")
		}

		if n.level > 0 {
			if last {
				out.WriteString("└── ")
			} else {
				out.WriteString("├── ")
			}
		}

		// Выводим имя
		name := filepath.Base(n.path)
		if n.isDir {
			out.WriteString("**" + name + "**")
		} else {
			// Выделяем важные файлы
			switch name {
			case "go.mod", "go.sum", "main.go":
				out.WriteString("**" + name + "**")
			default:
				out.WriteString(name)
			}
		}
		out.WriteString("\n")

		// Сортируем дочерние элементы: сначала папки, потом файлы
		sort.Slice(n.children, func(i, j int) bool {
			if n.children[i].isDir && !n.children[j].isDir {
				return true
			}
			if !n.children[i].isDir && n.children[j].isDir {
				return false
			}
			return n.children[i].path < n.children[j].path
		})

		// Рекурсивно выводим детей
		for i, child := range n.children {
			printNode(child, i == len(n.children)-1)
		}
	}

	// Выводим дерево
	out.WriteString("```\n")
	printNode(rootNode, true)
	out.WriteString("```\n")
	return nil
}

func shouldSkipDir(name string) bool {
	skipDirs := []string{
		"vendor", "node_modules", ".git", ".vscode", ".idea", "dist",
		"build", "__pycache__", "bin", "obj", "testdata", "coverage",
	}
	for _, dir := range skipDirs {
		if name == dir {
			return true
		}
	}
	return false
}

func shouldIncludeFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".go" || name == "go.mod" || ext == ".py"
}

func getFileLanguage(path string) string {
	switch {
	case strings.HasSuffix(path, ".mod"):
		return "mod"
	case strings.HasSuffix(path, ".sum"):
		return "mod"
	default:
		return "go"
	}
}
