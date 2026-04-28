// cmd/web/server.go — HTTP handlers, tool loop, and tool building.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vibex/agent/adapters"
	"vibex/agent/agents/background"
	"vibex/agent/agents/compact"
	"vibex/agent/agents/sessions"
	"vibex/agent/agents/runtime"
	rtools "vibex/agent/agents/runtime/tools"
	"vibex/agent/agents/skills"
	"vibex/agent/agents/subagent"
	"vibex/agent/internal/common"
	"vibex/agent/vibex/domain"
	vibexspec "vibex/agent/vibex/domain/spec"
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
  1. Use spec_validate to check the target spec YAML
  2. Use tdd_design to design test cases (RED phase)
  3. Use tdd_run to verify tests fail (RED)
  4. Implement the feature
  5. Use tdd_run to verify tests pass (GREEN)
  6. Use tdd_iterate for next behavior step
  7. Use spec_sync push after changes
  8. Use make_validate to verify all specs

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
- Never assume — always clarify ambiguous requirements
- After any code/spec change, always run make_validate`

// ── Tool Loop ─────────────────────────────────────────────────

// runToolLoop executes a tool-use turn via the LLMClient interface.
// The adapter handles all API-level differences (Responses vs Chat Completions).
// Returns (answer, turnItems, error). turnItems includes all tool calls/outputs from this turn.
func runToolLoop(
	threadID string,
	llm adapters.LLMClient,
	model string,
	tools []responses.ToolUnionParam,
	handlers map[string]rtools.Handler,
	messages []responses.ResponseInputItemUnionParam,
	skillState *skills.State,
	skillRegistry *skills.Registry,
) (string, []responses.ResponseInputItemUnionParam, error) {
	inputItems := append([]responses.ResponseInputItemUnionParam{}, messages...)
	bgMgr := background.NewManager()

	for step := 0; step < 20; step++ {
		if compacted, _ := compact.MicroCompact(inputItems, compact.DefaultKeepRecentToolResults); compacted != nil {
			inputItems = compacted
		}
		if compact.NeedsAutoCompact(inputItems, compact.DefaultAutoCompactCharLimit) {
			summary, _ := summarizeForAutoCompact(llm, model, inputItems)
			if summary != "" {
				inputItems = compact.AutoCompact(inputItems, summary, compact.DefaultAutoCompactKeepRecentK)
			}
		}

	reqInput := append([]responses.ResponseInputItemUnionParam{}, inputItems...)
	if notes := strings.TrimSpace(rtools.FormatBackgroundNotifications(bgMgr.DrainNotifications())); notes != "" {
		reqInput = append(reqInput, responses.ResponseInputItemParamOfMessage(notes, responses.EasyInputMessageRoleDeveloper))
	}
	// Inject workspace awareness at the start of every turn
	if wsRoot := strings.TrimSpace(cfg.WorkspaceDir); wsRoot != "" {
		wsAwareness := vibexspec.WorkspaceAwarenessContext(wsRoot)
		if wsAwareness != "" {
			reqInput = append(reqInput, responses.ResponseInputItemParamOfMessage(
				"## Workspace Awareness (auto-injected, do not skip)\n"+wsAwareness,
				responses.EasyInputMessageRoleDeveloper))
		}
	}
	reqInput = append(reqInput, responses.ResponseInputItemParamOfMessage(
		"Use todo_set to track progress. Use skill_load to activate skills.", responses.EasyInputMessageRoleDeveloper))
		if skillRegistry != nil {
			reqInput = append(reqInput, responses.ResponseInputItemParamOfMessage(skillRegistry.NamesContextMessage(), responses.EasyInputMessageRoleDeveloper))
			if ctx := strings.TrimSpace(skillState.ContextMessage(skillRegistry)); ctx != "" {
				reqInput = append(reqInput, responses.ResponseInputItemParamOfMessage(ctx, responses.EasyInputMessageRoleDeveloper))
			}
		}

		ctx := context.Background()
		text, toolCalls, err := llm.Chat(ctx, model, tools, reqInput)
		if err != nil {
			return "", inputItems, err
		}

		if text != "" {
			broadcastSSE(threadID, "message.delta", map[string]interface{}{
				"role": "assistant", "delta": text, "is_final": false,
			})
		}

		followUp := make([]responses.ResponseInputItemUnionParam, 0, len(toolCalls)*2)
		hasCalls := false
		for _, item := range toolCalls {
			if item.OfFunctionCall == nil {
				continue
			}
			hasCalls = true
			// 回放 function_call，保持 call_id 匹配
			followUp = append(followUp, item)

			var args map[string]any
			json.Unmarshal([]byte(item.OfFunctionCall.Arguments), &args)
			callID := item.OfFunctionCall.CallID
			broadcastSSE(threadID, "tool.called", map[string]interface{}{
				"toolName":     item.OfFunctionCall.Name, // camelCase for sse.ts
				"tool":         item.OfFunctionCall.Name, // snake_case for stores/sse.ts
				"invocationId": callID,                  // camelCase for sse.ts
				"call_id":      callID,                   // snake_case for stores/sse.ts
				"runId":        threadID,                 // camelCase: parent run
				"args":         args,
			})

			h, ok := handlers[item.OfFunctionCall.Name]
			if !ok {
				followUp = append(followUp, responses.ResponseInputItemParamOfFunctionCallOutput(item.OfFunctionCall.CallID, "unsupported tool"))
				continue
			}
			result := h(item.OfFunctionCall.Arguments)
			followUp = append(followUp, responses.ResponseInputItemParamOfFunctionCallOutput(callID, result))
			broadcastSSE(threadID, "tool.completed", map[string]interface{}{
				"toolName":     item.OfFunctionCall.Name,
				"tool":         item.OfFunctionCall.Name,
				"invocationId": callID,
				"call_id":      callID,
				"result":       result,
			})
		}

		if !hasCalls {
			// 发送最终消息：is_final=true，触发前端合并气泡并完成
			if text != "" {
				broadcastSSE(threadID, "message.delta", map[string]interface{}{
					"role": "assistant", "delta": strings.TrimSpace(text), "is_final": true,
				})
			}
			broadcastSSE(threadID, "run.completed", map[string]interface{}{
				"run_id": threadID, "runId": threadID, "summary": "Done."})
			return strings.TrimSpace(text), inputItems, nil
		}
		inputItems = append(inputItems, followUp...)
	}
	return "", inputItems, fmt.Errorf("tool loop exceeded 20 steps")
}

func summarizeForAutoCompact(llm adapters.LLMClient, model string, items []responses.ResponseInputItemUnionParam) (string, error) {
	input := append([]responses.ResponseInputItemUnionParam{}, items...)
	input = append(input, responses.ResponseInputItemParamOfMessage(
		"Summarize: key decisions, progress, TODO state, active skills, unresolved issues.",
		responses.EasyInputMessageRoleDeveloper))
	ctx := context.Background()
	return llm.SimpleChat(ctx, model, input)
}

// ── Build tools & handlers ──────────────────────────────────────

func buildToolsAndHandlers(threadID string, cfg common.Config,
	skillRegistry *skills.Registry) ([]responses.ToolUnionParam, map[string]rtools.Handler) {

	state := getThreadState(threadID)
	bgMgr := background.NewManager()
	subMgr := subagent.NewManager(4)

	runner := func(ctx context.Context, taskSummary string) (string, error) {
		childTodo := rtools.NewTodoStore()
		childSkills := skills.NewState()
		childSkills.SetActive(state.skillState.ActiveNames())
		childBg := background.NewManager()
		childSpecs := rtools.ParentSpecs(childTodo, childBg, nil, nil, childSkills, skillRegistry)
		childTools := rtools.BuildTools(childSpecs)
		childHandlers := rtools.BuildHandlers(childSpecs)
		childMsgs := []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(developerMessage, responses.EasyInputMessageRoleDeveloper),
			responses.ResponseInputItemParamOfMessage("Sub-agent task:\n"+strings.TrimSpace(taskSummary), responses.EasyInputMessageRoleUser),
		}
		answer, _, err := runToolLoop(threadID, llm, cfg.SubAgentModel, childTools, childHandlers, childMsgs, childSkills, skillRegistry)
		return answer, err
	}

	// Parent (base nanoClaudeCode) specs
	specs := rtools.ParentSpecs(state.todo, bgMgr, subMgr, runner, state.skillState, skillRegistry)

	// Vibex domain specs (spec tools + TDD tools)
	vibexReg := domain.NewRegistry(cfg.WorkspaceDir, broadcastSSE, SetStepType)
	vibexSpecs := vibexReg.ToolSpecs()
	specs = append(specs, vibexSpecs...)

	tools := rtools.BuildTools(specs)
	handlers := rtools.BuildHandlers(specs)

	// Merge vibex handlers (they use factory with broadcaster)
	for name, h := range vibexReg.ToolHandlers() {
		handlers[name] = h
	}

	return tools, handlers
}

// ── Agent turn ─────────────────────────────────────────────────

func runAgentTurn(threadID string, userInput string) (string, error) {
	state := getThreadState(threadID)

	state.mu.Lock()
	if len(state.messages) == 0 {
		state.messages = []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(developerMessage, responses.EasyInputMessageRoleDeveloper),
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
		"run_id":   runID,    // snake_case for stores/sse.ts
		"runId":    runID,    // camelCase for sse.ts
		"thread_id": threadID,
		"step_type": stepType,
		"model":    model,
	})
	if stepType != "" {
		broadcastSSE(threadID, "agent.step", map[string]string{"type": stepType, "model": model})
	}

	broadcastSSE(threadID, "run.planning", map[string]interface{}{
		"run_id": threadID,
		"status": "planning",
		"model": model,
	})
	broadcastSSE(threadID, "agent.thinking", map[string]string{"status": "processing", "model": model})

	tools, handlers := buildToolsAndHandlers(threadID, cfg, skillRegistry)
	answer, turnItems, err := runToolLoop(threadID, llm, model, tools, handlers, messages, state.skillState, skillRegistry)
	if err != nil {
		broadcastSSE(threadID, "run.failed", map[string]interface{}{
			"run_id": threadID, "runId": threadID,
			"error": err.Error(),
		})
		return "", err
	}

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
	ThreadID string `json:"threadId"`
	Input    string `json:"input"`
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

	broadcastSSE(req.ThreadID, "message.delta", map[string]interface{}{
		"role": "user", "delta": req.Input,
	})

	go func() {
		answer, err := runAgentTurn(req.ThreadID, req.Input)
		if err != nil {
			broadcastSSE(req.ThreadID, "error", map[string]interface{}{"error": err.Error()})
			return
		}
		// 注意：is_final=true 已由 runToolLoop 在 !hasCalls 时发送，
		// 此处不再重复 broadcast，避免前端出现两个相同内容的气泡。
		_ = answer // 已由 runToolLoop 的 SSE 事件消耗
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "queued", "threadId": req.ThreadID})
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
//               POST /api/step {threadId, stepType} → sets step type
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
		ID            string `json:"id"`
		SpecName      string `json:"spec_name"`
		SpecParent    string `json:"spec_parent"`
		Phase         string `json:"phase"`
		Status        string `json:"status"`
		Rounds        int    `json:"rounds"`
		CurrentRound  int    `json:"current_round"`
		HasDraft      bool   `json:"has_draft"`
		ConfirmedAt   string `json:"confirmed_at,omitempty"`
		UpdatedAt     string `json:"updated_at"`
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
			HasDraft: s.DerivedSpecDraft != "",
			ConfirmedAt: confirmedAt,
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
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
			"spec_parent":    session.SpecParent,
			"phase":         string(session.Phase),
			"status":        string(session.Status),
			"rounds":        session.Rounds,
			"current_round": session.CurrentRound,
			"draft":         session.DerivedSpecDraft,
			"yaml_content":   yamlContent,
			"confirmed_at":   session.ConfirmedAt,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Action  string `json:"action"` // "confirm"|"reject"|"start"|"qa"|"draft"
			Phase   string `json:"phase"`
			Draft   string `json:"draft"`
			Question string `json:"question"`
			Answer  string `json:"answer"`
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
