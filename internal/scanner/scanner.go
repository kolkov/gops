package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectJSFramework определяет JavaScript фреймворк проекта
func DetectJSFramework(rootDir string) string {
	// Проверка Angular
	if isAngularProject(rootDir) {
		return "Angular"
	}

	// Проверка React
	if isReactProject(rootDir) {
		return "React"
	}

	// Проверка Vue
	if isVueProject(rootDir) {
		return "Vue"
	}

	// Проверка Svelte
	if isSvelteProject(rootDir) {
		return "Svelte"
	}

	return "JavaScript"
}

func isAngularProject(rootDir string) bool {
	angularFiles := []string{"angular.json", "src/main.ts"}
	count := 0
	for _, file := range angularFiles {
		if _, err := os.Stat(filepath.Join(rootDir, file)); err == nil {
			count++
		}
	}
	return count >= 2
}

func isReactProject(rootDir string) bool {
	reactFiles := []string{"src/App.jsx", "src/App.tsx", "src/index.js"}
	count := 0
	for _, file := range reactFiles {
		if _, err := os.Stat(filepath.Join(rootDir, file)); err == nil {
			count++
		}
	}
	return count >= 2 || hasDependency(rootDir, "react")
}

func isVueProject(rootDir string) bool {
	vueFiles := []string{"vue.config.js", "src/App.vue", "src/main.js"}
	count := 0
	for _, file := range vueFiles {
		if _, err := os.Stat(filepath.Join(rootDir, file)); err == nil {
			count++
		}
	}
	return count >= 2 || hasDependency(rootDir, "vue")
}

func isSvelteProject(rootDir string) bool {
	svelteFiles := []string{"svelte.config.js", "src/App.svelte"}
	count := 0
	for _, file := range svelteFiles {
		if _, err := os.Stat(filepath.Join(rootDir, file)); err == nil {
			count++
		}
	}
	return count >= 2 || hasDependency(rootDir, "svelte")
}

func hasDependency(rootDir, dep string) bool {
	pkgPath := filepath.Join(rootDir, "package.json")
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), `"`+dep+`"`)
}
