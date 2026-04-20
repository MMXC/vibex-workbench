// Package domain provides the VibeX-specific tool domain.
// It wires vibex spec tools and TDD tools together with a shared broadcaster.
package domain

import (
	rt "vibex/agent/agents/runtime/tools"
	"vibex/agent/vibex/domain/spec"
	"vibex/agent/vibex/domain/tdd"
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

// ToolSpecs returns all vibex tool specs (spec tools + TDD tools).
func (r *Registry) ToolSpecs() []rt.Spec {
	var specs []rt.Spec
	specs = append(specs, spec.ToolSpecs(r.WorkspaceDir, r.Broadcaster, r.SetStepType)...)
	specs = append(specs, tdd.ToolSpecs()...)
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

	return handlers
}
