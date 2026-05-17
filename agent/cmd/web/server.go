// cmd/web/server.go — HTTP handlers, tool loop, and tool building.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"vibex-workbench/pkg/vibexpaths"
	"vibex/agent/adapters"
	"vibex/agent/agents/background"
	"vibex/agent/agents/runtime"
	rtools "vibex/agent/agents/runtime/tools"
	"vibex/agent/agents/sessions"
	"vibex/agent/agents/skills"
	"vibex/agent/agents/subagent"
	"vibex/agent/internal/common"
	"vibex/agent/vibex/domain"
	"vibex/agent/vibex/domain/rulesengine"
	"vibex/generators/memlace"

	"github.com/openai/openai-go/v3/responses"
)

// developerMessage is the system prompt loaded by all agent turns.
var developerMessage = `You are a VibeX Spec Governance Assistant. Follow this workflow:

GOAL routes (new project/feature idea):
  1. Use spec_designer to create a spec draft
  2. Ask clarifying questions until intent is unambiguous
  3. After confirmation → use spec_feature to break into feature specs
  4. Use tdd_design to generate test cases from the spec's io_contract
  5. Use canvas_update to show spec relationship on canvas

FEATURE routes (implementing existing spec):
  1. Use plan_graph_create to create a reviewable execution/tool graph
  2. Use tool_route_preview to route graph nodes to builtin/custom tools
  3. Ask for confirmation before write_file, bash, make_generate, or custom execution
  4. Use spec_validate to check the target spec YAML
  5. Use tdd_design to design test cases (RED phase)
  6. Use tdd_run to verify tests fail (RED)
  7. Implement only the confirmed graph nodes
  8. Use tdd_run to verify tests pass (GREEN)
  9. Use spec_sync push after changes
  10. Use make_validate to verify all specs

BUG routes:
  1. Use bug_report to create changelog entry
  2. Use tdd_design for regression test cases
  3. Fix the issue
  4. Use tdd_run to confirm GREEN

TDD Cycle:
  RED → GREEN → REFACTOR
  - tdd_design: creates test cases from spec io_contract (input/output/boundary/behavior)
  - tdd_run: executes tests, returns RED (fail) or GREEN (pass)
  - tdd_iterate: runs tests + shows next behavior step from spec

Core principles:
- Every spec must have: input, output, boundary, behavior fields filled
- Test cases are generated from io_contract, not from implementation
- Use canvas_update to show TDD cycle progress
- Prefer Plan Graph → Tool Routing Graph → confirmed execution over direct code generation
- Never assume — always clarify ambiguous requirements
- After any code/spec change, always run make_validate

Tooling:
- The tool list is the full registry for routing; each turn only invoke tools relevant to the current step — ignore unrelated tools (do not loop through every tool).
- read_file / write_file / bash: use paths relative to the workspace root when possible; absolute paths under the workspace are normalized automatically.
- bash on Windows uses Git Bash when available (set VIBEX_POSIX_SHELL to override).`

var (
	pptQueueOnce sync.Once
	pptTaskQueue chan func()
)

func ensurePPTQueue() {
	pptQueueOnce.Do(func() {
		pptTaskQueue = make(chan func(), 64)
		go func() {
			for task := range pptTaskQueue {
				task()
			}
		}()
	})
}

func enqueuePPTTask(task func()) bool {
	ensurePPTQueue()
	select {
	case pptTaskQueue <- task:
		return true
	default:
		return false
	}
}

type agentProfileConfig struct {
	DeveloperMessage string
	AllowedTools     map[string]struct{}
	RequiredSkills   []string
}

type profileJSON struct {
	DeveloperMessage string   `json:"developer_message"`
	AllowedTools     []string `json:"allowed_tools"`
	RequiredSkills   []string `json:"required_skills"`
}

// defaultSkillAgentToolNames — workspace / Git / CI style skill agents (shell + files + skills + bg + subagents).
var defaultSkillAgentToolNames = []string{
	"bash", "read_file", "write_file", "append_file", "todo_set",
	"bash_bg", "bg_wait", "bg_list",
	"skill_list", "skill_load", "skill_unload",
	"subagent_spawn", "subagent_wait",
}

func toAllowSet(names ...string) map[string]struct{} {
	if len(names) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		out[n] = struct{}{}
	}
	return out
}

func isSafeProfileName(profileName string) bool {
	n := strings.TrimSpace(profileName)
	if n == "" {
		return false
	}
	if strings.ContainsAny(n, `/\.`) {
		return false
	}
	return true
}

