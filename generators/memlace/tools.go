package memlace

import (
	"encoding/json"
	"fmt"
)

// Tool handlers — 符合 vibex-agent 的 func(string) string 签名。
// 注册方式：在 vibex-agent 的 handlers 中 import 此包，调用 RegisterHandlers。

// ── 全局单例（handler 无状态依赖）──────────────────────────────

var (
	globalStore *Store
	globalSess  *SessionManager
)

func ensureInit() error {
	if globalStore != nil && globalSess != nil {
		return nil
	}
	cfg := DefaultConfig("")
	store, err := NewStore(cfg)
	if err != nil {
		return fmt.Errorf("memlace init: %w", err)
	}
	sess, err := NewSessionManager(cfg)
	if err != nil {
		return fmt.Errorf("memlace session manager: %w", err)
	}
	globalStore = store
	globalSess = sess
	return nil
}

// ── Tool Schemas（供 agent 注册工具清单）───────────────────────

// ToolSchemas 返回所有 memlace tool 的 schema 描述。
func ToolSchemas() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "memlace_search",
			"description": "Search user preferences and project conventions from MemLace memory store. " +
				"Use this BEFORE asking clarifying questions to check if the preference already exists.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Search query (key, category, or value keywords)"},
					"target": map[string]interface{}{"type": "string", "description": "Filter by target: user|project (default: project)"},
					"limit": map[string]interface{}{"type": "integer", "description": "Max results (default: 10)"},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "memlace_write_pref",
			"description": "Write a user preference to MemLace store. Call this after a clarification round when the user confirms a choice.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target":   map[string]interface{}{"type": "string", "description": "user|project (default: project)"},
					"category": map[string]interface{}{"type": "string", "description": "Category: design|tech|workflow|ui|pref"},
					"key":      map[string]interface{}{"type": "string", "description": "Preference key"},
					"value":    map[string]interface{}{"type": "string", "description": "Preference value"},
					"source":   map[string]interface{}{"type": "string", "description": "clarification|observation|explicit"},
				},
				"required": []string{"key", "value"},
			},
		},
		{
			"name":        "memlace_clarification_start",
			"description": "Start a clarification session for a L2 spec. Call this when user clicks a L2 spec card.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"spec_name":   map[string]interface{}{"type": "string", "description": "L2 spec name"},
					"spec_parent": map[string]interface{}{"type": "string", "description": "L1 parent spec name"},
					"phase":       map[string]interface{}{"type": "string", "description": "tech_stack|mvp_prototype|frontend_split|user_stories"},
				},
				"required": []string{"spec_name", "phase"},
			},
		},
		{
			"name":        "memlace_clarification_qa",
			"description": "Record a clarification Q&A round. Call this after user answers a clarification question.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"spec_name": map[string]interface{}{"type": "string", "description": "L2 spec name"},
					"question":  map[string]interface{}{"type": "string", "description": "The question asked"},
					"answer":   map[string]interface{}{"type": "string", "description": "User answer"},
				},
				"required": []string{"spec_name", "question", "answer"},
			},
		},
		{
			"name":        "memlace_clarification_draft",
			"description": "Write the generated L2 spec draft to the clarification session.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"spec_name": map[string]interface{}{"type": "string", "description": "L2 spec name"},
					"draft":    map[string]interface{}{"type": "string", "description": "Generated L2 spec YAML content"},
				},
				"required": []string{"spec_name", "draft"},
			},
		},
		{
			"name":        "memlace_clarification_confirm",
			"description": "Confirm and lock a clarification session. Call this when user approves the final L2 spec.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"spec_name": map[string]interface{}{"type": "string", "description": "L2 spec name to confirm"},
				},
				"required": []string{"spec_name"},
			},
		},
		{
			"name":        "memlace_session_list",
			"description": "List all clarification sessions and their statuses.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// ── Tool Handlers ─────────────────────────────────────────────

// SearchHandler memlace_search tool handler.
func SearchHandler(args string) string {
	if err := ensureInit(); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	var params struct {
		Query  string `json:"query"`
		Target string `json:"target"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return `{"error": "invalid args"}`
	}
	if params.Target == "" {
		params.Target = "project"
	}
	searcher := NewSearcher(globalStore)
	results, err := searcher.Search(params.Query, params.Limit)
	if err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	out, _ := json.Marshal(map[string]interface{}{"query": params.Query, "count": len(results), "result": results})
	return string(out)
}

// WritePrefHandler memlace_write_pref tool handler.
func WritePrefHandler(args string) string {
	if err := ensureInit(); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	var params struct {
		Target   string `json:"target"`
		Category string `json:"category"`
		Key      string `json:"key"`
		Value    string `json:"value"`
		Source   string `json:"source"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return `{"error": "invalid args"}`
	}
	if params.Target == "" {
		params.Target = "project"
	}
	if params.Category == "" {
		params.Category = "pref"
	}
	if params.Source == "" {
		params.Source = "explicit"
	}
	entry := PreferenceEntry{
		Target:   params.Target,
		Category: params.Category,
		Key:      params.Key,
		Value:    params.Value,
		Source:   params.Source,
	}
	if err := globalStore.Add(entry); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	return fmt.Sprintf(`{"ok": true, "key": %q}`, params.Key)
}

// ClarificationStartHandler memlace_clarification_start tool handler.
func ClarificationStartHandler(args string) string {
	if err := ensureInit(); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	var params struct {
		SpecName   string `json:"spec_name"`
		SpecParent string `json:"spec_parent"`
		Phase      string `json:"phase"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return `{"error": "invalid args"}`
	}
	session := globalSess.CreateSession(params.SpecName, params.SpecParent, ClarificationPhase(params.Phase))
	questions := PhaseDefaultQuestions(ClarificationPhase(params.Phase))
	out, _ := json.Marshal(map[string]interface{}{
		"session_id": session.ID,
		"status":    session.Status,
		"phase":     params.Phase,
		"questions": questions,
	})
	return string(out)
}

// ClarificationQAHandler memlace_clarification_qa tool handler.
func ClarificationQAHandler(args string) string {
	if err := ensureInit(); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	var params struct {
		SpecName string `json:"spec_name"`
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return `{"error": "invalid args"}`
	}
	session, err := globalSess.AddRound(params.SpecName, params.Question, params.Answer)
	if err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	answered := make(map[string]bool)
	for _, r := range session.Rounds {
		answered[r.Question] = true
	}
	nextQuestions := PhaseDefaultQuestions(session.Phase)
	var remaining []string
	for _, q := range nextQuestions {
		if !answered[q] {
			remaining = append(remaining, q)
		}
	}
	out, _ := json.Marshal(map[string]interface{}{
		"session_id":    session.ID,
		"current_round": session.CurrentRound,
		"status":        session.Status,
		"next_questions": remaining,
		"done":          len(remaining) == 0,
	})
	return string(out)
}

// ClarificationDraftHandler memlace_clarification_draft tool handler.
func ClarificationDraftHandler(args string) string {
	if err := ensureInit(); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	var params struct {
		SpecName string `json:"spec_name"`
		Draft    string `json:"draft"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return `{"error": "invalid args"}`
	}
	if err := globalSess.SetDraft(params.SpecName, params.Draft); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	return fmt.Sprintf(`{"ok": true, "spec_name": %q}`, params.SpecName)
}

// ClarificationConfirmHandler memlace_clarification_confirm tool handler.
func ClarificationConfirmHandler(args string) string {
	if err := ensureInit(); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	var params struct {
		SpecName string `json:"spec_name"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return `{"error": "invalid args"}`
	}
	session := globalSess.GetSession(params.SpecName)
	if session == nil {
		return `{"error": "session not found"}`
	}
	if session.DerivedSpecDraft == "" {
		return `{"error": "no draft to confirm"}`
	}
	if err := globalSess.Confirm(params.SpecName); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	return fmt.Sprintf(`{"ok": true, "spec_name": %q}`, params.SpecName)
}

// SessionListHandler memlace_session_list tool handler.
func SessionListHandler(args string) string {
	if err := ensureInit(); err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	sessions := globalSess.ListSessions()
	var summaries []map[string]interface{}
	for _, s := range sessions {
		summaries = append(summaries, map[string]interface{}{
			"id": s.ID, "spec_name": s.SpecName, "phase": s.Phase,
			"status": s.Status, "rounds": len(s.Rounds),
			"has_draft": s.DerivedSpecDraft != "",
		})
	}
	if summaries == nil {
		summaries = []map[string]interface{}{}
	}
	out, _ := json.Marshal(map[string]interface{}{"count": len(sessions), "sessions": summaries})
	return string(out)
}

// PhaseDefaultQuestions 返回某阶段默认的澄清问题列表。
func PhaseDefaultQuestions(phase ClarificationPhase) []string {
	switch phase {
	case PhaseTechStack:
		return []string{
			"前端框架选什么？Svelte/SvelteKit 还是 React/Next.js？为什么？",
			"后端语言/运行时？Go 还是 Node.js？为什么？",
			"需要 AI 模型集成吗？什么模型？MiniMax/OpenAI/本地？",
			"数据存储方案？SQLite/PostgreSQL/内存？还是纯文件？",
			"部署目标是什么？本地桌面/Docker/VPS/Serverless？",
		}
	case PhaseMVPPrototype:
		return []string{
			"你想要什么交互形态？拖拽节点+连线 / 面板点击 / 纯文本？",
			"核心用户流程是什么？（按使用顺序说）",
			"有没有参考产品？类似 Figma/Notion/Linear 的哪些部分？",
			"MVP 必须有的功能是哪些？（最少 3 个）",
			"MVP 可以没有的功能是哪些？（第一版砍掉）",
		}
	case PhaseFrontendSplit:
		return []string{
			"前端状态管理用什么？Svelte store / Pinia / Zustand？",
			"API 调用层是否需要单独抽象？HTTP client 封装？",
			"胶水代码（路由/权限/事件桥接）放在哪层？",
			"前端代码生成物有哪些类型？组件/页面/store/路由？",
			"样式方案？Tailwind / 原生 CSS / CSS-in-JS？",
		}
	case PhaseUserStories:
		return []string{
			"用户是谁？他们的背景是什么？（描述 1-2 个典型用户画像）",
			"第一阶段必须交付的用户故事是什么？（按优先级排序）",
			"每个故事的验收标准是什么？（如何确认完成了）",
			"有没有需要拒绝的用户故事？（不在 MVP 范围内）",
			"各用户故事的依赖关系是什么？（哪些要先完成才能做下一个）",
		}
	}
	return []string{}
}

// ContextForAgent 读取 memlace，生成供 agent 注入 system prompt 的上下文字符串。
func ContextForAgent(limit int) string {
	if err := ensureInit(); err != nil {
		return ""
	}
	ctx, _ := globalStore.ReadContext(limit)
	if ctx == "" {
		return ""
	}
	return "\n" + ctx + "\n"
}
// NoOpHandler is a placeholder handler used in ToolSchemas.
// Real handlers are registered in ToolHandlers separately.
func NoOpHandler(args string) string {
	return `{"error": "not initialized"}`
}
