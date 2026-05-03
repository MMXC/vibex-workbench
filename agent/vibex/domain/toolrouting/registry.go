package toolrouting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Registry struct {
	WorkspaceDir string
	custom       []CustomToolConfig
}

func NewRegistry(workspaceDir string) *Registry {
	r := &Registry{WorkspaceDir: workspaceDir}
	r.custom = r.loadCustomTools()
	return r
}

func (r *Registry) ListTools() []ToolDescriptor {
	tools := builtinTools()
	for _, c := range r.custom {
		tools = append(tools, ToolDescriptor{
			Name:        c.Name,
			Kind:        c.Kind,
			Description: c.Description,
			Source:      "custom",
			Schema:      c.Schema,
			Permissions: c.Permissions,
			AIFillArea:  c.AIFillArea,
		})
	}
	return tools
}

func (r *Registry) CreatePlanGraph(goal, specPath, mode string, slotID ...string) PlanGraph {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		goal = "未命名目标"
	}
	if mode == "" {
		mode = "plan-first"
	}
	slot := ""
	if len(slotID) > 0 {
		slot = strings.TrimSpace(slotID[0])
	}
	if mode == "spec-slot-routing" || slot != "" {
		if mode == "" {
			mode = "spec-slot-routing"
		}
		return r.createSlotPlanGraph(goal, specPath, mode, slot)
	}
	nodes := []PlanNode{
		{
			ID: "inspect_spec", Kind: "spec.inspect", Title: "读取并理解 Spec",
			Description: "读取当前 spec、canonical 槽位和邻接关系，形成执行上下文。",
			Tool:        "read_spec_context", Inputs: map[string]any{"spec_path": specPath}, Outputs: []string{"spec_context"},
		},
		{
			ID: "design_code_graph", Kind: "graph.design", Title: "生成代码关系图",
			Description: "把需求拆成文件、依赖、测试和验证节点，不直接写代码。",
			Tool:        "design_code_graph", Inputs: map[string]any{"goal": goal}, Outputs: []string{"code_graph"},
		},
		{
			ID: "route_tools", Kind: "tool.route", Title: "路由到工具",
			Description: "根据节点 kind 和权限声明选择内置或自定义工具。",
			Tool:        "tool_route_preview", Outputs: []string{"route_preview"},
		},
		{
			ID: "user_confirm", Kind: "gate.confirm", Title: "用户确认执行图",
			Description: "展示计划图和工具路由，等待用户确认后才进入写盘或命令执行。",
			Tool:        "manual_confirmation", Outputs: []string{"confirmed_plan"}, RequiresConfirmation: true,
		},
		{
			ID: "validate", Kind: "workspace.validate", Title: "验证结果",
			Description: "运行 spec/code 相关校验，返回可见问题而不是静默继续。",
			Tool:        "make_validate", Outputs: []string{"validation_report"},
		},
	}
	return PlanGraph{
		Version:  "plan-graph/v1",
		Goal:     goal,
		SpecPath: specPath,
		SlotID:   slot,
		Mode:     mode,
		Nodes:    nodes,
		Edges: []PlanEdge{
			{From: "inspect_spec", To: "design_code_graph", Reason: "spec context constrains graph"},
			{From: "design_code_graph", To: "route_tools", Reason: "graph nodes need tool decisions"},
			{From: "route_tools", To: "user_confirm", Reason: "tool route must be reviewable"},
			{From: "user_confirm", To: "validate", Reason: "confirmed execution must be verified"},
		},
	}
}

