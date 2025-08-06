package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kolkov/gops/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectDetector(t *testing.T) {
	log := logger.New(logger.InfoLevel)

	t.Run("ConflictingFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte{}, 0644))

		detector := NewProjectDetector(tmpDir, log)
		_, err := detector.Detect()
		assert.ErrorIs(t, err, ErrConflictingFiles)
	})

	t.Run("NxMonorepo", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "apps"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "libs"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "nx.json"), []byte{}, 0644))

		detector := NewProjectDetector(tmpDir, log)
		projectType, err := detector.Detect()
		require.NoError(t, err)
		assert.Equal(t, NxMonorepo, projectType)
	})

	t.Run("GoProject", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte{}, 0644))

		detector := NewProjectDetector(tmpDir, log)
		projectType, err := detector.Detect()
		require.NoError(t, err)
		assert.Equal(t, Go, projectType)
	})

	t.Run("JSProject", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "index.js"), []byte{}, 0644))

		detector := NewProjectDetector(tmpDir, log)
		projectType, err := detector.Detect()
		require.NoError(t, err)
		assert.Equal(t, JS, projectType)
	})

	t.Run("AngularProject", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src", "app"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "angular.json"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "main.ts"), []byte{}, 0644))

		detector := NewProjectDetector(tmpDir, log)
		projectType, err := detector.Detect()
		require.NoError(t, err)
		assert.Equal(t, Angular, projectType)
	})

	t.Run("BrowserExtension", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "background.js"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "popup.html"), []byte{}, 0644))

		detector := NewProjectDetector(tmpDir, log)
		projectType, err := detector.Detect()
		require.NoError(t, err)
		assert.Equal(t, BrowserExtension, projectType)
	})

	t.Run("UnsupportedProject", func(t *testing.T) {
		tmpDir := t.TempDir()
		detector := NewProjectDetector(tmpDir, log)
		_, err := detector.Detect()
		assert.ErrorIs(t, err, ErrUnsupportedProject)
	})
}

func TestJSFrameworkDetection(t *testing.T) {
	t.Run("ReactProject", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "App.jsx"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"react": "^18.0.0"}}`), 0644))

		framework := DetectJSFramework(tmpDir)
		assert.Equal(t, "React", framework)
	})

	t.Run("VueProject", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)) // Создаем директорию
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "App.vue"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"vue": "^3.0.0"}}`), 0644))

		framework := DetectJSFramework(tmpDir)
		assert.Equal(t, "Vue", framework)
	})

	t.Run("SvelteProject", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "svelte.config.js"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"svelte": "^3.0.0"}}`), 0644))

		framework := DetectJSFramework(tmpDir)
		assert.Equal(t, "Svelte", framework)
	})

	t.Run("PlainJS", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "index.js"), []byte{}, 0644))
		framework := DetectJSFramework(tmpDir)
		assert.Equal(t, "JavaScript", framework)
	})
}
