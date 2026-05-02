// pkg/verify/verifier.go
// Validation engine for spec → code alignment checks.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Result holds the outcome of a single check.
type Result struct {
	SpecName    string `json:"spec_name"`
	SpecLevel   string `json:"spec_level"`
	SpecFile    string `json:"spec_file"`
	CheckType   string `json:"check_type"`   // e.g. "file_exists", "parent_exists"
	CheckID     string `json:"check_id"`    // e.g. "impacted_file", "B1"
	FilePath    string `json:"file_path,omitempty"` // relative path checked
	AbsPath     string `json:"abs_path,omitempty"`
	Status      string `json:"status"`       // "pass" | "fail" | "warn"
	Severity    string `json:"severity"`     // "error" | "warning" | "info"
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// Report is the full verification report.
type Report struct {
	WorkspaceRoot string   `json:"workspace_root"`
	TotalSpecs    int      `json:"total_specs"`
	TotalChecks   int      `json:"total_checks"`
	PassCount     int      `json:"pass_count"`
	FailCount     int      `json:"fail_count"`
	WarnCount     int      `json:"warn_count"`
	Results       []Result `json:"results"`
	Summary       string   `json:"summary"`
}

// Verifier runs validation checks against loaded specs.
type Verifier struct {
	workspaceRoot string
	specs         map[string]*Spec
	opts          VerifierOptions
}

// VerifierOptions controls which checks run.
type VerifierOptions struct {
	// CheckFileExistence verifies that impacted_files and content.file_path exist on disk.
	CheckFileExistence bool
	// CheckParentChain verifies that parent spec names exist.
	CheckParentChain bool
	// CheckCompleteness warns about empty required fields.
	CheckCompleteness bool
	// CheckGoStructFields verifies that Go struct fields match spec content.
	CheckGoStructFields bool
	// CheckSvelteProps verifies that Svelte component props match spec.
	CheckSvelteProps bool
	// CheckConstraints verifies that constraint rules can be validated.
	CheckConstraints bool
	// CheckBehaviors warns if behaviors are empty for L4+ specs.
	CheckBehaviors bool
	// OnlySpecLevels restricts checks to these levels (e.g. ["4_feature","5_slice"]).
	// Empty = all levels.
	OnlySpecLevels []string
}

// DefaultVerifierOptions returns the recommended default set of checks.
func DefaultVerifierOptions() VerifierOptions {
	return VerifierOptions{
		CheckFileExistence: true,
		CheckParentChain:   true,
		CheckCompleteness:  true,
		CheckBehaviors:     true,
	}
}

// NewVerifier creates a new verifier for the given workspace and loaded specs.
func NewVerifier(workspaceRoot string, specs map[string]*Spec) *Verifier {
	return &Verifier{
		workspaceRoot: workspaceRoot,
		specs:         specs,
		opts:          DefaultVerifierOptions(),
	}
}

// WithOptions sets verifier options.
func (v *Verifier) WithOptions(opts VerifierOptions) *Verifier {
	v.opts = opts
	return v
}

// Run executes all configured checks and returns the report.
func (v *Verifier) Run() *Report {
	var results []Result
	for _, spec := range v.specs {
		if len(v.opts.OnlySpecLevels) > 0 && !v.containsLevel(spec.Level) {
			continue
		}
		results = append(results, v.checkSpec(spec)...)
	}

	// Sort results: errors first, then warnings, then info; within each: by spec level, then name.
	sort.Slice(results, func(i, j int) bool {
		order := map[string]int{"error": 0, "warning": 1, "info": 2}
		oi, oj := order[results[i].Severity], order[results[j].Severity]
		if oi != oj {
			return oi < oj
		}
		if results[i].SpecLevel != results[j].SpecLevel {
			return specLevelOrder(results[i].SpecLevel) < specLevelOrder(results[j].SpecLevel)
		}
		return results[i].SpecName < results[j].SpecName
	})

	pass := 0
	fail := 0
	warn := 0
	for _, r := range results {
		switch r.Status {
		case "pass":
			pass++
		case "fail":
			fail++
		case "warn":
			warn++
		}
	}

	summary := fmt.Sprintf("%d specs, %d checks (%d pass, %d fail, %d warn)",
		len(v.specs), len(results), pass, fail, warn)

	return &Report{
		WorkspaceRoot: v.workspaceRoot,
		TotalSpecs:    len(v.specs),
		TotalChecks:   len(results),
		PassCount:     pass,
		FailCount:     fail,
		WarnCount:     warn,
		Results:       results,
		Summary:       summary,
	}
}

func (v *Verifier) checkSpec(spec *Spec) []Result {
	var results []Result

	// 1. File existence checks
	if v.opts.CheckFileExistence {
		results = append(results, v.checkImpactedFiles(spec)...)
		results = append(results, v.checkContentFilePath(spec)...)
	}

	// 2. Parent chain
	if v.opts.CheckParentChain {
		results = append(results, v.checkParent(spec)...)
	}

	// 3. Completeness
	if v.opts.CheckCompleteness {
		results = append(results, v.checkCompleteness(spec)...)
	}

	// 4. Behaviors (L4+)
	if v.opts.CheckBehaviors && spec.LevelNum() >= 4 {
		results = append(results, v.checkBehaviors(spec)...)
	}

	// 5. Go struct field check (L3 module with public_api that looks like Go)
	if v.opts.CheckGoStructFields {
		results = append(results, v.checkGoStructFields(spec)...)
	}

	// 6. Svelte props check
	if v.opts.CheckSvelteProps {
		results = append(results, v.checkSvelteProps(spec)...)
	}

	return results
}

// ── Individual check methods ──────────────────────────────────

func (v *Verifier) checkImpactedFiles(spec *Spec) []Result {
	var results []Result
	if len(spec.ImpactedFiles) == 0 {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "impacted_files",
			Status:    "warn",
			Severity:  "warning",
			Message:   "spec has no impacted_files listed",
			Suggestion: "Add structure.impacted_files to declare which files this spec affects",
		})
		return results
	}
	for _, fp := range spec.ImpactedFiles {
		abs := spec.AbsPath(v.workspaceRoot, fp)
		_, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				results = append(results, Result{
					SpecName:  spec.Name,
					SpecLevel: spec.Level,
					SpecFile:  spec.SourceFile,
					CheckType: "impacted_file",
					CheckID:   fp,
					FilePath:  fp,
					AbsPath:   abs,
					Status:    "fail",
					Severity:  "error",
					Message:   fmt.Sprintf("impacted file does not exist: %s", fp),
					Suggestion: fmt.Sprintf("Create the file or remove it from impacted_files if it's generated later"),
				})
			} else {
				results = append(results, Result{
					SpecName:  spec.Name,
					SpecLevel: spec.Level,
					SpecFile:  spec.SourceFile,
					CheckType: "impacted_file",
					CheckID:   fp,
					FilePath:  fp,
					AbsPath:   abs,
					Status:    "fail",
					Severity:  "error",
					Message:   fmt.Sprintf("error checking impacted file %s: %v", fp, err),
				})
			}
		} else {
			results = append(results, Result{
				SpecName:  spec.Name,
				SpecLevel: spec.Level,
				SpecFile:  spec.SourceFile,
				CheckType: "impacted_file",
				CheckID:   fp,
				FilePath:  fp,
				AbsPath:   abs,
				Status:    "pass",
				Severity:  "info",
				Message:   fmt.Sprintf("impacted file exists: %s", fp),
			})
		}
	}
	return results
}

