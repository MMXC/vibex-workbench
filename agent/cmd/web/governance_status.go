// agent/cmd/web/governance_status.go — Governance status update handler.
// SLICE-mvp-governance-status-update
package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// governanceStatusRequest is the POST body for /api/workspace/governance/status.
type governanceStatusRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	Refresh       bool   `json:"refresh"`
}

// governanceStatusHandler POST/GET /api/workspace/governance/status
// Returns current governance coverage and consistency status.
func governanceStatusHandler(w http.ResponseWriter, r *http.Request) {
	wsRoot := r.URL.Query().Get("workspaceRoot")
	if wsRoot == "" {
		wsRoot = cfg.WorkspaceDir
	}
	if wsRoot == "" {
		wsRoot = os.Getenv("WORKSPACE_ROOT")
	}
	if wsRoot == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}

	status := map[string]interface{}{
		"workspaceRoot": wsRoot,
		"ok":            true,
	}

	// panorama.json 仅认 .vibex/specs/_governance
	status["panorama"] = "missing"
	panoramaPath := filepath.Join(wsRoot, ".vibex", "specs", "_governance", "panorama.json")
	if _, err := os.Stat(panoramaPath); err == nil {
		status["panorama"] = "present"
	}

	specBucket := func(wsRelative string) string {
		s := filepath.ToSlash(wsRelative)
		if !strings.HasPrefix(s, ".vibex/specs/") {
			return "_other"
		}
		s = strings.TrimPrefix(s, ".vibex/specs/")
		parts := strings.Split(s, "/")
		if len(parts) == 0 || parts[0] == "" {
			return "_other"
		}
		return parts[0]
	}

	counts := map[string]int{}
	count := 0
	specsDir := filepath.Join(wsRoot, ".vibex", "specs")
	if _, err := os.Stat(specsDir); err == nil {
		_ = filepath.Walk(specsDir, func(full string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(full, ".yaml") && !strings.HasSuffix(full, ".yml") {
				return nil
			}
			relWS, relErr := filepath.Rel(wsRoot, full)
			if relErr != nil {
				return nil
			}
			key := filepath.ToSlash(relWS)
			count++
			b := specBucket(key)
			counts[b]++
			return nil
		})
	}

	status["total_specs"] = count
	status["by_directory"] = counts

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
