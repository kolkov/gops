package filesystem

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
	"golang.org/x/sync/semaphore"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

func ReadFile(path string, maxSize int64) ([]byte, error) {
	if strings.HasSuffix(strings.ToLower(path), "gops_config.yaml") {
		return nil, fmt.Errorf("gops config file should be skipped")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.Size() > maxSize {
		return nil, model.ErrFileTooLarge
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(buf, file)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ShouldSkipFile(name, outputConfigFilename, outputFilename string, excludedPatterns, importantFiles []string) bool {
	if name == "gops_config.yaml" {
		return true
	}

	if name == "go.sum" {
		return true
	}

	if name == filepath.Base(outputFilename) {
		return true
	}

	baseName := strings.TrimSuffix(outputConfigFilename, filepath.Ext(outputConfigFilename))
	docPattern := baseName + "*.md"
	if matched, _ := filepath.Match(docPattern, name); matched {
		return true
	}

	systemExcludes := []string{
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
	}
	for _, excl := range systemExcludes {
		if name == excl {
			return true
		}
	}

	if strings.HasPrefix(name, "~$") {
		return true
	}

	for _, pattern := range excludedPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	for _, important := range importantFiles {
		if name == important {
			return false
		}
	}

	return false
}

func shouldSkipDir(name string, excludedPatterns []string) bool {
	systemExcludes := []string{
		".idea", ".vscode", ".git", "node_modules",
		"vendor", "dist", "build", "__pycache__",
		"bin", "obj", "coverage", ".angular",
		".nx", "target", "out", "__tests__",
		"__snapshots__", ".next", ".nuxt", ".cache",
		"cypress", "e2e",
	}
	for _, excl := range systemExcludes {
		if name == excl {
			return true
		}
	}

	for _, pattern := range excludedPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

func shouldSkipByType(path string, cfg *model.ScanConfig) bool {
	for _, important := range cfg.ImportantFiles {
		if strings.HasSuffix(path, important) {
			return false
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	fileName := strings.ToLower(filepath.Base(path))

	switch {
	case !cfg.IncludeStyles && (ext == ".css" || ext == ".scss" || ext == ".sass" || ext == ".less"):
		return true
	case !cfg.IncludeMarkup && (ext == ".html" || ext == ".htm"):
		return true
	case !cfg.IncludeConfigs && (ext == ".json" || ext == ".yaml" || ext == ".yml" || strings.Contains(fileName, "config")):
		return true
	case !cfg.IncludeTests && (strings.Contains(fileName, ".spec.") || strings.Contains(fileName, ".test.") || strings.Contains(filepath.Dir(path), "test")):
		return true
	default:
		return false
	}
}

func ScanProject(root string, cfg *model.ScanConfig, logger *logger.Logger, processFile func(*model.ProjectFile)) error {
	sem := semaphore.NewWeighted(int64(cfg.ParallelWorkers))
	ctx := context.Background()

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if shouldSkipDir(info.Name(), cfg.ExcludedPatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if ShouldSkipFile(info.Name(), cfg.OutputConfigFilename, cfg.OutputFilename, cfg.ExcludedPatterns, cfg.ImportantFiles) {
			return nil
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		go func() {
			defer sem.Release(1)
			file := processSingleFile(path, root, cfg)
			processFile(file)
		}()

		return nil
	})

	if err := sem.Acquire(ctx, int64(cfg.ParallelWorkers)); err != nil {
		logger.Error("Failed to acquire semaphore", err)
	}

	return err
}

func processSingleFile(path, root string, cfg *model.ScanConfig) *model.ProjectFile {
	relPath, _ := filepath.Rel(root, path)
	file := &model.ProjectFile{
		Path: relPath,
		Lang: model.GetFileLanguage(path),
	}

	for _, important := range cfg.ImportantFiles {
		if strings.HasSuffix(path, important) {
			content, err := ReadFile(path, cfg.MaxFileSize)
			if err != nil {
				file.Skipped = true
			} else {
				file.Content = content
			}
			return file
		}
	}

	if info, err := os.Stat(path); err == nil && info.Size() > cfg.MaxFileSize {
		file.Skipped = true
		return file
	}

	if shouldSkipByType(path, cfg) {
		file.Skipped = true
		return file
	}

	content, err := ReadFile(path, cfg.MaxFileSize)
	if err != nil {
		file.Skipped = true
		return file
	}

	file.Content = content
	return file
}
