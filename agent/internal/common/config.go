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
	loadDotEnv()

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
	workspaceDir := getenv("WORKSPACE_DIR", "")
	if workspaceDir == "" {
		workspaceDir = inferWorkspaceDir()
	}

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

// loadDotEnv 从多处加载；同一键以后面的为准。
// 最后在「真正的 agent/.env」上再 Overload 一次，避免 ~/.vibex/agent.env 里裸域名覆盖掉 agent/.env 里的 .../v1。
func loadDotEnv() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	files := []string{
		filepath.Join(cwd, "..", "..", ".env"),
		filepath.Join(cwd, "..", ".env"),
		filepath.Join(cwd, ".env"),
		filepath.Join(cwd, "agent", ".env"),
	}
	for _, f := range files {
		_ = godotenv.Overload(f)
	}
	if home := homeDir(); home != "" {
		_ = godotenv.Overload(filepath.Join(home, ".vibex", "agent.env"))
	}
	// 强制以 agent 目录下的 .env 为最终覆盖（无论从仓库根还是从 agent/ 启动）
	var agentDotEnv string
	if strings.EqualFold(filepath.Base(cwd), "agent") {
		agentDotEnv = filepath.Join(cwd, ".env")
	} else {
		agentDotEnv = filepath.Join(cwd, "agent", ".env")
	}
	_ = godotenv.Overload(agentDotEnv)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return ""
}

// inferWorkspaceDir 在未设置 WORKSPACE_DIR 时：在 agent/ 下运行 → 上级为仓库根；否则 cwd 视为仓库根。
func inferWorkspaceDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "/root/vibex-workbench"
	}
	if strings.EqualFold(filepath.Base(cwd), "agent") {
		return filepath.Clean(filepath.Join(cwd, ".."))
	}
	return cwd
}

func normalizeBaseURL(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimRight(v, "/")
	v = strings.TrimSuffix(v, "/chat/completions")
	v = strings.TrimSuffix(v, "/responses")

	lower := strings.ToLower(v)
	// SDK 实际请求为 {BaseURL}/chat/completions；MiniMax 必须为 .../v1/chat/completions，否则会 404
	if strings.Contains(lower, "minimaxi.com") || strings.Contains(lower, "minimax.com") {
		if !strings.Contains(v, "/v1") && !strings.Contains(lower, "/anthropic/") {
			v = v + "/v1"
		}
	}
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
