// agent/vibex/domain/spec/governance_handlers.go — HTTP handlers for spec history and governance APIs.
// SLICE-spec-history-api
package spec

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SpecTreeNode represents a node in the spec tree response.
type SpecTreeNode struct {
	Path     string          `json:"path"`
	Name     string          `json:"name"`
	Level    string          `json:"level"`
	Parent   string          `json:"parent,omitempty"`
	Children []SpecTreeNode `json:"children,omitempty"`
}

// BuildSpecTree builds the full spec tree from the workspace.
func BuildSpecTree(workspaceDir string) ([]SpecTreeNode, error) {
	specsDir := filepath.Join(workspaceDir, "specs")
	var roots []SpecTreeNode

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
		rel, _ := filepath.Rel(workspaceDir, full)

		node := SpecTreeNode{
			Path:  rel,
			Name:  filepath.Base(rel),
			Level: inferLevel(rel),
		}
		_ = data // full YAML parsing deferred

		// Simple top-level: add to roots (real impl would build parent chain)
		roots = append(roots, node)
		return nil
	})

	return roots, err
}

func inferLevel(rel string) string {
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) > 0 {
		first := parts[0]
		if strings.HasPrefix(first, "L1") {
			return "L1"
		}
		if strings.HasPrefix(first, "L2") {
			return "L2"
		}
		if strings.HasPrefix(first, "L3") {
			return "L3"
		}
		if strings.HasPrefix(first, "L4") {
			return "L4"
		}
		if strings.HasPrefix(first, "L5") {
			return "L5"
		}
	}
	return "unknown"
}

// SpecHistoryEntry represents a git commit for a spec.
type SpecHistoryEntry struct {
	Commit   string `json:"commit"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	Message  string `json:"message"`
	SpecPath string `json:"spec_path"`
}

// GetSpecHistory returns git log for a given spec file.
func GetSpecHistory(workspaceDir, specPath string) ([]SpecHistoryEntry, error) {
	absPath := filepath.Join(workspaceDir, specPath)

	cmd := exec.Command("git", "log", "--oneline", "--format=%H|%an|%ad|%s", "--date=iso", "--", absPath)
	cmd.Dir = workspaceDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil // no git history
	}

	var entries []SpecHistoryEntry
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) >= 4 {
			entries = append(entries, SpecHistoryEntry{
				Commit:   parts[0],
				Author:   parts[1],
				Date:     parts[2],
				Message:  parts[3],
				SpecPath: specPath,
			})
		}
	}
	return entries, nil
}

// SpecTreeHandler GET /api/workspace/specs/tree
func SpecTreeHandler(workspaceDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		tree, err := BuildSpecTree(workspaceDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"tree": tree, "total": len(tree)})
	}
}