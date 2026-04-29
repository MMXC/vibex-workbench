package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"vibex/agent/agents/background"
	rtools "vibex/agent/agents/runtime/tools"
	"vibex/agent/agents/skills"
	"vibex/agent/agents/subagent"
	"vibex/agent/internal/common"
	"vibex/agent/adapters"
	"vibex/agent/agents/runtime"

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

// RunGoal 执行单个 goal，输出 JSON event lines 到 stdout
func RunGoal(ctx context.Context, goal string) error {
	cfg := common.LoadConfig()
	if cfg.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is empty")
	}

	rawClient := common.NewClient(cfg)
	llm := adapters.NewLLMClient(rawClient, cfg.BaseURL, cfg.Model)

	skillRegistry, err := skills.LoadRegistryFromDir(".skills")
	if err != nil {
		fmt.Printf("warning: failed to load .skills: %v\n", err)
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
		childTools := rtools.BuildTools(childSpecs)
		childHandlers := rtools.BuildHandlers(childSpecs)
		childMessages := []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage("You are a coding agent.", responses.EasyInputMessageRoleDeveloper),
			responses.ResponseInputItemParamOfMessage("Sub-agent task summary:\n"+strings.TrimSpace(taskSummary), responses.EasyInputMessageRoleUser),
		}
		answer, _, err := runtime.RunToolLoop(ctx, llm, cfg.SubAgentModel, childTools, childHandlers, childTodo, childMessages, nil, nil, childSkills, skillRegistry)
		if err != nil {
			return "", err
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return answer, nil
	}

	specs := rtools.ParentSpecs(todo, backgroundMgr, subAgentMgr, subAgentRunner, parentSkills, skillRegistry)
	tools := rtools.BuildTools(specs)
	handlers := rtools.BuildHandlers(specs)

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

	developerMsg := "You are a coding agent. Use tools `bash`, `read_file`, `write_file`, and `todo_set` when needed. You can manage skills with `skill_list`, `skill_load`, and `skill_unload`. Use `todo_set` only for non-trivial multi-step tasks (for example: code changes, file edits, debugging, or tasks requiring multiple actions). For simple single-turn Q&A, reply directly without creating TODO. If a TODO is started, keep it updated and reply directly once completed."
	messages := []responses.ResponseInputItemUnionParam{
		responses.ResponseInputItemParamOfMessage(developerMsg, responses.EasyInputMessageRoleDeveloper),
		responses.ResponseInputItemParamOfMessage(goal, responses.EasyInputMessageRoleUser),
	}

	fmt.Printf("Agent started (model=%s subagent=%s)\n", cfg.Model, cfg.SubAgentModel)

	answer, _, err := runtime.RunToolLoop(ctx, llm, cfg.Model, tools, wrappedHandlers, todo, messages, backgroundMgr, subAgentMgr, parentSkills, skillRegistry)
	if err != nil {
		emit("error", map[string]string{"message": err.Error()})
		return err
	}

	emit("done", map[string]string{"answer": answer})
	return nil
}
