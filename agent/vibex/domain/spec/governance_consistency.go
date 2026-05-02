// agent/vibex/domain/spec/governance_consistency.go — Consistency detection for spec governance.
// SLICE-spec-governance-consistency-engine
package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConsistencyIssue represents a governance consistency violation.
type ConsistencyIssue struct {
	Severity  string `json:"severity"` // error, warning, info
	SpecPath  string `json:"spec_path"`
	Rule      string `json:"rule"`
	Message   string `json:"message"`
	Field     string `json:"field,omitempty"`
	Fix       string `json:"fix,omitempty"`
}

// ConsistencyEngine checks spec governance consistency.
type ConsistencyEngine struct {
	workspaceDir string
}

// NewConsistencyEngine creates a new ConsistencyEngine.
func NewConsistencyEngine(workspaceDir string) *ConsistencyEngine {
	return &ConsistencyEngine{workspaceDir: workspaceDir}
}

// CheckAll runs consistency checks across all specs.
func (e *ConsistencyEngine) CheckAll() ([]ConsistencyIssue, error) {
	specsDir := filepath.Join(e.workspaceDir, "specs")
	var issues []ConsistencyIssue

	err := filepath.Walk(specsDir, func(full string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(full, ".yaml") && !strings.HasSuffix(full, ".yml") {
			return nil
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(e.workspaceDir, full)

		// Check 1: file must have spec.name field
		content := string(data)
		if !strings.Contains(content, "spec:") {
			issues = append(issues, ConsistencyIssue{
				Severity: "error",
				SpecPath: rel,
				Rule:     "spec-name-required",
				Message:  "spec.name is required",
				Fix:      "Add 'spec:\\n  name: <name>' to the YAML frontmatter",
			})
		}

		// Check 2: file must have structure.impacted_files
		if !strings.Contains(content, "impacted_files:") {
			issues = append(issues, ConsistencyIssue{
				Severity: "warning",
				SpecPath: rel,
				Rule:     "impacted-files-required",
				Message:  "structure.impacted_files is missing",
			})
		}

		return nil
	})

	return issues, err
}

// CheckSpec checks a single spec for consistency issues.
func (e *ConsistencyEngine) CheckSpec(specPath string) ([]ConsistencyIssue, error) {
	full := filepath.Join(e.workspaceDir, specPath)
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	var issues []ConsistencyIssue
	content := string(data)

	if !strings.Contains(content, "spec:") {
		issues = append(issues, ConsistencyIssue{
			Severity: "error",
			SpecPath: specPath,
			Rule:     "spec-name-required",
			Message:  "spec.name is required",
		})
	}

	return issues, nil
}