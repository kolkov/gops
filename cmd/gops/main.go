package main

import (
	"fmt"
	"os"
	"time"

	"project_scanner/internal/markdown"
	"project_scanner/internal/scanner"
)

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Ошибка получения текущей директории: %v\n", err)
		return
	}

	// Генерация имени выходного файла
	currentTime := time.Now()
	outputFile := markdown.GenerateOutputFilename(currentTime)

	// Создание генератора документации
	docGenerator := markdown.NewDocumentationGenerator(outputFile)
	defer docGenerator.Close()

	// Создание сканера
	projectScanner := scanner.NewProjectScanner(rootDir, outputFile)

	// Запрос настроек документации
	projectScanner.AskContentSettings()

	// Сканирование проекта
	if err := projectScanner.Scan(docGenerator); err != nil {
		fmt.Printf("Ошибка сканирования проекта: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nДокументация сохранена в %s\n", outputFile)
}
