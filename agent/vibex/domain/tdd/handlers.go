package tdd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	rt "vibex/agent/agents/runtime/tools"
)

// Broadcaster sends canvas events to the SSE layer.
// Passed via dependency injection instead of a global variable.
type Broadcaster func(threadID, event string, data interface{})

// ─────────────────────────────────────────────────────────────
// Handler Factories (inject broadcaster dependency)
// ─────────────────────────────────────────────────────────────

func MakeTddDesignHandler(workspaceDir string, bc Broadcaster) rt.Handler {
	return func(arguments string) string {
		var args struct {
			SpecPath      string `json:"spec_path"`
			TestLanguage  string `json:"test_language"`
			Framework     string `json:"framework"`
			MockSpecPaths string `json:"mock_spec_paths"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.SpecPath == "" {
			return "spec_path is required"
		}
		if args.TestLanguage == "" {
			args.TestLanguage = "python"
		}
		if args.Framework == "" {
			args.Framework = defaultFramework(args.TestLanguage)
		}

		specPath := args.SpecPath
		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(workspaceDir, args.SpecPath)
		}

		data, err := os.ReadFile(specPath)
		if err != nil {
			return "error reading spec: " + err.Error()
		}

		io := parseIOContract(string(data))
		if io.Input == "" && io.Output == "" {
			return "warning: spec has no io_contract fields. Generate test cases from behavior field only."
		}

		specMeta := parseSpecMeta(string(data))
		specDir := filepath.Dir(specPath)
		testsDir := filepath.Join(specDir, "tests")
		os.MkdirAll(testsDir, 0755)

		b := make([]byte, 4)
		rand.Read(b)
		testID := hex.EncodeToString(b)
		safeName := specMeta.Name
		if safeName == "" {
			safeName = "spec"
		}

		var testFile, testContent string
		switch args.TestLanguage {
		case "go":
			testFile, testContent = generateGoTests(safeName, specMeta.ID, io, args.Framework, testID)
		case "python":
			testFile, testContent = generatePythonTests(safeName, specMeta.ID, io, args.Framework, testID)
		case "typescript", "javascript":
			testFile, testContent = generateTSTests(safeName, specMeta.ID, io, args.Framework, testID)
		default:
			return "unsupported test_language: " + args.TestLanguage + ". Supported: go, python, typescript, javascript"
		}

		testPath := filepath.Join(testsDir, testFile)
		if err := os.WriteFile(testPath, []byte(testContent), 0644); err != nil {
			return "error writing test file: " + err.Error()
		}

		emitTDDCanvasNodes(specMeta.ID, testID, io, testPath, bc)

		cases := generateTestCasesFromIO(io)
		return fmt.Sprintf("TDD test cases designed:\nspec: %s\ntest_file: %s\nlanguage: %s\nframework: %s\ntest_count: %d\n\nTest cases generated from io_contract:\n%s\n\nNext: Run tests with tdd_run to see RED (failing) → then implement → GREEN (passing)",
			specMeta.ID, testPath, args.TestLanguage, args.Framework, len(cases), cases)
	}
}

func MakeTddRunHandler(workspaceDir string, bc Broadcaster) rt.Handler {
	return func(arguments string) string {
		var args struct {
			SpecPath    string `json:"spec_path"`
			TestFile    string `json:"test_file"`
			TestLanguage string `json:"test_language"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.TestFile == "" {
			return "test_file is required"
		}
		if args.TestLanguage == "" {
			ext := strings.ToLower(filepath.Ext(args.TestFile))
			switch ext {
			case ".go":
				args.TestLanguage = "go"
			case ".py":
				args.TestLanguage = "python"
			case ".ts", ".tsx":
				args.TestLanguage = "typescript"
			case ".js", ".jsx":
				args.TestLanguage = "javascript"
			default:
				return "cannot infer test_language from extension " + ext + ". Specify test_language explicitly."
			}
		}

		testPath := args.TestFile
		if !filepath.IsAbs(testPath) {
			testPath = filepath.Join(workspaceDir, args.TestFile)
		}

		var cmd *exec.Cmd
		var runDir string
		switch args.TestLanguage {
		case "go":
			cmd = exec.Command("go", "test", "-v", testPath)
			runDir = filepath.Dir(testPath)
		case "python":
			cmd = exec.Command("python3", "-m", "pytest", testPath, "-v")
			runDir = workspaceDir
		case "typescript":
			cmd = exec.Command("npx", "vitest", "run", testPath)
			runDir = workspaceDir
		case "javascript":
			cmd = exec.Command("npx", "jest", testPath)
			runDir = workspaceDir
		default:
			return "unsupported test_language: " + args.TestLanguage
		}
		cmd.Dir = runDir
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))

		status := "UNKNOWN"
		passed := 0
		failed := 0

		if err != nil {
			status = "RED"
			failed = countFailedTests(text)
		} else {
			status = "GREEN"
			passed = countPassedTests(text)
		}

		specID := args.SpecPath
		if args.SpecPath != "" && filepath.IsAbs(args.SpecPath) {
			specID = filepath.Base(args.SpecPath)
		}
		emitTDDCycleCanvas(specID, status, passed, failed, text, bc)

		result := fmt.Sprintf("TDD cycle result: **%s**\n\n", status)
		if status == "RED" {
			result += fmt.Sprintf("Tests failing: %d\n\n%s\n\n→ Implement the feature to make tests pass.", failed, text)
		} else {
			result += fmt.Sprintf("Tests passing: %d\n\n%s\n\nAll tests green! Feature is complete.", passed, text)
		}
		return result
	}
}

