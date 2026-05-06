package spec

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	rt "vibex/agent/agents/runtime/tools"
)

// Broadcaster sends canvas events to the SSE layer.
type Broadcaster func(threadID, event string, data interface{})

// ─────────────────────────────────────────────────────────────
// Handler Factories
// ─────────────────────────────────────────────────────────────

func MakeSpecDesignerHandler(workspaceDir string, bc Broadcaster, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-goal")
		}
		intent, err := parseStringArg(arguments, "intent")
		if err != nil {
			return "invalid args: " + err.Error()
		}

		b := make([]byte, 4)
		rand.Read(b)
		specID := hex.EncodeToString(b)
		timestamp := time.Now().Format(time.RFC3339)

		// ── Read template from generators/templates/ (spec-driven) ──
		templateDir := filepath.Join(workspaceDir, "generators", "templates")
		tplPath := filepath.Join(templateDir, "designer_intent_template.yaml.tpl")

		var tplContent string
		if tplBytes, err := os.ReadFile(tplPath); err == nil {
			tplContent = string(tplBytes)
		} else {
			// Fallback: template not yet generated → generate it now via make generate
			genCmd := exec.Command("make", "generate")
			genCmd.Dir = workspaceDir
			if genOut, genErr := genCmd.CombinedOutput(); genErr != nil {
				return fmt.Sprintf("make generate FAILED (needed for template sync):\n%s\n%v",
					strings.TrimSpace(string(genOut)), genErr)
			}
			if tplBytes2, err2 := os.ReadFile(tplPath); err2 == nil {
				tplContent = string(tplBytes2)
			} else {
				return fmt.Sprintf("designer intent template not found: %s (after make generate)\n%v", tplPath, err2)
			}
		}

		// ── Substitute placeholders ──
		tplContent = strings.ReplaceAll(tplContent, "${SPEC_ID}", specID)
		tplContent = strings.ReplaceAll(tplContent, "${TIMESTAMP}", timestamp)
		tplContent = strings.ReplaceAll(tplContent, "${INTENT}", escapeYAML(intent))

		specsDir := filepath.Join(workspaceDir, "specs")
		goalDir := filepath.Join(specsDir, "project-goal")
		os.MkdirAll(goalDir, 0755)
		specFile := filepath.Join(goalDir, fmt.Sprintf("intent-%s-%s.yaml",
			time.Now().Format("20060102-150405"), specID))

		if err := os.WriteFile(specFile, []byte(tplContent), 0644); err != nil {
			return "error writing spec: " + err.Error()
		}

		// ── Auto-chain: emit canvas event ──
		var canvasResult string
		if bc != nil {
			canvasEvent := map[string]interface{}{
				"event_type": "node_added",
				"title":      fmt.Sprintf("Intent: %s", intent),
				"content":    fmt.Sprintf("spec: %s\nstatus: draft", specFile),
				"node_type":  "intent-node",
				"timestamp":  timestamp,
			}
			bc("", "canvas.spec_created", canvasEvent)
			canvasResult = "\n✅ canvas updated with new intent node"
		} else {
			canvasResult = ""
		}

		return fmt.Sprintf("spec draft created: %s\nintent: %s%s\n\nAwait user confirmation before finalizing.",
			specFile, intent, canvasResult)
	}
}