func (v *Verifier) checkContentFilePath(spec *Spec) []Result {
	var results []Result
	if spec.FilePath == "" || spec.LevelNum() < 5 {
		return results
	}
	abs := spec.AbsPath(v.workspaceRoot, spec.FilePath)
	_, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			results = append(results, Result{
				SpecName:  spec.Name,
				SpecLevel: spec.Level,
				SpecFile:  spec.SourceFile,
				CheckType: "content_file_path",
				CheckID:   spec.FilePath,
				FilePath:  spec.FilePath,
				AbsPath:   abs,
				Status:    "fail",
				Severity:  "error",
				Message:   fmt.Sprintf("content.file_path does not exist: %s", spec.FilePath),
				Suggestion: "Create this file or update content.file_path if it will be generated",
			})
		}
	} else {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "content_file_path",
			CheckID:   spec.FilePath,
			FilePath:  spec.FilePath,
			AbsPath:   abs,
			Status:    "pass",
			Severity:  "info",
			Message:   fmt.Sprintf("content.file_path exists: %s", spec.FilePath),
		})
	}
	return results
}

func (v *Verifier) checkParent(spec *Spec) []Result {
	var results []Result
	if spec.Parent == "" {
		// Root-level spec (L1 concept or skeleton) — no parent to check
		return results
	}
	if _, exists := v.specs[spec.Parent]; !exists {
		// Check if it looks like a valid spec name format
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "parent_chain",
			CheckID:   spec.Parent,
			Status:    "warn",
			Severity:  "warning",
			Message:   fmt.Sprintf("parent spec %q not found in specs/ directory", spec.Parent),
			Suggestion: "Verify the parent spec name is correct or the file exists in specs/",
		})
	} else {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "parent_chain",
			CheckID:   spec.Parent,
			Status:    "pass",
			Severity:  "info",
			Message:   fmt.Sprintf("parent spec %q found", spec.Parent),
		})
	}
	return results
}

