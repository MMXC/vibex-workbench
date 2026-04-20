package common

import (
	"encoding/json"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type Config struct {
	BaseURL       string
	APIKey        string
	Model         string
	SubAgentModel string
	DebugHTTP     bool

	// Vibex-specific
	SkillsDir  string            // path to hermes skills, default ~/.hermes/skills
	WorkspaceDir string          // vibex workbench root, default /root/vibex-workbench
	StepModels map[string]string // step_type -> model name, loaded from models.yaml
}

func LoadConfig() Config {
	_ = godotenv.Overload(".env", "../.env", "../../.env", "~/.vibex/agent.env")

	model := getenv("OPENAI_MODEL", "gpt-4o")
	skillsDir := getenv("SKILLS_DIR", "")
	if skillsDir == "" {
		usr, err := user.Current()
		if err == nil {
			skillsDir = filepath.Join(usr.HomeDir, ".hermes", "skills")
		} else {
			skillsDir = "/root/.hermes/skills"
		}
	}
	workspaceDir := getenv("WORKSPACE_DIR", "/root/vibex-workbench")

	return Config{
		BaseURL:       normalizeBaseURL(getenv("OPENAI_BASE_URL", "https://api.openai.com/v1")),
		APIKey:        normalizeAPIKey(getenv("OPENAI_API_KEY", "")),
		Model:         model,
		SubAgentModel: getenv("SUBAGENT_MODEL", model),
		DebugHTTP:     getenvBool("DEBUG_HTTP", false),
		SkillsDir:     skillsDir,
		WorkspaceDir: workspaceDir,
		StepModels:    loadStepModels(workspaceDir),
	}
}

func NewClient(cfg Config) openai.Client {
	opts := []option.RequestOption{
		option.WithBaseURL(cfg.BaseURL),
		option.WithAPIKey(cfg.APIKey),
	}
	if cfg.DebugHTTP {
		opts = append(opts, option.WithDebugLog(log.New(os.Stderr, "[openai] ", log.LstdFlags|log.Lmicroseconds)))
	}
	return openai.NewClient(opts...)
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getenvBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func normalizeAPIKey(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "Bearer ")
	return strings.TrimSpace(v)
}

func normalizeBaseURL(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimRight(v, "/")
	v = strings.TrimSuffix(v, "/chat/completions")
	v = strings.TrimSuffix(v, "/responses")
	return v
}

// ── Step Model Routing ──────────────────────────────────────────

// jsonModels is the shape of models.json
type jsonModels struct {
	Default    string                       `json:"default"`
	StepModels map[string]stepModelEntry `json:"step_models"`
}

type stepModelEntry struct {
	Description string `json:"description"`
	Model       string `json:"model"`
}

// loadStepModels reads models.json from the agent directory.
func loadStepModels(workspaceDir string) map[string]string {
	cfgPath := filepath.Join(workspaceDir, "agent", "models.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Printf("[config] models.json not found at %s: %v", cfgPath, err)
		return nil
	}
	var m jsonModels
	if err := json.Unmarshal(data, &m); err != nil {
		log.Printf("[config] failed to parse models.json: %v", err)
		return nil
	}
	result := make(map[string]string)
	if m.Default != "" {
		result["_default"] = m.Default
	}
	for k, v := range m.StepModels {
		if v.Model != "" {
			result[k] = v.Model
		}
	}
	if len(result) > 0 {
		log.Printf("[config] loaded %d step models from models.json", len(result))
	}
	return result
}

// GetModelForStep returns the model for a given step type, or the default.
func (c Config) GetModelForStep(stepType string) string {
	if c.StepModels == nil {
		return c.Model
	}
	if m, ok := c.StepModels[stepType]; ok {
		return m
	}
	if d, ok := c.StepModels["_default"]; ok {
		return d
	}
	return c.Model
}
