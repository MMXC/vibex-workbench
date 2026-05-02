// agent/vibex/domain/spec/governance_view_builder.go — Per-spec view model builder.
// SLICE-spec-view-model-builder
package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ViewBuilder builds SpecViewModel objects for individual specs.
type ViewBuilder struct {
	workspaceDir string
}

// NewViewBuilder creates a new ViewBuilder.
func NewViewBuilder(workspaceDir string) *ViewBuilder {
	return &ViewBuilder{workspaceDir: workspaceDir}
}

// BuildView constructs the view model for a single spec.
func (b *ViewBuilder) BuildView(specPath string) (*SpecViewModel, error) {
	full := filepath.Join(b.workspaceDir, specPath)
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	vm := &SpecViewModel{
		Path: specPath,
		Name: filepath.Base(specPath),
		Metadata: make(map[string]interface{}),
	}

	content := string(data)
	vm.Status = extractFrontmatterField(content, "status")
	vm.Parent = extractFrontmatterField(content, "parent")
	vm.Description = extractFrontmatterField(content, "description")

	// Detect level from path
	vm.Level = inferLevel(specPath)

	// Add default actions
	vm.Actions = []ViewModelAction{
		{ID: "view-content", Label: "查看内容", RequiresConfirm: false},
		{ID: "open-parent", Label: "打开父级", RequiresConfirm: false},
	}

	// Add links
	vm.Links = []ViewModelLink{
		{Label: "View Raw YAML", URI: "file://" + specPath, Enabled: true},
	}

	// Add basic metadata
	vm.Metadata["file_size"] = len(data)

	return vm, nil
}

// BuildViewJSON returns the view model as JSON.
func (b *ViewBuilder) BuildViewJSON(specPath string) (string, error) {
	vm, err := b.BuildView(specPath)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(vm, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// BuildViewMarkdown returns a markdown representation of the view model.
func (b *ViewBuilder) BuildViewMarkdown(specPath string) (string, error) {
	vm, err := b.BuildView(specPath)
	if err != nil {
		return "", err
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("# %s", vm.Name))
	lines = append(lines, fmt.Sprintf("- **Path**: %s", vm.Path))
	lines = append(lines, fmt.Sprintf("- **Level**: %s", vm.Level))
	lines = append(lines, fmt.Sprintf("- **Status**: %s", vm.Status))
	if vm.Parent != "" {
		lines = append(lines, fmt.Sprintf("- **Parent**: %s", vm.Parent))
	}
	if vm.Description != "" {
		lines = append(lines, "")
		lines = append(lines, vm.Description)
	}
	return strings.Join(lines, "\n"), nil
}

// extractFrontmatterField extracts a field from YAML frontmatter.
func extractFrontmatterField(content, field string) string {
	lines := strings.Split(content, "\n")
	var inFrontmatter bool
	var inContent bool

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else if !inContent {
				inContent = true
				continue
			}
		}
		if inFrontmatter && !inContent {
			if strings.HasPrefix(line, field+":") {
				return strings.TrimSpace(strings.TrimPrefix(line, field+":"))
			}
		}
	}
	return ""
}