// agent/cmd/web/qa_runner.go — QA Playwright scenario runner handler.
// SLICE-qa-playwright-scenario-runner
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// qaRunnerRequest is the POST body for /api/workspace/qa/run.
type qaRunnerRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	Scenario      string `json:"scenario"` // "e2e", "smoke", "spec-verify"
	Tags          string `json:"tags"`      // comma-separated playwright tags
}

// qaRunnerResult is the response from a QA run.
type qaRunnerResult struct {
	OK       bool     `json:"ok"`
	Scenario string   `json:"scenario"`
	Passed   bool     `json:"passed"`
	Duration string   `json:"duration"`
	Output   string   `json:"output,omitempty"`
	Errors   []string `json:"errors,omitempty"`
	ExitCode int      `json:"exit_code"`
}

// qaRunnerHandler POST /api/workspace/qa/run
// Runs Playwright QA scenarios against the workbench.
func qaRunnerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req qaRunnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}

	wsRoot := req.WorkspaceRoot
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

	result := runQAScenario(wsRoot, req.Scenario, req.Tags)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// runQAScenario executes a QA scenario.
func runQAScenario(wsRoot, scenario, tags string) qaRunnerResult {
	start := time.Now()
	result := qaRunnerResult{
		Scenario: scenario,
		Duration: "0s",
	}

	var args []string

	switch scenario {
	case "spec-verify":
		// Run verify_specs as a QA check
		binPath := "./verify_specs"
		args = []string{"--workspace", wsRoot, "--format", "summary"}
		cmd := exec.Command(binPath, args...)
		cmd.Dir = wsRoot
		out, err := cmd.CombinedOutput()
		result.Output = string(out)
		result.ExitCode = 0
		if err != nil {
			if ex, ok := err.(*exec.ExitError); ok {
				result.ExitCode = ex.ExitCode()
			}
		}
		result.Passed = result.ExitCode == 0
	case "smoke":
		// Smoke test: check that key files exist
		keyFiles := []string{
			"specs/L1-goal/ENTRY.yaml",
			"agent/cmd/web/main.go",
			"frontend/src/App.svelte",
		}
		var missing []string
		for _, f := range keyFiles {
			if _, err := os.Stat(wsRoot + "/" + f); os.IsNotExist(err) {
				missing = append(missing, f)
			}
		}
		result.Passed = len(missing) == 0
		result.Errors = missing
		result.Output = fmt.Sprintf("smoke check: %d/%d files present", len(keyFiles)-len(missing), len(keyFiles))
		result.ExitCode = 0
		if !result.Passed {
			result.ExitCode = 1
		}
	case "e2e":
		// E2E: try to run playwright if available
		cmd := exec.Command("which", "playwright")
		if err := cmd.Run(); err != nil {
			result.Passed = false
			result.Output = "playwright not installed"
			result.ExitCode = 1
			return result
		}
		args = []string{"test"}
		if tags != "" {
			args = append(args, "--grep", tags)
		}
		cmd = exec.Command(args[0], args[1:]...)
		cmd.Dir = wsRoot
		out, err := cmd.CombinedOutput()
		result.Output = string(out)
		result.Passed = err == nil
		if err != nil {
			if ex, ok := err.(*exec.ExitError); ok {
				result.ExitCode = ex.ExitCode()
			}
		}
	default:
		result.Passed = false
		result.Output = "unknown scenario: " + scenario
		result.ExitCode = 1
	}

	result.OK = true
	result.Duration = time.Since(start).String()
	return result
}