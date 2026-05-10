package common

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type agentPromptConfig struct {
	PromptFile string            `json:"prompt_file"`
	Adapters   map[string]string `json:"adapters"`
}

// ResolveDeveloperMessage loads generic agent prompt from .agents and appends
// adapter-specific suffix. Falls back to fallback when config/files are missing.
func ResolveDeveloperMessage(workspaceDir, adapter, fallback string) string {
	cfgPath := filepath.Join(workspaceDir, ".agents", "agents", "general-agent.json")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return strings.TrimSpace(fallback)
	}
	var cfg agentPromptConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Printf("[agent-prompt] invalid config %s: %v", cfgPath, err)
		return strings.TrimSpace(fallback)
	}
	promptPath := strings.TrimSpace(cfg.PromptFile)
	if promptPath == "" {
		return strings.TrimSpace(fallback)
	}
	if !filepath.IsAbs(promptPath) {
		promptPath = filepath.Join(workspaceDir, filepath.Clean(promptPath))
	}
	content, err := os.ReadFile(promptPath)
	if err != nil {
		log.Printf("[agent-prompt] failed to read prompt %s: %v", promptPath, err)
		return strings.TrimSpace(fallback)
	}
	msg := strings.TrimSpace(string(content))
	if msg == "" {
		return strings.TrimSpace(fallback)
	}
	suffix := strings.TrimSpace(cfg.Adapters[strings.TrimSpace(adapter)])
	if suffix != "" {
		msg = msg + "\n\n" + suffix
	}
	return msg
}
