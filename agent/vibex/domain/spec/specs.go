package spec

import (
	rt "vibex/agent/agents/runtime/tools"
)

// ToolSpecs returns the 8 spec-tool specs for registration.
func ToolSpecs(workspaceDir string, bc Broadcaster, setStepType func(threadID, stepType string)) []rt.Spec {
	return []rt.Spec{
		{
			Name:        "spec_designer",
			Description: "Create a spec YAML draft from user intent (L1 goal). " +
				"Auto-emits canvas.spec_created SSE event to update the canvas. " +
				"After creation, await user confirmation before proceeding.",
			Parameters: objectSchema(
				reqField("intent", "string", "User intent in natural language"),
			),
			Handler: MakeSpecDesignerHandler(workspaceDir, bc, setStepType),
		},
		{
			Name:        "spec_feature",
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
			Name:        "spec_validate",
			Description: "Validate a spec YAML for syntax and required fields.",
			Parameters: objectSchema(
				reqField("spec_path", "string", "Path to the spec YAML file"),
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
			Description: "Run `make validate` in vibex-workbench to check all spec YAML files.",
			Parameters:  objectSchema(),
			Handler:     MakeMakeValidateHandler(workspaceDir, setStepType),
		},
		{
			Name:        "make_generate",
			Description: "Run `make generate` in vibex-workbench — the spec-to-code step. " +
				"Creates types.ts, *.Skeleton.svelte, and stubs from spec YAML. " +
				"Use after creating or updating a spec file. This is the core of spec-driven development.",
			Parameters:  objectSchema(),
			Handler:     MakeMakeGenerateHandler(workspaceDir, setStepType),
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
			Name:        "workspace_detect_state",
			Description: "Detect workspace state: empty (no specs/gen.py) / partial (specs only) / ready (all scaffolding present). " +
				"Returns state, detection signals, and next-step suggestions. " +
				"Use this when starting a new project or when the user asks about project status.",
			Parameters: objectSchema(
				optField("workspace_root", "string", "Path to workspace root (defaults to WORKSPACE_ROOT env var)"),
			),
			Handler: MakeWorkspaceDetectStateHandler(workspaceDir, setStepType),
		},
		{
			Name:        "workspace_scaffold",
			Description: "Scaffold a new VibeX project from scratch. " +
				"Creates the minimal directory structure: specs/, generators/, spec-templates/, Makefile, frontend/package.json. " +
				"Uses vibex-workbench scaffold files as templates. " +
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
