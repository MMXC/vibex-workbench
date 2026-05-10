# VibeX Workbench — Spec ↔ Code 对齐分析报告

> 生成时间：2026-05-10
> 分析工具：./verify_specs --workspace . --format json + 人工代码审查
> Spec 总数：102 个（L1×1, L2×1, L3×6, L4×34, L5×50）

---

## 一、概览

### 1.1 验证结果摘要

| 指标 | 数值 |
|------|------|
| Spec 总数 | 102 |
| 总检查项 | 508 |
| 通过 | 348 (68%) |
| 失败 | 13 (3%) |
| 警告 | 147 (29%) |

**13 个 FAIL 全是 `file_existence` / `content_file_path` 类型** — spec 声明了目标文件路径，但代码中不存在。这是 spec 领先于实现的正常状态，不代表代码有 bug。

**147 个 WARN** 主要来自：
- `impacted_files: []` 为空（L3/L4 模块级 spec 普遍缺失）
- `content.behaviors` 未定义（L4 功能 spec 普遍缺失）
- `lifecycle.current` 为空

### 1.2 Spec 状态分布

```
L1 (1 spec):   proposal (1)
L2 (1 spec):  proposal (1)
L3 (6 specs): proposal (6)       ← 全部未开始
L4 (34 specs): implementation (1), proposal (33)
L5 (50 specs): ready (2), proposal (48)
```

**仅 2 个 L5 处于 ready 状态：**
- `SLICE-spec-explorer-workspace-awareness`
- `SLICE-wails-binding-guard`

**仅 1 个 L4 处于 implementation 状态：**
- `FEAT-workspace-selector`（但 spec status 仍为 proposal，只有 lifecycle.current=implementation）

---

## 二、最小能力五环评估

> 评估标准：小羊提出的「空仓库 → 边做边长」五项最小能力

### 2.1 能力 1：绑定「当前仓库根」

| 维度 | 状态 | 说明 |
|------|------|------|
| 前端输入/选择目录 | ✅ | `/workspace` 页面支持手动输入 + 原生目录选择器 |
| WORKSPACE_ROOT 统一传递 | ✅ | localStorage → 各 API / store 统一注入 |
| 后端 App.SetWorkspaceRoot | ✅ | main.go Wails binding |
| agent 感知 workspace | ⚠️ | agent 从 `cfg.WorkspaceDir` 读取，但未从 frontend 动态更新 |
| SSE bridge 使用 workspace | ⚠️ | agent 读取 workspace 中的 spec，但 spec 不含 workspace root 信息 |

**坏味道：**
- `workspace-session-store.ts` 的 `workspaceRoot` 和 `workbench-layout-store.ts` 的 `workspaceRoot` 是两个独立状态，存在同步风险
- agent 的 workspace root 在启动时固定，不支持运行中切换

**改进建议：**
- 在 SSE 连接建立时，frontend 将 workspaceRoot 作为 query param 传给 agent：`/api/sse/{threadId}?workspaceRoot=...`
- 将 workspaceRoot 提升为全局单例 store，所有组件读取同一份

### 2.2 能力 2：空/半空/就绪三态探测

| 维度 | 状态 | 说明 |
|------|------|------|
| state_detector.py 三态判断 | ✅ | empty / partial / ready，逻辑清晰 |
| UI 显示状态+信号 | ✅ | /workspace 页面 state-card 展示 signals + suggestions |
| `lifecycle.current` 填充 | ❌ | 大量 spec 的 lifecycle.current 为空（verify warn） |
| 图谱/画布感知状态 | ❌ | GenericSpecGraph / GoalSpecCanvas 不读取 state 信息 |

**改进建议：**
- 新增 `make spec-lifecycle-sync` 目标：将 `lifecycle.current` 字段与实际文件状态同步（detect-state 结果回写 spec）
- BuildPanel / StatusBar 读取 workspace state，而非仅依赖用户输入

### 2.3 能力 3：首次落地脚手架

| 维度 | 状态 | 说明 |
|------|------|------|
| scaffolder.py 生成骨架 | ✅ | specs/L1 + gen.py + validate_specs.py + Makefile + README |
| 显式确认流程 | ⚠️ | scaffold 按钮有 confirm，但 confirm=true 静默写入 |
| 前端调用 | ✅ | /workspace 页面 `scaffold()` → POST /api/workspace/scaffold |
| 错误回显 | ⚠️ | 失败时显示 error string，无结构化错误分类 |
| 模板可配置 | ❌ | 只有一个 default 模板，不支持 agent/frontend 等场景化模板 |

