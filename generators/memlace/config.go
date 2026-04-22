package memlace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Config 决定 memlace 从哪里读偏好、从哪里写澄清。
// 用户通过 .memlace/config.yaml 或 .memlace/config.json 配置。
type Config struct {
	// MemoryStore 偏好记忆存储路径。
	// 支持两种模式：
	//   - "hermes": 指向 ~/.hermes/memories（链接 Hermes）
	//   - ".memlace": 本地 .memlace/local/ 存储
	MemoryStore string `json:"memory_store"`

	// Clarifications 澄清会话存储路径（默认 .memlace/clarifications）
	Clarifications string `json:"clarifications"`

	// WorkDir vibex-workbench 根目录（默认 cwd）
	WorkDir string `json:"work_dir"`

	// HermesLink 是否链接 Hermes 偏好（true = MemoryStore 指向 ~/.hermes/memories）
	HermesLink bool `json:"hermes_link"`
}

// DefaultConfig 返回默认配置，workDir 为空则用 cwd。
func DefaultConfig(workDir string) *Config {
	if workDir == "" {
		wd, _ := os.Getwd()
		workDir = wd
	}
	return &Config{
		MemoryStore:   ".memlace",
		Clarifications: ".memlace/clarifications",
		WorkDir:       workDir,
		HermesLink:   false,
	}
}

// LoadConfig 从 workDir/.memlace/config.json 加载配置，找不到则返回默认配置。
func LoadConfig(workDir string) *Config {
	cfg := DefaultConfig(workDir)
	path := filepath.Join(workDir, ".memlace", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg // 无配置文件就用默认
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		return cfg // 解析失败用默认
	}
	// 合并：loaded 的非零值覆盖默认值
	if loaded.MemoryStore != "" {
		cfg.MemoryStore = loaded.MemoryStore
	}
	if loaded.Clarifications != "" {
		cfg.Clarifications = loaded.Clarifications
	}
	if loaded.WorkDir != "" {
		cfg.WorkDir = loaded.WorkDir
	}
	cfg.HermesLink = loaded.HermesLink
	return cfg
}

// ResolveMemoryStore 返回偏好的绝对路径。
// "hermes" 映射到 ~/.hermes/memories，其他值相对于 WorkDir。
func (c *Config) ResolveMemoryStore() string {
	switch c.MemoryStore {
	case "hermes":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".hermes", "memories")
	case ".memlace":
		return filepath.Join(c.WorkDir, ".memlace", "local")
	default:
		if filepath.IsAbs(c.MemoryStore) {
			return c.MemoryStore
		}
		return filepath.Join(c.WorkDir, c.MemoryStore)
	}
}

// ResolveClarificationDir 返回澄清会话的绝对路径。
func (c *Config) ResolveClarificationDir() string {
	if filepath.IsAbs(c.Clarifications) {
		return c.Clarifications
	}
	return filepath.Join(c.WorkDir, c.Clarifications)
}

// EnsureDirs 确保所有必要目录存在。
func (c *Config) EnsureDirs() error {
	paths := []string{
		c.ResolveMemoryStore(),
		c.ResolveClarificationDir(),
	}
	for _, p := range paths {
		if err := os.MkdirAll(p, 0755); err != nil {
			return err
		}
	}
	return nil
}

// Save 将当前配置写回 .memlace/config.json。
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(c.WorkDir, ".memlace")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")
	return os.WriteFile(path, data, 0644)
}

// DetectHermesLink 检测 ~/.hermes/memories 是否存在，自动设置 HermesLink。
func DetectHermesLink() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".hermes", "memories"))
	return err == nil
}

// InitTemplate 返回 .memlace/config.json 的模板内容。
func InitTemplate() string {
	hermes := DetectHermesLink()
	memStore := ".memlace"
	if hermes {
		memStore = "hermes"
	}
	t := map[string]interface{}{
		"memory_store":   memStore,
		"clarifications": ".memlace/clarifications",
		"work_dir":       "", // 空 = 用 cwd
		"hermes_link":    hermes,
	}
	data, _ := json.MarshalIndent(t, "", "  ")
	return string(data)
}

// IsHermesLinked 检查是否链接了 Hermes。
func (c *Config) IsHermesLinked() bool {
	if c.HermesLink {
		return true
	}
	return c.MemoryStore == "hermes" || strings.HasSuffix(c.MemoryStore, ".hermes/memories")
}