func loadProfileJSON(path string, fromSkillDir bool) (agentProfileConfig, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return agentProfileConfig{}, false
	}
	var raw profileJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("[agent-profile] invalid profile json %s: %v", path, err)
		return agentProfileConfig{}, false
	}
	devMsg := strings.TrimSpace(raw.DeveloperMessage)
	allowed := toAllowSet(raw.AllowedTools...)
	if fromSkillDir && len(raw.AllowedTools) == 0 {
		allowed = toAllowSet(defaultSkillAgentToolNames...)
	}
	cfg := agentProfileConfig{
		DeveloperMessage: devMsg,
		AllowedTools:     allowed,
		RequiredSkills:   append([]string(nil), raw.RequiredSkills...),
	}
	cfg = enrichAgentProfileWhenEmptyMessage(cfg)
	return cfg, true
}

// mergeSkillCoLocatedAgentJSON overlays co-located `skills/<name>/agent.json`, `.vibex/agents/skills/...`, then `.vibex/.agents/skills/...` when `.vibex/agents/agents/<name>.json` exists.
func mergeSkillCoLocatedAgentJSON(base, name string, cfg agentProfileConfig) agentProfileConfig {
	paths := []string{
		filepath.Join(base, "skills", name, "agent.json"),
		filepath.Join(base, filepath.FromSlash(vibexpaths.AgentsRootRel), "skills", name, "agent.json"),
		filepath.Join(base, filepath.FromSlash(vibexpaths.AgentsDotAgentsRootRel), "skills", name, "agent.json"),
	}
	for _, p := range paths {
		cfg = mergeOneSkillAgentJSONOverlay(cfg, p)
	}
	return cfg
}

func mergeOneSkillAgentJSONOverlay(cfg agentProfileConfig, agentJSONPath string) agentProfileConfig {
	data, err := os.ReadFile(agentJSONPath)
	if err != nil {
		return cfg
	}
	var raw profileJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("[agent-profile] invalid overlay json %s: %v", agentJSONPath, err)
		return cfg
	}
	if len(cfg.RequiredSkills) == 0 && len(raw.RequiredSkills) > 0 {
		cfg.RequiredSkills = append([]string(nil), raw.RequiredSkills...)
	}
	if len(raw.AllowedTools) > 0 {
		cfg.AllowedTools = toAllowSet(raw.AllowedTools...)
	}
	return cfg
}

func enrichAgentProfileWhenEmptyMessage(p agentProfileConfig) agentProfileConfig {
	if strings.TrimSpace(p.DeveloperMessage) != "" {
		return p
	}
	if len(p.RequiredSkills) != 1 || skillRegistry == nil {
		return p
	}
	n := skills.NormalizeName(p.RequiredSkills[0])
	def, ok := skillRegistry.Get(n)
	if !ok {
		return p
	}
	p.DeveloperMessage = fmt.Sprintf(
		"You are the specialized `%s` agent. Follow the loaded SKILL.md workflow (steps, guardrails, output) exactly; avoid unrelated VibeX tools unless this skill explicitly requires them.\n\nSkill summary: %s",
		def.Name,
		def.Description,
	)
	return p
}

func loadAgentProfileFromFile(profileName string) (agentProfileConfig, bool) {
	if !isSafeProfileName(profileName) {
		return agentProfileConfig{}, false
	}
	name := strings.TrimSpace(strings.ToLower(profileName))
	base := cfg.WorkspaceDir

	if cfg, ok := loadProfileJSON(filepath.Join(base, filepath.FromSlash(vibexpaths.AgentsRootRel), "profiles", name+".json"), false); ok {
		return cfg, true
	}

	if msg, req, allow, ok := common.LoadSpecializedAgent(base, name, "web"); ok {
		cfg := agentProfileConfig{
			DeveloperMessage: strings.TrimSpace(msg),
			AllowedTools:     toAllowSet(allow...),
			RequiredSkills:   append([]string(nil), req...),
		}
		cfg = mergeSkillCoLocatedAgentJSON(base, name, cfg)
		cfg = enrichAgentProfileWhenEmptyMessage(cfg)
		return cfg, true
	}

	// 仅 skill 包：优先 `.vibex/.agents/skills`，其次 `.vibex/agents/skills`，最后 `<workspace>/skills`（与合并注册「后者覆盖」心智一致）。
	skillAgentPaths := []string{
		filepath.Join(base, filepath.FromSlash(vibexpaths.AgentsDotAgentsRootRel), "skills", name, "agent.json"),
		filepath.Join(base, filepath.FromSlash(vibexpaths.AgentsRootRel), "skills", name, "agent.json"),
		filepath.Join(base, "skills", name, "agent.json"),
	}
	for _, p := range skillAgentPaths {
		if cfg, ok := loadProfileJSON(p, true); ok {
			return cfg, true
		}
	}
	return agentProfileConfig{}, false
}