**坏味道：**
- scaffolder 的 `confirm` 参数在 UI 层硬编码为 `true`，没有让用户真正确认的 UI 流程
- scaffold 结果不展示具体创建了哪些文件，用户无法审查

**改进建议：**
- scaffold 前展示将要创建的文件预览（dry-run 模式）
- 支持 `--template` 参数：`default | agent-only | frontend-only`

### 2.4 能力 4：规格可读 + 至少一处可编辑

| 维度 | 状态 | 说明 |
|------|------|------|
| Spec 列表展示 | ✅ | /workspace 页面 spec-list 卡片，展示 level/name/状态徽章 |
| Spec 内容读取 | ✅ | ReadSpecFile Wails binding + /api/workspace/specs/read |
| Spec 图谱可视化 | ⚠️ | GenericSpecGraph 存在，但 generateFromSpecs 是 stub |
| Spec 编辑（写盘） | ❌ | WriteSpecFile binding 存在，但没有 UI 调用它 |
| Monaco 编辑器集成 | ❌ | SpecViewer 显示原始文本，无语法高亮 |
| L1 新建向导 | ⚠️ | NewL1Wizard.svelte 存在，但未集成到 /workspace 页面 |

**最关键的缺口：** spec 写闭环完全缺失。用户无法在 UI 中编辑任何 spec。

**改进建议（优先级排序）：**
1. 在 SpecViewer 中加入「编辑」按钮 → 打开 textarea → WriteSpecFile 回写（最快闭环）
2. Monaco editor 集成（spec-to-sveltekit generator 已支持，但未触发）
3. NewL1Wizard 嵌入 /workspace 页面的 spec-list 旁边

### 2.5 能力 5：生成/校验一触即达

| 维度 | 状态 | 说明 |
|------|------|------|
| `make lint-specs` | ✅ | validate_specs.py，YAML 语法 + 必填字段 |
| `make validate` | ✅ | validate_chain.py，parent chain + level consistency |
| `make generate` | ✅ | gen.py，调用各个 generator |
| verify_specs CLI | ✅ | file_existence / parent_chain / completeness / behaviors / go_struct / svelte_props |
| 前端调用 make | ✅ | BuildPanel → runMake() → /api/workspace/run-make |
| 前端展示校验结果 | ⚠️ | BuildPanel 显示 stdout/stderr，但没有结构化解析 |
| 前端 verify-specs 结果 | ⚠️ | /workspace 页面 runVerify() 展示，但无 UI 触发按钮（用户在 workbench 看不到） |

**坏味道：**
- `make generate` 的 gen.py 存在但 `generate` 脚本内容是空的（只有 pass）
- frontend/src/lib/services/code_gen_panel_services.ts 的 runGenerate 是 `throw NotImplemented`

**改进建议：**
- BuildPanel 应解析 make 输出，提取 `[lint-specs]`, `[validate]`, `[generate]` 的阶段结果
- verify-specs 结果应持久化到 `drift-state.json`，供 DriftPanel 使用

---

## 三、功能模块逐项分析

### 3.1 工作区入口（FEAT-workspace-selector）— ⭐ MVP 核心

**Spec 声明（status=implementation）：**
- `/workspace` 页面三态展示
- 原生目录选择器
- 「初始化脚手架」按钮（empty 态）
- 「校验/生成/进入 Workbench」按钮（partial/ready 态）
- localStorage 持久化 + 跳转

**实现情况：**
| 功能 | 状态 | 说明 |
|------|------|------|
| 目录输入+检测 | ✅ | detectState() → /api/workspace/detect-state |
| 原生目录选择器 | ✅ | openDirectoryNativeFirst() → Wails runtime |
| empty 态脚手架按钮 | ✅ | scaffold() → /api/workspace/scaffold |
| partial/ready 态按钮 | ✅ | runMake('validate'/'generate') + enterWorkbench() |
| localStorage 持久化 | ✅ | localStorage.setItem('vibex-workspace-root') |
| Workbench 内菜单打开目录 | ❌ | StatusBar 没有实现 menu:open-project |
| 完整路径校验 | ❌ | 输入相对路径不会被拒绝 |

