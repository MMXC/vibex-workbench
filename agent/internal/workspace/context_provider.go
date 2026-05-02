// agent/internal/workspace/context_provider.go — Provides workspace context to the agent.
// SLICE-agent-workspace-context-provider
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContextProvider provides workspace-scoped context data to the agent.
type ContextProvider struct {
	workspaceDir string
}

// NewContextProvider creates a new ContextProvider.
func NewContextProvider(workspaceDir string) *ContextProvider {
	return &ContextProvider{workspaceDir: workspaceDir}
}

// SpecIndex returns all spec paths under specs/ in this workspace.
func (p *ContextProvider) SpecIndex() ([]string, error) {
	specsDir := filepath.Join(p.workspaceDir, "specs")
	var paths []string
	err := filepath.Walk(specsDir, func(full string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (strings.HasSuffix(full, ".yaml") || strings.HasSuffix(full, ".yml")) {
			rel, err := filepath.Rel(p.workspaceDir, full)
			if err == nil {
				paths = append(paths, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk specs: %w", err)
	}
	return paths, nil
}

// WorkspacePath returns the configured workspace root.
func (p *ContextProvider) WorkspacePath() string {
	return p.workspaceDir
}

// Exists reports whether a given relative path exists within the workspace.
func (p *ContextProvider) Exists(relPath string) bool {
	full, err := filepath.Abs(filepath.Join(p.workspaceDir, relPath))
	if err != nil {
		return false
	}
	_, err = os.Stat(full)
	return err == nil
}