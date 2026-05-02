// agent/vibex/domain/spec/governance_view_schema.go — View model schema definitions per spec layer.
// SLICE-spec-layer-view-schema
package spec

// SpecViewModel represents the base view model for any spec.
type SpecViewModel struct {
	Path        string                 `json:"path"`
	Name        string                 `json:"name"`
	Level       string                 `json:"level"`
	Parent      string                 `json:"parent,omitempty"`
	Status      string                 `json:"status"`
	Children    []string               `json:"children,omitempty"`
	Description string                 `json:"description,omitempty"`
	Links       []ViewModelLink        `json:"links,omitempty"`
	Actions     []ViewModelAction      `json:"actions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ViewModelLink represents a clickable link in the spec view.
type ViewModelLink struct {
	Label   string `json:"label"`
	URI     string `json:"uri"`
	Enabled bool   `json:"enabled"`
}

// ViewModelAction represents a button or action in the spec view.
type ViewModelAction struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	URI          string `json:"uri,omitempty"`
	RequiresConfirm bool `json:"requires_confirmation"`
	Dangerous    bool   `json:"dangerous,omitempty"`
}

// L1ViewModel is the view model for L1 goal specs.
type L1ViewModel struct {
	SpecViewModel
	Goals           []string `json:"goals"`
	Stakeholders    []string `json:"stakeholders,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
}

// L2ViewModel is the view model for L2 feature specs.
type L2ViewModel struct {
	SpecViewModel
	Capabilities    []string `json:"capabilities"`
	DependsOn       []string `json:"depends_on,omitempty"`
}

// L5ViewModel is the view model for L5 component specs.
type L5ViewModel struct {
	SpecViewModel
	FilePath      string   `json:"file_path"`
	ImpactedFiles []string `json:"impacted_files,omitempty"`
	Generates     []string `json:"generates,omitempty"`
}

// SchemaForLevel returns the schema description for a given level.
func SchemaForLevel(level string) string {
	switch level {
	case "L1", "1_goal":
		return "L1 View Model: path, name, status, goals, stakeholders, acceptance_criteria, children (L2 paths), links, actions"
	case "L2", "2_feature":
		return "L2 View Model: path, name, status, parent, capabilities, depends_on, children (L3 paths), links, actions"
	case "L3", "3_module":
		return "L3 View Model: path, name, status, parent, modules, children, links, actions"
	case "L4", "4_feature":
		return "L4 View Model: path, name, status, parent, impacted_files, behaviors, links, actions"
	case "L5", "5_slice":
		return "L5 View Model: path, name, status, parent, file_path, impacted_files, generates, io_contract, links, actions"
	default:
		return "Base View Model: path, name, level, parent, status, description, children, links, actions, metadata"
	}
}