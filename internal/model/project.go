package model

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type Duration time.Duration

const (
	MaxFileSize = 2 * 1024 * 1024 // 2MB
)

var (
	ErrFileTooLarge = errors.New("file size exceeds maximum limit")
)

type ProjectMeta struct {
	Name    string
	Type    string
	RootDir string
}

type ProjectFile struct {
	Path    string
	Content []byte
	Lang    string
	Skipped bool
}

type NxProject struct {
	Name      string
	Type      string
	Root      string
	SourceDir string
}

type ScanConfig struct {
	IncludeTests         bool
	IncludeConfigs       bool
	IncludeMarkup        bool
	IncludeStyles        bool
	ExcludedPatterns     []string
	MaxFileSize          int64
	ParallelWorkers      int
	OutputFilename       string
	OutputConfigFilename string
}

func GetFileLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".jsx":
		return "jsx"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".html", ".htm":
		return "html"
	case ".scss", ".sass":
		return "scss"
	case ".css":
		return "css"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".go":
		return "go"
	default:
		return "text"
	}
}