func (r *Registry) createSlotPlanGraph(goal, specPath, mode, slotID string) PlanGraph {
	if mode == "" {
		mode = "spec-slot-routing"
	}
	slotKind, slotTitle, slotTool := slotRoute(slotID)
	nodes := []PlanNode{
		{
			ID: "inspect_spec", Kind: "spec.inspect", Title: "读取 Spec 槽位上下文",
			Description: "读取当前 spec 全文、canonical 槽位状态和邻接关系。",
			Tool:        "read_spec_context", Inputs: map[string]any{"spec_path": specPath, "slot_id": slotID}, Outputs: []string{"spec_context"},
		},
		{
			ID: "analyze_slot", Kind: slotKind, Title: slotTitle,
			Description: "基于当前槽位生成可审查的工具能力、澄清问题和路由依据，不直接写业务代码。",
			Tool:        slotTool, Inputs: map[string]any{"goal": goal, "spec_path": specPath, "slot_id": slotID}, Outputs: []string{"slot_analysis"},
		},
		{
			ID: "debug_route", Kind: "tool.route.debug", Title: "调试工具路由",
			Description: "解释为什么命中或没有命中工具，并给出 schema、prompt、examples 调整建议。",
			Tool:        "tool_route_debugger", Inputs: map[string]any{"slot_id": slotID}, Outputs: []string{"route_debug_report"},
		},
		{
			ID: "draft_missing_tool", Kind: "tool.draft.design", Title: "设计缺失工具草案",
			Description: "当现有工具不足时，生成 metadata-only 工具草案，包含 AI 个性化填充区。",
			Tool:        "tool_draft_designer", Inputs: map[string]any{"slot_id": slotID}, Outputs: []string{"tool_draft"},
		},
		{
			ID: "render_fireworks_graph", Kind: "graph.fireworks.render", Title: "渲染 Fireworks 技术图",
			Description: "将 plan graph 和 route preview 转为抽屉右侧可视化图谱数据。",
			Tool:        "fireworks_tech_graph_renderer", Outputs: []string{"fireworks_graph"},
		},
		{
			ID: "user_confirm", Kind: "gate.confirm", Title: "用户确认工具路由",
			Description: "用户确认工具路由、工具草案或后续实现计划后，才允许进入写盘或执行。",
			Tool:        "manual_confirmation", Outputs: []string{"confirmed_route"}, RequiresConfirmation: true,
		},
	}
	return PlanGraph{
		Version:  "plan-graph/v1",
		Goal:     goal,
		SpecPath: specPath,
		SlotID:   slotID,
		Mode:     mode,
		Nodes:    nodes,
		Edges: []PlanEdge{
			{From: "inspect_spec", To: "analyze_slot", Reason: "slot routing needs spec context"},
			{From: "analyze_slot", To: "debug_route", Reason: "route decisions must be explainable"},
			{From: "debug_route", To: "draft_missing_tool", Reason: "missing route may require tool draft"},
			{From: "debug_route", To: "render_fireworks_graph", Reason: "debug output feeds visualization"},
			{From: "draft_missing_tool", To: "render_fireworks_graph", Reason: "tool draft must be visible"},
			{From: "render_fireworks_graph", To: "user_confirm", Reason: "visual route must be confirmed"},
		},
	}
}

func slotRoute(slotID string) (kind, title, tool string) {
	switch strings.ToLower(slotID) {
	case "structure", "constraints":
		return "spec.structure.analyze", "分析结构与约束槽位", "spec_structure_analyzer"
	case "input", "output", "io":
		return "spec.io.validate", "校验输入输出契约", "spec_io_contract_validator"
	case "prototype":
		return "spec.prototype.plan", "规划原型验证", "spec_prototype_planner"
	case "implementation", "implement":
		return "implementation.route.plan", "规划实现工具路由", "implementation_route_planner"
	default:
		return "spec.structure.analyze", "分析 Spec 槽位", "spec_structure_analyzer"
	}
}

func (r *Registry) RouteGraph(graph PlanGraph) RoutePreview {
	tools := r.ListTools()
	decisions := make([]RouteDecision, 0, len(graph.Nodes))
	var warnings []string
	for _, node := range graph.Nodes {
		tool, ok := selectTool(node, tools)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("node %s (%s) has no route", node.ID, node.Kind))
			decisions = append(decisions, RouteDecision{
				NodeID: node.ID, NodeKind: node.Kind, Tool: "", Reason: "no matching tool",
				CanExecute: false, NeedsConfirm: true,
			})
			continue
		}
		decisions = append(decisions, RouteDecision{
			NodeID: node.ID, NodeKind: node.Kind, Tool: tool.Name, ToolSource: tool.Source,
			Reason:       fmt.Sprintf("matched node kind %q to tool kind %q", node.Kind, tool.Kind),
			CanExecute:   tool.Source == "builtin",
			NeedsConfirm: node.RequiresConfirmation || hasPermission(tool, "write") || hasPermission(tool, "execute"),
		})
	}
	return RoutePreview{GraphVersion: graph.Version, Decisions: decisions, Warnings: warnings}
}

