/** Mirrors agent/vibex/domain/rulesengine for UI/session serialization. */

export type RulesPipelinePhase =
	| 'start'
	| 'clarify'
	| 'strong_validation_plan'
	| 'user_verification'
	| 'user_confirm'
	| 'route_execute'
	| 'goal_check'
	| 'strong_terminal_check'
	| 'end'
	| 'repair_orchestrator';

export type StrongValidationItem = {
	id: string;
	command?: string;
	tool_call_template?: string;
	cwd?: string;
	timeout_sec?: number;
	expect_signal?: string;
	label?: string;
};

export type StrongValidationPlan = {
	plan_id: string;
	slot_binding?: string;
	spec_path?: string;
	items: StrongValidationItem[];
};

export type VerificationOutcome = 'passed' | 'failed';

export type VerificationSubmission = {
	submission_id: string;
	plan_id: string;
	slot_id: string;
	spec_path?: string;
	outcome: VerificationOutcome;
	artifact_refs?: string[];
	notes?: string;
};

export type RepairDecisionPayload = {
	target_phase: RulesPipelinePhase;
	reason_code: string;
	developer_hint?: string;
	allowed: boolean;
};

export type RepairEnvelopePayload = {
	failed_node_id?: string;
	error_message: string;
	failure_type: string;
	inputs_snapshot?: Record<string, unknown>;
	tool_name?: string;
	call_id?: string;
};