func ensureProfileSkillsLoaded(state *threadState, profile agentProfileConfig) error {
	if len(profile.RequiredSkills) == 0 {
		return nil
	}
	for _, skillName := range profile.RequiredSkills {
		name := strings.TrimSpace(skillName)
		if name == "" {
			continue
		}
		if _, _, err := state.skillState.Load(name, skillRegistry); err != nil {
			return fmt.Errorf("required skill load failed (%s): %w", name, err)
		}
	}
	return nil
}

func resolveAgentProfile(profileName string) agentProfileConfig {
	if fromFile, ok := loadAgentProfileFromFile(strings.TrimSpace(strings.ToLower(profileName))); ok {
		return fromFile
	}

	switch strings.TrimSpace(strings.ToLower(profileName)) {
	case "", "default", "spec-governance":
		return agentProfileConfig{
			DeveloperMessage: developerMessage,
			AllowedTools:     nil, // full registry
		}
	case "ppt-generator":
		return agentProfileConfig{
			DeveloperMessage: `You are a focused HTML PPT generation agent.
- Only work on the requested spec and target html file.
- Link the deck in YAML under the top-level spec block as spec.ppt_file (workspace-relative). Do not put ppt_file under prototype (prototype is for UI prototype artifacts).
- After generating the HTML, ensure the spec file sets spec.ppt_file to the output path.
- Prefer loading html-ppt skill if available, then generate/overwrite only the target file.
- Keep output deterministic and concise. Do not call unrelated governance/TDD tools.
- If skill is unavailable, still produce a single self-contained HTML deck with keyboard navigation.`,
			AllowedTools: toAllowSet(
				"read_file",
				"write_file",
				"append_file",
				"skill_list",
				"skill_load",
				"skill_unload",
			),
		}
	default:
		// Unknown profile falls back to default to keep API backward-compatible.
		return agentProfileConfig{
			DeveloperMessage: developerMessage,
			AllowedTools:     nil,
		}
	}
}

func filterSpecsByAllowlist(specs []rtools.Spec, allow map[string]struct{}) []rtools.Spec {
	if len(allow) == 0 {
		return specs
	}
	filtered := make([]rtools.Spec, 0, len(specs))
	for _, spec := range specs {
		if _, ok := allow[spec.Name]; ok {
			filtered = append(filtered, spec)
		}
	}
	return filtered
}

// sseToolLoopHooks maps runtime tool loop events to SSE (same thread as chat UI).
func sseToolLoopHooks(threadID string) *runtime.ToolLoopHooks {
	return &runtime.ToolLoopHooks{
		OnAssistantDelta: func(text string, isFinal bool) {
			broadcastSSE(threadID, "message.delta", map[string]interface{}{
				"role": "assistant", "delta": text, "is_final": isFinal,
			})
		},
		OnToolCalled: func(name, callID string, args map[string]any) {
			broadcastSSE(threadID, "tool.called", map[string]interface{}{
				"toolName": name, "tool": name, "invocationId": callID, "call_id": callID,
				"runId": threadID, "args": args,
			})
			// PS 工具：额外广播 agent:tool:call 事件，扩展侧订阅此事件驱动 DOM 操作
			if isPSTool(name) {
				broadcastSSE(threadID, "agent:tool:call", map[string]interface{}{
					"name":   name,
					"args":   args,
					"callId": callID,
				})
			}
		},
		OnToolCompleted: func(name, callID, result string) {
			broadcastSSE(threadID, "tool.completed", map[string]interface{}{
				"toolName": name, "tool": name, "invocationId": callID, "call_id": callID,
				"result": result,
			})
		},
		OnRepairDecision: func(decision rulesengine.RepairDecision, envelope rulesengine.RepairEnvelope) {
			broadcastSSE(threadID, "repair.decision", map[string]interface{}{
				"decision": decision,
				"envelope": envelope,
			})
		},
	}
}

// isPSTool returns true if name is a Prototype-Skill PS tool.
func isPSTool(name string) bool {
	switch name {
	case "ps_highlight", "ps_annotate", "ps_parse", "ps_bind", "ps_onboard", "ps_get_page_context":
		return true
	}
	return false
}

// ── Build tools & handlers ──────────────────────────────────────

