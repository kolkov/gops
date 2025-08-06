package docgen

import "github.com/kolkov/gops/internal/model"

type MockGenerator struct{}

func (m *MockGenerator) WriteHeader(meta *model.ProjectMeta)                              {}
func (m *MockGenerator) WriteTree(structure string)                                       {}
func (m *MockGenerator) WriteFileSection(file *model.ProjectFile)                         {}
func (m *MockGenerator) WriteNxStructure(projects []*model.NxProject, rootFiles []string) {}
func (m *MockGenerator) WriteProjectHeader(name, ptype, root string)                      {}
func (m *MockGenerator) WriteProjectTree(structure string)                                {}
func (m *MockGenerator) WriteModulesHeader()                                              {}
func (m *MockGenerator) Close() error                                                     { return nil }