func (r *Registry) RegisterCustomTool(cfg CustomToolConfig) (ToolDescriptor, error) {
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.Kind = strings.TrimSpace(cfg.Kind)
	if cfg.Name == "" || cfg.Kind == "" {
		return ToolDescriptor{}, fmt.Errorf("name and kind are required")
	}
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{1,63}$`).MatchString(cfg.Name) {
		return ToolDescriptor{}, fmt.Errorf("invalid tool name: %s", cfg.Name)
	}
	if cfg.ExecuteMode == "" {
		cfg.ExecuteMode = "metadata_only"
	}
	if cfg.ExecuteMode != "metadata_only" {
		return ToolDescriptor{}, fmt.Errorf("MVP only supports execute_mode=metadata_only")
	}
	dir := filepath.Join(r.WorkspaceDir, ".vibex", "tools")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolDescriptor{}, err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return ToolDescriptor{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, cfg.Name+".json"), data, 0644); err != nil {
		return ToolDescriptor{}, err
	}
	r.custom = r.loadCustomTools()
	return ToolDescriptor{Name: cfg.Name, Kind: cfg.Kind, Description: cfg.Description, Source: "custom", Schema: cfg.Schema, Permissions: cfg.Permissions, AIFillArea: cfg.AIFillArea}, nil
}

func builtinTools() []ToolDescriptor {
	return []ToolDescriptor{
		{Name: "read_spec_context", Kind: "spec.inspect", Description: "Read spec context and canonical slots.", Source: "builtin", Permissions: []string{"read"}},
		{Name: "design_code_graph", Kind: "graph.design", Description: "Create a code/dependency/test graph without writing files.", Source: "builtin", Permissions: []string{"plan"}},
		{Name: "tool_route_preview", Kind: "tool.route", Description: "Preview routing decisions for a plan graph.", Source: "builtin", Permissions: []string{"plan"}},
		{Name: "manual_confirmation", Kind: "gate.confirm", Description: "Require human confirmation before execution.", Source: "builtin", Permissions: []string{"confirm"}},
		{Name: "make_validate", Kind: "workspace.validate", Description: "Run validation after confirmed execution.", Source: "builtin", Permissions: []string{"execute"}},
		tool("spec_structure_analyzer", "spec.structure.analyze", "Analyze spec parent/children, dependencies, impacted files, constraints, and layer legality."),
		tool("spec_io_contract_validator", "spec.io.validate", "Validate spec input/output contracts for completeness, consistency, and testability."),
		tool("spec_prototype_planner", "spec.prototype.plan", "Plan prototype purpose, validation targets, and prototype file requirements."),
		tool("implementation_route_planner", "implementation.route.plan", "Create implementation routing plan without directly writing business code."),
		tool("tool_draft_designer", "tool.draft.design", "Design metadata-only tool drafts with schema and AI personalization fill area."),
		tool("tool_route_debugger", "tool.route.debug", "Debug route hits/misses and suggest schema, prompt, and example improvements."),
		tool("fireworks_tech_graph_renderer", "graph.fireworks.render", "Render plan graph and route preview into dynamic fireworks technical graph data."),
	}
}

func tool(name, kind, description string) ToolDescriptor {
	return ToolDescriptor{
		Name:        name,
		Kind:        kind,
		Description: description,
		Source:      "builtin",
		Permissions: []string{"plan"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"spec_path": map[string]any{"type": "string"},
				"slot_id":   map[string]any{"type": "string"},
				"context":   map[string]any{"type": "object"},
			},
		},
		AIFillArea: AIFillArea{
			PromptTemplate:       "",
			DomainRules:          []string{},
			Examples:             []any{},
			PersonalizationNotes: "Reserved for project-specific prompts, examples, and user preferences.",
		},
	}
}

func selectTool(node PlanNode, tools []ToolDescriptor) (ToolDescriptor, bool) {
	if node.Tool != "" {
		for _, tool := range tools {
			if tool.Name == node.Tool {
				return tool, true
			}
		}
	}
	for _, tool := range tools {
		if tool.Kind == node.Kind {
			return tool, true
		}
	}
	return ToolDescriptor{}, false
}

func hasPermission(tool ToolDescriptor, permission string) bool {
	for _, p := range tool.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (r *Registry) loadCustomTools() []CustomToolConfig {
	dir := filepath.Join(r.WorkspaceDir, ".vibex", "tools")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var tools []CustomToolConfig
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var cfg CustomToolConfig
		if json.Unmarshal(data, &cfg) == nil && cfg.Name != "" {
			tools = append(tools, cfg)
		}
	}
	return tools
}
