package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectJSFramework(t *testing.T) {
	t.Run("ReactDetection", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"react": "^18.0.0"}}`), 0644))
		framework := DetectJSFramework(tmpDir)
		assert.Equal(t, "React", framework)
	})

	t.Run("VueDetection", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"vue": "^3.0.0"}}`), 0644))
		framework := DetectJSFramework(tmpDir)
		assert.Equal(t, "Vue", framework)
	})

	// Обновим тест для Angular
	t.Run("AngularDetection", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "angular.json"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "main.ts"), []byte{}, 0644)) // Добавляем второй файл
		framework := DetectJSFramework(tmpDir)
		assert.Equal(t, "Angular", framework)
	})

	t.Run("SvelteDetection", func(t *testing.T) {
		tmpDir := t.TempDir()
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
