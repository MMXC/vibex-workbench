package common

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// AgentPromptConfig matches `.agents/agents/*.json` — same schema as general-agent plus optional tool/skill routing for specialized agents.
type AgentPromptConfig struct {
	PromptFile     string            `json:"prompt_file"`
	Adapters       map[string]string `json:"adapters"`
	RequiredSkills []string          `json:"required_skills"`
	AllowedTools   []string          `json:"allowed_tools"`
	// LabelZH is optional UI metadata for slash-command menus (not used at runtime resolution).
	LabelZH string `json:"label_zh"`
}

// ApplyAgentPromptConfig resolves prompt_file + adapter suffix into one developer message (empty if unreadable).
func ApplyAgentPromptConfig(workspaceDir string, cfg *AgentPromptConfig, adapter string) string {
	if cfg == nil {
		return ""
	}
	promptPath := strings.TrimSpace(cfg.PromptFile)
	if promptPath == "" {
		return ""
	}
	if !filepath.IsAbs(promptPath) {
		promptPath = filepath.Join(workspaceDir, filepath.Clean(promptPath))
	}
	content, err := os.ReadFile(promptPath)
	if err != nil {
		log.Printf("[agent-prompt] failed to read prompt %s: %v", promptPath, err)
		return ""
	}
	msg := strings.TrimSpace(string(content))
	if msg == "" {
		return ""
	}
	suffix := strings.TrimSpace(cfg.Adapters[strings.TrimSpace(adapter)])
	if suffix != "" {
		msg = msg + "\n\n" + suffix
	}
	return msg
}

// ResolveDeveloperMessage loads the default agent from `.agents/agents/general-agent.json`.
func ResolveDeveloperMessage(workspaceDir, adapter, fallback string) string {
	cfgPath := filepath.Join(workspaceDir, ".agents", "agents", "general-agent.json")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return strings.TrimSpace(fallback)
	}
	var cfg AgentPromptConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Printf("[agent-prompt] invalid config %s: %v", cfgPath, err)
		return strings.TrimSpace(fallback)
	}
	msg := ApplyAgentPromptConfig(workspaceDir, &cfg, adapter)
	if msg == "" {
		return strings.TrimSpace(fallback)
	}
	return msg
}

// LoadSpecializedAgent reads `.agents/agents/<slug>.json` (same schema as general-agent).
// Used when chat passes agent_profile=<slug>. Does not read general-agent.json (use ResolveDeveloperMessage for that).
func LoadSpecializedAgent(workspaceDir, slug, adapter string) (developerMessage string, requiredSkills []string, allowedTools []string, ok bool) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" || slug == "general-agent" {
		return "", nil, nil, false
	}
	cfgPath := filepath.Join(workspaceDir, ".agents", "agents", slug+".json")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", nil, nil, false
	}
	var cfg AgentPromptConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Printf("[agent-prompt] invalid specialized agent %s: %v", cfgPath, err)
		return "", nil, nil, false
	}
	msg := ApplyAgentPromptConfig(workspaceDir, &cfg, adapter)
	return msg, cfg.RequiredSkills, cfg.AllowedTools, true
}
