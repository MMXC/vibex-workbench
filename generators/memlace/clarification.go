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

// ClarificationStatus 澄清会话状态。
type ClarificationStatus string

const (
	StatusDraft       ClarificationStatus = "draft"
	StatusInProgress  ClarificationStatus = "in_progress"
	StatusConfirmed   ClarificationStatus = "confirmed"
	StatusRejected    ClarificationStatus = "rejected"
)

// Round 一轮澄清对话。
type Round struct {
	Round     int       `json:"round"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Confirmed bool      `json:"confirmed"`
	At        time.Time `json:"at"`
}

// ClarificationSession 一次澄清会话。
type ClarificationSession struct {
	ID                string              `json:"id"`
	SpecName         string              `json:"spec_name"`          // 目标 L2 spec 名
	SpecParent       string              `json:"spec_parent"`        // L1 parent 名
	Phase            ClarificationPhase  `json:"phase"`              // L2 澄清阶段
	Status           ClarificationStatus `json:"status"`
	Rounds           []Round             `json:"rounds"`
	CurrentRound     int                 `json:"current_round"`
	DerivedSpecDraft string              `json:"derived_spec_draft"` // 生成的 L2 spec 草稿内容
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	ConfirmedAt      *time.Time          `json:"confirmed_at,omitempty"`
	ConfirmedBy      string              `json:"confirmed_by,omitempty"`
}

// ClarificationPhase L2 澄清的四个阶段。
type ClarificationPhase string

const (
	PhaseTechStack     ClarificationPhase = "tech_stack"      // ① 技术选型
	PhaseMVPPrototype  ClarificationPhase = "mvp_prototype"   // ② MVP 原型
	PhaseFrontendSplit ClarificationPhase = "frontend_split"  // ③ 前端分层
	PhaseUserStories   ClarificationPhase = "user_stories"    // ④ 功能/用户故事
)

// PhaseMeta 每个阶段的元数据：默认问题模板。
var PhaseMeta = map[ClarificationPhase]struct {
	DisplayName string
	Description string
}{
	PhaseTechStack: {
		DisplayName: "技术选型",
		Description: "确认技术栈、框架选型与落地验证标准",
	},
	PhaseMVPPrototype: {
		DisplayName: "MVP 原型",
		Description: "通过可交互原型确认产品形态",
	},
	PhaseFrontendSplit: {
		DisplayName: "前后端分层",
		Description: "确认前后端各层边界与胶水代码引入方式",
	},
	PhaseUserStories: {
		DisplayName: "功能/用户故事",
		Description: "按阶段扩展优先级的功能规格",
	},
}

// SessionManager 管理澄清会话的创建、轮次和确认。
type SessionManager struct {
	sessions    map[string]*ClarificationSession
	dir         string
	activePhase ClarificationPhase
}

// NewSessionManager 打开澄清存储目录。
func NewSessionManager(cfg *Config) (*SessionManager, error) {
	dir := cfg.ResolveClarificationDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create clarification dir: %w", err)
	}
	m := &SessionManager{
		sessions:    make(map[string]*ClarificationSession),
		dir:         dir,
		activePhase: PhaseTechStack,
	}
	// 加载已有会话
	if err := m.loadAll(); err != nil {
		return nil, err
	}
	return m, nil
}

// CreateSession 创建或恢复一个澄清会话。
// 如果 specName 已存在则恢复，不重复创建。
func (m *SessionManager) CreateSession(specName, specParent string, phase ClarificationPhase) *ClarificationSession {
	// 优先恢复已有的 in_progress 会话
	for _, s := range m.sessions {
		if s.SpecName == specName && s.Status != StatusConfirmed && s.Status != StatusRejected {
			m.activePhase = s.Phase
			return s
		}
	}

	s := &ClarificationSession{
		ID:          fmt.Sprintf("clf-%d", time.Now().UnixNano()),
		SpecName:    specName,
		SpecParent:  specParent,
		Phase:       phase,
		Status:      StatusDraft,
		Rounds:      []Round{},
		CurrentRound: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.sessions[specName] = s
	m.activePhase = phase
	return s
}

// GetSession 读取一个会话。
func (m *SessionManager) GetSession(specName string) *ClarificationSession {
	return m.sessions[specName]
}

// AddRound 添加一轮问答。
func (m *SessionManager) AddRound(specName, question, answer string) (*ClarificationSession, error) {
	s, ok := m.sessions[specName]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", specName)
	}
	s.CurrentRound++
	r := Round{
		Round:    s.CurrentRound,
		Question: question,
		Answer:   answer,
		At:       time.Now(),
	}
	s.Rounds = append(s.Rounds, r)
	s.Status = StatusInProgress
	s.UpdatedAt = time.Now()

	if err := m.saveSession(s); err != nil {
		return nil, err
	}
	return s, nil
}

// SetDraft 设置 L2 spec 草稿内容。
func (m *SessionManager) SetDraft(specName, draft string) error {
	s, ok := m.sessions[specName]
	if !ok {
		return fmt.Errorf("session not found: %s", specName)
	}
	s.DerivedSpecDraft = draft
	s.UpdatedAt = time.Now()
	return m.saveSession(s)
}

// Confirm 确认澄清完成，锁定会话。
func (m *SessionManager) Confirm(specName string) error {
	s, ok := m.sessions[specName]
	if !ok {
		return fmt.Errorf("session not found: %s", specName)
	}
	s.Status = StatusConfirmed
	now := time.Now()
	s.ConfirmedAt = &now
	s.UpdatedAt = now
	return m.saveSession(s)
}

// Reject 拒绝/撤回澄清。
func (m *SessionManager) Reject(specName string) error {
	s, ok := m.sessions[specName]
	if !ok {
		return fmt.Errorf("session not found: %s", specName)
	}
	s.Status = StatusRejected
	s.UpdatedAt = time.Now()
	return m.saveSession(s)
}

// ListSessions 列出所有会话。
func (m *SessionManager) ListSessions() []*ClarificationSession {
	var all []*ClarificationSession
	for _, s := range m.sessions {
		all = append(all, s)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})
	return all
}

// ActivePhase 返回当前激活的澄清阶段。
func (m *SessionManager) ActivePhase() ClarificationPhase {
	return m.activePhase
}

// sessionFilePath 返回会话文件路径。
func (m *SessionManager) sessionFilePath(specName string) string {
	safe := strings.ReplaceAll(specName, "/", "-")
	return filepath.Join(m.dir, safe+".clf.json")
}

// saveSession 持久化会话到 JSON 文件。
func (m *SessionManager) saveSession(s *ClarificationSession) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	path := m.sessionFilePath(s.SpecName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

// loadAll 加载 dir 下所有 .clf.json 文件。
func (m *SessionManager) loadAll() error {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".clf.json") {
			continue
		}
		path := filepath.Join(m.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var s ClarificationSession
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		m.sessions[s.SpecName] = &s
	}
	return nil
}

// ToYAML 将澄清结果转为 L2 spec YAML 片段（供 agent 生成完整 spec 用）。
func (s *ClarificationSession) ToYAML() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("spec:\n  name: %q\n", s.SpecName))
	sb.WriteString(fmt.Sprintf("  parent: %q\n", s.SpecParent))
	sb.WriteString(fmt.Sprintf("  clarification_phase: %q\n", s.Phase))
	sb.WriteString("  clarification_rounds:\n")
	for _, r := range s.Rounds {
		sb.WriteString(fmt.Sprintf("    - round: %d\n", r.Round))
		sb.WriteString(fmt.Sprintf("      question: %q\n", r.Question))
		sb.WriteString(fmt.Sprintf("      answer: %q\n", r.Answer))
	}
	if s.DerivedSpecDraft != "" {
		sb.WriteString("  derived_spec_draft: |\n")
		for _, line := range strings.Split(s.DerivedSpecDraft, "\n") {
			sb.WriteString("    " + line + "\n")
		}
	}
	return sb.String()
}
