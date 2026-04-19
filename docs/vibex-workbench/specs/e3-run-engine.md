# Spec — E3: Run Engine

> PRD: `prd.md` E3 章节
> 规范层级: L4 Feature Spec

## Run 状态追踪

### runStore 扩展

```typescript
// 新增 toolInvocations 数组
interface RunState {
  runs: Run[];
  active_run_id: string | null;
  toolInvocations: ToolInvocation[];  // 新增
}

// 事件流映射
'run.started'    → runStore.createRun + toolInvocations = []
'tool.called'    → toolInvocations.push(invocation)
'tool.completed' → toolInvocations[i].status = 'completed'
'run.completed'  → active_run_id = null
```

### 状态机

| Run 状态 | 触发事件 | Canvas 节点 | Composer 显示 |
|----------|----------|-------------|-------------|
| pending | (初始) | — | — |
| executing | run.started | Run 节点（蓝色边框）| "运行中..." |
| tool_called | tool.called | Tool 节点出现 | — |
| tool_completed | tool.completed | Tool 节点变绿 | — |
| completed | run.completed | Run 节点变绿 | 显示 result_summary |
| failed | run.failed | Run 节点变红 | 显示 error_message |

---

## UI 状态规范

### Composer 运行态

#### 理想态
```
┌────────────────────────────────────────┐
│  [文本][图片][文件][URL]               │
│ ┌────────────────────────────────────┐ │
│ │                                    │ │
│ └────────────────────────────────────┘ │
│  Ctrl+Enter 发送              [发送↵] │
└────────────────────────────────────────┘
```

#### 加载态（Run 执行中）
```
┌────────────────────────────────────────┐
│  运行中 ████████░░░░░  3/3 工具已调用   │
│ ┌────────────────────────────────────┐ │
│ │ [用户输入内容]                      │ │
│ └────────────────────────────────────┘ │
│  [停止运行]                    [取消]  │
└────────────────────────────────────────┘
```
- 进度条：线性，`width` 从 0 到 100%，`background: #4f46e5`
- 工具计数：「3/3 工具已调用」
- 停止按钮：`background: transparent`, `border: 1px solid #ef4444`, `color: #ef4444`

#### 错误态
```
┌────────────────────────────────────────┐
│  ⚠️ 运行失败：网络超时                  │
│ ┌────────────────────────────────────┐ │
│ │ [用户输入内容]                      │ │
│ └────────────────────────────────────┘ │
│                           [重新运行]  │
└────────────────────────────────────────┘
```
- 错误文案：具体错误信息，禁止只写"出错"
- 重新运行按钮：`background: #4f46e5`, `color: white`

---

## Canvas Run 节点规范

### 节点四态

#### 理想态（completed）
- 节点背景：`#1a2e1a`，边框：`#22c55e` 2px
- 状态图标：✓
- 内容：Run 标题 + 执行时间

#### 加载态（executing）
- 节点背景：`#1a1a2e`，边框：`#4f46e5` 2px，animated pulse
- 状态图标：旋转 loading
- 内容：Run 标题 + "运行中..."

#### 错误态（failed）
- 节点背景：`#2e1a1a`，边框：`#ef4444` 2px
- 状态图标：✗
- 内容：Run 标题 + 错误摘要

#### 空状态
- Canvas 无节点时显示占位：
```
┌──────────────────────────────────────┐
│                                      │
│        [插图: 空画布]                 │
│     发送消息开始你的第一个 Run        │
│                                      │
└──────────────────────────────────────┘
```
- 禁止只留白

---

## 验收标准

```typescript
// F3.1 — Run 状态追踪
expect(runStore.activeRun).not.toBeNull();     // run.started 后
expect(runStore.toolInvocations.length).toBe(3);  // mock run 3 tools

// F3.2 — Run 结果展示
// completed 后
expect(canvasNode('run-id').data.status).toBe('completed');
expect(composerFooter).toContainText('完成');
```

### DoD
- [ ] `runStore.toolInvocations` 数组在 Run 期间正确更新
- [ ] Composer 显示运行进度（进度条 + 工具计数）
- [ ] Run 完成后 Composer 底部显示摘要
- [ ] Run 失败后 Composer 显示具体错误信息
- [ ] Canvas Run 节点四态定义完整
- [ ] Canvas 空状态有引导文案
- [ ] 所有间距使用 8 倍数，所有颜色使用 CSS Token