**坏味道：**
- C5 约束（完整路径校验）在实现中未落地，用户输入 `vibex-workbench` 不会报错
- `ready` 判断只依赖 state，不验证 specs/ 和 gen.py 是否真正存在

**更优实现：**
- 合并 `ready` 判断：`state?.state === 'ready' || state?.state === 'partial'` 改为直接用 state state，但加 path 存在性二次确认

### 3.2 工作区状态检测（FEAT-state-detection）— ✅ 完整实现

**实现情况：**
- `generators/state_detector.py`：empty / partial / ready 三态，signals 数组，suggestions 数组
- `/api/workspace/detect-state` SvelteKit 路由
- 逻辑与 spec 完全对齐

**唯一警告：** `lifecycle.current` 未填充

### 3.3 脚手架生成（FEAT-scaffolding）— ✅ 基础完整，缺确认

**实现情况：**
- `generators/scaffolder.py`：生成 L1 + gen.py + validate_specs.py + Makefile + README
- `/api/workspace/scaffold` SvelteKit 路由
- scaffolder 支持 dry-run（`--dry-run` 参数）

**坏味道：**
- `/workspace` 页面 scaffold 时 confirm=true，绕过 dry-run
- 没有展示生成预览，用户无法知道会创建什么

### 3.4 Make 集成（FEAT-make-integration）— ✅ 完整实现

**实现情况：**
- Wails binding：`RunMake(target, workspace)` → main.go
- `/api/workspace/run-make` SvelteKit 路由
- BuildPanel 显示 output + exitCode badge
- Makefile 有 `lint-specs`, `validate`, `generate` 目标

**警告：** `make generate` 调用的 gen.py 是空壳

### 3.5 Spec 校验与验证（FEAT-mvp-acceptance-gates）— ⚠️ 工具就绪，UI 断链

**实现情况：**
| 组件 | 状态 |
|------|------|
| `./verify_specs` CLI | ✅ 6 类检查（508 checks，13 fail，348 pass） |
| Wails binding | ✅ `VerifySpecs()` |
| `/api/workspace/verify-specs` 路由 | ✅ |
| Workbench BuildPanel 集成 | ❌ BuildPanel 不调用 verify |
| /workspace 页面 | ⚠️ `runVerify()` 存在但需要手动触发 |
| 结果持久化到 drift-state.json | ❌ |

**关键缺口：**
- verify_specs 不写回 `lifecycle.verification` 字段
- verify 结果不同步到 DriftPanel

### 3.6 Spec 编辑（FEAT-spec-write）— ⚠️ 初稿过时；以 §九复核为准

**实现情况（2026-05-10 复核）：**
| 组件 | 状态 |
|------|------|
| Wails WriteSpecFile | ✅ |
| HTTP `POST /api/workspace/specs/write` | ⚠️ **无** SvelteKit `+server.ts`；由 **`main.go` 反代** 至 agent `workspaceSpecWriteHandler`，**浏览器 dev** 下 `wailsWriteSpecFile` 走该路径 |
| SpecViewer 编辑 + Monaco | ✅ **已实现**（`SpecViewer.svelte`：`wailsWriteSpecFile` + Monaco） |
| NewL1Wizard | ⚠️ 组件存在但未嵌入 `/workspace` |
| 写盘后列表/图谱刷新 | ⚠️ 依赖既有刷新链路，非全自动 |

**初稿「完全没有 UI / 只能看不能改」已不再成立；** 仍缺统一 SK 独占路由与体验打磨，详见 **§九**。

### 3.7 图谱可视化（FEAT-canvas-expand + FEAT-spec-graph-expansion）— ⚠️ 骨架存在，核心逻辑 stub

**实现情况：**
| 组件 | 状态 |
|------|------|
| GenericSpecGraph.svelte | ⚠️ 存在，但 fetch 端点不存在 |
| GoalSpecCanvas.svelte | ⚠️ 基础渲染，展开功能缺失 |
| CanvasRenderer.svelte | ⚠️ skeleton 文件存在 |
| generateFromGraph | ❌ NotImplemented |
| generateFromSpecs | ❌ NotImplemented |
| toCanvasNodes | ❌ NotImplemented |
| addChildTrigger | ❌ ClarificationPanel 未嵌入 |
| SpecTreeMap | ⚠️ SLICE-spec-tree-map 组件存在 |

