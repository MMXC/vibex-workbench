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

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/chat", chatHandler)
	http.HandleFunc("/api/sse/", sseHandler)
	http.HandleFunc("/api/threads/", historyHandler)
	http.HandleFunc("/api/skills", skillsHandler)
	http.HandleFunc("/api/step", stepHandler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-quit; log.Println("[VibeX Agent] shutdown"); os.Exit(0) }()

	log.Println("[VibeX Agent] listening on :33338")
	log.Println("  SSE:    GET  http://localhost:33338/api/sse/<threadId>")
	log.Println("  Chat:   POST http://localhost:33338/api/chat")
	log.Fatal(http.ListenAndServe(":33338", nil))
}