func MakeTddIterateHandler(workspaceDir string, bc Broadcaster) rt.Handler {
	return func(arguments string) string {
		var args struct {
			SpecPath    string `json:"spec_path"`
			TestFile    string `json:"test_file"`
			TestLanguage string `json:"test_language"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "invalid args: " + err.Error()
		}
		if args.TestFile == "" {
			return "test_file is required"
		}

		// Run the tests first
		runArgs, _ := json.Marshal(map[string]string{
			"spec_path":     args.SpecPath,
			"test_file":     args.TestFile,
			"test_language": args.TestLanguage,
		})
		runResult := MakeTddRunHandler(workspaceDir, bc)(string(runArgs))

		specPath := args.SpecPath
		if !filepath.IsAbs(specPath) && specPath != "" {
			specPath = filepath.Join(workspaceDir, args.SpecPath)
		}

		io := IOPair{Input: "", Output: "", Boundary: "", Behavior: ""}
		if specPath != "" {
			if data, err := os.ReadFile(specPath); err == nil {
				io = parseIOContract(string(data))
			}
		}

		nextSteps := extractBehaviorSteps(io.Behavior)
		cycles := buildTDDCycles(nextSteps)

		result := runResult + "\n\nTDD Cycle progression:\n" + cycles + "\n\n→ Use canvas_update to visualize current cycle status."
		return result
	}
}

// TddDesignHandler is the default handler using the global broadcaster (for backward compat).
var TddDesignHandler rt.Handler
var TddRunHandler rt.Handler
var TddIterateHandler rt.Handler

// ─────────────────────────────────────────────────────────────
// Test Case Generators
// ─────────────────────────────────────────────────────────────

type IOPair struct {
	Input    string
	Output   string
	Boundary string
	Behavior string
}

type TestCase struct {
	Name        string
	Input       string
	Expected    string
	Boundary    string
	BehaviorIdx int
}

func generateGoTests(name, specID string, io IOPair, framework, testID string) (string, string) {
	filename := fmt.Sprintf("%s_test.go", sanitizeName(name))
	cases := generateTestCasesFromIO(io)
	var b strings.Builder
	b.WriteString("package tests\n\n")
	b.WriteString("import \"testing\"\n\n")
	b.WriteString(fmt.Sprintf("// Test cases for: %s\n", specID))
	b.WriteString(fmt.Sprintf("// Generated by VibeX Agent at %s\n\n", time.Now().Format(time.RFC3339)))

	for i, tc := range cases {
		funcName := sanitizeName(tc.Name)
		if funcName == "" {
			funcName = fmt.Sprintf("TestCase%d", i+1)
		}
		b.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", funcName))
		b.WriteString(fmt.Sprintf("\t// Input: %s\n", escapeComment(tc.Input)))
		b.WriteString(fmt.Sprintf("\t// Expected Output: %s\n", escapeComment(tc.Expected)))
		b.WriteString(fmt.Sprintf("\t// Boundary: %s\n", escapeComment(tc.Boundary)))
		b.WriteString("\n")
		b.WriteString("\t// TODO: implement the function under test\n")
		b.WriteString("\tt.Errorf(\"test not implemented: RED phase — write implementation to make this GREEN\\n\")\n")
		b.WriteString("}\n\n")
	}
	return filename, b.String()
}

func generatePythonTests(name, specID string, io IOPair, framework, testID string) (string, string) {
	filename := fmt.Sprintf("test_%s.py", sanitizeName(name))
	cases := generateTestCasesFromIO(io)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Test cases for: %s\n", specID))
	b.WriteString(fmt.Sprintf("# Generated by VibeX Agent at %s\n", time.Now().Format(time.RFC3339)))
	b.WriteString("import pytest\n\n\n")

	for i, tc := range cases {
		funcName := sanitizeName(tc.Name)
		if funcName == "" {
			funcName = fmt.Sprintf("test_case_%d", i+1)
		}
		b.WriteString(fmt.Sprintf("def %s():\n", funcName))
		b.WriteString(fmt.Sprintf("    \"\"\"Test: %s\n", escapeComment(tc.Name)))
		b.WriteString(fmt.Sprintf("    Input: %s\n", escapeComment(tc.Input)))
		b.WriteString(fmt.Sprintf("    Expected: %s\n", escapeComment(tc.Expected)))
		b.WriteString(fmt.Sprintf("    Boundary: %s\n", escapeComment(tc.Boundary)))
		b.WriteString("    \"\"\"\n")
		b.WriteString("    # TODO: implement the function under test\n")
		b.WriteString("    raise NotImplementedError(\"RED phase — write implementation to make this GREEN\")\n\n\n")
	}
	return filename, b.String()
}

func generateTSTests(name, specID string, io IOPair, framework, testID string) (string, string) {
	if framework == "" {
		framework = "vitest"
	}
	filename := fmt.Sprintf("%s.test.ts", sanitizeName(name))
	cases := generateTestCasesFromIO(io)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("// Test cases for: %s\n", specID))
	b.WriteString(fmt.Sprintf("// Generated by VibeX Agent at %s\n", time.Now().Format(time.RFC3339)))

	if framework == "jest" {
		b.WriteString("import { describe, it, expect } from '@jest/globals'\n\n")
	} else {
		b.WriteString("import { describe, it, expect } from 'vitest'\n\n")
	}

	for i, tc := range cases {
		funcName := sanitizeName(tc.Name)
		if funcName == "" {
			funcName = fmt.Sprintf("testCase%d", i+1)
		}
		b.WriteString(fmt.Sprintf("describe('%s', () => {\n", specID))
		b.WriteString(fmt.Sprintf("  it('%s', () => {\n", escapeComment(tc.Name)))
		b.WriteString(fmt.Sprintf("    // Input: %s\n", escapeComment(tc.Input)))
		b.WriteString(fmt.Sprintf("    // Expected Output: %s\n", escapeComment(tc.Expected)))
		b.WriteString(fmt.Sprintf("    // Boundary: %s\n", escapeComment(tc.Boundary)))
		b.WriteString("    // TODO: implement\n")
		b.WriteString("    expect(true).toBe(false) // RED phase\n")
		b.WriteString("  })\n")
		b.WriteString("})\n\n")
	}
	return filename, b.String()
}

// ─────────────────────────────────────────────────────────────
// IO Contract Parsing
// ─────────────────────────────────────────────────────────────

func parseIOContract(content string) IOPair {
	io := IOPair{}

	re := regexp.MustCompile(`(?i)io_contract:\s*\n((?:\s+.+\n?)+)`)
	if m := re.FindStringSubmatch(content); len(m) > 1 {
		ioBlock := m[1]
		lines := strings.Split(ioBlock, "\n")
		current := ""
		for _, line := range lines {
			fieldMatch := regexp.MustCompile(`^\s+(input|output|boundary|behavior)\s*:\s*(.*)$`)
			if fm := fieldMatch.FindStringSubmatch(line); len(fm) > 0 {
				if current != "" {
					setIOField(&io, current, strings.TrimSpace(getFieldValue(ioBlock, current)))
				}
				current = strings.ToLower(fm[1])
			}
		}
		if current != "" {
			setIOField(&io, current, strings.TrimSpace(getFieldValue(ioBlock, current)))
		}
	}

	if io.Input == "" {
		io.Input = extractFieldValue(content, "input")
		io.Output = extractFieldValue(content, "output")
		io.Boundary = extractFieldValue(content, "boundary")
		io.Behavior = extractFieldValue(content, "behavior")
	}

	return io
}

func setIOField(io *IOPair, field, value string) {
	switch field {
	case "input":
		io.Input = value
	case "output":
		io.Output = value
	case "boundary":
		io.Boundary = value
	case "behavior":
		io.Behavior = value
	}
}

func getFieldValue(block, field string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?i)%s\s*:\s*(.*?)(?=\n\s+[a-z_]+\s*:|$)`, field))
	if m := re.FindStringSubmatch(block); len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractFieldValue(content, field string) string {
	reBlock := regexp.MustCompile(fmt.Sprintf(`(?im)%s\s*[|>]\s*\n((?:\s+.+\n?)+)`, field))
	if m := reBlock.FindStringSubmatch(content); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	reSingle := regexp.MustCompile(fmt.Sprintf(`(?im)%s\s*:\s*(.+)`, field))
	if m := reSingle.FindStringSubmatch(content); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func parseSpecMeta(content string) struct{ ID, Name, Parent string } {
	meta := struct{ ID, Name, Parent string }{}
	idRe := regexp.MustCompile(`(?im)^\s*id\s*:\s*"?([^"\n]+)"?`)
	nameRe := regexp.MustCompile(`(?im)^\s*name\s*:\s*"?([^"\n]+)"?`)
	parentRe := regexp.MustCompile(`(?im)^\s*parent\s*:\s*"?([^"\n]+)"?`)

	if m := idRe.FindStringSubmatch(content); len(m) > 1 {
		meta.ID = strings.TrimSpace(m[1])
	}
	if m := nameRe.FindStringSubmatch(content); len(m) > 1 {
		meta.Name = strings.TrimSpace(m[1])
	}
	if m := parentRe.FindStringSubmatch(content); len(m) > 1 {
		meta.Parent = strings.TrimSpace(m[1])
	}
	return meta
}

// ─────────────────────────────────────────────────────────────
// Test Case Generation from IO
// ─────────────────────────────────────────────────────────────

func generateTestCasesFromIO(io IOPair) []TestCase {
	cases := []TestCase{}

	if io.Input != "" && io.Output != "" {
		cases = append(cases, TestCase{
			Name:     "Happy path: input produces expected output",
			Input:    io.Input,
			Expected: io.Output,
			Boundary: io.Boundary,
		})
	}

	if io.Boundary != "" {
		boundaryLines := strings.Split(io.Boundary, "\n")
		for _, line := range boundaryLines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, "not") || strings.Contains(line, "only") ||
				strings.Contains(line, "max") || strings.Contains(line, "min") ||
				strings.Contains(line, "at least") || strings.Contains(line, "at most") {
				cases = append(cases, TestCase{
					Name:     "Boundary: " + line,
					Input:    line,
					Expected: "Behaves correctly at boundary",
					Boundary: line,
				})
			}
		}
	}

	steps := extractBehaviorSteps(io.Behavior)
	for i, step := range steps {
		if len(step) > 5 {
			cases = append(cases, TestCase{
				Name:        fmt.Sprintf("Behavior step %d: %s", i+1, firstFewWords(step, 6)),
				Input:       step,
				Expected:    "Behavior step executes correctly",
				BehaviorIdx: i + 1,
			})
		}
	}

	if io.Input != "" {
		cases = append(cases, TestCase{
			Name:     "Edge case: empty/null input",
			Input:    "(empty)",
			Expected: "Handles empty input gracefully",
			Boundary: "empty input",
		})
	}

	return cases
}

