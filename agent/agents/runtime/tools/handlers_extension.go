// handlers_extension.go — Prototype-Skill PS 工具集（供 Agent 通过 extension 驱动原型页）
// 这些工具不直接操作文件系统，而是通过 SSE + HTTP 回调与 Chrome 扩展通信，
// 由扩展在原型页面中执行实际的高亮/标注/解析等操作。
package tools

// psHighlightResult 供 extension tool-result handler 读取
type psHighlightResult struct {
	Highlighted int      `json:"highlighted"`
	Missed      []string `json:"missed"`
}

// psAnnotateResult
type psAnnotateResult struct {
	Annotated int `json:"annotated"`
}

// psParseResult
type psParseResult struct {
	URL       string         `json:"url"`
	Title     string         `json:"title"`
	Nodes     int            `json:"nodes"`
	SpecTree  any           `json:"spec_tree"`
}

// psBindResult
type psBindResult struct {
	Bound      bool   `json:"bound"`
	DataAttr   string `json:"data_ps_attr"`
}

// psOnboardResult
type psOnboardResult struct {
	StepsRun int `json:"steps_run"`
}

// psGetContextResult
type psGetContextResult struct {
	URL             string `json:"url"`
	Title           string `json:"title"`
	WorkspaceRoot   string `json:"workspace_root"`
	SpecPath       string `json:"spec_path"`
	PrototypeFile  string `json:"prototype_file"`
	SessionThreadID string `json:"session_thread_id"`
}

// ── PS 工具 handler（Go Agent 端桩）
// 这些 handler 不在 Go 中真正执行 DOM 操作，而是：
// 1. 广播 SSE 事件 → extension 接收并执行
// 2. 将工具调用 ID 注册到 pendingResult 表
// 3. 返回占位文本，告知 Agent "等待 extension 执行"
// 实际的 DOM 结果由 extension 通过 POST /api/extension/tool-result 回调。

func psHighlightHandler(arguments string) string {
	return "[PS-HIGHLIGHT] 请求已通过 SSE 广播到扩展。等待扩展执行结果。工具调用ID: " +
		ExtractCallID(arguments) + "。如扩展未连接，结果将不可用。"
}

func psAnnotateHandler(arguments string) string {
	return "[PS-ANNOTATE] 请求已通过 SSE 广播到扩展。等待扩展执行结果。"
}

func psParseHandler(arguments string) string {
	return "[PS-PARSE] 请求已通过 SSE 广播到扩展。等待扩展执行结果。"
}

func psBindHandler(arguments string) string {
	return "[PS-BIND] 请求已通过 SSE 广播到扩展。等待扩展执行结果。"
}

func psOnboardHandler(arguments string) string {
	return "[PS-ONBOARD] 请求已通过 SSE 广播到扩展。等待扩展执行结果。"
}

func psGetPageContextHandler(arguments string) string {
	return "[PS-GET-PAGE-CONTEXT] 请求已通过 SSE 广播到扩展。等待扩展执行结果。"
}

// ExtractCallID 尝试从 arguments JSON 中提取 call_id（供日志用）
func ExtractCallID(arguments string) string {
	// arguments 可能是 {"callId": "call_xxx", ...} 格式
	// 简单字符串查找，避免引入 json 解析依赖
	const prefix = `"callId":`
	for i := 0; i < len(arguments)-len(prefix)-2; i++ {
		if arguments[i:i+len(prefix)] == prefix {
			start := i + len(prefix)
			// 跳过空格和引号
			for start < len(arguments) && (arguments[start] == ' ' || arguments[start] == '"') {
				start++
			}
			end := start
			for end < len(arguments) && arguments[end] != '"' && arguments[end] != ',' && arguments[end] != '}' {
				end++
			}
			return arguments[start:end]
		}
	}
	return "unknown"
}

// PS工具规格（供 buildToolsAndHandlers 注册）
var PSSpecs = []Spec{
	{
		Name:        "ps_highlight",
		Description: "在原型页面中高亮指定 CSS 选择器的元素。扩展执行后通过 POST /api/extension/tool-result 回调结果。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"selectors":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "CSS选择器数组"},
				"color":        map[string]any{"type": "string", "description": "高亮颜色，默认#7170ff"},
				"duration_ms":  map[string]any{"type": "integer", "description": "持续毫秒，默认3000"},
				"callId":       map[string]any{"type": "string", "description": "工具调用ID（框架自动注入）"},
			},
			"required": []any{"selectors"},
		},
		Handler: psHighlightHandler,
	},
	{
		Name:        "ps_annotate",
		Description: "在原型页面显示/隐藏节点标注。扩展执行后回调结果。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode":             map[string]any{"type": "string", "enum": []any{"show", "hide", "toggle"}, "description": "显示/隐藏/切换"},
				"show_spec_links":  map[string]any{"type": "boolean", "description": "是否显示spec绑定链接"},
				"callId":           map[string]any{"type": "string"},
			},
		},
		Handler: psAnnotateHandler,
	},
	{
		Name:        "ps_parse",
		Description: "解析原型页面为Spec树（DOM遍历）。扩展执行后返回节点结构。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"max_depth":      map[string]any{"type": "integer", "description": "最大深度，默认3"},
				"include_hidden": map[string]any{"type": "boolean", "description": "是否包含隐藏元素"},
				"callId":         map[string]any{"type": "string"},
			},
		},
		Handler: psParseHandler,
	},
	{
		Name:        "ps_bind",
		Description: "将Spec节点绑定到DOM选择器（写入data-ps-*属性）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"spec_name": map[string]any{"type": "string", "description": "Spec节点名称"},
				"selector":  map[string]any{"type": "string", "description": "CSS选择器"},
				"layer":     map[string]any{"type": "string", "description": "层级L3/L4/L5"},
				"callId":    map[string]any{"type": "string"},
			},
			"required": []any{"spec_name", "selector"},
		},
		Handler: psBindHandler,
	},
	{
		Name:        "ps_onboard",
		Description: "在原型页面执行引导演示（onboard steps）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"steps": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"title":   map[string]any{"type": "string"},
							"body":    map[string]any{"type": "string"},
							"target":  map[string]any{"type": "string"},
							"ms":      map[string]any{"type": "integer"},
						},
					},
				},
				"callId": map[string]any{"type": "string"},
			},
			"required": []any{"steps"},
		},
		Handler: psOnboardHandler,
	},
	{
		Name:        "ps_get_page_context",
		Description: "获取当前原型页面的完整上下文信息（工作区根/原型文件路径/SSE threadId）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"callId": map[string]any{"type": "string"},
			},
		},
		Handler: psGetPageContextHandler,
	},
}
