// pkg/verify/loader.go
// Loads and parses spec YAML files from the workspace.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader loads spec YAML files.
type Loader struct {
	workspaceRoot string
}

// NewLoader creates a new spec loader for the given workspace root.
func NewLoader(workspaceRoot string) *Loader {
	return &Loader{workspaceRoot: workspaceRoot}
}

// LoadAll loads all spec YAML files recursively under the specs/ directory.
// It returns specs keyed by spec.name.
func (l *Loader) LoadAll() (map[string]*Spec, error) {
	specsDir := filepath.Join(l.workspaceRoot, "specs")
	info, err := os.Stat(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("specs/ directory does not exist at %s", specsDir)
		}
		return nil, fmt.Errorf("stat specs/: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("specs/ is not a directory")
	}

	specs := make(map[string]*Spec)
	if err := l.walkDir(specsDir, specs); err != nil {
		return nil, err
	}
	return specs, nil
}

func (l *Loader) walkDir(dir string, specs map[string]*Spec) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		full := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if err := l.walkDir(full, specs); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		spec, err := l.loadFile(full)
		if err != nil {
			// Log and skip individual bad files rather than failing entirely
			fmt.Fprintf(os.Stderr, "[verify] warning: skipping %s: %v\n", full, err)
			continue
		}
		if spec.Name == "" {
			fmt.Fprintf(os.Stderr, "[verify] warning: skipping %s: spec.name is empty\n", full)
			continue
		}
		if _, exists := specs[spec.Name]; exists {
			fmt.Fprintf(os.Stderr, "[verify] warning: duplicate spec name %q (files: %s and existing), skipping duplicate\n",
				spec.Name, full)
			continue
		}
		specs[spec.Name] = spec
	}
	return nil
}

// loadFile parses a single YAML spec file.
func (l *Loader) loadFile(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Strip leading/trailing BOM if present
	data = stripBOM(data)

	var raw struct {
		Spec struct {
			Name    string `yaml:"name"`
			Level   string `yaml:"level"`
			Parent  string `yaml:"parent"`
			Status  string `yaml:"status"`
		} `yaml:"spec"`
		Meta struct {
			Module string `yaml:"module"`
			Owner  string `yaml:"owner"`
		} `yaml:"meta"`
		Lifecycle struct {
			Current string `yaml:"current"`
			Updated string `yaml:"updated"`
		} `yaml:"lifecycle"`
		Display struct {
			Title   string `yaml:"title"`
			Summary string `yaml:"summary"`
		} `yaml:"display"`
		Structure struct {
			Parent        string   `yaml:"parent"`
			Children      []string `yaml:"children"`
			Dependencies   []string `yaml:"dependencies"`
			ImpactedFiles []string `yaml:"impacted_files"`
		} `yaml:"structure"`
		Content struct {
			FilePath        string            `yaml:"file_path"`
			FileType        string            `yaml:"file_type"`
			Behaviors       []Behavior        `yaml:"behaviors"`
			UserStories     []UserStory       `yaml:"user_stories"`
			TestScenarios   []TestScenario    `yaml:"test_scenarios"`
			GenerationRules []GenRule         `yaml:"generation_rules"`
			Verification    *Verification     `yaml:"verification"`
			Constraints     []Constraint      `yaml:"constraints"`
			L4L5Lineage     *L4L5Lineage     `yaml:"l4_l5_lineage"`
			L3L4Lineage     *L3L4Lineage     `yaml:"l3_l4_lineage"`
		} `yaml:"content"`
		Prototype struct {
			File       string   `yaml:"file"`
			Validates  []string `yaml:"validates"`
			Status     string   `yaml:"status"`
		} `yaml:"prototype"`
	}

	// Try parsing the top-level "spec:" wrapper first
	hasSpecWrapper := false
	if err := yaml.Unmarshal(data, &raw); err == nil && raw.Spec.Name != "" {
		hasSpecWrapper = true
	}

	// Also try parsing without the wrapper (for flat YAML with no "spec:" root key)
	if !hasSpecWrapper {
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("yaml unmarshal: %w", err)
		}
	}

	spec := &Spec{
		SourceFile:       path,
		Name:            raw.Spec.Name,
		Level:           raw.Spec.Level,
		Parent:          raw.Spec.Parent,
		Status:          raw.Spec.Status,
		Module:          raw.Meta.Module,
		Owner:           raw.Meta.Owner,
		LifecycleCurrent: raw.Lifecycle.Current,
		LifecycleUpdated: raw.Lifecycle.Updated,
		Title:           raw.Display.Title,
		Summary:         raw.Display.Summary,
		ImpactedFiles:   raw.Structure.ImpactedFiles,
		StructureDeps:   raw.Structure.Dependencies,
		Children:        raw.Structure.Children,
		FilePath:        raw.Content.FilePath,
		FileType:        raw.Content.FileType,
		Behaviors:       raw.Content.Behaviors,
		UserStories:     raw.Content.UserStories,
		TestScenarios:   raw.Content.TestScenarios,
		GenerationRules: raw.Content.GenerationRules,
		Verification:    raw.Content.Verification,
		Constraints:     raw.Content.Constraints,
		L4L5Lineage:     raw.Content.L4L5Lineage,
		L3L4Lineage:     raw.Content.L3L4Lineage,
		PrototypeFile:    raw.Prototype.File,
		PrototypeValidates: raw.Prototype.Validates,
		PrototypeStatus: raw.Prototype.Status,
	}

	// If spec.name is still empty, try the top-level yaml key directly
	if spec.Name == "" {
		// Try without the wrapper
		var flat struct {
			Name    string `yaml:"name"`
			Level   string `yaml:"level"`
			Parent  string `yaml:"parent"`
			Status  string `yaml:"status"`
		}
		if err := yaml.Unmarshal(data, &flat); err == nil && flat.Name != "" {
			spec.Name = flat.Name
			spec.Level = flat.Level
			spec.Parent = flat.Parent
			spec.Status = flat.Status
		}
	}

	return spec, nil
}

// stripBOM removes UTF-8 BOM from the start of data.
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}
