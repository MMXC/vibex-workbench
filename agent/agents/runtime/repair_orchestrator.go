package runtime

import (
	"strings"

	"vibex/agent/vibex/domain/rulesengine"
)

const maxRepairAttempts = 12

// DecideRepair returns a bounded reroute decision from a repair envelope (spec SLICE-agent-repair-orchestrator-runtime).
func DecideRepair(env rulesengine.RepairEnvelope, loopStep int) rulesengine.RepairDecision {
	if loopStep >= maxRepairAttempts {
		return rulesengine.RepairDecision{
			TargetPhase:   rulesengine.PhaseEnd,
			ReasonCode:    "repair_exhausted",
			DeveloperHint: "Tool loop repair budget exhausted; summarize blockers for the user.",
			Allowed:       false,
		}
	}

	ft := strings.TrimSpace(env.FailureType)
	switch ft {
	case rulesengine.FailurePolicyBlocked, rulesengine.FailureIntentAmbiguous:
		return rulesengine.RepairDecision{
			TargetPhase: rulesengine.PhaseClarify,
			ReasonCode:  "repair_clarify",
			DeveloperHint: "Clarify intent and use structured function_call tool outputs only; " +
				"avoid pseudo tool-call text or narrative file-write claims.",
			Allowed: true,
		}
	case rulesengine.FailureToolMissing:
		return rulesengine.RepairDecision{
			TargetPhase:   rulesengine.PhaseRouteExecute,
			ReasonCode:    "repair_tool_routing",
			DeveloperHint: "Pick a registered tool name from the tool list; verify required JSON arguments.",
			Allowed:       true,
		}
	case rulesengine.FailureExecutionFailed:
		return rulesengine.RepairDecision{
			TargetPhase:   rulesengine.PhaseRouteExecute,
			ReasonCode:    "repair_execution",
			DeveloperHint: "Fix arguments or preconditions, then retry with the same or an alternative tool.",
			Allowed:       true,
		}
	case rulesengine.FailureGoalNotReached:
		return rulesengine.RepairDecision{
			TargetPhase:   rulesengine.PhaseStrongValidationPlan,
			ReasonCode:    "repair_goal",
			DeveloperHint: "Produce explicit tool calls or a concrete plan; empty replies are not sufficient.",
			Allowed:       true,
		}
	case rulesengine.FailureOrchestrationOutOfScope:
		return rulesengine.RepairDecision{
			TargetPhase:   rulesengine.PhaseRepairOrchestrator,
			ReasonCode:    "orchestration_out_of_scope",
			DeveloperHint: "Sub-orchestration must stay within the engine node whitelist.",
			Allowed:       false,
		}
	default:
		return rulesengine.RepairDecision{
			TargetPhase:   rulesengine.PhaseClarify,
			ReasonCode:    "repair_generic",
			DeveloperHint: "Repair the last failure and continue with structured tool calls.",
			Allowed:       true,
		}
	}
}
