package scanner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/kolkov/gops/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrConflictingFiles   = errors.New("conflicting project files detected")
	ErrUnsupportedProject = errors.New("unsupported project type")
)

const (
	Go               = "go"
	JS               = "js"
	Angular          = "angular"
	BrowserExtension = "browser-extension"
	NxMonorepo       = "nx"
	Unknown          = "unknown"
)

type projectDetector struct {
	rootDir string
	logger  *logger.Logger
}

func (d *projectDetector) Detect() (string, error) {
	d.logger.Debug("Starting project detection",
		zap.String("directory", d.rootDir))

	// Проверка конфликтующих файлов
	if d.hasConflictingFiles() {
		d.logger.Error("Conflicting files found", zap.Error(ErrConflictingFiles))
		return "", ErrConflictingFiles
	}

	// Последовательность детектирования от специфичного к общему
	switch {
	case d.isNxMonorepo():
		d.logger.Info("Detected Nx monorepo")
		return NxMonorepo, nil

	case d.isBrowserExtension():
		d.logger.Info("Detected browser extension project")
		return BrowserExtension, nil

	case d.isAngularProject():
		d.logger.Info("Detected Angular project")
		return Angular, nil

	case d.isGoProject():
		d.logger.Info("Detected Go project")
		return Go, nil

	case d.isJSProject():
		d.logger.Info("Detected JavaScript project")
		return JS, nil
	}

	d.logger.Warn("Project type could not be determined")
	return Unknown, ErrUnsupportedProject
}

func (d *projectDetector) hasConflictingFiles() bool {
	goModPath := filepath.Join(d.rootDir, "go.mod")
	pkgJsonPath := filepath.Join(d.rootDir, "package.json")

	_, goModExists := os.Stat(goModPath)
	_, pkgJsonExists := os.Stat(pkgJsonPath)

	return goModExists == nil && pkgJsonExists == nil
}

func (d *projectDetector) isNxMonorepo() bool {
	// Проверка специфичных для Nx файлов
	nxFiles := []string{"nx.json", "workspace.json"}
	for _, file := range nxFiles {
		if _, err := os.Stat(filepath.Join(d.rootDir, file)); err == nil {
			return true
		}
	}

	// Проверка структуры директорий
	requiredDirs := []string{"apps", "libs"}
	dirCount := 0
	for _, dir := range requiredDirs {
		if fi, err := os.Stat(filepath.Join(d.rootDir, dir)); err == nil && fi.IsDir() {
			dirCount++
		}
	}
	return dirCount >= 2
}

func (d *projectDetector) isBrowserExtension() bool {
	// Основные файлы расширений браузера
	extensionFiles := []string{
		"manifest.json",
		"background.js",
		"content-script.js",
		"popup.html",
		"service-worker.js",
	}

	foundCount := 0
	for _, file := range extensionFiles {
		if _, err := os.Stat(filepath.Join(d.rootDir, file)); err == nil {
			foundCount++
		}
	}
	return foundCount >= 3
}

func (d *projectDetector) isAngularProject() bool {
	angularFiles := []string{
		"angular.json",
		"src/main.ts",
		"src/index.html",
	}

	foundCount := 0
	for _, file := range angularFiles {
		if _, err := os.Stat(filepath.Join(d.rootDir, file)); err == nil {
			foundCount++
		}
	}
	return foundCount >= 2 || hasDependency(d.rootDir, "@angular/core")
}

func (d *projectDetector) isGoProject() bool {
	// Проверка go.mod
	if _, err := os.Stat(filepath.Join(d.rootDir, "go.mod")); err == nil {
		return true
	}

	// Проверка наличия Go файлов в корне
	files, err := os.ReadDir(d.rootDir)
	if err != nil {
		return false
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name(), ".go") {
			return true
		}
	}
	return false
}

func (d *projectDetector) isJSProject() bool {
	// Проверка package.json
	if _, err := os.Stat(filepath.Join(d.rootDir, "package.json")); err == nil {
		return true
	}

	// Проверка наличия JS/TS файлов
	files, err := os.ReadDir(d.rootDir)
	if err != nil {
		return false
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := strings.ToLower(file.Name())
		switch {
		case strings.HasSuffix(name, ".js"),
			strings.HasSuffix(name, ".ts"),
			strings.HasSuffix(name, ".jsx"),
			strings.HasSuffix(name, ".tsx"),
			strings.HasSuffix(name, ".mjs"),
			strings.HasSuffix(name, ".cjs"):
			return true
		}
	}

	// Проверка вложенных директорий
	jsDirs := []string{"src", "app", "lib"}
	for _, dir := range jsDirs {
		if fi, err := os.Stat(filepath.Join(d.rootDir, dir)); err == nil && fi.IsDir() {
			return true
		}
	}

	return false
}