**坏味道：**
- GenericSpecGraph 第 367 行 fetch 的是 `${SSE_URL}/api/agent/execute`（agent 端点），不是 spec 端点
- Canvas store 有两个版本：`canvas-store.ts` 和 `canvasStore.ts`（重复）
- `dsl-canvas-store.ts` 和 `dslCanvas.ts` 又是另一套 DSL 风格

### 3.8 规范查看器（FEAT-spec-editor）— ⚠️ 展示就绪，编辑缺失

**实现情况：**
| 组件 | 状态 |
|------|------|
| SpecExplorer.svelte | ⚠️ 显示 spec 树，但无 click→展开逻辑 |
| SpecViewer.svelte | ⚠️ 显示 spec 内容，无编辑模式 |
| SpecDetailPanel.svelte | ⚠️ panel 形式，内容展示 |
| SpecInteractiveMarkdown | ⚠️ Markdown 渲染，有 checkbox 交互 |
| spec-explorer-store.ts | ✅ 状态管理 |

**坏味道：**
- SpecExplorer 没有搜索/过滤功能（spec 超过 100 个后会难以浏览）
- SpecViewer 和 SpecDetailPanel 功能重叠，可能合并

### 3.9 构建面板（FEAT-build-panel）— ⚠️ 基础可用，problems 面板缺失

**实现情况：**
- BuildPanel.svelte 显示 make output + exitCode
- `runMake()` 调用 `/api/workspace/run-make`
- FEAT-command-output-problems-panel 的 problems 显示未实现

**坏味道：**
- BuildPanel 没有 tabs（lint-specs / validate / generate / verify 混在一起）
- 没有输出折叠，长输出会撑满整个 panel

### 3.10 Spec 历史（FEAT-spec-history-rollback）— ❌ UI 存在，后端缺失

**实现情况：**
- `SpecHistoryPanel.svelte`（242 行）存在
- 调用 `/api/workspace/specs/history?specPath=...`
- **路由不存在**，所有 history 操作都失败

### 3.11 Spec Drift 检测（FEAT-drift-panel）— ❌ UI 存在，后端缺失

