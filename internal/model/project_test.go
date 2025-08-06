package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFileLanguage(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"main.go", "go"},
		{"script.js", "javascript"},
		{"component.tsx", "tsx"},
		{"styles.css", "css"},
		{"styles.scss", "scss"},
		{"index.html", "html"},
		{"config.json", "json"},
		{"README.md", "markdown"},
		{"Dockerfile", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetFileLanguage(tt.file))
		})
	}
}
