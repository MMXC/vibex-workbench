package toolrouting

// PlanGraph is the first-class artifact between spec intent and execution.
// It is intentionally declarative: nodes describe work, the router decides tools.
type PlanGraph struct {
	Version  string     `json:"version"`
	Goal     string     `json:"goal"`
	SpecPath string     `json:"spec_path,omitempty"`
	SlotID   string     `json:"slot_id,omitempty"`
	Mode     string     `json:"mode"`
	Nodes    []PlanNode `json:"nodes"`
	Edges    []PlanEdge `json:"edges"`
}

type PlanNode struct {
	ID                   string         `json:"id"`
	Kind                 string         `json:"kind"`
	Title                string         `json:"title"`
	Description          string         `json:"description"`
	Tool                 string         `json:"tool,omitempty"`
	Inputs               map[string]any `json:"inputs,omitempty"`
	Outputs              []string       `json:"outputs,omitempty"`
	RequiresConfirmation bool           `json:"requires_confirmation"`
}

type PlanEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason,omitempty"`
}

type ToolDescriptor struct {
	Name        string         `json:"name"`
	Kind        string         `json:"kind"`
	Description string         `json:"description"`
	Source      string         `json:"source"`
	Schema      map[string]any `json:"schema,omitempty"`
	Permissions []string       `json:"permissions,omitempty"`
	AIFillArea  AIFillArea     `json:"ai_fill_area,omitempty"`
}

type CustomToolConfig struct {
	Name        string         `json:"name"`
	Kind        string         `json:"kind"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema,omitempty"`
	Permissions []string       `json:"permissions,omitempty"`
	AIFillArea  AIFillArea     `json:"ai_fill_area,omitempty"`
	ExecuteMode string         `json:"execute_mode"`
	Command     string         `json:"command,omitempty"`
}

type AIFillArea struct {
	PromptTemplate       string   `json:"prompt_template,omitempty"`
	DomainRules          []string `json:"domain_rules,omitempty"`
	Examples             []any    `json:"examples,omitempty"`
	PersonalizationNotes string   `json:"personalization_notes,omitempty"`
}

type RouteDecision struct {
	NodeID       string `json:"node_id"`
	NodeKind     string `json:"node_kind"`
	Tool         string `json:"tool"`
	ToolSource   string `json:"tool_source"`
	Reason       string `json:"reason"`
	CanExecute   bool   `json:"can_execute"`
	NeedsConfirm bool   `json:"needs_confirm"`
}

type RoutePreview struct {
	GraphVersion string          `json:"graph_version"`
	Decisions    []RouteDecision `json:"decisions"`
	Warnings     []string        `json:"warnings,omitempty"`
}
