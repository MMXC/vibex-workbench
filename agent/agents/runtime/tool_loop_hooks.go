package runtime

import "vibex/agent/vibex/domain/rulesengine"

// ToolLoopHooks wires optional callbacks for SSE/UI (e.g. cmd/web). Nil hooks = CLI — no side effects.
type ToolLoopHooks struct {
	OnAssistantDelta func(text string, isFinal bool)
	OnToolCalled     func(name, callID string, args map[string]any)
	OnToolCompleted  func(name, callID, result string)
	OnRepairDecision func(decision rulesengine.RepairDecision, envelope rulesengine.RepairEnvelope)
}