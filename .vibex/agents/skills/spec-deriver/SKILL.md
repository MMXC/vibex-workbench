---
name: spec-deriver
description: |
  根据 UI spec（L3/L4/L5）推导对应的数据契约、状态模型、事件总线规范。
  读取已标注原型的 L3/L4/L5 spec，解析其 io / io_contract / behavior / content.ui_spec.acceptance 字段，
  形式化推导出 DATA（数据契约）/ STATE（状态模型）/ EVT（事件总线）三类派生 spec。
  同一源 spec 可生成多个同类型派生 spec（用不同名称区分）。
  派生 spec 放入现有层级目录（DATA/STATE → L4-feature/；EVT → L5-slice/），仅用前缀区分。
  触发词：推导spec、推导数据契约、推导状态、推导事件、derive spec。
---

# spec-deriver（Spec 推导器）

**规范路径**：与本目录 **`agent.json`** 成对放置于 **`.vibex/agents/skills/spec-deriver/`**。

---

## 0. 前置条件

### 0.1 输入

| 参数 | 来源 | 说明 |
|------|------|------|
| `workspace_root` | 用户指定 | 工作区根目录 |
| `source_spec_path` | 用户指定或自动扫描 | 源 UI spec 路径（可选，默认扫描所有 L3/L4/L5） |

### 0.2 前置检查

1. 使用 `read_file` 确认源 spec 存在且包含 `io:` 或 `io_contract:` 字段
2. 确认目标目录存在（使用现有层级目录，不新建）：
   - DATA / STATE（L4 源）→ `.vibex/specs/L4-feature/`
   - EVT（L5 源）→ `.vibex/specs/L5-slice/`
   - EVT（L3/L4 源）→ `.vibex/specs/L4-feature/`
3. 如目录不存在，创建空目录

---

## 1. 派生目录结构（核心规则）

**禁止创建新平级目录。所有派生 spec 必须放进现有层级目录，仅用前缀区分。**

```
.vibex/specs/
├── L4-feature/          ← L4 源派生：DATA / STATE（L5 源但 level 4_feature 的 EVT 也放这里）
│   └── DATA-{entity}-{action}.yaml
│   └── STATE-{store-name}.yaml
│   └── EVT-{component}-{event}.yaml      # L4 源的 EVT
└── L5-slice/            ← L5 源派生：EVT（L5 源特有）
    └── EVT-{component}-{event}.yaml
    └── STATE-{component-detail}.yaml     # 组件级状态（如编辑态）
```

**层级归属规则**：

| 派生类型 | 源 spec 层级 | `spec.level` | 写入目录 |
|---------|------------|-------------|---------|
| DATA | L4 feature | `4_feature` | `L4-feature/` |
| STATE | L4 feature | `4_feature` | `L4-feature/` |
| EVT | L4 feature | `4_feature` | `L4-feature/` |
| EVT | L5 slice | `5_slice` | `L5-slice/` |
| STATE | L5 slice | `5_slice` | `L5-slice/` |

---

## 2. 派生规范

### 2.1 三类派生 spec

| 类型 | 前缀 | `meta.type` | `spec.parent` | `spec.level` | 写入目录 | 从哪里推导 |
|------|------|------------|--------------|-------------|---------|-----------|
| 数据契约 | `DATA-` | `data_contract` | 源 spec name | 源 spec level | 源 spec 同级目录 | `io.input` / `io.output` / `io_contract` / `constraints` |
| 状态模型 | `STATE-` | `state_model` | 源 spec name | 源 spec level | 源 spec 同级目录 | `content.ui_spec.states` / `content.changes` / `io_contract.behavior` |
| 事件总线 | `EVT-` | `event_bus` | 源 spec name | 源 spec level | 源 spec 同级目录 | `io_contract.behavior`、`content.ui_spec.acceptance` |

**`spec.parent` 指向源 spec 的 `name` 字段**（不是文件路径）。

### 2.2 DATA 数据契约格式

