package tdd

import (
	rt "vibex/agent/agents/runtime/tools"
)

// ToolSpecs returns the 3 TDD tool specs for registration.
func ToolSpecs() []rt.Spec {
	return []rt.Spec{
		{
			Name:        "tdd_design",
			Description: "Parse a spec YAML's io_contract (input/output/boundary/behavior) and generate test cases for the specified language. Creates RED phase tests that will initially fail.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"spec_path": map[string]any{
						"type":        "string",
						"description": "Path to the spec YAML file to generate tests from",
					},
					"test_language": map[string]any{
						"type":        "string",
						"description": "Language for tests: go | python | typescript | javascript (default: python)",
					},
					"framework": map[string]any{
						"type":        "string",
						"description": "Testing framework: testing | pytest | jest | vitest (auto-detected from language if omitted)",
					},
					"mock_spec_paths": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of additional spec paths for mock/stub generation",
					},
				},
				"required":             []any{"spec_path"},
				"additionalProperties": false,
			},
			Handler: TddDesignHandler,
		},
		{
			Name:        "tdd_run",
			Description: "Execute the generated test suite. Returns RED (tests fail) or GREEN (tests pass). Updates Canvas with the TDD cycle node.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"test_path": map[string]any{
						"type":        "string",
						"description": "Path to the test file or test directory to run",
					},
					"spec_path": map[string]any{
						"type":        "string",
						"description": "Path to the spec YAML (for canvas emission)",
					},
					"language": map[string]any{
						"type":        "string",
						"description": "Language: go | python | typescript | javascript",
					},
					"verbose": map[string]any{
						"type":        "boolean",
						"description": "Show full test output (default: false, shows summary only)",
					},
				},
				"required":             []any{"test_path", "language"},
				"additionalProperties": false,
			},
			Handler: TddRunHandler,
		},
		{
			Name:        "tdd_iterate",
			Description: "Run tests and extract the next behavior step from the spec. Combines tdd_run + spec behavior extraction for iterative TDD.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"test_path":    map[string]any{"type": "string", "description": "Path to the test file or directory"},
					"spec_path":    map[string]any{"type": "string", "description": "Path to the spec YAML (for behavior step extraction)"},
					"language":     map[string]any{"type": "string", "description": "Language: go | python | typescript | javascript"},
					"current_step": map[string]any{"type": "integer", "description": "Current behavior step number (1-indexed, for extracting next step)"},
				},
				"required":             []any{"test_path", "spec_path", "language"},
				"additionalProperties": false,
			},
			Handler: TddIterateHandler,
		},
	}
}