func buildToolsAndHandlers(threadID string, cfg common.Config,
	skillRegistry *skills.Registry, allowedTools map[string]struct{}) ([]responses.ToolUnionParam, map[string]rtools.Handler, *background.Manager, *subagent.Manager) {

	state := getThreadState(threadID)
	bgMgr := background.NewManager()
	subMgr := subagent.NewManager(4)

	runner := func(ctx context.Context, taskSummary string) (string, error) {
		childTodo := rtools.NewTodoStore()
		childSkills := skills.NewState()
		childSkills.SetActive(state.skillState.ActiveNames())
		childBg := background.NewManager()
		childSpecs := rtools.ParentSpecs(childTodo, childBg, nil, nil, childSkills, skillRegistry)
		childVibexReg := domain.NewRegistry(cfg.WorkspaceDir, broadcastSSE, SetStepType)
		childSpecs = append(childSpecs, childVibexReg.ToolSpecs()...)
		childTools := rtools.BuildTools(childSpecs)
		childHandlers := rtools.BuildHandlers(childSpecs)
		for name, handler := range childVibexReg.ToolHandlers() {
			childHandlers[name] = handler
		}
		childMsgs := []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(developerMessage, responses.EasyInputMessageRoleDeveloper),
			responses.ResponseInputItemParamOfMessage("Sub-agent task:\n"+strings.TrimSpace(taskSummary), responses.EasyInputMessageRoleUser),
		}
		answer, _, err := runtime.RunToolLoop(ctx, llm, cfg.SubAgentModel, childTools, childHandlers, childTodo, childMsgs, childBg, nil, childSkills, skillRegistry, sseToolLoopHooks(threadID))
		return answer, err
	}

	// Parent (base nanoClaudeCode) specs
	specs := rtools.ParentSpecs(state.todo, bgMgr, subMgr, runner, state.skillState, skillRegistry)

	// Vibex domain specs (spec tools + TDD tools)
	vibexReg := domain.NewRegistry(cfg.WorkspaceDir, broadcastSSE, SetStepType)
	vibexSpecs := vibexReg.ToolSpecs()
	specs = append(specs, vibexSpecs...)

	// Prototype-Skill PS 工具：供 Agent 通过 SSE+HTTP 回调与 Chrome 扩展通信
	specs = append(specs, rtools.PSSpecs...)

	specs = filterSpecsByAllowlist(specs, allowedTools)
	tools := rtools.BuildTools(specs)
	handlers := rtools.BuildHandlers(specs)

	// Merge vibex handlers (they use factory with broadcaster)
	for name, h := range vibexReg.ToolHandlers() {
		if len(allowedTools) == 0 {
			handlers[name] = h
			continue
		}
		if _, ok := allowedTools[name]; ok {
			handlers[name] = h
		}
	}

	// Wrap PS tool handlers: register pending result, wait for extension callback, return real result
	for _, name := range []string{"ps_highlight", "ps_annotate", "ps_parse", "ps_bind", "ps_onboard", "ps_get_page_context"} {
		if len(allowedTools) > 0 {
			if _, ok := allowedTools[name]; !ok {
				continue
			}
		}
		baseHandler := handlers[name]
		handlers[name] = wrapPSToolHandler(name, baseHandler, threadID)
	}

	return tools, handlers, bgMgr, subMgr
}

// wrapPSToolHandler wraps a PS tool handler so it waits for extension callback before returning.
func wrapPSToolHandler(name string, base rtools.Handler, threadID string) rtools.Handler {
	return func(args string) string {
		// Extract callId from args JSON
		callID := rtools.ExtractCallID(args)
		registerWrite, waitForResult := RegisterPSToolCall(callID)
		_ = registerWrite // stored in global registry for extensionToolResultHandler to write

		// Call base handler (broadcasts SSE event, returns placeholder text)
		_ = base(args)

		// Check if extension already responded (non-blocking)
		if existing, ok := GetPSResult(callID); ok {
			return formatPSResult(name, existing)
		}

		// Block and wait for extension callback
		pending := waitForResult()
		return formatPSResult(name, pending)
	}
}

func formatPSResult(name string, r psToolResult) string {
	if r.Error != "" {
		return "[PS-" + name + "] extension error: " + r.Error
	}
	if r.Result == nil {
		return "[PS-" + name + "] no result from extension"
	}
	data, _ := json.Marshal(r.Result)
	return "[PS-" + name + "] result: " + string(data)
}

// ── Agent turn ─────────────────────────────────────────────────

