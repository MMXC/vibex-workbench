package main

import (
	"encoding/json"
	"net/http"

	"vibex/agent/vibex/domain/toolrouting"
)

func agentToolsHandler(w http.ResponseWriter, r *http.Request) {
	reg := toolrouting.NewRegistry(cfg.WorkspaceDir)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, map[string]any{"tools": reg.ListTools()})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func agentCustomToolHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req toolrouting.CustomToolConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	reg := toolrouting.NewRegistry(cfg.WorkspaceDir)
	tool, err := reg.RegisterCustomTool(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tool": tool})
}

func agentPlanGraphHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Goal     string `json:"goal"`
		SpecPath string `json:"spec_path"`
		Mode     string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	reg := toolrouting.NewRegistry(cfg.WorkspaceDir)
	writeJSON(w, map[string]any{"graph": reg.CreatePlanGraph(req.Goal, req.SpecPath, req.Mode)})
}

func agentToolRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Graph toolrouting.PlanGraph `json:"graph"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	reg := toolrouting.NewRegistry(cfg.WorkspaceDir)
	writeJSON(w, map[string]any{"route": reg.RouteGraph(req.Graph)})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