```yaml
---
spec:
  version: "0.1"
  level: "{源spec.level值}"       # ← 如 "4_feature" 或 "5_slice"
  name: "DATA-{entity}-{action}"
  parent: "{源spec.name}"          # ← 指向源 spec 的 name 字段
  status: "proposal"
  prototype_ref: "{相对路径}"      # ← 如 "../../prototypes/…"（可为空字符串）
meta:
  type: "data_contract"
  owner: "user"
  created: "{今天日期}"
  updated: "{今天日期}"
source_ui_spec: "{源spec.name}"
derived_from: "{io.input/output 原文}"
io:
  input:
    - source: "{io.input 原文}"
      fields:
        - name: string
          type: string
          required: boolean
          description: string
  output:
    - source: "{io.output 原文}"
      fields:
        - name: string
          type: string
          required: boolean
          description: string
api:
  endpoint: string
  method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" | "WAILS"
  description: string
constraints:
  - name: string
    rule: string
    validation: string
```

**method 枚举**：

| method | 含义 |
|--------|------|
| `GET` / `POST` 等 | 标准 HTTP 方法 |
| `WAILS` | 通过 Wails binding 直接调用 Go 方法，不走 HTTP |

### 2.3 STATE 状态模型格式

```yaml
---
spec:
  version: "0.1"
  level: "{源spec.level值}"
  name: "STATE-{store-name}"
  parent: "{源spec.name}"
  status: "proposal"
  prototype_ref: "{相对路径}"
meta:
  type: "state_model"
  owner: "user"
  created: "{今天日期}"
  updated: "{今天日期}"
source_ui_spec: "{源spec.name}"
state:
  initial: {}                      # ← 所有字段的默认值
  shape:
    - key: string
      type: string                 # ← TypeScript 类型字符串，如 "string | null"
      default: any
      description: string
  persistence: "memory" | "localStorage" | "sessionStorage" | "store2disk"
  readers:
    - component: string
      uses: string                 # ← 字段名列表
  writers:
    - component: string
      mutates: string             # ← 字段名列表
derived_from_behavior: "{来源描述}"
```

### 2.4 EVT 事件总线格式

```yaml
---
spec:
  version: "0.1"
  level: "{源spec.level值}"
  name: "EVT-{component}-{event}"
  parent: "{源spec.name}"
  status: "proposal"
  prototype_ref: "{相对路径}"
meta:
  type: "event_bus"
  owner: "user"
  created: "{今天日期}"
  updated: "{今天日期}"
source_ui_spec: "{源spec.name}"
event:
  name: string                     # ← 事件名，建议用 "source:subject:verb" 格式
  trigger: string                  # ← 触发时机描述
  direction: "emit" | "listen" | "bidirectional"
  payload:
    - field: string
      type: string
      description: string
  consumers:
    - component: string
      reaction: string             # ← 消费者收到事件后做什么
derived_from_behavior: "{来源描述}"
```

---

## 3. 推导工作流程

### 步骤一：扫描源 spec

扫描 `.vibex/specs/L3-module/`、`.vibex/specs/L4-feature/` 和 `.vibex/specs/L5-slice/`，选取包含以下字段的 spec：

- `io:` 有内容
- `io_contract:` 有内容
- `content.ui_spec.acceptance` 有内容
- `content.changes` 有内容
- `constraints:` 有事件/状态描述

输出扫描结果清单：

```
### 待推导源 spec

| # | spec 名称 | 层级 | `spec.level` | 包含字段 | 可推导类型 | 写入目录 |
|---|-----------|------|------------|---------|-----------|---------|
| 1 | FEAT-scaffolding | L4 | 4_feature | io, io_contract.behavior, content.behaviors | DATA + STATE | L4-feature/ |
| 2 | FEAT-workspace-session-persistence | L4 | 4_feature | io, io_contract.behavior, content.session_fields | DATA + STATE | L4-feature/ |
| 3 | FEAT-spec-write_L5 | L5 | 5_slice | io, io_contract, content.changes, content.ui_spec.acceptance | EVT | L5-slice/ |
```

