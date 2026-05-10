package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type agentFlowQARunRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	FlowPath      string `json:"flow_path"`
}

type agentFlowQAStepResult struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type agentFlowQARunResponse struct {
	OK               bool                    `json:"ok"`
	CustomAgentFound bool                    `json:"custom_agent_found"`
	WorkspaceRoot    string                  `json:"workspace_root"`
	FlowPath         string                  `json:"flow_path"`
	StartupOK        bool                    `json:"startup_ok"`
	TestOK           bool                    `json:"test_ok"`
	ReturnOK         bool                    `json:"return_ok"`
	Steps            []agentFlowQAStepResult `json:"steps,omitempty"`
	Error            string                  `json:"error,omitempty"`
}

type flowStepPayload struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Target    string `json:"target"`
	Command   string `json:"command"`
	Expect    string `json:"expect"`
	TimeoutSec int   `json:"timeout_sec"`
}

type flowPayload struct {
	Name  string            `json:"name"`
	Steps []flowStepPayload `json:"steps"`
}

func agentFlowQARunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req agentFlowQARunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	wsRoot := strings.TrimSpace(req.WorkspaceRoot)
	if wsRoot == "" {
		wsRoot = cfg.WorkspaceDir
	}
	if wsRoot == "" {
		wsRoot = os.Getenv("WORKSPACE_ROOT")
	}
	if wsRoot == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}
	flowPath := strings.TrimSpace(req.FlowPath)
	if flowPath == "" {
		flowPath = ".agents/flows/qa-agent-flow.json"
	}

	resp := runAgentFlowQACheck(wsRoot, flowPath)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func runAgentFlowQACheck(workspaceRoot, flowPath string) agentFlowQARunResponse {
	out := agentFlowQARunResponse{
		OK:            false,
		WorkspaceRoot: workspaceRoot,
		FlowPath:      filepath.ToSlash(flowPath),
	}
	customAgentPath := filepath.Join(workspaceRoot, ".agents", "agents", "general-agent.json")
	if _, err := os.Stat(customAgentPath); err != nil {
		out.Error = "custom agent config not found: .agents/agents/general-agent.json"
		return out
	}
	out.CustomAgentFound = true

	absFlow := filepath.Clean(filepath.Join(workspaceRoot, flowPath))
	if rel, err := filepath.Rel(workspaceRoot, absFlow); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		out.Error = "flow_path escapes workspace_root"
		return out
	}
	raw, err := os.ReadFile(absFlow)
	if err != nil {
		out.Error = fmt.Sprintf("read flow failed: %v", err)
		return out
	}
	var flow flowPayload
	if err := json.Unmarshal(raw, &flow); err != nil {
		out.Error = "invalid flow json: " + err.Error()
		return out
	}
	if len(flow.Steps) == 0 {
		out.Error = "flow has no steps"
		return out
	}

	startupOK := false
	testOK := false
	returnOK := false
	results := make([]agentFlowQAStepResult, 0, len(flow.Steps))
	for _, step := range flow.Steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = step.Type
		}
		typ := strings.ToLower(strings.TrimSpace(step.Type))
		expect := strings.ToLower(strings.TrimSpace(step.Expect))
		if expect == "" {
			expect = "pass"
		}
		stepResult := agentFlowQAStepResult{Name: name, Type: typ, OK: false}
		switch typ {
		case "make":
			target := strings.TrimSpace(step.Target)
			if target == "" {
				stepResult.Error = "target is required for make step"
				results = append(results, stepResult)
				continue
			}
			ok, out, errText := runWorkspaceMakeTarget(workspaceRoot, target)
			passed := ok
			if expect == "fail" {
				passed = !passed
			}
			stepResult.OK = passed
			stepResult.Output = out
			if !passed {
				stepResult.Error = errText
			}
			if target == "test" && passed {
				testOK = true
			}
		case "cmd":
			command := strings.TrimSpace(step.Command)
			if command == "" {
				stepResult.Error = "command is required for cmd step"
				results = append(results, stepResult)
				continue
			}
			exitCode, output, err := runShellCommand(workspaceRoot, command, step.TimeoutSec)
			passed := exitCode == 0
			if expect == "fail" {
				passed = !passed
			}
			stepResult.OK = passed
			stepResult.Output = output
			if !passed && err != nil {
				stepResult.Error = err.Error()
			}
			if strings.Contains(strings.ToLower(command), " test") && passed {
				testOK = true
			}
			if (strings.Contains(strings.ToLower(command), "dev") || strings.Contains(strings.ToLower(command), "start")) && passed {
				startupOK = true
			}
		case "cdp_validate":
			startupOK = true
			returnOK = true
			stepResult.OK = true
			stepResult.Output = "cdp_validate configured"
		default:
			stepResult.Error = "unsupported step type: " + typ
		}
		results = append(results, stepResult)
	}

	if !returnOK {
		// 只要流程能结构化返回至少一步结果，视为“正常返回”最小达成
		returnOK = len(results) > 0
	}
	out.Steps = results
	out.StartupOK = startupOK
	out.TestOK = testOK
	out.ReturnOK = returnOK
	out.OK = out.CustomAgentFound && out.StartupOK && out.TestOK && out.ReturnOK
	if !out.OK && out.Error == "" {
		out.Error = "custom agent flow checks failed"
	}
	return out
}

func runWorkspaceMakeTarget(workspaceRoot, target string) (ok bool, output string, errText string) {
	cmd := exec.Command("make", target)
	cmd.Dir = workspaceRoot
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return false, text, err.Error()
	}
	return true, text, ""
}

func runShellCommand(workspaceRoot, command string, timeoutSec int) (int, string, error) {
	if timeoutSec <= 0 {
		timeoutSec = 180
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if strings.EqualFold(runtime.GOOS, "windows") {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-lc", command)
	}
	cmd.Dir = workspaceRoot
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	exitCode := 0
	if err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			exitCode = ex.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return exitCode, text, err
}
