// cmd/web/main.go — VibeX Agent Web Server entry point.
// Bridges the nanoClaudeCode runtime with SSE for the Canvas workbench.
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"vibex/agent/adapters"
	"vibex/agent/agents/skills"
	"vibex/agent/internal/common"
)

func main() {
	cfg = common.LoadConfig()
	developerMessage = common.ResolveDeveloperMessage(cfg.WorkspaceDir, "web", developerMessage)
	if cfg.APIKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}
	if cfg.WorkspaceDir != "" {
		_ = os.Setenv("WORKSPACE_DIR", cfg.WorkspaceDir)
	}

	rawClient := common.NewClient(cfg)
	llm = adapters.NewLLMClient(rawClient, cfg.BaseURL, cfg.Model)

	var err error
	skillRegistry, err = skills.LoadWorkspaceSkillsRegistry(cfg.WorkspaceDir)
	if err != nil {
		log.Printf("warning: skills registry merge failed for %s: %v", cfg.WorkspaceDir, err)
		skillRegistry = skills.NewRegistry()
	}
	log.Printf("[VibeX Agent] adapter=%s | model=%s | workspace=%s | skills=%d (merged skills/ + .agents/skills)",
		llm.AdapterName(), cfg.Model, cfg.WorkspaceDir, skillRegistry.Count())

	os.MkdirAll(".sessions", 0755)

	// CORS：前端 dev（5173）调用 33338 时需预检 OPTIONS
	withCORS := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", "*")
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			// EventSource / fetch 跨域预检常见会带 Cache-Control / Accept 等；过窄会导致预检失败，
			// 浏览器报错后实际拿到的是阻断页或非 SSE MIME，表现为 text/html。
			h.Set("Access-Control-Allow-Headers",
				"Accept, Authorization, Cache-Control, Content-Type, Pragma, X-Requested-With")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}

	http.HandleFunc("/health", withCORS(healthHandler))
	http.HandleFunc("/api/chat", withCORS(chatHandler))
	http.HandleFunc("/api/agent/execute", withCORS(chatHandler))
	http.HandleFunc("/api/sse/", withCORS(sseHandler))
	http.HandleFunc("/api/threads/", withCORS(historyHandler))
	http.HandleFunc("/api/skills", withCORS(skillsHandler))
	http.HandleFunc("/api/step", withCORS(stepHandler))
	http.HandleFunc("/api/clarifications", withCORS(clarificationsHandler))
	http.HandleFunc("/api/clarifications/", withCORS(clarificationHandler))
	http.HandleFunc("/api/workspace/specs/read", withCORS(workspaceSpecReadHandler))
	http.HandleFunc("/api/workspace/specs/write", withCORS(workspaceSpecWriteHandler))
	http.HandleFunc("/api/workspace/file/", withCORS(workspaceFileHandler))
	http.HandleFunc("/api/workspace/specs/list", withCORS(workspaceSpecsListHandler))
	http.HandleFunc("/api/workspace/specs/init-layout", withCORS(workspaceSpecsInitLayoutHandler))
	http.HandleFunc("/api/workspace/tree", withCORS(workspaceTreeHandler))
	http.HandleFunc("/api/workspace/read-file", withCORS(workspaceReadFileHandler))
	http.HandleFunc("/api/workspace/git/status", withCORS(workspaceGitStatusHandler))
	http.HandleFunc("/api/workspace/git/commit-spec", withCORS(workspaceGitCommitBySpecHandler))
	http.HandleFunc("/api/workspace/specs/convention", withCORS(workspaceSpecsConventionHandler))
	http.HandleFunc("/api/workspace/detect-state", withCORS(workspaceDetectStateHandler))
	http.HandleFunc("/api/workspace/scaffold", withCORS(workspaceScaffoldHandler))
	http.HandleFunc("/api/workspace/spec-bootstrap", withCORS(workspaceSpecsBootstrapHandler))
	http.HandleFunc("/api/workspace/run-make", withCORS(workspaceRunMakeHandler))
	http.HandleFunc("/api/workspace/verify-specs", withCORS(workspaceVerifySpecsHandler))
	http.HandleFunc("/api/workspace/agent-commands", withCORS(workspaceAgentCommandsHandler))
	http.HandleFunc("/api/workspace/qa/run", withCORS(qaRunnerHandler))
	http.HandleFunc("/api/workspace/agent-flow-qa/run", withCORS(agentFlowQARunHandler))
	http.HandleFunc("/api/workspace/cdp/validate", withCORS(cdpValidateHandler))
	http.HandleFunc("/api/workspace/governance/status", withCORS(governanceStatusHandler))
	http.HandleFunc("/api/agent/tools", withCORS(agentToolsHandler))
	http.HandleFunc("/api/agent/tools/custom", withCORS(agentCustomToolHandler))
	http.HandleFunc("/api/agent/plan-graph", withCORS(agentPlanGraphHandler))
	http.HandleFunc("/api/agent/tool-route", withCORS(agentToolRouteHandler))
	http.HandleFunc("/api/workbench/inspector/ws", inspectorWebSocketHandler)
	http.HandleFunc("/api/workbench/inspector/snapshot", withCORS(inspectorSnapshotHandler))
	http.HandleFunc("/api/workbench/inspector/trace", withCORS(inspectorTraceHandler))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-quit; log.Println("[VibeX Agent] shutdown"); os.Exit(0) }()

	log.Println("[VibeX Agent] listening on :33338")
	log.Println("  SSE:       GET  http://localhost:33338/api/sse/<threadId>")
	log.Println("  Chat:      POST http://localhost:33338/api/chat")
	log.Println("  Inspector: WS   ws://localhost:33338/api/workbench/inspector/ws")
	log.Println("  Inspector: GET  http://localhost:33338/api/workbench/inspector/snapshot")
	log.Fatal(http.ListenAndServe(":33338", nil))
}
