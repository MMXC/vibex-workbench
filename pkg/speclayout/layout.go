// Package speclayout creates canonical specs/L1–L5 directory buckets under a workspace root.
package speclayout

import (
	"fmt"
	"os"
	"path/filepath"
)

// CanonicalDirs are paths relative to workspace root (POSIX slashes for JSON/logging).
// Keep in sync with frontend CANONICAL_SPEC_SUBDIRS in spec-layout-dirs.ts.
var CanonicalDirs = []string{
	"specs/L1-goal",
	"specs/L2-skeleton",
	"specs/L3-module",
	"specs/L4-feature",
	"specs/L5-slice",
	"specs/_governance",
	".agents/flows",
}

// EnsureCanonicalDirs creates standard VibeX specs hierarchy; idempotent (MkdirAll).
func EnsureCanonicalDirs(workspaceRoot string) (created []string, skipped []string, err error) {
	root := filepath.Clean(workspaceRoot)
	if root == "" || root == "." {
		return nil, nil, fmt.Errorf("workspace root required")
	}
	for _, rel := range CanonicalDirs {
		full := filepath.Join(root, filepath.FromSlash(rel))
		exists := false
		if st, statErr := os.Stat(full); statErr == nil && st.IsDir() {
			exists = true
		}
		if exists {
			skipped = append(skipped, rel)
			continue
		}
		if err := os.MkdirAll(full, 0o755); err != nil {
			return created, skipped, fmt.Errorf("mkdir %s: %w", rel, err)
		}
		created = append(created, rel)
	}
	return created, skipped, nil
}