### 步骤二：分 spec 推导

对每个源 spec，逐一执行：

**DATA 推导**：
1. 解析 `io.input` 和 `io.output` 字段，提取名词（entity）和动词（action）
2. 从 `io_contract` 提取 API endpoint（`/api/...`）
3. 从 `constraints` 提取约束规则，映射到 `api.constraints`
4. 决定 `api.method`：HTTP 方法或 `"WAILS"`
5. `spec.parent` = 源 spec 的 `name`
6. `spec.level` = 源 spec 的 `level`
7. 写入目录：源 spec 同级目录（DATA/STATE 源为 L4 → `L4-feature/`）

**STATE 推导**：
1. 分析 `content.ui_spec.states`（状态列表：loading/empty/error/normal/editing 等）
2. 分析 `content.changes` 中的 `section: state` 区块
3. 分析 `io_contract.behavior` 中涉及状态变更的描述
4. 确定 `persistence`：`memory` / `localStorage` / `sessionStorage`
5. 派生 readers / writers（按 impacted_files 或 behavior 描述推断）
6. `spec.parent` = 源 spec `name`；`spec.level` = 源 spec `level`
7. 写入目录：源 spec 同级目录

**EVT 推导**：
1. 从 `content.ui_spec.acceptance` 中提取：
   - 包含 "成功" / "→" 的行 → 对应 `EVT-*-success` 事件
   - 包含 "失败" / "error" 的行 → 对应 `EVT-*-error` 事件
2. 从 `io_contract.behavior` 提取成功/失败两条路径
3. 同一 UI 操作通常产生 **一对**：success + error
4. 确定事件名格式：`{component}:{subject}:{result}`
5. `spec.parent` = 源 spec `name`；`spec.level` = 源 spec `level`
6. 写入目录：源 spec 同级目录（L4 源 → `L4-feature/`；L5 源 → `L5-slice/`）

### 步骤三：校验与去重

1. 检查目标目录中是否已存在同名派生 spec
2. 如已存在：比较内容，有差异则追加 `changelog` 版本历史，**不覆盖**
3. 如不存在：标记为新建

### 步骤四：确认与写盘

输出完整派生计划（含目录归属）：

```
### 派生计划

| 源 spec | 源层级 | 派生类型 | 文件名 | 写入目录 | 状态 |
|---------|--------|---------|--------|---------|------|
| FEAT-scaffolding | L4 | DATA | DATA-scaffold-init.yaml | L4-feature/ | 新建 |
| FEAT-scaffolding | L4 | STATE | STATE-scaffold-wizard.yaml | L4-feature/ | 新建 |
| FEAT-workspace-session-persistence | L4 | DATA | DATA-workspace-session.yaml | L4-feature/ | 新建 |
| FEAT-workspace-session-persistence | L4 | STATE | STATE-workspace-session.yaml | L4-feature/ | 新建 |
| FEAT-spec-write_L5 | L5 | EVT | EVT-spec-save-success.yaml | L5-slice/ | 新建 |
| FEAT-spec-write_L5 | L5 | EVT | EVT-spec-save-error.yaml | L5-slice/ | 新建 |
```

向用户确认后写盘。

---

## 4. 推导规则

### 4.1 entity/action 提取规则

| UI 描述 | entity | action | DATA 名称 |
|---------|--------|--------|-----------|
| 读取 workspace 列表 | workspace | list | `DATA-workspace-list` |
| 保存 spec 文件 | spec | save | `DATA-spec-save` |
| 初始化脚手架 | scaffold | init | `DATA-scaffold-init` |
| 刷新文件树 | explorer | refresh | `DATA-explorer-refresh` |
| 提交对话消息 | chat | submit | `DATA-chat-submit` |
| 切换 Dock Tab | dock | tab-switch | `DATA-dock-tab-switch` |

### 4.2 state 命名规则

