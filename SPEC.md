# VibeX Workbench 三栏布局重设计 SPEC

> 重写目标：将 vibex-workbench 从「四栏 Resizable 布局」改为「左 Agent Hub / 中 SpecPilot 原型 / 右 ProtoDesign 规范面板」三栏布局。

## 设计愿景

保留现有 vibex-workbench 的暗色主题和交互风格，将布局重构为三栏固定面板，中心填充 SpecPilot 原型预览，左右为工具面板。

```
┌──────────────────────────────────────────────────────────────┐
│ [工作区]  [Agent ▼]  [Spec Console | Task Console | PPT] [⚙] │  ← 顶部导航栏
├─────────┬────────────────────────────────────┬───────────────┤
│         │                                    │               │
│  Agent  │   SpecPilot 原型预览 (iframe)       │  ProtoDesign  │
│  Hub    │   全屏填充，支持热刷新              │               │
│         │                                    │  • Spec 说明  │
│ 专业Agent│                                   │  • DataCenter │
│ 面板集合 │                                   │  • EventCenter│
│         │                                    │  • MF注册表   │
│         │                                    │  • 适配器     │
│         │                                    │               │
├─────────┴────────────────────────────────────┴───────────────┤
│ [SpecPilot: DC:7890 ● | MF:5177 ●] [Agent: ●] [Backend: ●]  │  ← 底部状态栏
└──────────────────────────────────────────────────────────────┘
```

---

## L1 整体架构

### 布局模型

| 区域 | 默认宽度 | 行为 |
|------|------|------|
| Left Agent Hub | 340px | 可拖动调整宽度（min 200px / max 500px），可折叠到 0 |
| Center Prototype | flex:1（填满） | 全屏 iframe，热刷新 |
| Right ProtoDesign | 380px | 可拖动调整宽度（min 240px / max 560px），可折叠到 0 |
| Top NavBar | 42px | 常驻 |
| Bottom StatusBar | 24px | 常驻 |

### 路由设计

无 SPA 路由变化——整个布局是单页，panel 内容通过 Svelte 状态切换。

### 状态管理

- `activeWorkspace` — 当前 workspace 根目录
- `activeAgent` — 当前选中的专业 Agent（SpecSlot / TaskAgent / PPTAgent）
- `activeView` — 顶部视图切换（SpecConsole / TaskConsole / PPT）
- `spStatus` — SpecPilot 服务状态（dcRunning / mfRunning / ports）
- `activeProtoDesignTab` — 右侧面板当前 Tab（Spec / DC / EC / MF / Adapter）

---

## L2 组件清单

### 1. TopNavBar

**文件**: `frontend/src/lib/components/layout/TopNavBar.svelte`

**位置**: `position: fixed; top: 0; left: 0; right: 0; height: 42px;`

**元素**:
- `[工作区名]` — 显示当前 workspace 目录名，点击可切换
- `[视图标签行]` — 三个标签：Spec Console / Task Console / PPT
- `[⚙ 设置]` — 打开设置面板

**样式**: 背景 `#10131a`，底部 border `#303746`，字体 Inter 12px

### 2. Left Agent Hub Panel

**文件**: `frontend/src/lib/components/agents/AgentHubPanel.svelte`

**位置**: `position: fixed; top: 42px; left: 0; bottom: 24px; width: 320px;`

**内容**: 左侧工具面板，承载 SpecSlot 对话能力（复用现有 ChatPane + tool calling）。Panel 可折叠（点击边缘箭头收起），收起时仅显示 40px 宽的图标列。

**子组件**:
- `AgentContentArea` — SpecSlot 聊天区域
  - 复用 `SpecSlotChatPane` + `ToolChatBubble`
  - 复用 spec slot 选择和 tool calling 逻辑

**样式**: 背景 `#0b0d12`，右侧分隔线 `#1e1e2e`

### 3. Center SpecPilot Preview

**文件**: `frontend/src/lib/components/prototype/SpecPilotPreview.svelte`

**位置**: `position: fixed; top: 42px; left: 320px; right: 360px; bottom: 24px;`

**内容**:
- iframe 全屏加载 `http://localhost:{mfPort}/preview`
- 如果 SpecPilot 未启动，显示「SpecPilot 未启动 · 点击底部状态栏启动」
- iframe 样式: `border: none; width: 100%; height: 100%`

**通信**: 通过 `postMessage` bridge 与 iframe 通信（复用现有 bridgeHandler）

### 4. Right ProtoDesign Panel

