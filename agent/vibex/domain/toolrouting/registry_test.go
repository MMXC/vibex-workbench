package toolrouting

import "testing"

func TestBuiltinToolsIncludeSlotRoutingTools(t *testing.T) {
	reg := NewRegistry(t.TempDir())
	tools := reg.ListTools()
	required := map[string]bool{
		"spec_structure_analyzer":       false,
		"spec_io_contract_validator":    false,
		"spec_prototype_planner":        false,
		"implementation_route_planner":  false,
		"tool_draft_designer":           false,
		"tool_route_debugger":           false,
		"fireworks_tech_graph_renderer": false,
	}
	for _, tool := range tools {
		if _, ok := required[tool.Name]; ok {
			required[tool.Name] = true
			if tool.AIFillArea.PersonalizationNotes == "" {
				t.Fatalf("tool %s missing ai_fill_area personalization notes", tool.Name)
			}
		}
	}
	for name, found := range required {
		if !found {
			t.Fatalf("missing builtin tool %s", name)
		}
	}
}

func TestCreateSlotPlanGraphRoutesBySlot(t *testing.T) {
	reg := NewRegistry(t.TempDir())
	cases := []struct {
		slot     string
		wantKind string
		wantTool string
	}{
		{slot: "structure", wantKind: "spec.structure.analyze", wantTool: "spec_structure_analyzer"},
		{slot: "input", wantKind: "spec.io.validate", wantTool: "spec_io_contract_validator"},
		{slot: "output", wantKind: "spec.io.validate", wantTool: "spec_io_contract_validator"},
		{slot: "prototype", wantKind: "spec.prototype.plan", wantTool: "spec_prototype_planner"},
		{slot: "implementation", wantKind: "implementation.route.plan", wantTool: "implementation_route_planner"},
	}
	for _, tc := range cases {
		graph := reg.CreatePlanGraph("test", "specs/L4-feature/foo.yaml", "spec-slot-routing", tc.slot)
		if graph.SlotID != tc.slot {
			t.Fatalf("slot %s: graph slot id = %q", tc.slot, graph.SlotID)
		}
		var found bool
		for _, node := range graph.Nodes {
			if node.ID == "analyze_slot" {
				found = true
				if node.Kind != tc.wantKind || node.Tool != tc.wantTool {
					t.Fatalf("slot %s: node kind/tool = %s/%s, want %s/%s", tc.slot, node.Kind, node.Tool, tc.wantKind, tc.wantTool)
				}
			}
		}
		if !found {
			t.Fatalf("slot %s: missing analyze_slot node", tc.slot)
		}
	}
}

func TestRouteGraphHitsSlotTools(t *testing.T) {
	reg := NewRegistry(t.TempDir())
	graph := reg.CreatePlanGraph("implement safely", "specs/L5-slice/foo.yaml", "spec-slot-routing", "implementation")
	route := reg.RouteGraph(graph)
	found := map[string]bool{}
	for _, decision := range route.Decisions {
		found[decision.Tool] = true
	}
	for _, tool := range []string{"implementation_route_planner", "tool_route_debugger", "tool_draft_designer", "fireworks_tech_graph_renderer"} {
		if !found[tool] {
			t.Fatalf("route preview missing %s; decisions=%+v", tool, route.Decisions)
		}
	}
	if len(route.Warnings) != 0 {
		t.Fatalf("unexpected route warnings: %+v", route.Warnings)
	}
}