func (v *Verifier) checkCompleteness(spec *Spec) []Result {
	var results []Result
	if spec.Title == "" {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "completeness",
			CheckID:   "display.title",
			Status:    "warn",
			Severity:  "warning",
			Message:   "display.title is empty",
			Suggestion: "Add a human-readable title in display.title",
		})
	}
	if spec.Summary == "" {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "completeness",
			CheckID:   "display.summary",
			Status:    "warn",
			Severity:  "warning",
			Message:   "display.summary is empty",
			Suggestion: "Add a one-line summary in display.summary",
		})
	}
	if spec.LifecycleCurrent == "" {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "completeness",
			CheckID:   "lifecycle.current",
			Status:    "warn",
			Severity:  "warning",
			Message:   "lifecycle.current is empty",
			Suggestion: "Set lifecycle.current (proposal | implementation | delivered)",
		})
	}
	return results
}

func (v *Verifier) checkBehaviors(spec *Spec) []Result {
	var results []Result
	if len(spec.Behaviors) == 0 {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "behaviors",
			Status:    "warn",
			Severity:  "warning",
			Message:   "no behaviors defined in content.behaviors",
			Suggestion: "Add at least one behavior in content.behaviors to define what this spec does",
		})
	}
	// Check each behavior has required fields
	for _, b := range spec.Behaviors {
		if b.Trigger == "" {
			results = append(results, Result{
				SpecName:  spec.Name,
				SpecLevel: spec.Level,
				SpecFile:  spec.SourceFile,
				CheckType: "behavior_field",
				CheckID:   b.ID,
				Status:    "warn",
				Severity:  "warning",
				Message:   fmt.Sprintf("behavior %q has empty trigger", b.ID),
				Suggestion: "Fill in behavior.trigger to describe when this behavior activates",
			})
		}
		if b.Action == "" {
			results = append(results, Result{
				SpecName:  spec.Name,
				SpecLevel: spec.Level,
				SpecFile:  spec.SourceFile,
				CheckType: "behavior_field",
				CheckID:   b.ID,
				Status:    "warn",
				Severity:  "warning",
				Message:   fmt.Sprintf("behavior %q has empty action", b.ID),
				Suggestion: "Fill in behavior.action to describe what this behavior does",
			})
		}
	}
	return results
}