func runAgentTurn(threadID string, userInput string, profile agentProfileConfig) (string, error) {
	state := getThreadState(threadID)
	if err := ensureProfileSkillsLoaded(state, profile); err != nil {
		return "", err
	}

	state.mu.Lock()
	if len(state.messages) == 0 {
		devMsg := profile.DeveloperMessage
		if strings.TrimSpace(devMsg) == "" {
			devMsg = developerMessage
		}
		state.messages = []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(devMsg, responses.EasyInputMessageRoleDeveloper),
		}
	}
	state.messages = append(state.messages, responses.ResponseInputItemParamOfMessage(userInput, responses.EasyInputMessageRoleUser))
	messages := make([]responses.ResponseInputItemUnionParam, len(state.messages))
	copy(messages, state.messages)
	state.mu.Unlock()

	// Step model routing: look up model for current step type, fall back to cfg.Model
	state.mu.RLock()
	stepType := state.stepType
	state.mu.RUnlock()
	model := cfg.GetModelForStep(stepType)

	// S2 SSE lifecycle: emit run lifecycle events aligned with frontend expectations
	runID := threadID + "-run-" + fmt.Sprintf("%d", time.Now().Unix())
	broadcastSSE(threadID, "run.started", map[string]interface{}{
		"run_id":    runID, // snake_case for stores/sse.ts
		"runId":     runID, // camelCase for sse.ts
		"thread_id": threadID,
		"step_type": stepType,
		"model":     model,
	})
	if stepType != "" {
		broadcastSSE(threadID, "agent.step", map[string]string{"type": stepType, "model": model})
	}

	broadcastSSE(threadID, "run.planning", map[string]interface{}{
		"run_id": threadID,
		"status": "planning",
		"model":  model,
	})
	broadcastSSE(threadID, "agent.thinking", map[string]string{"status": "processing", "model": model})

	tools, handlers, bgMgr, subMgr := buildToolsAndHandlers(threadID, cfg, skillRegistry, profile.AllowedTools)
	answer, turnItems, err := runtime.RunToolLoop(context.Background(), llm, model, tools, handlers, state.todo, messages, bgMgr, subMgr, state.skillState, skillRegistry, sseToolLoopHooks(threadID))
	if err != nil {
		broadcastSSE(threadID, "run.failed", map[string]interface{}{
			"run_id": threadID, "runId": threadID,
			"error": err.Error(),
		})
		return "", err
	}
	broadcastSSE(threadID, "run.completed", map[string]interface{}{
		"run_id": threadID, "runId": threadID, "summary": "Done.",
	})

	// Self-reflection: analyze this turn and execute automatable improvements.
	// Results are broadcast via SSE so the frontend can show them.
	if refl := runtime.RunSelfReflectionIfWorthy(context.Background(), llm, model, turnItems, answer); refl != "" {
		broadcastSSE(threadID, "agent.self_reflection", map[string]string{
			"summary": refl,
		})
	}

	state.mu.Lock()
	state.messages = append(state.messages, responses.ResponseInputItemParamOfMessage(answer, responses.EasyInputMessageRoleAssistant))
	state.mu.Unlock()

	go saveSession(threadID, state)
	return answer, nil
}

func saveSession(threadID string, state *threadState) {
	state.mu.RLock()
	messages := make([]responses.ResponseInputItemUnionParam, len(state.messages))
	copy(messages, state.messages)
	state.mu.RUnlock()

	store := sessions.NewStore(fmt.Sprintf(".sessions/%s", threadID))
	sessionID, _ := store.Save(state.activeSession, messages, state.todo, state.skillState)
	if sessionID != "" && state.activeSession == "" {
		state.mu.Lock()
		state.activeSession = sessionID
		state.mu.Unlock()
	}
}

// ── HTTP Handlers ─────────────────────────────────────────────

