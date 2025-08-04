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

	currentTime := time.Now()
	outputFile := markdown.GenerateOutputFilename(currentTime)

	docGenerator := markdown.NewDocumentationGenerator(outputFile)
	defer docGenerator.Close()

	projectScanner := scanner.NewProjectScanner(rootDir, outputFile)

	if err := projectScanner.InitializeScanner(); err != nil {
		fmt.Printf("Ошибка инициализации сканера: %v\n", err)
		return
	}

	projectScanner.AskContentSettings()

	if err := projectScanner.Scan(docGenerator); err != nil {
		fmt.Printf("Ошибка сканирования проекта: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nДокументация сохранена в %s\n", outputFile)
}
