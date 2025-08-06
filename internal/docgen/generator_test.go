package docgen

import (
	"testing"

	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewMarkdownGenerator(t *testing.T) {
	log := logger.New(logger.InfoLevel)
	gen := NewMarkdownGenerator("test.md", log)
	assert.NotNil(t, gen)
}

func TestGeneratorInterface(t *testing.T) {
	var gen Generator = &MockGenerator{}
	assert.NotNil(t, gen)
}
