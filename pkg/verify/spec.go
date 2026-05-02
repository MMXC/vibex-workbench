// pkg/verify/spec.go
// Spec data structures mirroring the YAML format.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Spec represents a parsed spec YAML file.
type Spec struct {
	// SourceFile is the absolute path to the YAML file on disk.
	SourceFile string `json:"source_file"`

	// Spec-level fields
	Name    string `yaml:"name"`
	Level   string `yaml:"level"`    // e.g. "4_feature", "3_module", "5_slice"
	Parent  string `yaml:"parent"`   // parent spec name, "" for root
	Status  string `yaml:"status"`   // proposal/implementation/delivered

	// Meta
	Module string `yaml:"module"`
	Owner  string `yaml:"owner"`

	// Lifecycle
	LifecycleCurrent string `yaml:"lifecycle.current"`
	LifecycleUpdated string `yaml:"lifecycle.updated"`

	// Display
	Title   string `yaml:"display.title"`
	Summary string `yaml:"display.summary"`

	// Structure
	ImpactedFiles  []string `yaml:"structure.impacted_files"`
	StructureDeps  []string `yaml:"structure.dependencies"`
	Children       []string `yaml:"structure.children"`

	// Content (level-dependent)
	FilePath   string            `yaml:"content.file_path"`    // L5
	FileType   string            `yaml:"content.file_type"`    // L5: svelte, go, ts, etc.
	Behaviors  []Behavior        `yaml:"content.behaviors"`
	UserStories []UserStory      `yaml:"content.user_stories"`
	TestScenarios []TestScenario `yaml:"content.test_scenarios"`
	GenerationRules []GenRule    `yaml:"content.generation_rules"`
	Verification   *Verification `yaml:"content.verification"`  // L5
	Constraints    []Constraint  `yaml:"content.constraints"`
	L4L5Lineage    *L4L5Lineage  `yaml:"content.l4_l5_lineage"`
	L3L4Lineage    *L3L4Lineage  `yaml:"content.l3_l4_lineage"`

	// Prototype
	PrototypeFile    string   `yaml:"prototype.file"`
	PrototypeValidates []string `yaml:"prototype.validates"`
	PrototypeStatus  string   `yaml:"prototype.status"`
}

// Behavior represents a single behavior entry in L4/L5 specs.
type Behavior struct {
	ID             string `yaml:"id"`
	Trigger        string `yaml:"trigger"`
	Action         string `yaml:"action"`
	ExpectedResult string `yaml:"expected_result"`
	ErrorHandling  string `yaml:"error_handling"`
}

// UserStory represents a user story.
type UserStory struct {
	ID         string `yaml:"id"`
	AsA        string `yaml:"as_a"`
	IWant      string `yaml:"i_want"`
	SoThat     string `yaml:"so_that"`
	Acceptance string `yaml:"acceptance"`
}

// TestScenario represents a Gherkin-style test scenario.
type TestScenario struct {
	ID         string `yaml:"id"`
	Scenario   string `yaml:"scenario"`
	Given      string `yaml:"given"`
	When       string `yaml:"when"`
	Then       string `yaml:"then"`
	Evaluation string `yaml:"evaluation"`
}

// GenRule is a generation rule from L5 content.generation_rules.
type GenRule struct {
	Rule   string `yaml:"rule"`
	Detail string `yaml:"detail"`
}

// Verification is the verification section of L5 content.verification.
type Verification struct {
	Commands       []string `yaml:"commands"`
	ManualChecks   []string `yaml:"manual_checks"`
	FailureConditions []string `yaml:"failure_conditions"`
}

// Constraint is a constraint entry.
type Constraint struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Rule       string `yaml:"rule"`
	Validation string `yaml:"validation"`
}

// L4L5Lineage is the l4_l5_lineage section.
type L4L5Lineage struct {
	Summary        string        `yaml:"summary"`
	WhichFiles     []LineageFile `yaml:"which_files"`
	GenerationOrder []string     `yaml:"generation_order"`
}

// L3L4Lineage is the l3_l4_lineage section.
type L3L4Lineage struct {
	Summary       string           `yaml:"summary"`
	WhichFeatures []LineageFeature `yaml:"which_features"`
}

// LineageFile maps a file to its source behavior.
type LineageFile struct {
	File          string `yaml:"file"`
	GeneratedFrom string `yaml:"generated_from"`
}

// LineageFeature maps a feature to its used APIs.
type LineageFeature struct {
	Feature string   `yaml:"feature"`
	UsesAPI []string `yaml:"uses_api"`
}

// Level returns the numeric level as an int (e.g. 4 from "4_feature").
func (s *Spec) LevelNum() int {
	// spec.level format: "4_feature", "3_module", "5_slice", "2_skeleton", "1_concept"
	parts := strings.Split(s.Level, "_")
	if len(parts) == 0 {
		return 0
	}
	var n int
	fmt.Sscanf(parts[0], "%d", &n)
	return n
}

// AbsPath resolves a relative path against the workspace root.
// Returns the absolute path, or "" if the path is empty.
func (s *Spec) AbsPath(workspaceRoot, relPath string) string {
	if relPath == "" {
		return ""
	}
	// If already absolute, return as-is
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(workspaceRoot, relPath)
}

// Exists reports whether the file at relPath exists in workspaceRoot.
func (s *Spec) Exists(workspaceRoot, relPath string) bool {
	if relPath == "" {
		return false
	}
	abs := s.AbsPath(workspaceRoot, relPath)
	_, err := os.Stat(abs)
	return err == nil
}
