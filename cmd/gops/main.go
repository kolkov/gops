package main

import (
	"context"
	"flag"
	"github.com/kolkov/gops/internal/app"
	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/pkg/logger"
	"log"
	"path/filepath"
)

func main() {
	cfgPath := flag.String("c", "./gops_config.yaml", "Path to config file")
	verbose := flag.Bool("v", false, "Verbose output")
	rootDir := flag.String("d", ".", "Project root directory")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	logLevel := logger.InfoLevel
	if *verbose {
		logLevel = logger.DebugLevel
	}
	zapLogger := logger.New(logLevel)
	defer zapLogger.Sync()

	absRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		zapLogger.Fatal("Failed to get absolute path", err)
	}

	outputFile := app.GenerateOutputFilename(cfg)

	scanner := app.NewProjectScanner(
		absRoot,
		outputFile,
		cfg,
		zapLogger,
	)

	if err := scanner.Run(ctx); err != nil {
		zapLogger.Fatal("Scan failed", err)
	}

	zapLogger.Info("Documentation generated successfully",
		"file", outputFile,
		"project", absRoot,
	)
}