type chatRequest struct {
	ThreadID       string `json:"threadId"`
	Input          string `json:"input"`
	WorkspaceRoot  string `json:"workspaceRoot"`
	WorkspaceRoot2 string `json:"workspace_root"`
	WorkRootDir    string `json:"workRootDir"`
	WorkRootDir2   string `json:"work_root_dir"`
	AgentProfile   string `json:"agent_profile"`
	AgentProfile2  string `json:"agentProfile"`
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}
	if req.ThreadID == "" {
		req.ThreadID = "default"
	}
	if req.Input == "" {
		http.Error(w, "input is required", http.StatusBadRequest)
		return
	}
	wsCandidate := firstNonEmpty(
		req.WorkspaceRoot,
		req.WorkspaceRoot2,
		req.WorkRootDir,
		req.WorkRootDir2,
	)
	if strings.TrimSpace(wsCandidate) == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}
	if _, err := effectiveWorkspaceRoot(wsCandidate); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	broadcastSSE(req.ThreadID, "message.delta", map[string]interface{}{
		"role": "user", "delta": req.Input,
	})

	profileName := strings.TrimSpace(strings.ToLower(firstNonEmpty(req.AgentProfile, req.AgentProfile2)))
	runTask := func() {
		if profileName == "ppt-generator" {
			if err := ensurePPTAssetsCopied(cfg.WorkspaceDir); err != nil {
				log.Printf("[ppt] ensure assets failed: %v", err)
			}
		}
		profile := resolveAgentProfile(profileName)
		answer, err := runAgentTurn(req.ThreadID, req.Input, profile)
		if err != nil {
			broadcastSSE(req.ThreadID, "error", map[string]interface{}{"error": err.Error()})
			return
		}
		// is_final / run.completed 由 runtime.RunToolLoop + sseToolLoopHooks 发送。
		_ = answer // 已由 runToolLoop 的 SSE 事件消耗
	}

	if profileName == "ppt-generator" {
		broadcastSSE(req.ThreadID, "ppt.queue", map[string]interface{}{
			"threadId": req.ThreadID,
			"status":   "queued",
		})
		if !enqueuePPTTask(runTask) {
			http.Error(w, "ppt generation queue is full", http.StatusTooManyRequests)
			return
		}
	} else {
		go runTask()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "queued", "threadId": req.ThreadID})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func effectiveWorkspaceRoot(workspaceRoot string) (string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return cfg.WorkspaceDir, nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("invalid workspaceRoot: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("workspaceRoot unavailable: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspaceRoot is not a directory: %s", abs)
	}
	if cfg.WorkspaceDir != abs {
		cfg.WorkspaceDir = abs
		cfg.SkillsDir = filepath.Join(abs, "skills")
		if reg, err := skills.LoadWorkspaceSkillsRegistry(abs); err != nil {
			log.Printf("warning: skills registry merge failed for %s: %v", abs, err)
			skillRegistry = skills.NewRegistry()
		} else {
			skillRegistry = reg
		}
		_memLaceMgr = nil
		_memLaceMgrOnce = false
	}
	return cfg.WorkspaceDir, nil
}

type historyResponse struct {
	ThreadID string          `json:"threadId"`
	Messages []sessions.Item `json:"messages"`
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	threadID := strings.TrimPrefix(r.URL.Path, "/api/threads/")
	if threadID == "" {
		http.Error(w, "missing threadId", http.StatusBadRequest)
		return
	}

	state := getThreadState(threadID)
	state.mu.RLock()
	defer state.mu.RUnlock()

	items := sessions.EncodeMessages(state.messages)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(historyResponse{ThreadID: threadID, Messages: items})
}

func skillsHandler(w http.ResponseWriter, r *http.Request) {
	if skillRegistry == nil {
		json.NewEncoder(w).Encode(map[string]any{"skills": []any{}, "count": 0})
		return
	}
	defs := skillRegistry.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"skills": defs, "count": len(defs)})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	_ = r
	count := 0
	if skillRegistry != nil {
		count = skillRegistry.Count()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"port":          33338,
		"model":         cfg.Model,
		"step_models":   cfg.StepModels,
		"skills_count":  count,
		"workspace_dir": cfg.WorkspaceDir,
		"skills_dir":    cfg.SkillsDir,
	})
}