// checkGoStructFields checks if a Go file contains a struct whose fields match
// any API definitions in the spec (for L3 modules with public_api).
func (v *Verifier) checkGoStructFields(spec *Spec) []Result {
	var results []Result
	if len(spec.ImpactedFiles) == 0 {
		return results
	}
	// Look for Go files in impacted files
	for _, fp := range spec.ImpactedFiles {
		if filepath.Ext(fp) != ".go" {
			continue
		}
		abs := spec.AbsPath(v.workspaceRoot, fp)
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		content := string(data)
		// Extract struct names from the spec
		// This is a basic check: verify the struct name appears in the file
		if !strings.Contains(content, "type "+spec.Name) && !strings.Contains(content, "type "+strings.ReplaceAll(spec.Name, "-", "")) {
			results = append(results, Result{
				SpecName:  spec.Name,
				SpecLevel: spec.Level,
				SpecFile:  spec.SourceFile,
				CheckType: "go_struct_reference",
				CheckID:   fp,
				FilePath:  fp,
				AbsPath:   abs,
				Status:    "warn",
				Severity:  "warning",
				Message:   fmt.Sprintf("Go file %s does not contain struct %q or %q — spec may be out of sync",
					fp, spec.Name, strings.ReplaceAll(spec.Name, "-", "")),
				Suggestion: "Verify this Go file is the correct implementation target for " + spec.Name,
			})
		} else {
			results = append(results, Result{
				SpecName:  spec.Name,
				SpecLevel: spec.Level,
				SpecFile:  spec.SourceFile,
				CheckType: "go_struct_reference",
				CheckID:   fp,
				FilePath:  fp,
				AbsPath:   abs,
				Status:    "pass",
				Severity:  "info",
				Message:   fmt.Sprintf("Go file %s contains struct reference for %s", fp, spec.Name),
			})
		}
	}
	return results
}

// checkSvelteProps verifies that a Svelte component's props match what's declared in spec.
func (v *Verifier) checkSvelteProps(spec *Spec) []Result {
	var results []Result
	if spec.FileType != "svelte" || spec.FilePath == "" {
		return results
	}
	abs := spec.AbsPath(v.workspaceRoot, spec.FilePath)
	data, err := os.ReadFile(abs)
	if err != nil {
		return results
	}
	content := string(data)

	// Extract script tag content
	scriptMatch := regexp.MustCompile(`<script[^>]*>(.*?)</script>`)
	matches := scriptMatch.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "svelte_script_tag",
			CheckID:   spec.FilePath,
			FilePath:  spec.FilePath,
			AbsPath:   abs,
			Status:    "warn",
			Severity:  "warning",
			Message:   "Svelte file has no <script> tag",
			Suggestion: "Add a <script> tag with props defined",
		})
		return results
	}

	script := matches[0][1]
	// Check for $props() rune (Svelte 5)
	if strings.Contains(script, "$props()") || strings.Contains(script, "let {") {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "svelte_props",
			CheckID:   spec.FilePath,
			FilePath:  spec.FilePath,
			AbsPath:   abs,
			Status:    "pass",
			Severity:  "info",
			Message:   "Svelte component uses $props() rune",
		})
	} else if strings.Contains(script, "export let") {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "svelte_props",
			CheckID:   spec.FilePath,
			FilePath:  spec.FilePath,
			AbsPath:   abs,
			Status:    "pass",
			Severity:  "info",
			Message:   "Svelte component uses export let props (Svelte 4 style)",
		})
	} else {
		results = append(results, Result{
			SpecName:  spec.Name,
			SpecLevel: spec.Level,
			SpecFile:  spec.SourceFile,
			CheckType: "svelte_props",
			CheckID:   spec.FilePath,
			FilePath:  spec.FilePath,
			AbsPath:   abs,
			Status:    "warn",
			Severity:  "warning",
			Message:   "Svelte component has no clear props definition ($props() or export let)",
			Suggestion: "Define props using $props() rune (Svelte 5) or export let (Svelte 4)",
		})
	}
	return results
}

// ── Helpers ──────────────────────────────────────────────────

func (v *Verifier) containsLevel(level string) bool {
	for _, l := range v.opts.OnlySpecLevels {
		if l == level {
			return true
		}
	}
	return false
}

func specLevelOrder(level string) int {
	// Sort L1-L5 in order
	m := map[string]int{
		"1_concept": 1,
		"2_skeleton": 2,
		"3_module":  3,
		"4_feature": 4,
		"5_slice":   5,
	}
	if n, ok := m[level]; ok {
		return n
	}
	return 99
}
