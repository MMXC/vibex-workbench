package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ── confirmed spec draft payload ────────────────────────────────────

// confirmedSpecDraft is the payload sent by the agent after user confirmation.
type confirmedSpecDraft struct {
	WorkspaceRoot  string `json:"workspace_root"`
	Path          string `json:"path"`
	ParentName    string `json:"parent_name"`
	TargetLevel   string `json:"target_level"` // L2|L3|L4|L5
	YamlText      string `json:"yaml_text"`
	ConfirmationID string `json:"confirmation_id"`
}

// validationErrorCategory classifies make validate failures for user feedback.
type validationErrorCategory struct {
	Category string `json:"category"` // yaml_parse_error | parent_not_found | level_mismatch | missing_l5_boundary
	Message  string `json:"message"`
	Suggest  string `json:"suggest"`
}

// handleAgentSpecWriteProtocol handles confirmed spec drafts from the agent.
// It assembles a specs/write request, calls write, then runs make validate,
// and classifies the result for the agent thread.
func handleAgentSpecWriteProtocol(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var draft confirmedSpecDraft
	if err := json.NewDecoder(r.Body).Decode(&draft); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	// ── Gate: reject if no confirmation ID (draft not confirmed) ──
	if draft.ConfirmationID == "" {
		http.Error(w, "spec write protocol: confirmation_id required (draft not confirmed)", http.StatusForbidden)
		return
	}

	// ── Workspace root normalization ──
	wsRoot, errCode := normalizeWorkspaceRoot(draft.WorkspaceRoot)
	if errCode != "" {
		http.Error(w, errCode+": workspace root", http.StatusBadRequest)
		return
	}

	// ── Path guard: must live under specs/ in workspace root ──
	safePath := filepath.Join(wsRoot, "specs", draft.Path)
	if !strings.HasPrefix(filepath.Clean(safePath), filepath.Join(wsRoot, "specs")) {
		http.Error(w, "path_traversal: spec path must be under specs/", http.StatusForbidden)
		return
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(safePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		http.Error(w, "failed to create parent directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ── Parent guard: verify yaml_text parent matches parent_name ──
	if !parentGuardMatches(draft.YamlText, draft.ParentName) {
		http.Error(w, "parent_mismatch: yaml spec.parent != parent_name", http.StatusBadRequest)
		return
	}

	// ── Write spec file ──
	if err := os.WriteFile(safePath, []byte(draft.YamlText), 0644); err != nil {
		http.Error(w, "write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ── Run make validate in workspace root ──
	cat, msg := runValidateAndClassify(wsRoot)

	resp := struct {
		Path    string                   `json:"path"`
		Written bool                     `json:"written"`
		Validate validationErrorCategory `json:"validate"`
	}{
		Path:    draft.Path,
		Written: true,
		Validate: validationErrorCategory{
			Category: cat,
			Message:  msg,
			Suggest:  suggestForCategory(cat),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// parentGuardMatches checks that the spec.parent in yamlText equals expectedParent.
func parentGuardMatches(yamlText, expectedParent string) bool {
	lines := strings.Split(yamlText, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "parent:") {
			val := strings.TrimPrefix(trimmed, "parent:")
			val = strings.TrimSpace(val)
			// Strip YAML quotes
			val = strings.Trim(val, "\"")
			val = strings.Trim(val, "'")
			return val == expectedParent
		}
	}
	// If no parent field found, allow (spec might not need parent)
	return true
}

// runValidateAndClassify runs make validate in wsRoot and returns error category + message.
func runValidateAndClassify(wsRoot string) (category, message string) {
	cmd := exec.Command("make", "validate")
	cmd.Dir = wsRoot
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err == nil {
		return "ok", "validate passed"
	}

	// Classify the error
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "yaml") && (strings.Contains(lower, "error") || strings.Contains(lower, "parse")):
		return "yaml_parse_error", extractYamlError(output)
	case strings.Contains(lower, "parent") && strings.Contains(lower, "not found"):
		return "parent_not_found", "spec parent reference not found in parent chain"
	case strings.Contains(lower, "level") && (strings.Contains(lower, "mismatch") || strings.Contains(lower, "invalid")):
		return "level_mismatch", "spec level does not match parent level"
	case strings.Contains(lower, "l5") && strings.Contains(lower, "boundary"):
		return "missing_l5_boundary", "L5 spec missing required L5 boundary field"
	default:
		return "validation_error", firstLine(output)
	}
}

func extractYamlError(output string) string {
	lines := strings.Split(output, "\n")
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), "error") || strings.Contains(l, "line ") {
			return strings.TrimSpace(l)
		}
	}
	return firstLine(output)
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func suggestForCategory(cat string) string {
	switch cat {
	case "yaml_parse_error":
		return "Check YAML syntax: indentation, quotes, and special characters"
	case "parent_not_found":
		return "Verify the parent spec exists and the name matches exactly"
	case "level_mismatch":
		return "Ensure the spec level (L2/L3/L4/L5) matches the parent's level+1"
	case "missing_l5_boundary":
		return "Add required L5 boundary fields (io_contract, content, changelog)"
	case "ok":
		return "Spec validated successfully"
	default:
		return "Run 'make validate' locally for details"
	}
}