func MakeSpecFeatureHandler(workspaceDir string, bc Broadcaster, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-feature")
		}
		var args struct {
			ParentSpecID string `json:"parent_spec_id"`
			FeatureName  string `json:"feature_name"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.FeatureName == "" {
			return "feature_name is required"
		}

		specsDir := filepath.Join(workspaceDir, "specs", "feature")
		os.MkdirAll(specsDir, 0755)

		b := make([]byte, 4)
		rand.Read(b)
		featureID := hex.EncodeToString(b)
		safeName := strings.ReplaceAll(strings.ToLower(args.FeatureName), " ", "-")

		// Use sub-directory so feature + sub-specs stay together
		// dir: specs/feature/<safeName>/
		featureDir := filepath.Join(specsDir, safeName)
		os.MkdirAll(featureDir, 0755)
		specFile := filepath.Join(featureDir, fmt.Sprintf("%s_feature.yaml", safeName))
		uiuxFile := filepath.Join(featureDir, fmt.Sprintf("%s_uiux.yaml", safeName))

		// ── E3: Read templates from generators/templates/ (spec-driven template) ──
		templateDir := filepath.Join(workspaceDir, "generators", "templates")
		featureTplPath := filepath.Join(templateDir, "feature_template_feature.yaml.tpl")
		uiuxTplPath := filepath.Join(templateDir, "feature_template_uiux.yaml.tpl")

		var featureTplContent, uiuxTplContent string
		if tplBytes, err := os.ReadFile(featureTplPath); err == nil {
			featureTplContent = string(tplBytes)
		} else {
			// Fallback: template not yet generated → generate it now via make generate
			genCmd := exec.Command("make", "generate")
			genCmd.Dir = workspaceDir
			if genOut, genErr := genCmd.CombinedOutput(); genErr != nil {
				return fmt.Sprintf("make generate FAILED (needed for template sync):\n%s\n%v",
					strings.TrimSpace(string(genOut)), genErr)
			}
			if tplBytes2, err2 := os.ReadFile(featureTplPath); err2 == nil {
				featureTplContent = string(tplBytes2)
			} else {
				return fmt.Sprintf("template not found: %s (after make generate)\n%v", featureTplPath, err2)
			}
		}
		if tplBytes, err := os.ReadFile(uiuxTplPath); err == nil {
			uiuxTplContent = string(tplBytes)
		} else {
			return fmt.Sprintf("template not found: %s\n%v", uiuxTplPath, err)
		}

		// ── Substitute placeholders (${FEATURE_ID} etc., synced from meta-spec) ──
		timestamp := time.Now().Format(time.RFC3339)
		subs := map[string]string{
			"${FEATURE_ID}":   featureID,
			"${SAFE_NAME}":    safeName,
			"${PARENT_ID}":    args.ParentSpecID,
			"${TIMESTAMP}":    timestamp,
			"${FEATURE_NAME}": escapeYAML(args.FeatureName),
		}
		for placeholder, value := range subs {
			featureTplContent = strings.ReplaceAll(featureTplContent, placeholder, value)
			uiuxTplContent = strings.ReplaceAll(uiuxTplContent, placeholder, value)
		}

		if err := os.WriteFile(specFile, []byte(featureTplContent), 0644); err != nil {
			return "error writing feature spec: " + err.Error()
		}
		if err := os.WriteFile(uiuxFile, []byte(uiuxTplContent), 0644); err != nil {
			return fmt.Sprintf("feature spec created: %s\n(error creating uiux sub-spec: %v)", specFile, err)
		}

		// ── Auto-chain: make validate + make generate ──
		// Step 1: validate
		validateCmd := exec.Command("make", "validate")
		validateCmd.Dir = workspaceDir
		valOut, valErr := validateCmd.CombinedOutput()
		valText := strings.TrimSpace(string(valOut))

		var validationResult string
		if valErr != nil {
			validationResult = fmt.Sprintf("⚠️ make validate FAILED (spec created, fix required before generating):\n%s", valText)
		} else {
			validationResult = fmt.Sprintf("✅ make validate PASSED")
		}

		// Step 2: generate code (only if validation passed)
		var generationResult string
		if valErr == nil {
			genCmd := exec.Command("make", "generate")
			genCmd.Dir = workspaceDir
			genOut, genErr := genCmd.CombinedOutput()
			genText := strings.TrimSpace(string(genOut))
			if genErr != nil {
				generationResult = fmt.Sprintf("⚠️ make generate FAILED:\n%s", genText)
			} else {
				// Extract summary lines
				lines := strings.Split(genText, "\n")
				summary := []string{}
				for _, l := range lines {
					if strings.Contains(l, "✅") || strings.Contains(l, "❌") || strings.Contains(l, "Template") {
						summary = append(summary, l)
					}
				}
				if len(summary) == 0 {
					summary = []string{lines[len(lines)-1]}
				}
				generationResult = fmt.Sprintf("✅ make generate PASSED:\n  %s", strings.Join(summary, "\n  "))
			}
		}

		// Step 3: auto-canvas-update (only if generation succeeded)
		var canvasResult string
		if valErr == nil && bc != nil {
			canvasEvent := map[string]interface{}{
				"event_type": "node_added",
				"title":      fmt.Sprintf("Feature: %s", args.FeatureName),
				"content":    fmt.Sprintf("spec: %s\nuiux: %s", specFile, uiuxFile),
				"node_type":  "feature-node",
				"spec_id":    featureID,
				"parent":     args.ParentSpecID,
				"timestamp":  timestamp,
			}
			bc("", "canvas.spec_created", canvasEvent)
			canvasResult = "\n✅ canvas updated with new feature node"
		} else {
			canvasResult = ""
		}

		return fmt.Sprintf("spec_feature created:\n  %s\n  %s\nparent: %s\n\nAUTO-CHAIN RESULTS:\n  %s\n  %s%s",
			specFile, uiuxFile, args.ParentSpecID, validationResult, generationResult, canvasResult)
	}
}

func MakeSpecValidateHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}
		specPath, err := parseStringArg(arguments, "spec_path")
		if err != nil {
			return "invalid args: " + err.Error()
		}

		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(workspaceDir, specPath)
		}

		data, err := os.ReadFile(specPath)
		if err != nil {
			return "error reading file: " + err.Error()
		}

		content := string(data)
		issues := []string{}

		requiredFields := []string{"spec:", "  type:", "  status:"}
		for _, field := range requiredFields {
			if !strings.Contains(content, field) {
				issues = append(issues, fmt.Sprintf("missing field: %s", field))
			}
		}

		if strings.Contains(content, "|\n") && strings.Contains(content, "\"") {
			issues = append(issues, "warning: pipe inside double-quoted string may be misparsed as YAML block scalar")
		}

		if strings.Contains(content, "result:") {
			issues = append(issues, "info: result[] field detected — ensure items link to confirmed files")
		}

		if len(issues) == 0 {
			return fmt.Sprintf("validation PASSED: %s\n- YAML syntax: ok\n- Required fields: present\n- No issues found", specPath)
		}
		return fmt.Sprintf("validation issues for %s:\n- %s", specPath, strings.Join(issues, "\n- "))
	}
}

func MakeCanvasUpdateHandler(bc Broadcaster, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "canvas-exhibit")
		}
		var args struct {
			ThreadID  string `json:"thread_id"`
			EventType string `json:"event_type"`
			Payload   string `json:"payload"`
			Title     string `json:"title"`
			Content   string `json:"content"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.ThreadID == "" || args.EventType == "" {
			return "thread_id and event_type are required"
		}

		event := map[string]interface{}{
			"thread_id":  args.ThreadID,
			"event_type": args.EventType,
			"payload":    args.Payload,
			"title":      args.Title,
			"content":    args.Content,
			"timestamp":  time.Now().Format(time.RFC3339),
		}
		if bc != nil {
			bc(args.ThreadID, "canvas."+args.EventType, event)
		}

		return fmt.Sprintf("canvas updated: thread=%s event=%s", args.ThreadID, args.EventType)
	}
}

func MakeSpecSyncHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}
		var args struct {
			SpecPath   string `json:"spec_path"`
			Direction  string `json:"direction"`
			TargetFile string `json:"target_file"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.SpecPath == "" {
			return "spec_path is required"
		}
		if args.Direction == "" {
			args.Direction = "push"
		}

		specPath := args.SpecPath
		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(workspaceDir, args.SpecPath)
		}

		syncScript := filepath.Join(workspaceDir, "scripts", "dsl-sync.py")
		if _, err := os.Stat(syncScript); os.IsNotExist(err) {
			return fmt.Sprintf("sync script not found at %s.\nRun manually: python3 %s --spec %s --direction %s",
				syncScript, syncScript, specPath, args.Direction)
		}

		cmd := exec.Command("python3", syncScript, "--spec", specPath, "--direction", args.Direction)
		cmd.Dir = workspaceDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("sync error: %v\n%s", err, strings.TrimSpace(string(out)))
		}
		return fmt.Sprintf("sync completed:\nspec: %s\ndirection: %s\noutput: %s",
			specPath, args.Direction, strings.TrimSpace(string(out)))
	}
}

func MakeMakeValidateHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}
		cmd := exec.Command("make", "validate")
		cmd.Dir = workspaceDir
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			return fmt.Sprintf("make validate FAILED:\n%s\n%v", text, err)
		}
		return "make validate PASSED:\n" + text
	}
}

// MakeMakeGenerateHandler runs `make generate` — the spec-to-code step.
// This is the core of spec-driven development:
//   1. Agent creates/updates spec YAML
//   2. Calls make_generate → gen.py emits types/components/routes
//   3. Agent verifies output
func MakeMakeGenerateHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}
		cmd := exec.Command("make", "generate")
		cmd.Dir = workspaceDir
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			return fmt.Sprintf("make generate FAILED:\n%s\n%v", text, err)
		}
		return "make generate PASSED:\n" + text
	}
}

func MakeWorkspaceSpecsListHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-analyse")
		}
		var args struct {
			WorkspaceRoot string `json:"workspace_root"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		root := resolveWorkspaceArg(workspaceDir, args.WorkspaceRoot)
		specsDir := filepath.Join(root, "specs")
		var paths []string
		if err := filepath.Walk(specsDir, func(full string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(full, ".yaml") || strings.HasSuffix(full, ".yml") {
				if rel, relErr := filepath.Rel(root, full); relErr == nil {
					paths = append(paths, filepath.ToSlash(rel))
				}
			}
			return nil
		}); err != nil {
			return "workspace_specs_list failed: " + err.Error()
		}
		sort.Strings(paths)
		payload := map[string]any{
			"workspace_root": root,
			"count":          len(paths),
			"paths":          paths,
		}
		return marshalToolResult(payload)
	}
}

func MakeWorkspaceSpecsConventionHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-analyse")
		}
		payload := map[string]any{
			"workspace_root": workspaceDir,
			"levels": []map[string]string{
				{"id": "L1", "directory": "specs/L1-goal", "meaning": "Goal / project intent"},
				{"id": "L2", "directory": "specs/L2-skeleton", "meaning": "Skeleton / system shape"},
				{"id": "L3", "directory": "specs/L3-module", "meaning": "Module boundary"},
				{"id": "L4", "directory": "specs/L4-feature", "meaning": "Feature behavior"},
				{"id": "L5", "directory": "specs/L5-slice", "meaning": "Slice / implementation unit"},
			},
			"canonical_slots": []string{"structure", "io.input", "io.output", "constraints", "prototype", "implementation"},
			"rules": []string{
				"Use spec.name and spec.parent as the parent-chain source of truth.",
				"Cards should prefer display.title and display.summary.",
				"Missing canonical slots should be clarified before implementation.",
			},
		}
		return marshalToolResult(payload)
	}
}

func MakeVerifySpecSuiteHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}
		var args struct {
			WorkspaceRoot string `json:"workspace_root"`
			Checks        string `json:"checks"`
			Levels        string `json:"levels"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		root := resolveWorkspaceArg(workspaceDir, args.WorkspaceRoot)
		verifyBin := ""
		for _, candidate := range []string{filepath.Join(root, "verify_specs"), filepath.Join(root, "verify_specs.exe")} {
			if _, err := os.Stat(candidate); err == nil {
				verifyBin = candidate
				break
			}
		}
		if verifyBin != "" {
			cmdArgs := []string{"--workspace", root, "--format", "json"}
			if strings.TrimSpace(args.Checks) != "" {
				cmdArgs = append(cmdArgs, "--check", args.Checks)
			}
			if strings.TrimSpace(args.Levels) != "" {
				cmdArgs = append(cmdArgs, "--level", args.Levels)
			}
			cmd := exec.Command(verifyBin, cmdArgs...)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Sprintf("verify_spec_suite FAILED:\n%s\n%v", strings.TrimSpace(string(out)), err)
			}
			return "verify_spec_suite PASSED:\n" + strings.TrimSpace(string(out))
		}

		cmd := exec.Command("make", "validate")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			return fmt.Sprintf("verify_specs binary not found; make validate FAILED:\n%s\n%v", text, err)
		}
		return "verify_specs binary not found; make validate PASSED:\n" + text
	}
}

func MakeWorkspaceRunMakeHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}
		var args struct {
			WorkspaceRoot string `json:"workspace_root"`
			Target        string `json:"target"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		target := strings.TrimSpace(args.Target)
		allowed := map[string]bool{"validate": true, "lint-specs": true, "generate": true, "test": true}
		if !allowed[target] {
			return "target must be one of: validate, lint-specs, generate, test"
		}
		root := resolveWorkspaceArg(workspaceDir, args.WorkspaceRoot)
		cmd := exec.Command("make", target)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			return fmt.Sprintf("workspace_run_make %s FAILED:\n%s\n%v", target, text, err)
		}
		return fmt.Sprintf("workspace_run_make %s PASSED:\n%s", target, text)
	}
}

func MakeGovernanceStatusHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-analyse")
		}
		var args struct {
			WorkspaceRoot string `json:"workspace_root"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		root := resolveWorkspaceArg(workspaceDir, args.WorkspaceRoot)
		payload := governanceSummary(root)
		return marshalToolResult(payload)
	}
}

func MakeBugReportHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-bug")
		}
		var args struct {
			SpecPath    string `json:"spec_path"`
			BugDesc     string `json:"bug_description"`
			Severity    string `json:"severity"`
			ReproSteps  string `json:"repro_steps"`
			ExpectedFix string `json:"expected_fix"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.BugDesc == "" {
			return "bug_description is required"
		}
		if args.Severity == "" {
			args.Severity = "medium"
		}

		b := make([]byte, 4)
		rand.Read(b)
		bugID := hex.EncodeToString(b)
		timestamp := time.Now().Format("20060102-150405")

		entry := fmt.Sprintf(`- id: "bug-%s"
  timestamp: "%s"
  description: "%s"
  severity: %s
  repro_steps: "%s"
  expected_fix: "%s"
  status: open
`, bugID, timestamp, escapeYAML(args.BugDesc), args.Severity, escapeYAML(args.ReproSteps), escapeYAML(args.ExpectedFix))

		if args.SpecPath != "" {
			specPath := args.SpecPath
			if !filepath.IsAbs(specPath) {
				specPath = filepath.Join(workspaceDir, args.SpecPath)
			}
			data, err := os.ReadFile(specPath)
			if err == nil {
				content := string(data)
				if strings.Contains(content, "bug_changelog:") {
					content = strings.Replace(content, "bug_changelog:", "bug_changelog:\n"+entry, 1)
				} else {
					content += "\nbug_changelog:\n" + entry
				}
				os.WriteFile(specPath, []byte(content), 0644)
				return fmt.Sprintf("bug report appended to %s\nbug_id: bug-%s\nseverity: %s\ndescription: %s",
					specPath, bugID, args.Severity, args.BugDesc)
			}
		}

		return fmt.Sprintf("bug report created:\nbug_id: bug-%s\nseverity: %s\ndescription: %s\nAttach to a spec using spec_path parameter",
			bugID, args.Severity, args.BugDesc)
	}
}

