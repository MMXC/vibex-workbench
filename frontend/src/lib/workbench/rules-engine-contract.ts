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
	failed_node_ref?: {
		node_id: string;
		kind?: string;
		graph_id?: string;
		label?: string;
	};
	error_message: string;
	failure_type: string;
	inputs_snapshot?: Record<string, unknown>;
	tool_name?: string;
	call_id?: string;
};

export type ValidationItemRunResult = {
	item_id: string;
	ok: boolean;
	channel: 'tdd_run' | 'qa_playwright' | 'verify_specs' | 'make_validate' | 'cdp_validate' | 'unknown';
	source?: 'custom_agent_flow' | 'legacy_qa' | 'builtin';
	started_at: string;
	finished_at?: string;
	output?: string;
	error?: string;
	exit_code?: number;
	not_implemented?: boolean;
};

export type OrchestrationTracePayload = {
	phase: RulesPipelinePhase;
	node: {
		node_id: string;
		kind?: string;
		graph_id?: string;
		label?: string;
	};
	previous_phase?: RulesPipelinePhase;
	parent_invocation_id?: string;
	run_id?: string;
	child_run_id?: string;
	timestamp_unix?: number;
	payload_ref?: string;
	outcome_summary?: string;
};

export type TestEnvDeploymentMode = 'user_managed' | 'agent_managed' | 'wails_embedded';

export type CDPTargetEnvRef = {
	deployment?: TestEnvDeploymentMode;
	host?: string;
	port?: number;
	timeout_sec?: number;
	session_id?: string;
};

export type CDPAssertion = {
	id?: string;
	type: string;
	selector?: string;
	value?: string;
};

export type CDPValidationStep = {
	id?: string;
	url?: string;
	actions?: Record<string, unknown>[];
	assertions?: CDPAssertion[];
	timeout_sec?: number;
};

export type CDPValidationPlan = {
	plan_id: string;
	target_env: CDPTargetEnvRef;
	entry_url?: string;
	steps: CDPValidationStep[];
	screenshot_on_fail?: boolean;
};
