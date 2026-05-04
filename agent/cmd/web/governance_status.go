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

	// Check if panorama.json exists
	dotGov := wsRoot + "/specs/_governance/panorama.json"
	if _, err := os.Stat(dotGov); err == nil {
		status["panorama"] = "present"
	} else {
		status["panorama"] = "missing"
	}

	// Count specs by level without shelling out, so Windows builds behave the same.
	counts := map[string]int{}
	count := 0
	specsDir := filepath.Join(wsRoot, "specs")
	_ = filepath.Walk(specsDir, func(full string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(full, ".yaml") && !strings.HasSuffix(full, ".yml") {
			return nil
		}
		count++
		if rel, relErr := filepath.Rel(specsDir, full); relErr == nil {
			parts := strings.Split(filepath.ToSlash(rel), "/")
			if len(parts) > 0 {
				counts[parts[0]]++
			}
		}
		return nil
	})
	status["total_specs"] = count
	status["by_directory"] = counts

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