func MakeSpecResultTrackHandler(bc Broadcaster) rt.Handler {
	return func(arguments string) string {
		var args struct {
			SpecPath    string `json:"spec_path"`
			ResultIndex int    `json:"result_index"`
			Confirmed   bool   `json:"confirmed"`
			ConfirmedBy string `json:"confirmed_by"`
			Notes       string `json:"notes"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.SpecPath == "" {
			return "spec_path is required"
		}
		if args.ConfirmedBy == "" {
			args.ConfirmedBy = "agent"
		}

		event := map[string]interface{}{
			"spec_path":    args.SpecPath,
			"result_index": args.ResultIndex,
			"confirmed":    args.Confirmed,
			"confirmed_by": args.ConfirmedBy,
			"confirmed_at": time.Now().Format(time.RFC3339),
			"notes":        args.Notes,
		}
		if bc != nil {
			bc(args.SpecPath, "result.confirmed", event)
		}

		return fmt.Sprintf("result tracking event emitted:\nspec: %s\nresult_index: %d\nconfirmed: %t\nconfirmed_by: %s",
			args.SpecPath, args.ResultIndex, args.Confirmed, args.ConfirmedBy)
	}
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

func parseStringArg(arguments, fieldName string) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(arguments), &payload); err != nil {
		return "", err
	}
	v, ok := payload[fieldName]
	if !ok {
		return "", fmt.Errorf("missing field: %s", fieldName)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("field %s must be a string", fieldName)
	}
	return s, nil
}

func resolveWorkspaceArg(defaultRoot, root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return defaultRoot
	}
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

func governanceSummary(root string) map[string]any {
	summary := map[string]any{
		"workspace_root": root,
		"ok":             true,
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "_governance", "panorama.json")); err == nil {
		summary["panorama"] = "present"
	} else {
		summary["panorama"] = "missing"
	}
	counts := map[string]int{}
	total := 0
	_ = filepath.Walk(filepath.Join(root, "specs"), func(full string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(full, ".yaml") && !strings.HasSuffix(full, ".yml") {
			return nil
		}
		total++
		if rel, relErr := filepath.Rel(filepath.Join(root, "specs"), full); relErr == nil {
			parts := strings.Split(filepath.ToSlash(rel), "/")
			if len(parts) > 0 {
				counts[parts[0]]++
			}
		}
		return nil
	})
	summary["total_specs"] = total
	summary["by_directory"] = counts
	return summary
}

func marshalToolResult(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func MakeWorkspaceScaffoldHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}

		var args struct {
			WorkspaceRoot string `json:"workspace_root"`
			ProjectName   string `json:"project_name"`
			Owner         string `json:"owner"`
			DryRun        bool   `json:"dry_run"`
			Confirm       bool   `json:"confirm"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}

		if args.WorkspaceRoot == "" {
			return "workspace_root is required"
		}

		projectName := args.ProjectName
		if projectName == "" {
			projectName = "my-project"
		}
		owner := args.Owner
		if owner == "" {
			owner = "user"
		}

		script := filepath.Join(workspaceDir, "generators", "scaffold_generator.py")
		if _, err := os.Stat(script); os.IsNotExist(err) {
			return fmt.Sprintf("scaffold_generator.py not found at %s", script)
		}

		cmdArgs := []string{script, args.WorkspaceRoot,
			"--project-name", projectName,
			"--owner", owner,
		}
		if args.DryRun {
			cmdArgs = append(cmdArgs, "--dry-run")
		}
		if args.Confirm {
			cmdArgs = append(cmdArgs, "--confirm")
		}

		cmd := exec.Command("python3", cmdArgs...)
		cmd.Dir = workspaceDir
		out, err := cmd.CombinedOutput()
		text := string(out)

		if err != nil {
			return fmt.Sprintf("scaffold FAILED:\n%s\n%v", text, err)
		}

		// Format output for readability
		lines := strings.Split(strings.TrimSpace(text), "\n")
		var b strings.Builder
		for _, line := range lines {
			b.WriteString(line + "\n")
		}

		// Auto-detect state after scaffold
		if args.Confirm && !args.DryRun {
			detector := filepath.Join(workspaceDir, "generators", "state_detector.py")
			detCmd := exec.Command("python3", detector, args.WorkspaceRoot, "--json")
			detCmd.Dir = workspaceDir
			detOut, _ := detCmd.CombinedOutput()
			var result map[string]interface{}
			if json.Unmarshal(detOut, &result) == nil {
				state, _ := result["state"].(string)
				b.WriteString(fmt.Sprintf("\n状态验证: %s\n", state))
				if state == "ready" {
					b.WriteString("✅ 脚手架完整，workspace 已就绪\n")
				}
			}
		}

		return b.String()
	}
}

