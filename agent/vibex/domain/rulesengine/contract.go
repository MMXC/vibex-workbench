// Package rulesengine holds shared contract types for the rules pipeline (FEAT-agent-rules-engine).
package rulesengine

// RulesPipelinePhase identifies a stage in the default ordered pipeline.
type RulesPipelinePhase string

const (
	PhaseStart                 RulesPipelinePhase = "start"
	PhaseClarify               RulesPipelinePhase = "clarify"
	PhaseStrongValidationPlan  RulesPipelinePhase = "strong_validation_plan"
	PhaseUserVerification      RulesPipelinePhase = "user_verification"
	PhaseUserConfirm           RulesPipelinePhase = "user_confirm"
	PhaseRouteExecute          RulesPipelinePhase = "route_execute"
	PhaseGoalCheck             RulesPipelinePhase = "goal_check"
	PhaseStrongTerminalCheck   RulesPipelinePhase = "strong_terminal_check"
	PhaseEnd                   RulesPipelinePhase = "end"
	PhaseRepairOrchestrator    RulesPipelinePhase = "repair_orchestrator"
)

// Failure types align with FilterEngine classification (extend with product-specific codes).
const (
	FailureToolMissing              = "tool_missing"
	FailureIntentAmbiguous          = "intent_ambiguous"
	FailurePolicyBlocked            = "policy_blocked"
	FailureExecutionFailed          = "execution_failed"
	FailureGoalNotReached           = "goal_not_reached"
	FailureOrchestrationOutOfScope  = "orchestration_out_of_scope"
)

// StrongValidationItem is one executable check (shell command and/or tool template).
type StrongValidationItem struct {
	ID               string `json:"id"`
	Command          string `json:"command,omitempty"`
	ToolCallTemplate string `json:"tool_call_template,omitempty"`
	Cwd              string `json:"cwd,omitempty"`
	TimeoutSec       int    `json:"timeout_sec,omitempty"`
	ExpectSignal     string `json:"expect_signal,omitempty"` // red | green | either
	Label            string `json:"label,omitempty"`
}

// StrongValidationPlan groups executable validation entries for a slot.
type StrongValidationPlan struct {
	PlanID      string               `json:"plan_id"`
	SlotBinding string               `json:"slot_binding,omitempty"`
	SpecPath    string               `json:"spec_path,omitempty"`
	Items       []StrongValidationItem `json:"items"`
}

// VerificationOutcome is the user-submitted result for user_verification.
type VerificationOutcome string

const (
	OutcomePassed VerificationOutcome = "passed"
	OutcomeFailed VerificationOutcome = "failed"
)

// VerificationSubmission persists the RED/GREEN gate outcome for a plan + slot.
type VerificationSubmission struct {
	SubmissionID string              `json:"submission_id"`
	PlanID       string              `json:"plan_id"`
	SlotID       string              `json:"slot_id"`
	SpecPath     string              `json:"spec_path,omitempty"`
	Outcome      VerificationOutcome `json:"outcome"`
	ArtifactRefs []string            `json:"artifact_refs,omitempty"`
	Notes        string              `json:"notes,omitempty"`
}

// RepairEnvelope is raised when any gate or tool execution fails (propagate upward).
type RepairEnvelope struct {
	FailedNodeID   string         `json:"failed_node_id,omitempty"`
	FailedNodeRef  *AgentNodeRef  `json:"failed_node_ref,omitempty"` // optional structured failure site for audit / UI
	ErrorMessage   string         `json:"error_message"`
	FailureType    string         `json:"failure_type"`
	InputsSnapshot map[string]any `json:"inputs_snapshot,omitempty"`
	ToolName       string         `json:"tool_name,omitempty"`
	CallID         string         `json:"call_id,omitempty"`
}

