package tools

import (
	"vibex/agent/agents/background"
	"vibex/agent/agents/skills"
	"vibex/agent/agents/subagent"
)

func baseSpecs(todo *TodoStore) []Spec {
	return []Spec{
		newTool(
			"bash",
			"Execute a shell command in current workspace and return stdout/stderr.",
			bashHandler,
			reqString("command", "Shell command to run"),
		),
		newTool(
			"read_file",
			"Read file contents from workspace.",
			readFileHandler,
			reqString("path", "Relative file path to read"),
			optInteger("limit", "Max bytes to return (optional)"),
		),
		newTool(
			"write_file",
			"Write file contents into workspace.",
			writeFileHandler,
			reqString("path", "Relative file path to write"),
			reqString("content", "File content to write"),
		),
		newTool(
			"append_file",
			"Append file contents into workspace (chunked writes).",
			appendFileHandler,
			reqString("path", "Relative file path to append"),
			reqString("content", "Chunk content to append"),
		),
		{
			Name:        "todo_set",
			Description: "Replace TODO state with latest full list and current task.",
			Parameters:  todoSetSchema(),
			Handler:     todoSetHandler(todo),
		},
	}
}

func ParentSpecs(
	todo *TodoStore,
	backgroundMgr *background.Manager,
	manager *subagent.Manager,
	runner subagent.Runner,
	skillState *skills.State,
	skillRegistry *skills.Registry,
) []Spec {
	specs := append([]Spec{}, baseSpecs(todo)...)
	specs = append(specs,
		Spec{
			Name:        "bash_bg",
			Description: "Start a background shell command and return a task id immediately.",
			Parameters:  objectSchemaFromFields(reqString("command", "Shell command to run in background")),
			Handler:     bashBgHandler(backgroundMgr),
		},
		Spec{
			Name:        "bg_wait",
			Description: "Wait for one background task to finish, with optional timeout.",
			Parameters:  backgroundWaitSchema(),
			Handler:     bgWaitHandler(backgroundMgr),
		},
		Spec{
			Name:        "bg_list",
			Description: "List known background tasks and their statuses.",
			Parameters:  skillListSchema(),
			Handler:     bgListHandler(backgroundMgr),
		},
		Spec{
			Name:        "skill_list",
			Description: "List all skills from SKILLS_DIR (default <workspace>/skills) and show active flags.",
			Parameters:  skillListSchema(),
			Handler:     skillListHandler(skillRegistry, skillState),
		},
		Spec{
			Name:        "skill_load",
			Description: "Activate one skill by name for subsequent turns.",
			Parameters:  skillNameSchema("Skill name to activate."),
			Handler:     skillLoadHandler(skillRegistry, skillState),
		},
		Spec{
			Name:        "skill_unload",
			Description: "Deactivate one skill by name.",
			Parameters:  skillNameSchema("Skill name to deactivate."),
			Handler:     skillUnloadHandler(skillState),
		},
		Spec{
			Name:        "subagent_spawn",
			Description: "Create and start one sub-agent job asynchronously.",
			Parameters:  subAgentSpawnSchema(),
			Handler:     subAgentSpawnHandler(manager, runner),
		},
		Spec{
			Name:        "subagent_wait",
			Description: "Wait for specific sub-agent jobs (or all when omitted) and return summaries.",
			Parameters:  subAgentWaitSchema(),
			Handler:     subAgentWaitHandler(manager),
		},
		// SpecPilot four-layer data capabilities
		Spec{
			Name:        "dc_list",
			Description: "List all DataCenter keys and values. Returns JSON with data map.",
			Parameters:  objectSchemaFromFields(),
			Handler:     dcListHandler,
		},
		Spec{
			Name:        "dc_get",
			Description: "Get a single DataCenter key value.",
			Parameters:  objectSchemaFromFields(reqString("key", "DataCenter key name")),
			Handler:     dcGetHandler,
		},
		Spec{
			Name:        "dc_set",
			Description: "Set a DataCenter key/value pair.",
			Parameters:  objectSchemaFromFields(reqString("key", "Key name"), reqString("value", "Value to set")),
			Handler:     dcSetHandler,
		},
		Spec{
			Name:        "ec_history",
			Description: "Get EventCenter event history (last 50 events).",
			Parameters:  objectSchemaFromFields(),
			Handler:     ecHistoryHandler,
		},
		Spec{
			Name:        "ec_emit",
			Description: "Emit an event to the EventCenter.",
			Parameters:  objectSchemaFromFields(reqString("event", "Event name"), reqString("payload", "JSON payload string")),
			Handler:     ecEmitHandler,
		},
		Spec{
			Name:        "ec_subscribe",
			Description: "Subscribe a component to an event.",
			Parameters:  objectSchemaFromFields(reqString("event", "Event name"), reqString("subscriber", "Component name")),
			Handler:     ecSubscribeHandler,
		},
		Spec{
			Name:        "ad_list",
			Description: "List all adapters and show which is currently active.",
			Parameters:  objectSchemaFromFields(),
			Handler:     adListHandler,
		},
		Spec{
			Name:        "ad_switch",
			Description: "Switch the active data adapter (mock/http/ws/grpc).",
			Parameters:  objectSchemaFromFields(reqString("adapter", "Adapter name: mock, http, ws, grpc")),
			Handler:     adSwitchHandler,
		},
		Spec{
			Name:        "ad_query",
			Description: "Query data through the current active adapter.",
			Parameters:  objectSchemaFromFields(reqString("query", "SQL-style query")),
			Handler:     adQueryHandler,
		},
		Spec{
			Name:        "spec_list",
			Description: "List all registered specs in the SpecRegistry.",
			Parameters:  objectSchemaFromFields(),
			Handler:     specListHandler,
		},
		Spec{
			Name:        "spec_get",
			Description: "Get a single spec's full details.",
			Parameters:  objectSchemaFromFields(reqString("name", "Spec name")),
			Handler:     specGetHandler,
		},
		Spec{
			Name:        "spec_binding",
			Description: "Check field binding coverage for a spec.",
			Parameters:  objectSchemaFromFields(reqString("name", "Spec name")),
			Handler:     specBindingHandler,
		},
		Spec{
			Name:        "mf_list",
			Description: "List all registered Module Federation components.",
			Parameters:  objectSchemaFromFields(),
			Handler:     mfListHandler,
		},
		Spec{
			Name:        "mf_register",
			Description: "Register a Module Federation component.",
			Parameters:  objectSchemaFromFields(reqString("name", "Component name"), reqString("path", "Component path")),
			Handler:     mfRegisterHandler,
		},
		Spec{
			Name:        "mf_resolve",
			Description: "Resolve MF components automatically from a spec name.",
			Parameters:  objectSchemaFromFields(reqString("spec", "Spec name")),
			Handler:     mfResolveHandler,
		},
	)
	return specs
}

func ChildSpecs(todo *TodoStore) []Spec {
	return append([]Spec{}, baseSpecs(todo)...)
}