**文件**: `frontend/src/lib/components/proto/ProtoDesignPanel.svelte`

**位置**: `position: fixed; top: 42px; right: 0; bottom: 24px; width: 360px;`

**Tab 结构**（5 个 Tab）:

| Tab | 内容 |
|-----|------|
| Spec | 当前 spec 的 title / description / path / status |
| DataCenter | DC Key-Value 表格，可实时编辑值（PUT /api/dc/key） |
| EventCenter | 事件列表 + 发布表单（POST /api/ec/publish） |
| MF | MF 组件注册表（GET /api/components） |
| Adapter | 适配器连接状态列表 |

**DataCenter Tab**:
- 表格两列：`Key` | `Value`
- 每行可编辑，blur 或 Enter 提交 PUT
- 顶部有「+ 新建」按钮
- 实时轮询 `/api/dc/list`（10s 间隔）

**EventCenter Tab**:
- 事件列表（注册过的事件）
- 底部发布表单：`Event name` + `Data (JSON)` + `发布`按钮
- SSE 订阅 `/api/ec/subscribe` 实时显示发布事件

**样式**: Tab 行在顶部，12px Inter，字号 11px，活跃 tab 下划线 `#7c3aed`

### 5. Bottom StatusBar

**文件**: `frontend/src/lib/components/workbench/StatusBar.svelte`（已有，复用）

**位置**: `position: fixed; bottom: 0; left: 0; right: 0; height: 24px;`

**内容**（从右到左）:
- `[SpecPilot: DC:7890 ● | MF:5177 ●]` — 点击可展开/收起底层面板（已有）
- `[Agent: ●]` — 当前 Agent 状态
- `[Backend: 后端就绪]` — backend 连接状态
- `[VibeX Workbench]` — 版本标签

---

## L3 实现计划

### Phase 1: 骨架替换（不改现有逻辑，只改布局）

1. 新建 `LayoutShell.svelte` — 三栏骨架容器
2. 新建 `TopNavBar.svelte`
3. 新建 `AgentHubPanel.svelte`（先空壳，只渲染 SpecSlot Agent 内容）
4. 新建 `SpecPilotPreview.svelte`（直接渲染 iframe）
5. 新建 `ProtoDesignPanel.svelte`（先空壳，只有 Tab 框架）
6. 替换 `+page.svelte` 根组件为 `LayoutShell`
7. 验证：make wails-dev 能跑，布局正确显示

### Phase 2: 填充 Agent Hub

1. 把现有 `SpecSlotDrawer` 逻辑移到 `AgentHubPanel`
2. 接入 `specSlotSessionStore`
3. 接入 tool calling（复用现有 ToolChatBubble）
4. 实现 Agent Tab 切换（SpecSlot / Tasks / PPT）

### Phase 3: 填充 ProtoDesign Panel

1. 实现 Spec Tab（读取当前 spec 元数据）
2. 实现 DataCenter Tab（HTTP 读写 DC）
3. 实现 EventCenter Tab（SSE 订阅）
4. 实现 MF Tab（GET /api/components）
5. 实现 Adapter Tab（适配器列表）

### Phase 4: 集成 SpecPilot

1. 确认 SpecPilot CLI 已嵌入 `.specpilot/cli/`
2. 实现 workspace 切换时自动 `specpilot init`
3. StatusBar bootstrap 按钮触发 `specpilot start`
4. iframe 热刷新（preview 文件变更时自动 reload）

---

## L4 技术约束

- **不删除现有组件文件** — 暂时保留，以后逐步迁移
- **复用现有 stores** — `specSlotSessionStore`、`specpilotStatusStore`
- **复用现有 Go handlers** — `handlers_specpilot.go`
- **SpecPilot CLI** — 放在 `.specpilot/cli/`，通过 Go handler 调用
- **样式变量** — 复用现有 CSS 变量（`--wb-bg`、`--wb-border` 等）

---

## L5 验收标准

- [ ] `make wails-dev` 启动后，三栏布局正确显示
- [ ] 左侧 Agent Hub 显示 SpecSlot 聊天
- [ ] 中央 iframe 加载 SpecPilot preview（或显示未启动提示）
- [ ] 右侧 ProtoDesign 面板有 5 个 Tab
- [ ] 底部 StatusBar 显示 SpecPilot 状态
- [ ] SpecSlot Agent 可以正常对话（复用现有 tool calling）
- [ ] DataCenter Tab 可实时读写 DC 数据
