package memlace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PreferenceEntry 一条偏好记录。
type PreferenceEntry struct {
	ID        string    `json:"id"`
	Target    string    `json:"target"`    // "user"|"project"|"session"
	Category  string    `json:"category"`  // "design"|"tech"|"workflow"|"ui"|"pref"
	Key       string    `json:"key"`       // 偏好键名
	Value     string    `json:"value"`     // 偏好值
	Source    string    `json:"source"`    // 来源：clarification|observation|explicit
	CreatedAt time.Time `json:"created_at"`
}

// Store 管理偏好记忆的 JSON 文件存储（无外部依赖）。
type Store struct {
	cfg     *Config
	entries []PreferenceEntry
	loaded  bool
}

// NewStore 打开偏好存储。
func NewStore(cfg *Config) (*Store, error) {
	s := &Store{cfg: cfg}
	return s, s.load()
}

// load 从 JSON 文件加载所有偏好。
func (s *Store) load() error {
	if s.loaded {
		return nil
	}
	path := s.prefFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.entries = []PreferenceEntry{}
			s.loaded = true
			return nil
		}
		return err
	}
	if err := json.Unmarshal(data, &s.entries); err != nil {
		return err
	}
	s.loaded = true
	return nil
}

// save 持久化所有偏好到 JSON 文件。
func (s *Store) save() error {
	dir := filepath.Dir(s.prefFilePath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.prefFilePath(), data, 0644)
}

func (s *Store) prefFilePath() string {
	store := s.cfg.ResolveMemoryStore()
	return filepath.Join(store, ".memlace", "preferences.json")
}

// Add 写入一条偏好，按 key+target 去重。
func (s *Store) Add(entry PreferenceEntry) error {
	if entry.ID == "" {
		entry.ID = newID()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	// 去重：同 target+key 则覆盖
	for i, e := range s.entries {
		if e.Target == entry.Target && e.Key == entry.Key {
			entry.ID = e.ID
			s.entries[i] = entry
			return s.save()
		}
	}
	s.entries = append(s.entries, entry)
	return s.save()
}

// GetByKey 精确查询。
func (s *Store) GetByKey(target, key string) *PreferenceEntry {
	for _, e := range s.entries {
		if e.Target == target && e.Key == key {
			return &e
		}
	}
	return nil
}

// Search 全文搜索偏好（内存版，无 FTS 依赖）。
func (s *Store) Search(query string, limit int) ([]PreferenceEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}
	var results []PreferenceEntry
	for _, e := range s.entries {
		score, _ := scoreEntry(e, q)
		if score > 0 {
			results = append(results, e)
		}
	}
	// 按分数降序，取前 limit
	sort.Slice(results, func(i, j int) bool {
		s1, _ := scoreEntry(results[i], q)
		s2, _ := scoreEntry(results[j], q)
		return s1 > s2
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// List 返回某 target 下所有偏好。
func (s *Store) List(target string, limit int) ([]PreferenceEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	var out []PreferenceEntry
	for _, e := range s.entries {
		if e.Target == target || e.Target == "user" {
			out = append(out, e)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ReadContext 生成供 agent 注入的上下文字符串。
func (s *Store) ReadContext(limit int) (string, error) {
	if limit <= 0 {
		limit = 20
	}
	entries, err := s.List("project", limit)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", nil
	}
	var lines []string
	lines = append(lines, "## MemLace User Preferences")
	for _, e := range entries {
		lines = append(lines, "  - ["+e.Category+"/"+e.Source+"] "+e.Key+" = "+e.Value)
	}
	return strings.Join(lines, "\n"), nil
}


func newID() string {
	return time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", time.Now().Nanosecond())
}
