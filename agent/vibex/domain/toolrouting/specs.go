package toolrouting

import (
	"encoding/json"

	rt "vibex/agent/agents/runtime/tools"
)

func ToolSpecs(workspaceDir string) []rt.Spec {
	r := NewRegistry(workspaceDir)
	return []rt.Spec{
		{
			Name:        "plan_graph_create",
			Description: "Create a plan graph from a user goal/spec. This does not write code; it creates a reviewable execution graph.",
			Parameters: objectSchema(
				reqField("goal", "string", "User goal or implementation intent"),
				optField("spec_path", "string", "Optional spec path that anchors the graph"),
				optField("slot_id", "string", "Optional canonical spec slot id for slot-aware routing"),
				optField("mode", "string", "Routing mode, default plan-first"),
			),
			Handler: func(arguments string) string {
				var req struct {
					Goal     string `json:"goal"`
					SpecPath string `json:"spec_path"`
					SlotID   string `json:"slot_id"`
					Mode     string `json:"mode"`
				}
				if err := json.Unmarshal([]byte(arguments), &req); err != nil {
					return "invalid args: " + err.Error()
				}
				return encode(r.CreatePlanGraph(req.Goal, req.SpecPath, req.Mode, req.SlotID))
			},
		},
		{
			Name:        "tool_route_preview",
			Description: "Route a plan graph to builtin/custom tools. Preview only; custom tools are not executed in MVP.",
			Parameters: objectSchema(
				reqField("graph_json", "string", "PlanGraph JSON from plan_graph_create or user-edited graph"),
			),
			Handler: func(arguments string) string {
				var req struct {
					GraphJSON string `json:"graph_json"`
				}
				if err := json.Unmarshal([]byte(arguments), &req); err != nil {
					return "invalid args: " + err.Error()
				}
				var graph PlanGraph
				if err := json.Unmarshal([]byte(req.GraphJSON), &graph); err != nil {
					return "invalid graph_json: " + err.Error()
				}
				return encode(r.RouteGraph(graph))
			},
		},
		{
			Name:        "tool_registry_list",
			Description: "List builtin and custom tools available to the plan graph router.",
			Parameters:  objectSchema(),
			Handler: func(arguments string) string {
				return encode(r.ListTools())
			},
		},
		{
			Name:        "custom_tool_register",
			Description: "Register a custom routing target. MVP stores metadata only and will not execute arbitrary commands.",
			Parameters: objectSchema(
				reqField("name", "string", "Tool name, e.g. design_lint"),
				reqField("kind", "string", "Plan node kind this tool handles, e.g. graph.design"),
				optField("description", "string", "Tool description"),
				optField("execute_mode", "string", "Must be metadata_only in MVP"),
			),
			Handler: func(arguments string) string {
				var cfg CustomToolConfig
				if err := json.Unmarshal([]byte(arguments), &cfg); err != nil {
					return "invalid args: " + err.Error()
				}
				tool, err := r.RegisterCustomTool(cfg)
				if err != nil {
					return "custom tool rejected: " + err.Error()
				}
				return encode(tool)
			},
		},
	}
}

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
		props[f.Name] = map[string]any{"type": f.Type, "description": f.Description}
		if f.Required {
			req = append(req, f.Name)
		}
	}
	result := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(req) > 0 {
		result["required"] = req
	}
	return result
}

func reqField(name, typ, desc string) schemaField {
	return schemaField{Name: name, Type: typ, Description: desc, Required: true}
}

func optField(name, typ, desc string) schemaField {
	return schemaField{Name: name, Type: typ, Description: desc}
}

func encode(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return `{"error":"failed to encode result"}`
	}
	return string(data)
}