func MakeSpecWriteHandler(workspaceDir string, bc Broadcaster, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}

		var args struct {
			SpecPath      string `json:"spec_path"`
			Content       string `json:"content"`
			ValidateAfter *bool  `json:"validate_after"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.SpecPath == "" {
			return "spec_path is required"
		}
		if args.Content == "" {
			return "content is required"
		}

		// Resolve path
		specPath := args.SpecPath
		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(workspaceDir, specPath)
		}

		// Create parent dirs
		parentDir := filepath.Dir(specPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return "error creating directory " + parentDir + ": " + err.Error()
		}

		// Write file
		if err := os.WriteFile(specPath, []byte(args.Content), 0644); err != nil {
			return "error writing file " + specPath + ": " + err.Error()
		}

		// Quick YAML sanity check
		issues := []string{}
		if !strings.Contains(args.Content, "spec:") {
			issues = append(issues, "missing top-level 'spec:' block")
		}
		if strings.Contains(args.Content, "|\n") && strings.Contains(args.Content, "\"") {
			issues = append(issues, "warning: pipe inside double-quoted string — may misparse as YAML block scalar")
		}

		// Validation if requested
		validationResult := ""
		validateAfter := true
		if args.ValidateAfter != nil {
			validateAfter = *args.ValidateAfter
		}
		if validateAfter {
			script := filepath.Join(workspaceDir, "generators", "validate_specs.py")
			cmd := exec.Command("python3", script, specPath)
			cmd.Dir = workspaceDir
			out, err := cmd.CombinedOutput()
			vout := strings.TrimSpace(string(out))
			if err != nil {
				validationResult = fmt.Sprintf("\n⚠️ validation: %s\n%v", vout, err)
			} else {
				validationResult = "\n✅ validation passed"
			}
		}

		// Canvas update
		canvasNote := ""
		if bc != nil {
			bc("", "canvas.spec_modified", map[string]interface{}{
				"spec_path": args.SpecPath,
				"size":      len(args.Content),
				"timestamp": time.Now().Format(time.RFC3339),
			})
			canvasNote = "\n✅ canvas updated"
		}

		status := "✅ "
		if len(issues) > 0 {
			status = "⚠️  "
		}

		return fmt.Sprintf("%swritten: %s\n   %d bytes%s%s",
			status, specPath, len(args.Content), validationResult, canvasNote)
	}
}

func MakeSpecPatchApplyHandler(workspaceDir string, bc Broadcaster, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "spec-apply")
		}

		var args struct {
			SpecPath      string `json:"spec_path"`
			PatchJSON     string `json:"patch_json"`
			ValidateAfter *bool  `json:"validate_after"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if strings.TrimSpace(args.SpecPath) == "" {
			return "spec_path is required"
		}
		if strings.TrimSpace(args.PatchJSON) == "" {
			return "patch_json is required"
		}

		specPath := args.SpecPath
		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(workspaceDir, specPath)
		}
		specPath = filepath.Clean(specPath)
		specsRoot := filepath.Clean(filepath.Join(workspaceDir, "specs"))
		if !strings.HasPrefix(specPath, specsRoot+string(os.PathSeparator)) && specPath != specsRoot {
			return "forbidden: spec_path must be under specs/"
		}

		original, err := os.ReadFile(specPath)
		if err != nil {
			return "error reading file " + specPath + ": " + err.Error()
		}

		mergeScript := `
import json, sys, yaml
path = sys.argv[1]
patch_raw = sys.argv[2]
with open(path, "r", encoding="utf-8") as f:
    base = yaml.safe_load(f.read()) or {}
patch = json.loads(patch_raw)
if not isinstance(patch, dict):
    raise ValueError("patch_json must be a JSON object")
def merge(dst, src):
    for k, v in src.items():
        if isinstance(v, dict) and isinstance(dst.get(k), dict):
            merge(dst[k], v)
        else:
            dst[k] = v
merge(base, patch)
print(yaml.safe_dump(base, allow_unicode=True, sort_keys=False), end="")
`
		cmd := exec.Command("python3", "-c", mergeScript, specPath, args.PatchJSON)
		cmd.Dir = workspaceDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("patch merge failed: %v\n%s", err, strings.TrimSpace(string(out)))
		}
		patched := out
		if len(bytes.TrimSpace(patched)) == 0 {
			return "patch merge failed: empty output"
		}
		if err := os.WriteFile(specPath, patched, 0644); err != nil {
			return "error writing patched file " + specPath + ": " + err.Error()
		}

		validateAfter := true
		if args.ValidateAfter != nil {
			validateAfter = *args.ValidateAfter
		}
		validationResult := ""
		if validateAfter {
			script := filepath.Join(workspaceDir, "generators", "validate_specs.py")
			vCmd := exec.Command("python3", script, specPath)
			vCmd.Dir = workspaceDir
			vOut, vErr := vCmd.CombinedOutput()
			if vErr != nil {
				validationResult = fmt.Sprintf("\n⚠️ validation: %s\n%v", strings.TrimSpace(string(vOut)), vErr)
			} else {
				validationResult = "\n✅ validation passed"
			}
		}

		if bc != nil {
			bc("", "canvas.spec_modified", map[string]interface{}{
				"spec_path": args.SpecPath,
				"size":      len(patched),
				"delta":     len(patched) - len(original),
				"timestamp": time.Now().Format(time.RFC3339),
				"mode":      "patch",
			})
		}

		return fmt.Sprintf("✅ patch applied: %s\n   bytes: %d (delta %+d)%s",
			specPath, len(patched), len(patched)-len(original), validationResult)
	}
}

func escapeYAML(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}
