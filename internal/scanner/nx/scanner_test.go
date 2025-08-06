package nx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kolkov/gops/internal/model"
	"github.com/kolkov/gops/pkg/logger"
)

func TestNxScanner(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Nx-like structure:
	//   tmpDir/
	//   ├── apps/
	//   │   ├── app1/
	//   │   │   └── project.json
	//   │   └── app2/
	//   │       └── project.json
	//   ├── libs/
	//   │   └── lib1/
	//   │       └── project.json
	//   ├── tools/
	//   │   └── tool1/
	//   │       └── project.json
	//   ├── nx.json
	//   └── package.json

	// Create project directories
	projects := []struct {
		dir  string
		name string
	}{
		{"apps/app1", "app1"},
		{"apps/app2", "app2"},
		{"libs/lib1", "lib1"},
		{"tools/tool1", "tool1"},
	}

	for _, p := range projects {
		dir := filepath.Join(tmpDir, p.dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}

		// Create project.json
		projectFile := filepath.Join(dir, "project.json")
		if err := os.WriteFile(projectFile, []byte(`{"name": "`+p.name+`"}`), 0644); err != nil {
			t.Fatalf("Failed to create project.json: %v", err)
		}
	}

	// Create root files
	rootFiles := []string{"nx.json", "package.json"}
	for _, file := range rootFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", file, err)
		}
	}

	cfg := &model.ScanConfig{
		OutputFilename:       "output.md",
		OutputConfigFilename: "config.yaml",
	}

	logger := logger.New(logger.InfoLevel)
	scanner := NewScanner(tmpDir, "output.md", cfg, logger)

	t.Run("LoadProjects", func(t *testing.T) {
		if err := scanner.loadProjects(); err != nil {
			t.Fatalf("loadProjects failed: %v", err)
		}

		if len(scanner.projects) != 4 {
			t.Fatalf("Expected 4 projects, got %d", len(scanner.projects))
		}

		// Verify project types
		expectedTypes := map[string]string{
			"app1":  "apps",
			"app2":  "apps",
			"lib1":  "libs",
			"tool1": "tools",
		}

		for _, p := range scanner.projects {
			if p.Type != expectedTypes[p.Name] {
				t.Errorf("Project %s: expected type %s, got %s",
					p.Name, expectedTypes[p.Name], p.Type)
			}
		}
	})

	t.Run("FilterProjects", func(t *testing.T) {
		scanner.projects = []*model.NxProject{
			{Name: "p1", Type: "apps"},
			{Name: "p2", Type: "apps"},
			{Name: "p3", Type: "libs"},
		}

		tests := []struct {
			input    string
			expected []string
		}{
			{"all", []string{"p1", "p2", "p3"}},
			{"1,3", []string{"p1", "p3"}},
			{"2", []string{"p2"}},
			{"invalid", []string{}},
			{"", []string{"p1", "p2", "p3"}},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result := scanner.filterProjects(tt.input)
				if len(result) != len(tt.expected) {
					t.Fatalf("Expected %d projects, got %d",
						len(tt.expected), len(result))
				}
				for i, name := range tt.expected {
					if result[i].Name != name {
						t.Errorf("Project %d: expected %s, got %s",
							i, name, result[i].Name)
					}
				}
			})
		}
	})

	t.Run("GetRootFiles", func(t *testing.T) {
		files, err := scanner.getRootFiles()
		if err != nil {
			t.Fatalf("getRootFiles failed: %v", err)
		}

		expected := []string{"nx.json", "package.json"}
		if len(files) != len(expected) {
			t.Fatalf("Expected %d files, got %d", len(expected), len(files))
		}

		for i, file := range expected {
			if files[i] != file {
				t.Errorf("File %d: expected %s, got %s", i, file, files[i])
			}
		}
	})
}
