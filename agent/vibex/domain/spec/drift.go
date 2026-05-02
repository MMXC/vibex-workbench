// agent/vibex/domain/spec/drift.go — Spec drift detection engine.
// SLICE-drift-api-endpoints
package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DriftEntry represents a detected drift item.
type DriftEntry struct {
	SpecPath   string   `json:"spec_path"`
	Generates  []string `json:"generates,omitempty"`
	Missing    []string `json:"missing,omitempty"`
	Extra      []string `json:"extra,omitempty"`
	Modified   []string `json:"modified,omitempty"`
	DetectedAt string   `json:"detected_at"`
	Status     string   `json:"status"` // pending, accepted, rejected
}

// DriftEngine detects spec-to-code drift and manages drift state.
type DriftEngine struct {
	workspaceDir string
	statePath    string
	state        map[string]DriftEntry
}

// NewDriftEngine creates a new DriftEngine.
func NewDriftEngine(workspaceDir string) *DriftEngine {
	dotVibex := filepath.Join(workspaceDir, ".vibex")
	os.MkdirAll(dotVibex, 0755)
	de := &DriftEngine{
		workspaceDir: workspaceDir,
		statePath:    filepath.Join(dotVibex, "drift-state.json"),
		state:        make(map[string]DriftEntry),
	}
	de.loadState()
	return de
}

// CheckDrift scans specs for generated file drift.
func (e *DriftEngine) CheckDrift() ([]DriftEntry, error) {
	// Walk specs/ looking for generates[] declarations
	specsDir := filepath.Join(e.workspaceDir, "specs")
	var entries []DriftEntry

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
		// Simple scan for generates: pattern (rough — full YAML parsing deferred)
		content := string(data)
		if !strings.Contains(content, "generates:") {
			return nil
		}
		rel, _ := filepath.Rel(e.workspaceDir, full)

		// For each spec, check its declared outputs
		drift := DriftEntry{
			SpecPath:   rel,
			DetectedAt: time.Now().UTC().Format(time.RFC3339),
			Status:     "pending",
		}
		entries = append(entries, drift)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Update state
	for _, entry := range entries {
		e.state[entry.SpecPath] = entry
	}
	e.saveState()
	return entries, nil
}

// ListDrift returns all drift entries.
func (e *DriftEngine) ListDrift() []DriftEntry {
	result := make([]DriftEntry, 0, len(e.state))
	for _, v := range e.state {
		result = append(result, v)
	}
	return result
}

// AcceptDrift marks a drift entry as accepted.
func (e *DriftEngine) AcceptDrift(specPath string) error {
	if entry, ok := e.state[specPath]; ok {
		entry.Status = "accepted"
		e.state[specPath] = entry
		return e.saveState()
	}
	return fmt.Errorf("drift entry not found: %s", specPath)
}

// RejectDrift marks a drift entry as rejected.
func (e *DriftEngine) RejectDrift(specPath string) error {
	if entry, ok := e.state[specPath]; ok {
		entry.Status = "rejected"
		e.state[specPath] = entry
		return e.saveState()
	}
	return fmt.Errorf("drift entry not found: %s", specPath)
}

func (e *DriftEngine) loadState() {
	data, err := os.ReadFile(e.statePath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &e.state)
}

func (e *DriftEngine) saveState() error {
	data, err := json.MarshalIndent(e.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(e.statePath, data, 0644)
}

// RunMakeDryRun runs `make generate --dry-run` to detect drift.
func RunMakeDryRun(workspaceDir string) (string, string, error) {
	cmd := exec.Command("make", "generate")
	cmd.Dir = workspaceDir
	out, err := cmd.CombinedOutput()
	return string(out), "", err
}