| UI 组件 | store 名称 | STATE 名称 |
|---------|-----------|-----------|
| titlebar | titlebarStore | `STATE-titlebar` |
| scaffold wizard | scaffoldWizardStore | `STATE-scaffold-wizard` |
| workspace session | workspaceSessionStore | `STATE-workspace-session` |
| spec-explorer | specExplorerStore | `STATE-spec-explorer` |
| spec viewer 编辑态 | specViewerStore | `STATE-spec-viewer` |

### 4.3 event 命名规则（格式：`{component}:{subject}:{result}`）

| 触发时机 | 源层级 | 成功事件名 | 失败事件名 | EVT 名称对 |
|---------|--------|-----------|-----------|-----------|
| 保存 spec | L5 | `spec:save:success` | `spec:save:error` | `EVT-spec-save-success` / `EVT-spec-save-error` |
| 初始化脚手架 | L4 | `scaffold:done` | `scaffold:error` | `EVT-scaffold-done` / `EVT-scaffold-error` |
| 点击菜单按钮 | L4 | `menu:click` | — | `EVT-titlebar-menu-click` |
| 切换 activity | L4 | `workbench:activity:switch` | — | `EVT-activitybar-switch` |

**注**：同一源 UI 操作通常产生 **一对** EVT（success + error）。只有纯展示性行为（无失败路径）才只生成一个。

### 4.4 同一源多派生规则

| 场景 | 派生数量 | 处理 |
|------|---------|------|
| `content.ui_spec.acceptance` 中多个条件分支 | 每个分支派生子 EVT | "保存成功 → X" / "失败 → Y" → EVT-spec-save-success + EVT-spec-save-error |
| `io.output` 中多个独立输出 | 每个输出派生子 DATA | 按 entity-action 组合命名 |
| `content.changes` 中多个 state 区块 | 合并到同一个 STATE | 同组件的状态放在一个 STATE |

### 4.5 推导边界

- **只推导，不实现**：不写业务逻辑代码，只生成 spec
- **幂等**：已存在的派生 spec 不覆盖，追加 `changelog` 版本历史
- **严格溯源**：每个派生 spec 必须有 `source_ui_spec` 字段指向源
- **禁止新建平级目录**：只用 `L4-feature/` 和 `L5-slice/`

---

## 5. 示例推导

**源 spec**：`FEAT-spec-write_L5`（L5 slice → `level: 5_slice` → 写入 `L5-slice/`）

**派生计划**：EVT → `L5-slice/EVT-spec-save-success.yaml` + `L5-slice/EVT-spec-save-error.yaml`

→ `L5-slice/EVT-spec-save-success.yaml`：
```yaml
spec:
  level: "5_slice"          # ← 源 spec 的 level
  name: "EVT-spec-save-success"
  parent: "FEAT-spec-write_L5"  # ← 源 spec 的 name
meta:
  type: "event_bus"
source_ui_spec: "FEAT-spec-write_L5"
event:
  name: spec:save:success
  trigger: 用户点击「保存」后，wailsWriteSpecFile 返回成功
  direction: emit
  payload:
    - field: path
      type: string
      description: 保存的 spec 文件相对路径
  consumers:
    - component: SpecViewer
      reaction: 退出编辑模式，切换只读视图
```

→ `L5-slice/EVT-spec-save-error.yaml`：
```yaml
spec:
  level: "5_slice"
  name: "EVT-spec-save-error"
  parent: "FEAT-spec-write_L5"
meta:
  type: "event_bus"
source_ui_spec: "FEAT-spec-write_L5"
event:
  name: spec:save:error
  trigger: 用户点击「保存」后，wailsWriteSpecFile 调用失败
  direction: emit
  payload:
    - field: error
      type: string
      description: 错误信息
  consumers:
    - component: SpecViewer
      reaction: 保留编辑状态，显示 saveErr
```

---

## 6. 禁止事项

- **禁止**创建新平级目录（`L4-data/`、`L4-state/`、`L4-event/` 等）
- 不推导未在源 spec 中明确描述的 DATA/STATE/EVT
- 不写业务逻辑代码
- 不覆盖已存在的派生 spec（只追加版本历史）
- 不推导跨 workspace 的外部 API 契约