func extractBehaviorSteps(behavior string) []string {
	if behavior == "" {
		return nil
	}
	var steps []string
	lines := strings.Split(behavior, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		numRe := regexp.MustCompile(`^[0-9]+[.)>:]\s*(.+)`)
		if m := numRe.FindStringSubmatch(line); len(m) > 1 {
			steps = append(steps, m[1])
			continue
		}
		if strings.HasPrefix(line, "→") || strings.HasPrefix(line, "->") {
			steps = append(steps, strings.TrimPrefix(strings.TrimPrefix(line, "→"), "->"))
		}
	}
	return steps
}

func buildTDDCycles(steps []string) string {
	var b strings.Builder
	colors := []string{"🔴 RED", "🟢 GREEN", "🔵 REFACTOR", "🟡 YELLOW (review)"}
	for i, step := range steps {
		color := colors[i%len(colors)]
		b.WriteString(fmt.Sprintf("  %s | Step %d: %s\n", color, i+1, firstFewWords(step, 20)))
	}
	return b.String()
}

func firstFewWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) <= n {
		return s
	}
	return strings.Join(words[:n], " ") + "..."
}

// ─────────────────────────────────────────────────────────────
// Canvas Emission Helpers
// ─────────────────────────────────────────────────────────────

func emitTDDCanvasNodes(specID, testID string, io IOPair, testPath string, bc Broadcaster) {
	cases := generateTestCasesFromIO(io)
	if bc == nil {
		return
	}

	nodes := make([]map[string]interface{}, 0, len(cases)+3)

	nodes = append(nodes, map[string]interface{}{
		"type":   "tdd_phase",
		"phase":  "RED",
		"label":  "RED: Write failing tests",
		"spec_id": specID,
		"test_file": testPath,
		"test_count": len(cases),
		"color":  "#ef4444",
	})
	nodes = append(nodes, map[string]interface{}{
		"type":   "tdd_phase",
		"phase":  "GREEN",
		"label":  "GREEN: Implement to pass",
		"spec_id": specID,
		"color":  "#22c55e",
	})
	nodes = append(nodes, map[string]interface{}{
		"type":   "tdd_phase",
		"phase":  "REFACTOR",
		"label":  "REFACTOR: Clean up code",
		"spec_id": specID,
		"color":  "#3b82f6",
	})

	for i, tc := range cases {
		nodes = append(nodes, map[string]interface{}{
			"type":     "test_case",
			"case_id":  i + 1,
			"name":     tc.Name,
			"input":    tc.Input,
			"expected": tc.Expected,
			"status":  "RED",
			"spec_id": specID,
			"test_id": testID,
		})
	}

	bc(specID, "canvas.tdd_nodes", map[string]interface{}{
		"spec_id":    specID,
		"test_id":    testID,
		"test_file":  testPath,
		"test_count": len(cases),
		"phases":     []string{"RED", "GREEN", "REFACTOR"},
		"nodes":      nodes,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}

func emitTDDCycleCanvas(specID, status string, passed, failed int, output string, bc Broadcaster) {
	if bc == nil {
		return
	}
	color := "#ef4444"
	phase := "RED"
	if status == "GREEN" {
		color = "#22c55e"
		phase = "GREEN"
	}

	bc(specID, "canvas.tdd_cycle", map[string]interface{}{
		"spec_id":  specID,
		"phase":    phase,
		"status":   status,
		"color":    color,
		"passed":   passed,
		"failed":   failed,
		"output":   firstFewWords(output, 500),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ─────────────────────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────────────────────

func sanitizeName(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if len(s) > 0 && s[0] >= 'a' && s[0] <= 'z' {
		runes := []rune(s)
		runes[0] = rune(runes[0] - 'a' + 'A')
		s = string(runes)
	}
	return s
}

func escapeComment(s string) string {
	s = strings.ReplaceAll(s, "*/", "*")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func defaultFramework(lang string) string {
	switch lang {
	case "go":
		return "testing"
	case "python":
		return "pytest"
	case "typescript":
		return "vitest"
	case "javascript":
		return "jest"
	default:
		return "pytest"
	}
}

func countFailedTests(output string) int {
	re := regexp.MustCompile(`(?i)(FAIL|failed|error|failing|✗)`)
	return len(re.FindAllString(output, -1))
}

func countPassedTests(output string) int {
	re := regexp.MustCompile(`(?i)(PASS|passed|ok|✓|success)`)
	return len(re.FindAllString(output, -1))
}