// stepHandler: GET /api/step?thread=xxx → returns current step type
//
//	POST /api/step {threadId, stepType} → sets step type
func stepHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	threadID := r.URL.Query().Get("thread")
	if threadID == "" {
		if r.Method == http.MethodPost {
			var req struct {
				ThreadID string `json:"threadId"`
				StepType string `json:"stepType"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			threadID = req.ThreadID
			if req.StepType != "" {
				SetStepType(req.ThreadID, req.StepType)
				broadcastSSE(req.ThreadID, "agent.step", map[string]string{"type": req.StepType, "model": cfg.GetModelForStep(req.StepType)})
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "stepType": req.StepType})
			return
		}
		http.Error(w, "thread query param required", http.StatusBadRequest)
		return
	}
	state := getThreadState(threadID)
	state.mu.RLock()
	st := state.stepType
	state.mu.RUnlock()
	model := cfg.GetModelForStep(st)
	json.NewEncoder(w).Encode(map[string]interface{}{"threadId": threadID, "stepType": st, "model": model})
}

// SetStepType updates the current step type for a thread.
// Called by Vibex tool handlers to self-report their step type.
func SetStepType(threadID, stepType string) {
	state := getThreadState(threadID)
	state.mu.Lock()
	state.stepType = stepType
	state.mu.Unlock()
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// Package-level vars set by main.go.
var (
	cfg           common.Config
	llm           adapters.LLMClient
	skillRegistry *skills.Registry
)

// ── SpecPilot HTTP Handlers (for StatusBar button) ─────────────────

// specpilotStatusHandler: GET /api/specpilot/status
func specpilotStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsRoot := cfg.WorkspaceDir
	if wsRoot == "" {
		http.Error(w, `{"error": "no workspace set"}`, http.StatusBadRequest)
		return
	}
	dp := rtools.SpecpilotDCPort()
	mp := rtools.SpecpilotMFPort()
	installed, dcRunning, mfRunning := rtools.SpecpilotServiceStatus(wsRoot)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"installed":  installed,
		"dcRunning": dcRunning,
		"mfRunning": mfRunning,
		"dcPort":    dp,
		"mfPort":    mp,
	})
}

// specpilotBootstrapHandler: POST /api/specpilot/bootstrap { component?: string }
// Starts DC API + MF dev server, returns URLs.
func specpilotBootstrapHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsRoot := cfg.WorkspaceDir
	if wsRoot == "" {
		http.Error(w, `{"error": "no workspace set"}`, http.StatusBadRequest)
		return
	}
	var req struct {
		Component string `json:"component"`
	}
	if r.Method == http.MethodPost {
		json.NewDecoder(r.Body).Decode(&req)
	}
	component := req.Component
	if component == "" {
		component = "Dashboard"
	}
	dp := rtools.SpecpilotDCPort()
	mp := rtools.SpecpilotMFPort()
	result := rtools.SpecpilotBootstrap(wsRoot, component, dp, mp)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// specpilotDCHandler: GET /api/specpilot/dc → read .specpilot/dc_state.json
func specpilotDCHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsRoot := cfg.WorkspaceDir
	path := filepath.Join(wsRoot, ".specpilot", "dc_state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "total": 0})
		return
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "total": 0})
		return
	}
	entries, _ := raw["data"].(map[string]any)
	if entries == nil {
		entries = map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": entries, "total": len(entries)})
}

// specpilotDCSetHandler: POST /api/specpilot/dc/set { key, value }
func specpilotDCSetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsRoot := cfg.WorkspaceDir
	path := filepath.Join(wsRoot, ".specpilot", "dc_state.json")
	var req struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "missing key"})
		return
	}
	var raw map[string]any
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &raw)
	}
	if raw == nil {
		raw = make(map[string]any)
	}
	entries, _ := raw["data"].(map[string]any)
	if entries == nil {
		entries = make(map[string]any)
	}
	entries[req.Key] = req.Value
	raw["data"] = entries
	raw["total"] = len(entries)
	if out, err := json.Marshal(raw); err == nil {
		os.WriteFile(path, out, 0644)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "key": req.Key})
}

// specpilotECHistoryHandler: GET /api/specpilot/ec/history?limit=20
func specpilotECHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsRoot := cfg.WorkspaceDir
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	path := filepath.Join(wsRoot, ".specpilot", "ec_state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}
	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		records = []map[string]any{}
	}
	if len(records) > limit {
		records = records[len(records)-limit:]
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(records)
}

// specpilotMFComponentsHandler: GET /api/specpilot/mf/components
func specpilotMFComponentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsRoot := cfg.WorkspaceDir
	path := filepath.Join(wsRoot, ".specpilot", "mf_registry.json")
	data, err := os.ReadFile(path)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"components": []any{}, "total": 0})
		return
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "components": []any{}, "total": 0})
		return
	}
	comps, _ := raw["components"].(map[string]any)
	if comps == nil {
		comps = map[string]any{}
	}
	var items []map[string]any
	for name, info := range comps {
		if m, ok := info.(map[string]any); ok {
			items = append(items, map[string]any{
				"name":     name,
				"path":     m["path"],
				"status":   m["registered"],
				"contract": m["contract"],
			})
		}
	}
	if items == nil {
		items = []map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"components": items, "total": len(items)})
}

// ── MemLace Clarification API ─────────────────────────────────

// memLaceMgr is a singleton, initialized on first use.
var (
	_memLaceMgr     *memlace.SessionManager
	_memLaceMgrOnce bool
)

func getMemLaceMgr() *memlace.SessionManager {
	if _memLaceMgrOnce {
		return _memLaceMgr
	}
	_memLaceMgrOnce = true
	mCfg := memlace.DefaultConfig(cfg.WorkspaceDir)
	mgr, err := memlace.NewSessionManager(mCfg)
	if err != nil {
		return nil
	}
	_memLaceMgr = mgr
	return mgr
}

// clarificationsHandler: GET /api/clarifications → list all sessions
func clarificationsHandler(w http.ResponseWriter, r *http.Request) {
	mgr := getMemLaceMgr()
	if mgr == nil {
		http.Error(w, "memlace unavailable", http.StatusServiceUnavailable)
		return
	}
	sessions := mgr.ListSessions()
	type SessionSummary struct {
		ID           string `json:"id"`
		SpecName     string `json:"spec_name"`
		SpecParent   string `json:"spec_parent"`
		Phase        string `json:"phase"`
		Status       string `json:"status"`
		Rounds       int    `json:"rounds"`
		CurrentRound int    `json:"current_round"`
		HasDraft     bool   `json:"has_draft"`
		ConfirmedAt  string `json:"confirmed_at,omitempty"`
		UpdatedAt    string `json:"updated_at"`
	}
	var out []SessionSummary
	for _, s := range sessions {
		confirmedAt := ""
		if s.ConfirmedAt != nil {
			confirmedAt = s.ConfirmedAt.Format(time.RFC3339)
		}
		out = append(out, SessionSummary{
			ID: s.ID, SpecName: s.SpecName, SpecParent: s.SpecParent,
			Phase: string(s.Phase), Status: string(s.Status),
			Rounds: len(s.Rounds), CurrentRound: s.CurrentRound,
			HasDraft:    s.DerivedSpecDraft != "",
			ConfirmedAt: confirmedAt,
			UpdatedAt:   s.UpdatedAt.Format(time.RFC3339),
		})
	}
	if out == nil {
		out = []SessionSummary{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"count": len(out), "sessions": out})
}

// clarificationHandler: GET /api/clarifications/:specName → get session detail
// POST /api/clarifications/:specName {action, ...} → perform action
func clarificationHandler(w http.ResponseWriter, r *http.Request) {
	// Extract specName from path: /api/clarifications/:specName
	parts := strings.TrimPrefix(r.URL.Path, "/api/clarifications/")
	specName := strings.TrimSuffix(parts, "/")
	if specName == "" || specName == "/" {
		http.Error(w, "specName required", http.StatusBadRequest)
		return
	}
	mgr := getMemLaceMgr()
	if mgr == nil {
		http.Error(w, "memlace unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		session := mgr.GetSession(specName)
		if session == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		// Render as YAML for the spec draft if confirmed
		yamlContent := ""
		if session.Status == memlace.StatusConfirmed && session.DerivedSpecDraft != "" {
			yamlContent = session.DerivedSpecDraft
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            session.ID,
			"spec_name":     session.SpecName,
			"spec_parent":   session.SpecParent,
			"phase":         string(session.Phase),
			"status":        string(session.Status),
			"rounds":        session.Rounds,
			"current_round": session.CurrentRound,
			"draft":         session.DerivedSpecDraft,
			"yaml_content":  yamlContent,
			"confirmed_at":  session.ConfirmedAt,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Action   string `json:"action"` // "confirm"|"reject"|"start"|"qa"|"draft"
			Phase    string `json:"phase"`
			Draft    string `json:"draft"`
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		switch req.Action {
		case "start":
			session := mgr.CreateSession(specName, "", memlace.ClarificationPhase(req.Phase))
			questions := memlace.PhaseDefaultQuestions(memlace.ClarificationPhase(req.Phase))
			json.NewEncoder(w).Encode(map[string]interface{}{
				"session_id": session.ID, "status": session.Status,
				"phase": req.Phase, "questions": questions,
			})
		case "qa":
			if req.Question == "" || req.Answer == "" {
				http.Error(w, "question and answer required for qa action", http.StatusBadRequest)
				return
			}
			session, err := mgr.AddRound(specName, req.Question, req.Answer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true, "round": session.CurrentRound, "rounds": session.Rounds,
				"draft": session.DerivedSpecDraft, "status": session.Status,
			})
		case "draft":
			if req.Draft == "" {
				http.Error(w, "draft content required", http.StatusBadRequest)
				return
			}
			if err := mgr.SetDraft(specName, req.Draft); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"ok": "true", "status": "draft_saved"})
		case "confirm":
			session := mgr.GetSession(specName)
			if session == nil {
				http.Error(w, "session not found", http.StatusNotFound)
				return
			}
			// 如果 body 带了 draft 内容，先更新 draft
			if req.Draft != "" {
				if err := mgr.SetDraft(specName, req.Draft); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if session.DerivedSpecDraft == "" {
				http.Error(w, "no draft to confirm", http.StatusBadRequest)
				return
			}
			if err := mgr.Confirm(specName); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"ok": "true", "status": "confirmed"})
		case "reject":
			if err := mgr.Reject(specName); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"ok": "true", "status": "rejected"})
		default:
			http.Error(w, "unknown action: "+req.Action, http.StatusBadRequest)
		}
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// workspaceSpecHandler: GET /api/workspace/specs/read?path=<relative>
// Reads a spec file from the workspace and returns { path, content }.
func workspaceSpecHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "path query param required", http.StatusBadRequest)
		return
	}
	// Disallow path traversal: ensure the resolved path is inside WorkspaceDir
	absPath := filepath.Join(cfg.WorkspaceDir, filepath.Clean(relPath))
	if !strings.HasPrefix(absPath, cfg.WorkspaceDir) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"path": relPath, "content": string(data)})
}
