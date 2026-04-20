// cmd/web/server.go — HTTP handlers, tool loop, and tool building.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"vibex/agent/adapters"
	"vibex/agent/agents/background"
	"vibex/agent/agents/compact"
	"vibex/agent/agents/sessions"
	rtools "vibex/agent/agents/runtime/tools"
	"vibex/agent/agents/skills"
	"vibex/agent/agents/subagent"
	"vibex/agent/internal/common"
	"vibex/agent/vibex/domain"

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
func runToolLoop(
	threadID string,
	llm adapters.LLMClient,
	model string,
	tools []responses.ToolUnionParam,
	handlers map[string]rtools.Handler,
	messages []responses.ResponseInputItemUnionParam,
	skillState *skills.State,
	skillRegistry *skills.Registry,
) (string, error) {
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
			return "", err
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
			broadcastSSE(threadID, "tool.called", map[string]interface{}{
				"tool": item.OfFunctionCall.Name, "call_id": item.OfFunctionCall.CallID, "args": args,
			})

			h, ok := handlers[item.OfFunctionCall.Name]
			if !ok {
				followUp = append(followUp, responses.ResponseInputItemParamOfFunctionCallOutput(item.OfFunctionCall.CallID, "unsupported tool"))
				continue
			}
			result := h(item.OfFunctionCall.Arguments)
			followUp = append(followUp, responses.ResponseInputItemParamOfFunctionCallOutput(item.OfFunctionCall.CallID, result))
			broadcastSSE(threadID, "tool.completed", map[string]interface{}{
				"tool": item.OfFunctionCall.Name, "call_id": item.OfFunctionCall.CallID, "result": result,
			})
		}

		if !hasCalls {
			broadcastSSE(threadID, "run.completed", map[string]string{"summary": "Done."})
			return strings.TrimSpace(text), nil
		}
		inputItems = append(inputItems, followUp...)
	}
	return "", fmt.Errorf("tool loop exceeded 20 steps")
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
		return runToolLoop(threadID, llm, cfg.SubAgentModel, childTools, childHandlers, childMsgs, childSkills, skillRegistry)
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
	if stepType != "" {
		broadcastSSE(threadID, "agent.step", map[string]string{"type": stepType, "model": model})
	}

	broadcastSSE(threadID, "agent.thinking", map[string]string{"status": "processing", "model": model})

	tools, handlers := buildToolsAndHandlers(threadID, cfg, skillRegistry)
	answer, err := runToolLoop(threadID, llm, model, tools, handlers, messages, state.skillState, skillRegistry)
	if err != nil {
		return "", err
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
		broadcastSSE(req.ThreadID, "message.delta", map[string]interface{}{
			"role": "assistant", "delta": answer, "is_final": true,
		})
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
