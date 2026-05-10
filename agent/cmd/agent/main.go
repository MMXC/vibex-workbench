package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"vibex/agent/adapters"
	"vibex/agent/agents/background"
	"vibex/agent/agents/runtime"
	rtools "vibex/agent/agents/runtime/tools"
	"vibex/agent/agents/skills"
	"vibex/agent/agents/subagent"
	"vibex/agent/internal/common"
	"vibex/agent/vibex/domain"

	"github.com/openai/openai-go/v3/responses"
)

func main() {
	// 检测 --goal flag：单次 CLI 模式
	goalFlag := flag.String("goal", "", "run single goal and exit")
	workspaceFlag := flag.String("workspace", "", "workspace root path")
	flag.Parse()

	if *goalFlag != "" {
		// ── CLI 单次模式 ──────────────────────────────────────────
		workspace := *workspaceFlag
		if workspace != "" {
			if err := os.Chdir(workspace); err != nil {
				fmt.Fprintf(os.Stderr, `{"type":"error","data":{"message":"chdir failed: %v"}}`+"\n", err)
				os.Exit(1)
			}
		}
		ctx := context.Background()
		if err := RunGoal(ctx, *goalFlag); err != nil {
			fmt.Fprintf(os.Stderr, `{"type":"error","data":{"message":`+jsonMarshal(err.Error())+`}}`+"\n")
			os.Exit(1)
		}
		return
	}

	// ── 原有交互模式 ───────────────────────────────────────────
	if err := runtime.RunInteractive(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// jsonMarshal 简单转义 JSON 字符串值
func jsonMarshal(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// emit 向 stdout 打印 JSON event line
func emit(eventType string, data interface{}) {
	line := struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}{Type: eventType, Data: data}
	b, _ := json.Marshal(line)
	fmt.Println(string(b))
}

func emitBroadcast(threadID, event string, data interface{}) {
	emit("domain_event", map[string]interface{}{
		"thread_id": threadID,
		"event":     event,
		"data":      data,
	})
}

func noopSetStepType(threadID, stepType string) {}

// RunGoal 执行单个 goal，输出 JSON event lines 到 stdout
func RunGoal(ctx context.Context, goal string) error {
	cfg := common.LoadConfig()
	if cfg.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is empty")
	}

	rawClient := common.NewClient(cfg)
	llm := adapters.NewLLMClient(rawClient, cfg.BaseURL, cfg.Model)

	skillRegistry, err := skills.LoadWorkspaceSkillsRegistry(cfg.WorkspaceDir)
	if err != nil {
		fmt.Printf("warning: failed to merge skills for workspace %s: %v\n", cfg.WorkspaceDir, err)
		skillRegistry = skills.NewRegistry()
	}
	parentSkills := skills.NewState()
	todo := rtools.NewTodoStore()
	backgroundMgr := background.NewManager()
	subAgentMgr := subagent.NewManager(4)
	subAgentRunner := func(ctx context.Context, taskSummary string) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		childTodo := rtools.NewTodoStore()
		childSkills := skills.NewState()
		childSkills.SetActive(parentSkills.ActiveNames())
		childSpecs := rtools.ChildSpecs(childTodo)
		childVibexReg := domain.NewRegistry(cfg.WorkspaceDir, emitBroadcast, noopSetStepType)
		childSpecs = append(childSpecs, childVibexReg.ToolSpecs()...)
		childTools := rtools.BuildTools(childSpecs)
		childHandlers := rtools.BuildHandlers(childSpecs)
		for name, handler := range childVibexReg.ToolHandlers() {
			childHandlers[name] = handler
		}
		childMessages := []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage("You are a coding agent.", responses.EasyInputMessageRoleDeveloper),
			responses.ResponseInputItemParamOfMessage("Sub-agent task summary:\n"+strings.TrimSpace(taskSummary), responses.EasyInputMessageRoleUser),
		}
		answer, _, err := runtime.RunToolLoop(ctx, llm, cfg.SubAgentModel, childTools, childHandlers, childTodo, childMessages, nil, nil, childSkills, skillRegistry, nil)
		if err != nil {
			return "", err
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return answer, nil
	}

	specs := rtools.ParentSpecs(todo, backgroundMgr, subAgentMgr, subAgentRunner, parentSkills, skillRegistry)
	vibexReg := domain.NewRegistry(cfg.WorkspaceDir, emitBroadcast, noopSetStepType)
	specs = append(specs, vibexReg.ToolSpecs()...)
	tools := rtools.BuildTools(specs)
	handlers := rtools.BuildHandlers(specs)
	for name, handler := range vibexReg.ToolHandlers() {
		handlers[name] = handler
	}

	// 包装 handlers，输出 JSON event lines
	wrappedHandlers := make(map[string]rtools.Handler)
	for name, handler := range handlers {
		h := handler
		wrappedHandlers[name] = func(args string) string {
			out := h(args)
			emit("tool_output", map[string]string{"name": name, "output": out})
			return out
		}
	}

	developerMsg := common.ResolveDeveloperMessage(
		cfg.WorkspaceDir,
		"goal-cli",
		"You are a VibeX coding agent. Use VibeX domain tools such as spec_designer, spec_write, spec_validate, make_validate, canvas_update, TDD, toolrouting, and memlace when working with specs or the workbench flow. Use generic tools like bash, read_file, write_file, and todo_set only when the domain tools do not cover the task. Use todo_set only for non-trivial multi-step tasks. If a TODO is started, keep it updated and reply directly once completed.",
	)
	messages := []responses.ResponseInputItemUnionParam{
		responses.ResponseInputItemParamOfMessage(developerMsg, responses.EasyInputMessageRoleDeveloper),
		responses.ResponseInputItemParamOfMessage(goal, responses.EasyInputMessageRoleUser),
	}

	fmt.Printf("Agent started (model=%s subagent=%s workspace=%s)\n", cfg.Model, cfg.SubAgentModel, cfg.WorkspaceDir)

	answer, _, err := runtime.RunToolLoop(ctx, llm, cfg.Model, tools, wrappedHandlers, todo, messages, backgroundMgr, subAgentMgr, parentSkills, skillRegistry, nil)
	if err != nil {
		emit("error", map[string]string{"message": err.Error()})
		return err
	}

	emit("done", map[string]string{"answer": answer})
	return nil
}