// RepairDecision is a bounded reroute suggestion inside the engine whitelist.
type RepairDecision struct {
	TargetPhase   RulesPipelinePhase `json:"target_phase"`
	ReasonCode    string             `json:"reason_code"`
	DeveloperHint string             `json:"developer_hint,omitempty"`
	Allowed       bool               `json:"allowed"`
}

// PhaseTransitionEvent is emitted when the pipeline phase changes.
type PhaseTransitionEvent struct {
	Phase         RulesPipelinePhase `json:"phase"`
	PreviousPhase RulesPipelinePhase `json:"previous_phase,omitempty"`
	TimestampUnix int64              `json:"timestamp_unix,omitempty"`
	PayloadRef    string             `json:"payload_ref,omitempty"`
}

// DynamicSlotInvocation records a dynamic execution slot run.
type DynamicSlotInvocation struct {
	SlotKind      string `json:"slot_kind"`
	PayloadRef    string `json:"payload_ref,omitempty"`
	ResultSummary string `json:"result_summary,omitempty"`
}

// DynamicExecutionSlotRegistry maps engine node_kind to a stable handler reference name.
type DynamicExecutionSlotRegistry struct {
	Entries map[string]string `json:"entries"` // node_kind -> handler_ref
}

// AgentNodeKind classifies an orchestration node for whitelist, routing and audit.
// Built-in kinds are enumerated below; product-specific kinds use the same string
// namespace and must appear in the engine node whitelist (see FEAT-agent-rules-engine).
type AgentNodeKind string

const (
	NodeKindPlanGraphStep AgentNodeKind = "plan_graph_step"
	NodeKindToolRoute     AgentNodeKind = "tool_route"
	NodeKindSubagent      AgentNodeKind = "subagent"
	NodeKindStrongGate    AgentNodeKind = "strong_validation_gate"
	NodeKindUserGate      AgentNodeKind = "user_verification_gate"
	NodeKindRepairHook    AgentNodeKind = "repair_orchestrator"
)

// AgentNodeRef is a stable handle for one node in an orchestration graph or repair chain.
// Use NodeID consistently with RepairEnvelope.FailedNodeID when referring to the same failure site.
type AgentNodeRef struct {
	NodeID  string        `json:"node_id"`
	Kind    AgentNodeKind `json:"kind,omitempty"`
	GraphID string        `json:"graph_id,omitempty"` // plan graph / run tree id when applicable
	Label   string        `json:"label,omitempty"`
}

// AgentNodeRegistryEntry maps a node kind to a handler hook name (complements DynamicExecutionSlotRegistry).
// Prefer registering by Kind for extension packs; FixedNodeID optionally pins a singleton gate node.
type AgentNodeRegistryEntry struct {
	Kind        AgentNodeKind `json:"kind"`
	HandlerRef  string        `json:"handler_ref"`
	FixedNodeID string        `json:"fixed_node_id,omitempty"`
	Description string        `json:"description,omitempty"`
}

// AgentNodeRegistry is the engine-level whitelist of orchestration node kinds (extend via YAML/rule packs later).
type AgentNodeRegistry struct {
	Entries []AgentNodeRegistryEntry `json:"entries"`
}

// OrchestrationTraceEvent records one visible step for replay, SSE and cross-run lineage (agent node-centric).
type OrchestrationTraceEvent struct {
	Phase               RulesPipelinePhase `json:"phase"`
	Node                AgentNodeRef       `json:"node"`
	PreviousPhase       RulesPipelinePhase `json:"previous_phase,omitempty"`
	ParentInvocationID  string             `json:"parent_invocation_id,omitempty"` // tool call or upstream span
	RunID               string             `json:"run_id,omitempty"`               // session/run slice id
	ChildRunID          string             `json:"child_run_id,omitempty"`         // subagent or forked run
	TimestampUnix       int64              `json:"timestamp_unix,omitempty"`
	PayloadRef          string             `json:"payload_ref,omitempty"`
	OutcomeSummary      string             `json:"outcome_summary,omitempty"`
}
