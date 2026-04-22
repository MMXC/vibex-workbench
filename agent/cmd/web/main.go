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
	if cfg.APIKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	rawClient := common.NewClient(cfg)
	llm = adapters.NewLLMClient(rawClient, cfg.BaseURL, cfg.Model)

	var err error
	skillRegistry, err = skills.LoadRegistryFromDir(cfg.SkillsDir)
	if err != nil {
		log.Printf("warning: skills dir %s not found: %v", cfg.SkillsDir, err)
		skillRegistry = skills.NewRegistry()
	}
	log.Printf("[VibeX Agent] adapter=%s | model=%s | workspace=%s | skills=%d from %s",
		llm.AdapterName(), cfg.Model, cfg.WorkspaceDir, skillRegistry.Count(), cfg.SkillsDir)

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
	http.HandleFunc("/api/sse/", withCORS(sseHandler))
	http.HandleFunc("/api/threads/", withCORS(historyHandler))
	http.HandleFunc("/api/skills", withCORS(skillsHandler))
	http.HandleFunc("/api/step", withCORS(stepHandler))
	http.HandleFunc("/api/clarifications", withCORS(clarificationsHandler))
	http.HandleFunc("/api/clarifications/", withCORS(clarificationHandler))
	http.HandleFunc("/api/workspace/specs/read", withCORS(workspaceSpecHandler))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-quit; log.Println("[VibeX Agent] shutdown"); os.Exit(0) }()

	log.Println("[VibeX Agent] listening on :33338")
	log.Println("  SSE:    GET  http://localhost:33338/api/sse/<threadId>")
	log.Println("  Chat:   POST http://localhost:33338/api/chat")
	log.Fatal(http.ListenAndServe(":33338", nil))
}