**实现情况：**
- `DriftPanel.svelte`（304 行）存在，读取 `/api/spec/drift/list`
- `agent/vibex/domain/spec/drift.go`（Go DriftEngine）存在
- **SvelteKit 路由不存在**（/api/spec/drift/* vs /api/workspace/specs/drift/* 路径不匹配）

**坏味道：**
- DriftEngine 在 Go agent 侧，但 UI 通过 SvelteKit 调用，两者不在同一进程
- drift-state.json 路径约定缺失

### 3.12 Go Agent 工具集 — ⚠️ 基础框架完整，深度功能缺失

**已实现：**
- `spec_designer` — spec_designer.go（spec 新建）
- `spec_feature` — spec_feature.go（L4 拆解）
- `spec_validate` — validate_specs.py 包装
- `make_validate` — make validate 包装
- `canvas_update` — SSE canvas 更新
- `tdd_design` — io_contract → test cases
- `tdd_run` — 测试执行

**缺失：**
- `spec_sync` — spec ↔ code 双向同步
- `spec_result_track` — 结果追踪
- agent task state machine（FEAT-agent-task-state-machine）：spec 存在，代码完全不存在
- agent tool routing graph（FEAT-agent-tool-routing-graph）：spec 存在，`agent/vibex/domain/toolrouting/` 部分实现
- agent rules engine（FEAT-agent-rules-engine）：`agent/vibex/domain/rulesengine/` 部分实现，未集成到 tool loop

### 3.13 Design Kit 原型系统 — ⚠️ HTTP API 完整，拖拽提取缺失

**已实现（SLICE-workspace-design-kit-http-api）：**
- `pkg/designkit/designkit.go`：DesignKitStatus / Scaffold / Extract
- Wails binding：`App.DesignKitStatus/Scaffold/Extract`
- SvelteKit 路由：`/api/workspace/design-kit/{scaffold,extract,status}`

**缺失：**
- 原生 OS 拖拽文件到 workbench → 自动 extract
- `SLICE-workspace-design-kit-http-api` spec 说"支持本地图片拖拽提取"，未实现

### 3.14 IDE Chrome — ⚠️ 组件就绪，集成不完整

**已实现：**
- ActivityBar.svelte、WorkbenchTitlebar.svelte、WorkbenchCenterTabs.svelte、R2Dock.svelte、StatusBar.svelte、AiChatColumn.svelte
- WorkbenchLayoutResizable.svelte（可拖拽 resize）

**缺失/不完整：**
- TitleBar 的窗口控制按钮（最小化/最大化/关闭）— 依赖 Wails runtime，可能未连接
- menu:open-project 事件 → workspace root 更新 → StatusBar 刷新：未实现
- ActivityBar 的 spec explorer / canvas 等图标点击 → 切换 center view：未连接
- AiChatColumn 的 agent 对话流：SSE 连接但响应格式未确认

### 3.15 Slot 系统（SpecSlot*）— ⚠️ 复杂，集成渐进中

**已实现：**
- SpecSlotA2UIStage.svelte（490 行）、SpecSlotVisualPane（458 行）、SpecSlotChatPane（284 行）、SpecSlotDrawer（173 行）、SpecSlotPrototypeKitBar（308 行）
- spec-slot-session-store.ts（687 行）：状态管理

**问题（2026-05-10 复核）：**
- **`SpecSlotDrawer` 已在 `workbench/+page.svelte` 中使用**；初稿「完全未 import」不准确
- 其余 Slot 子组件嵌入深度仍不均衡；session 状态来源需持续收敛
- 设计面较大，MVP 需控制范围

---

## 四、代码质量审查

### 4.1 重复代码

| 位置 | 问题 |
|------|------|
| `canvas-store.ts` + `canvasStore.ts` | 两套 canvas store，功能重叠 |
| `dsl-canvas-store.ts` + `dslCanvas.ts` | 同上，DSL 风格版本 |
| `frontend/src/lib/stores/` 中多个 `*Store.ts` | 没有统一 store 约定，`.ts` vs `.svelte.ts` 混用 |
| `/api/spec/drift/*` vs `/api/workspace/specs/drift/*` | DriftPanel 用前者，Go agent 用后者，路径不统一 |

### 4.2 未实现的服务层

```
frontend/src/lib/services/code_gen_panel_services.ts:
  ❌ runGenerate() → throw NotImplemented
  ❌ getDiff() → throw NotImplemented

frontend/src/lib/services/dsl_canvas_services.ts:
  ❌ generateFromGraph() → throw NotImplemented
  ❌ generateFromSpecs() → throw NotImplemented
  ❌ toCanvasNodes() → throw NotImplemented
  ❌ autoLayout() → throw NotImplemented
  ❌ fitToScreen() → throw NotImplemented
  ❌ buildSequences() → throw NotImplemented

frontend/src/lib/services/routing_panel_services.ts:
  ⚠️ 未检查
```

这些 NotImplemented 服务被组件引用但会 panic，是运行时隐患。

### 4.3 YAML 质量问题（12 个文件）

**问题类型 1：Trailing empty document（12 个文件）**
```
specs/L4-feature/FEAT-agent-rules-engine.yaml
specs/L4-feature/FEAT-agent-task-state-machine.yaml
specs/L4-feature/FEAT-agent-tool-routing-graph.yaml
specs/L4-feature/FEAT-workspace-design-prototype-gate.yaml
specs/L5-slice/SLICE-agent-custom-tool-registry.yaml
specs/L5-slice/SLICE-agent-plan-graph-router-api.yaml
...（共 12 个）
```
每个文件末尾有多余的 `---`，被 `yaml.safe_load()` 解析为两个文档（第二个空），导致 `safe_load_all` 返回空 doc。这些文件目前能工作（Go verify 用的 `safe_load` 拒绝多文档），需要清理。

**问题类型 2：缺失 spec.status（8 个文件）**
```
specs/L5-slice/SLICE-build-panel-output-display.yaml
specs/L5-slice/SLICE-canvas-expand-add-child-trigger.yaml
specs/L5-slice/SLICE-canvas-expand-children-render.yaml
specs/L5-slice/SLICE-canvas-expand-spec-exists-conflict.yaml
specs/L5-slice/SLICE-mvp-governance-status-update.yaml
specs/L5-slice/SLICE-qa-playwright-scenario-runner.yaml
specs/L5-slice/SLICE-workspace-lifecycle-binding.yaml
specs/L5-slice/SLICE-workspace-lifecycle-partial-scaffold.yaml
```
`spec.status` 字段完全缺失（prototype.status 存在但不是 spec.status）。

**问题类型 3：lifecycle.current 普遍为空**
verify 工具报告 147 个 `completeness` 警告，大量 spec 的 `lifecycle.current` 未填充。

### 4.4 架构不一致

| 问题 | 说明 |
|------|------|
| Workspace API 在两个地方 | SvelteKit routes (`/api/workspace/*`) + Go agent (`agent/cmd/web/server.go`)，两套路由体系 |
| Agent SSE vs Wails binding | Agent SSE 用于 chat，SvelteKit routes 用于 workspace API，Wails 用于 native 调用，边界模糊 |
| Verify 结果不驱动 Drift | verify_specs 输出 13 fail，但 DriftPanel 不读取这些结果 |
| Spec read 路径不一致 | Wails: `ReadSpecFile(root, path)` vs SvelteKit: `/api/workspace/specs/read?workspaceRoot=&path=` |

---

## 五、Spec 覆盖矩阵

### 5.1 按模块

| 模块 | Spec 数 | 实现度 | 说明 |
|------|---------|--------|------|
| workspace-root | L3×1 + L4×3 + L5×3 | 80% | selector✅ state✅ lifecycle⚠️ |
| scaffolding | L3×1 + L4×1 + L5×3 | 75% | scaffolder✅ preview❌ confirm⚠️ |
| build-panel | L3×1 + L4×2 + L5×1 | 60% | runMake✅ problems❌ output⚠️ |
| spec-editor | L3×1 + L4×14 + L5×15 | 30% | read✅ write❌ history❌ |
| canvas/graph | L4×2 + L5×10 | 20% | skeleton✅ core❌ |
| ide-chrome | L3×1 + L4×6 + L5×6 | 50% | components✅ wiring❌ |
| agent | L3×1 + L4×7 + L5×20 | 40% | tools✅ routing❌ state-machine❌ |
| governance | L4×1 + L5×5 | 15% | viewer⚠️ auto-update❌ |
| design-kit | L4×1 + L5×2 | 60% | HTTP API✅ drag❌ |

### 5.2 按 spec 层级

| 层级 | 总数 | ✅完整 | ⚠️部分 | ❌未开始 |
|------|------|--------|--------|----------|
| L1 (goal) | 1 | 0 | 1 | 0 |
| L2 (skeleton) | 1 | 0 | 1 | 0 |
| L3 (module) | 6 | 0 | 3 | 3 |
| L4 (feature) | 34 | 1 | 10 | 23 |
| L5 (slice) | 50 | 2 | 12 | 36 |

---

## 六、后续路线

### 6.1 P0 — 让 MVP 真正可闭环（当前 session 优先级）

```
1. [5 min] 修复 trailing empty doc YAML（12 个文件，删除末尾 ---）
2. [5 min] 修复缺失 spec.status（8 个文件，补 status: proposal）
3. [30 min] 在 SpecViewer 中加「编辑模式」按钮 → textarea → WriteSpecFile
   → 核心价值：用户可以改 spec 了
4. [15 min] BuildPanel 加 verify-specs tab → 调用 /api/workspace/verify-specs
5. [10 min] scaffold confirm=true 改为 dry-run preview（用户可见将创建的文件）
```

### 6.2 P1 — 消除 NotImplemented（下一阶段）

```
1. [1h] gen.py 补全 generate 逻辑（目前是空 pass）
2. [30 min] /api/workspace/specs/write 路由实现（WriteSpecFile 包装）
3. [30 min] /api/workspace/specs/history 路由实现（git log spec-file）
4. [1h] dsl_canvas_services 核心方法实现（generateFromSpecs → CanvasNodes）
5. [30 min] Canvas addChildTrigger → ClarificationPanel 集成
```

### 6.3 P2 — 产品体验打磨

```
1. Monaco editor 集成到 SpecViewer（语法高亮 + YAML validation）
2. SpecExplorer 搜索/过滤
3. BuildPanel output 折叠 + 语法高亮
4. DriftPanel 接入 verify 结果
5. Agent workspace root 动态更新（SSE query param）
```

### 6.4 P3 — 高级功能

```
1. Canvas auto-layout + fit-to-screen
2. Agent tool routing graph 可视化
3. QA Playwright 集成
4. Implementation journal
5. Agent task state machine
```

---

## 七、YAML 修复脚本

```python
#!/usr/bin/env python3
"""修复 trailing empty doc 和缺失 spec.status"""
import os, sys, re

BASE = '/home/agent/vibex-workbench/specs'

# 1. 修复 trailing empty doc（12 个文件）
trailing_empty = [
    'specs/L4-feature/FEAT-agent-rules-engine.yaml',
    'specs/L4-feature/FEAT-agent-task-state-machine.yaml',
    'specs/L4-feature/FEAT-agent-tool-routing-graph.yaml',
    'specs/L4-feature/FEAT-workspace-design-prototype-gate.yaml',
    'specs/L5-slice/SLICE-agent-custom-tool-registry.yaml',
    'specs/L5-slice/SLICE-agent-plan-graph-router-api.yaml',
    'specs/L5-slice/SLICE-agent-repair-orchestrator-runtime.yaml',
    'specs/L5-slice/SLICE-agent-rules-engine-contract.yaml',
    'specs/L5-slice/SLICE-agent-task-state-machine-runtime.yaml',
    'specs/L5-slice/SLICE-spec-slot-prototype-kit-ui.yaml',
    'specs/L5-slice/SLICE-spec-slot-verification-bar.yaml',
    'specs/L5-slice/SLICE-workspace-design-kit-http-api.yaml',
]

for rel in trailing_empty:
    fpath = os.path.join(BASE, '..', rel)
    with open(fpath) as f:
        content = f.read()
    # 去掉末尾的 \n---\n
    fixed = content.rstrip()
    if fixed.endswith('---'):
        fixed = fixed[:-3].rstrip()
    if fixed != content:
        with open(fpath, 'w') as f:
            f.write(fixed + '\n')
        print(f'Fixed: {rel}')

# 2. 修复缺失 spec.status（8 个文件）
missing_status = [
    'specs/L5-slice/SLICE-build-panel-output-display.yaml',
    'specs/L5-slice/SLICE-canvas-expand-add-child-trigger.yaml',
    'specs/L5-slice/SLICE-canvas-expand-children-render.yaml',
    'specs/L5-slice/SLICE-canvas-expand-spec-exists-conflict.yaml',
    'specs/L5-slice/SLICE-mvp-governance-status-update.yaml',
    'specs/L5-slice/SLICE-qa-playwright-scenario-runner.yaml',
    'specs/L5-slice/SLICE-workspace-lifecycle-binding.yaml',
    'specs/L5-slice/SLICE-workspace-lifecycle-partial-scaffold.yaml',
]

for rel in missing_status:
    fpath = os.path.join(BASE, '..', rel)
    with open(fpath) as f:
        content = f.read()
    # 在 spec block 的 name 后面加 status
    # 匹配: spec:\n  name: XXX\n  level: ...
    # 改为: spec:\n  name: XXX\n  status: proposal\n  level: ...
    fixed = re.sub(
        r'(^spec:\n  name: .+?\n)(  level:)',
        r'\1  status: proposal\n\2',
        content,
        flags=re.MULTILINE
    )
    if fixed != content:
        with open(fpath, 'w') as f:
            f.write(fixed)
        print(f'Fixed status: {rel}')
```

---

## 八、总结

### 8.1 好在哪里

1. **五项最小能力的基础层完整**：workspace root 绑定、三态检测、脚手架生成、spec 读取、make 调用全部通了
2. **verify_specs CLI 工具质量高**：6 类检查，覆盖全面，结果准确
3. **Wails binding 规范**：统一的 `map[string]any` 选项格式，`isWails()` guard，`window.go.main.App.*` 约定
4. **SvelteKit API routes 架构清晰**：9 个路由覆盖 workspace 核心操作，职责分明
5. **Go agent 工具框架成熟**：spec_designer / spec_feature / make_validate / tdd_* 工具集完整

### 8.2 主要问题

1. **Spec 写闭环缺失**（MVP 最大缺口）
2. **多个 NotImplemented 服务层**（代码能跑但核心功能 stub）
3. **YAML 质量 20+ 处问题**（12 trailing doc + 8 missing status）
4. **Canvas / Graph 可视化** 核心逻辑全 stub
5. **Drift / History API** 前端有 UI 后端无路由
6. **Slot 系统** 未集成，可能是过度工程
7. **重复 store**（canvas×2, dsl-canvas×2）
8. **gen.py 是空壳**（make generate 实际上什么都没做）

### 8.3 推荐行动

**立即（5 min）：** 修复 12 个 trailing empty doc + 8 个 missing spec.status

**本周（2h）：** SpecViewer 加编辑按钮 + BuildPanel 加 verify tab → MVP 五环闭环

**下一阶段（1-2 days）：** gen.py 补全 → Canvas generateFromSpecs → spec history → API routes 补全

---

## 九、代码库复核（相对 §三初稿，2026-05-10）

以下逐项对照当前仓库 **复核**：报告初稿中部分结论已过时或与命名不符，其余缺口仍以代码为准。

### 9.1 初稿表述不准确（已更正理解）

| 初稿结论 | 复核结果 |
|----------|----------|
| 「Slot 系统在 workbench/+page 中未被 import」 | **不准确**：`workbench/+page.svelte` 已 import 并使用 **`SpecSlotDrawer`**（槽位抽屉）；其余 Slot 组件集成度仍不均衡 |
| 「workbench-layout-store 与 workspace-session-store 双份 workspaceRoot」 | **不准确**：**`workbench-layout-store.ts` 仅存布局像素**，不含 `workspaceRoot`。仍存在 **`workspace-session-store`** 与 **`spec-explorer-store`** 两套路径状态，同步风险仍在 |
| 「Spec 写闭环完全缺失 / SpecViewer 仅 `<pre>`」 | **不准确**：**`SpecViewer.svelte`** 已实现 **编辑模式 + Monaco + `wailsWriteSpecFile`**（浏览器 dev 下走 `POST /api/workspace/specs/write` 同源代理） |
| 「gen.py 只有 pass、make generate 空壳」 | **不准确**：根目录 **`generators/gen.py`** 为完整生成器（千行级）；若 Makefile 某目标仍指向空脚本，需对照具体 Makefile 目标而非断定 gen.py 为空 |
| 「menu:open-project 未实现 / StatusBar 未接」 | **部分不准确**：**`+layout.svelte`** 已 **`eventsOn('menu:open-project', handleOpenProject)`**；Workbench 标题栏与 StatusBar 是否每次刷新显示需与具体组件对齐，但事件链路存在 |

### 9.2 初稿结论仍成立（建议 spec 继续标的缺口）

| 主题 | 复核要点 |
|------|----------|
| **SvelteKit 专用路由** | 无 SvelteKit `frontend/src/routes/api/workspace/specs/write/+server.ts`：写请求由 **`wails-filesystem`** 走 **Wails `WriteSpecFile`** 或 **`fetch('/api/workspace/specs/write')`** 由 **`main.go` appHandler 反代** 至 agent **`workspaceSpecWriteHandler`**；spec 不应写死「仅 SK 路由」 |
| **FEAT-workspace-selector C5** | **`/workspace`** 输入框 **未见**「非完整路径一律拒绝」的严格校验；与约束 C5 仍有差距 |
| **dsl_canvas_services / routing_panel_services** | **`NotImplemented` 仍存在**，运行时若调用会抛错 |
| **BuildPanel** | **未集成 verify-specs**；**`/workspace`** 页面仍有 **`runVerify()` + 按钮** |
| **SpecHistoryPanel / DriftPanel** | 仍请求 **`/api/workspace/specs/history`**、**`/api/spec/drift/*`**；**未见**对应 **`frontend/src/routes/api/**`** 实现文件 → **UI 与后端路由缺口仍在** |
| **GenericSpecGraph** | 仍存在经 **`agentApiUrl('/api/agent/execute')`** 的路径；图谱生成管线与「纯 spec API」混用问题仍在 |

### 9.3 建议 spec 文档用语（与本仓库传输模型一致）

- **spec 写入**：表述为 **「Wails `WriteSpecFile` 优先，其次同源 HTTP `/api/workspace/specs/write`（由桌面壳反代至 agent）」**，避免要求「必须存在 SK `specs/write/+server.ts`」。
- **VERIFY**：区分 **入口页 `/workspace` 上的 verify** 与 **Workbench BuildPanel** 是否调用 verify；分别立项验收。
