package spec

import (
	rt "vibex/agent/agents/runtime/tools"
)

// ToolSpecs returns the 8 spec-tool specs for registration.
func ToolSpecs(workspaceDir string, bc Broadcaster, setStepType func(threadID, stepType string)) []rt.Spec {
	return []rt.Spec{
		{
			Name: "spec_designer",
			Description: "Create a spec YAML draft from user intent (L1 goal). " +
				"Auto-emits canvas.spec_created SSE event to update the canvas. " +
				"After creation, await user confirmation before proceeding.",
			Parameters: objectSchema(
				reqField("intent", "string", "User intent in natural language"),
			),
			Handler: MakeSpecDesignerHandler(workspaceDir, bc, setStepType),
		},
		{
			Name: "spec_feature",
			Description: "Break a confirmed goal spec into a feature spec (L4). " +
				"Creates both feature and uiux sub-spec in specs/feature/<name>/. " +
				"AUTO-CHAIN (no user action needed): " +
				"(1) make validate → (2) make generate → (3) canvas.spec_created SSE emitted. " +
				"After this tool, the full pipeline is complete — go to next task.",
			Parameters: objectSchema(
				reqField("parent_spec_id", "string", "ID of the parent goal/methodology spec"),
				reqField("feature_name", "string", "Name of the feature (spaces become hyphens)"),
			),
			Handler: MakeSpecFeatureHandler(workspaceDir, bc, setStepType),
		},
		{
			Name: "spec_validate",
			Description: "Validate a spec YAML for syntax and required fields. " +
				"When editing specs for the user's opened project in Workbench, pass workspace_root so paths resolve under that repo (not the workbench install dir).",
			Parameters: objectSchema(
				reqField("spec_path", "string", "Relative path under specs/ from workspace root (e.g. specs/L1-goal/foo.yaml), or absolute path inside that workspace"),
				optField("workspace_root", "string", "Open project root; defaults to agent WORKSPACE_DIR when omitted"),
			),
			Handler: MakeSpecValidateHandler(workspaceDir, setStepType),
		},
		{
			Name:        "canvas_update",
			Description: "Update Canvas visualization for the current thread. Show progress nodes, behavior flows, spec relationships.",
			Parameters: objectSchema(
				reqField("thread_id", "string", "Thread/canvas ID"),
				reqField("event_type", "string", "Event: node_added, edge_added, highlight, status_change, spec_linked"),
				optField("payload", "string", "JSON payload"),
				optField("title", "string", "Display title"),
				optField("content", "string", "Node content"),
			),
			Handler: MakeCanvasUpdateHandler(bc, setStepType),
		},
		{
			Name:        "spec_sync",
			Description: "Sync spec changes with prototypes/generated code. push=spec→code, pull=code→spec.",
			Parameters: objectSchema(
				reqField("spec_path", "string", "Path to spec YAML"),
				optField("direction", "string", "push or pull"),
				optField("target_file", "string", "Specific target file"),
			),
			Handler: MakeSpecSyncHandler(workspaceDir, setStepType),
		},
		{
			Name:        "make_validate",
			Description: "Run `make validate` in the current workspace to check all spec YAML files.",
			Parameters:  objectSchema(),
			Handler:     MakeMakeValidateHandler(workspaceDir, setStepType),
		},
		{
			Name: "make_generate",
			Description: "Run `make generate` in the current workspace — the spec-to-code step. " +
				"Creates types.ts, *.Skeleton.svelte, and stubs from spec YAML. " +
				"Use after creating or updating a spec file. This is the core of spec-driven development.",
			Parameters: objectSchema(),
			Handler:    MakeMakeGenerateHandler(workspaceDir, setStepType),
		},
		{
			Name:        "bug_report",
			Description: "Create a bug-changelog entry. Attach to a spec or create standalone.",
			Parameters: objectSchema(
				optField("spec_path", "string", "Spec path to attach bug to"),
				reqField("bug_description", "string", "Bug description"),
				optField("severity", "string", "critical/high/medium/low"),
				optField("repro_steps", "string", "Steps to reproduce"),
				optField("expected_fix", "string", "Expected behavior after fix"),
			),
			Handler: MakeBugReportHandler(workspaceDir, setStepType),
		},
		{
			Name:        "spec_result_track",
			Description: "Mark a result confirmed/pending. Emits SSE for frontend Result Tracker.",
			Parameters: objectSchema(
				reqField("spec_path", "string", "Spec path"),
				reqField("result_index", "integer", "Index in result[] (0-based)"),
				reqField("confirmed", "boolean", "Whether confirmed"),
				optField("confirmed_by", "string", "agent or user"),
				optField("notes", "string", "Additional notes"),
			),
			Handler: MakeSpecResultTrackHandler(bc),
		},
		{
			Name: "workspace_detect_state",
			Description: "Detect workspace state: empty (no specs/gen.py) / partial (specs only) / ready (all scaffolding present). " +
				"Returns state, detection signals, and next-step suggestions. " +
				"Use this when starting a new project or when the user asks about project status.",
			Parameters: objectSchema(
				optField("workspace_root", "string", "Path to workspace root (defaults to WORKSPACE_ROOT env var)"),
			),
			Handler: MakeWorkspaceDetectStateHandler(workspaceDir, setStepType),
		},
		{
			Name:        "workspace_specs_list",
			Description: "List all spec YAML files in the current workspace. Use this for natural-language requests like '列出 specs' or '当前有哪些规格'.",
			Parameters: objectSchema(
				optField("workspace_root", "string", "Path to workspace root (defaults to current WORKSPACE_DIR)"),
			),
			Handler: MakeWorkspaceSpecsListHandler(workspaceDir, setStepType),
		},
		{
			Name:        "workspace_specs_convention",
			Description: "Describe the VibeX spec hierarchy, naming convention, canonical slots, and parent-chain rules.",
			Parameters:  objectSchema(),
			Handler:     MakeWorkspaceSpecsConventionHandler(workspaceDir, setStepType),
		},
		{
			Name:        "verify_spec_suite",
			Description: "Run the structured spec verification suite for parent chains, completeness, behaviors, and file existence. Falls back to make validate when verify_specs binary is unavailable.",
			Parameters: objectSchema(
				optField("workspace_root", "string", "Path to workspace root (defaults to current WORKSPACE_DIR)"),
				optField("checks", "string", "Optional comma-separated checks"),
				optField("levels", "string", "Optional comma-separated levels"),
			),
			Handler: MakeVerifySpecSuiteHandler(workspaceDir, setStepType),
		},
		{
			Name:        "workspace_run_make",
			Description: "Run an allowlisted Makefile target in the current workspace and summarize output. Prefer this over generic bash for validate/generate/lint-specs.",
			Parameters: objectSchema(
				reqField("target", "string", "Make target: validate, lint-specs, generate, or test"),
				optField("workspace_root", "string", "Path to workspace root (defaults to current WORKSPACE_DIR)"),
			),
			Handler: MakeWorkspaceRunMakeHandler(workspaceDir, setStepType),
		},
		{
			Name: "workspace_agent_flow_qa",
			Description: "Run a user-workspace custom QA flow for generate/run/red-green/screenshot. " +
				"Reads flow JSON from the opened workspace (default .agents/flows/qa-agent-flow.json) " +
				"and executes steps sequentially. Supports step types: make/cmd/cdp_validate.",
			Parameters: objectSchema(
				reqField("workspace_root", "string", "Opened project root path (required)"),
				optField("flow_path", "string", "Relative path to workflow JSON in workspace (default .agents/flows/qa-agent-flow.json)"),
				optField("stop_on_failure", "boolean", "Stop pipeline on first failed step (default true)"),
			),
			Handler: MakeWorkspaceAgentFlowQAHandler(workspaceDir, setStepType),
		},
		{
			Name: "cdp_validate",
			Description: "Run CDP validation steps against a real dev/test browser endpoint. " +
				"Use target_env.host/port (e.g. 127.0.0.1:9222) and provide steps with assertions. " +
				"Supports text_contains / selector_visible / url_matches checks.",
			Parameters: objectSchema(
				reqField("plan_id", "string", "Validation plan ID"),
				reqField("target_env", "object", "CDP endpoint: deployment/host/port/timeout_sec/session_id"),
				optField("entry_url", "string", "Initial URL to open before steps"),
				reqField("steps", "array", "Validation steps [{id,url,actions[],assertions[],timeout_sec}]"),
				optField("screenshot_on_fail", "boolean", "Capture screenshots under .vibex/cdp-snapshots on assertion failure"),
			),
			Handler: MakeCDPValidateHandler(workspaceDir, setStepType),
		},
		{
			Name:        "governance_status",
			Description: "Summarize workspace governance state including panorama presence and spec counts by level.",
			Parameters: objectSchema(
				optField("workspace_root", "string", "Path to workspace root (defaults to current WORKSPACE_DIR)"),
			),
			Handler: MakeGovernanceStatusHandler(workspaceDir, setStepType),
		},
		{
			Name: "workspace_scaffold",
			Description: "Scaffold a new VibeX project from scratch. " +
				"Creates the minimal directory structure: specs/, generators/, spec-templates/, Makefile, frontend/package.json. " +
				"Uses built-in VibeX scaffold templates shipped with the Workbench. " +
				"IMPORTANT: Must call state_detector FIRST to check current state. " +
				"Do NOT scaffold into an existing 'ready' workspace.",
			Parameters: objectSchema(
				reqField("workspace_root", "string", "Target workspace root path"),
				optField("project_name", "string", "Project name (kebab-case, used for spec filenames)"),
				optField("owner", "string", "Owner username"),
				optField("dry_run", "boolean", "Preview files without writing (default false)"),
				optField("confirm", "boolean", "Skip confirmation prompt (default false)"),
			),
			Handler: MakeWorkspaceScaffoldHandler(workspaceDir, setStepType),
		},
		{
			Name: "workspace_specs_bootstrap",
			Description: "Generate L1–L5 placeholder specs via workspace-bootstrap skill execute " +
				"(workspace_bootstrap_contract chain: goal → skeleton → MOD shell → FEAT starter → SLICE). " +
				"Target must be a writable workspace root. Call workspace_detect_state first. " +
				"Requires confirm=true. Use overwrite=true to replace existing bootstrap files. " +
				"Typically after workspace_scaffold when spec-templates/ exists, or on any repo that already has specs/ layout. " +
				"Fallback to legacy generators/spec_workspace_bootstrap.py only when skill execute is unavailable.",
			Parameters: objectSchema(
				reqField("workspace_root", "string", "Target workspace root path"),
				optField("project_slug", "string", "Kebab-case id for spec names (default: directory basename)"),
				optField("project_name", "string", "Alias for project_slug"),
				optField("owner", "string", "Owner username for YAML meta"),
				optField("overwrite", "boolean", "Overwrite generated bootstrap YAMLs if they already exist (default false)"),
				optField("confirm", "boolean", "Must be true to write"),
			),
			Handler: MakeWorkspaceSpecsBootstrapHandler(workspaceDir, setStepType),
		},
		{
			Name: "spec_write",
			Description: "Write or overwrite a spec YAML file at a given path. " +
				"Use this to save edited spec content back to disk. " +
				"MUST pass workspace_root when working on a repo opened in Workbench so files land in that project (not the Workbench install directory). " +
				"Auto-creates parent directories if needed. " +
				"After writing, runs a quick validation check and emits canvas.spec_modified event.",
			Parameters: objectSchema(
				reqField("spec_path", "string", "Relative path under specs/ from workspace root (e.g. specs/L1-goal/my-goal.yaml)"),
				reqField("content", "string", "Full YAML content to write"),
				optField("workspace_root", "string", "Open project root; defaults to agent WORKSPACE_DIR when omitted"),
				optField("validate_after", "boolean", "Run validation after write (default true)"),
			),
			Handler: MakeSpecWriteHandler(workspaceDir, bc, setStepType),
		},
		{
			Name: "spec_patch_apply",
			Description: "Apply a partial JSON patch to an existing spec YAML (field-level merge, no full-file rewrite). " +
				"Only allows paths under specs/. MUST pass workspace_root when patching specs for the user's opened project.",
			Parameters: objectSchema(
				reqField("spec_path", "string", "Relative path under specs/ from workspace root"),
				reqField("patch_json", "string", "JSON object string to merge into YAML (e.g. {\"prototype\":{\"status\":\"prototype\"}})"),
				optField("workspace_root", "string", "Open project root; defaults to agent WORKSPACE_DIR when omitted"),
				optField("validate_after", "boolean", "Run validation after patch (default true)"),
			),
			Handler: MakeSpecPatchApplyHandler(workspaceDir, bc, setStepType),
		},
	}
}

// ─────────────────────────────────────────────────────────────
// Schema Helpers
// ─────────────────────────────────────────────────────────────

type schemaField struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

func objectSchema(fields ...schemaField) map[string]any {
	props := make(map[string]any)
	req := make([]string, 0)
	for _, f := range fields {
		props[f.Name] = map[string]any{
			"type":        f.Type,
			"description": f.Description,
		}
		if f.Required {
			req = append(req, f.Name)
		}
	}
	result := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(req) > 0 {
		result["required"] = req
	}
	result["additionalProperties"] = false
	return result
}

func reqField(name, typ, desc string) schemaField {
	return schemaField{Name: name, Type: typ, Description: desc, Required: true}
}

func optField(name, typ, desc string) schemaField {
	return schemaField{Name: name, Type: typ, Description: desc, Required: false}
}
