// agent/cmd/web/governance_status.go — Governance status update handler.
// SLICE-mvp-governance-status-update
package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
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

	// Count specs by level
	specsDir := wsRoot + "/specs"
	cmd := exec.Command("find", specsDir, "-name", "*.yaml", "-o", "-name", "*.yml")
	cmd.Dir = wsRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		count := 0
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				count++
			}
		}
		status["total_specs"] = count
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}