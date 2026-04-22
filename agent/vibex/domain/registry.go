// Package domain provides the VibeX-specific tool domain.
// It wires vibex spec tools, TDD tools, and memlace together with a shared broadcaster.
package domain

import (
	rt "vibex/agent/agents/runtime/tools"
	"vibex/agent/vibex/domain/spec"
	"vibex/agent/vibex/domain/tdd"
	"vibex/generators/memlace"
)

// Registry holds all vibex-specific tools for a given workspace.
type Registry struct {
	WorkspaceDir string
	Broadcaster  func(threadID, event string, data interface{})
	SetStepType func(threadID, stepType string) // updates per-thread step type
}

// NewRegistry creates a new vibex tool registry.
func NewRegistry(workspaceDir string, bc func(threadID, event string, data interface{}), setStepType func(threadID, stepType string)) *Registry {
	return &Registry{
		WorkspaceDir: workspaceDir,
		Broadcaster:  bc,
		SetStepType: setStepType,
	}
}

// ToolSpecs returns all vibex tool specs (spec tools + TDD tools + memlace).
func (r *Registry) ToolSpecs() []rt.Spec {
	var specs []rt.Spec
	specs = append(specs, spec.ToolSpecs(r.WorkspaceDir, r.Broadcaster, r.SetStepType)...)
	specs = append(specs, tdd.ToolSpecs()...)
	// MemLace tools
	memSpecs := memlace.ToolSchemas()
	for _, ms := range memSpecs {
		specs = append(specs, rt.Spec{
			Name:        ms["name"].(string),
			Description: ms["description"].(string),
			Parameters:  ms["parameters"].(map[string]any),
			Handler:     memlace.NoOpHandler, // real handlers set in ToolHandlers
		})
	}
	return specs
}

// ToolHandlers returns a map from tool name → handler for vibex tools.
func (r *Registry) ToolHandlers() map[string]rt.Handler {
	handlers := make(map[string]rt.Handler)

	// Spec tools
	for _, s := range spec.ToolSpecs(r.WorkspaceDir, r.Broadcaster, r.SetStepType) {
		handlers[s.Name] = s.Handler
	}

	// TDD tools
	handlers["tdd_design"] = tdd.MakeTddDesignHandler(r.WorkspaceDir, r.Broadcaster)
	handlers["tdd_run"] = tdd.MakeTddRunHandler(r.WorkspaceDir, r.Broadcaster)
	handlers["tdd_iterate"] = tdd.MakeTddIterateHandler(r.WorkspaceDir, r.Broadcaster)

	// MemLace tools
	handlers["memlace_search"] = memlace.SearchHandler
	handlers["memlace_write_pref"] = memlace.WritePrefHandler
	handlers["memlace_clarification_start"] = memlace.ClarificationStartHandler
	handlers["memlace_clarification_qa"] = memlace.ClarificationQAHandler
	handlers["memlace_clarification_draft"] = memlace.ClarificationDraftHandler
	handlers["memlace_clarification_confirm"] = memlace.ClarificationConfirmHandler
	handlers["memlace_session_list"] = memlace.SessionListHandler

	return handlers
}
