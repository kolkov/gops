package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/kolkov/gops/internal/app"
	"github.com/kolkov/gops/internal/config"
	"github.com/kolkov/gops/pkg/logger"
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

	outputFile := generateOutputFilename(cfg)

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

func generateOutputFilename(cfg *config.Config) string {
	if cfg.Output.Filename == "" {
		return fmt.Sprintf("project_docs_%s.md", time.Now().Format("20060102_150405"))
	}

	if cfg.Output.AppendTimestamp {
		ext := filepath.Ext(cfg.Output.Filename)
		base := cfg.Output.Filename[:len(cfg.Output.Filename)-len(ext)]
		return fmt.Sprintf("%s_%s%s", base, time.Now().Format("20060102_150405"), ext)
	}

	return cfg.Output.Filename
}
