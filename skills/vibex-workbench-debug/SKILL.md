---
name: vibex-workbench-debug
description: vibex-workbench 本地桌面工作台常见问题调试技能
category: devops
---
repo_tracked: true


# vibex-workbench-debug

## Terminal Deadlock

**症状**：terminal 所有命令立即返回 exit 130 (SIGINT)，所有工具调用中断。

**根因**：在当前 session 用 `&` 启动了 `vibex-agent-web` 后台进程。SIGINT 传染给所有后续命令。

**解法**：从外部 terminal 执行 `pkill -9 vibex-agent-web`。不要在同一 session 试图杀进程。

**防御**：用 `nohup ./vibex-agent-web > /tmp/log 2>&1 &` 替代直接 `&`，或用 screen/tmux 隔离。

## Workspace Root 不生效

**症状**：API 总是用默认路径 `/root/vibex-workbench`，忽略用户传入的 `workspace_root`。

**根因**：Go struct JSON tag 驼峰 vs 前端 TypeScript 下划线，unmarshal 静默失败。

```go
// 错
WorkspaceRoot string `json:"workspaceRoot"`
// 对
WorkspaceRoot string `json:"workspace_root"`
```

**修复**：`sed -i 's/`json:"workspaceRoot"`/`json:"workspace_root"`/' agent/cmd/web/workspace_handlers.go`

## 编译错误：重复声明

**症状**：`redeclared in this block`

**场景**：`handlers.go` 和 `handlers_workspace.go` 都有同名函数（如 `MakeWorkspaceDetectStateHandler`）。保留 `handlers_workspace.go` 的原生 Go 版本，删掉 `handlers.go` 里的重复块。

## API 404 但 handler 已注册

**根因**：改了代码但运行的是旧 binary。

**解法**：`pkill vibex-agent-web && go build -o vibex-agent-web ./cmd/web/ && ./vibex-agent-web &`

## Coverage Report 漏计 L5

**根因**：`spec_coverage.py` 只认 `level: 5_slice`，漏 `level: 5_implementation`。

**修复**：两处改为 `('5_slice', '5_implementation')`。


## 核心架构

### Spec 层级结构
```
L1 project-goal/        ← vibex-workbench-goal.yaml
L2 architecture/         ← vibex-workbench-skeleton.yaml (唯一 L2)
L3 module/               ← specs/module/MOD-<name>_module.yaml（5 个独立文件！）
L4 feature/              ← specs/feature/<name>/<name>_feature.yaml
L5 sub-specs/            ← <name>_uiux/service/data/test.yaml
```

### ⚠️ 关键陷阱：parent 引用规则

**两种 spec 结构**（区分 vibex-spec vs vibex-workbench）：
- `vibex-spec/specs/` — 14 个文件，无 L3 module 层，feature 直接 parent L2
- `vibex-workbench/specs/` — 32 个文件，有 L3 module 层，feature parent MOD-* 文件

**旧版 validator 的 bug**（LEVEL_PARENT 硬编码错误）：
```python
# ❌ 旧版（错误）
LEVEL_PARENT = {
    "4_feature": "2_architecture",   # ← 硬编码 feature → L2
}
# validator 在 specs/architecture/ 查找 MOD-xxx，找不到 → 5 个错误

# ✅ 新版（正确）
LEVEL_PARENT = {
    "4_feature": "3_module",   # ← feature → L3 module
}
# get_spec_path 识别 MOD-* 前缀 → 路由到 specs/module/MOD-<name>_module.yaml
```

### 当前 spec 结构（L3 module 层已独立文件）

```
specs/
├── architecture/vibex-workbench-skeleton.yaml
├── module/MOD-code-generator_module.yaml
├── module/MOD-dsl-visualizer_module.yaml
├── module/MOD-router_module.yaml
├── module/MOD-spec-engine_module.yaml
├── module/MOD-workbench-shell_module.yaml
└── feature/
    ├── spec-editor/spec-editor_feature.yaml  parent: MOD-spec-engine
    ├── workbench-shell/workbench-shell_feature.yaml  parent: MOD-workbench-shell
    ├── dsl-canvas/dsl-canvas_feature.yaml  parent: MOD-dsl-visualizer
    ├── code-gen-panel/code-gen-panel_feature.yaml  parent: MOD-code-generator
    └── routing-panel/routing-panel_feature.yaml  parent: MOD-router
```

## 修复流程（2026-04-20 心跳验证）

### Step 1: 诊断是哪种失败模式
```bash
cd /root/vibex-workbench && make lint-specs 2>&1
# 模式A: "parent 'MOD-xxx' 未找到" → 修 validate_specs.py
# 模式B: 32 个 specs 都有 parent 错误 → validate_specs.py LEVEL_PARENT 需要改
```

### Step 2: 检查当前 validator 状态
```bash
grep -A2 '"4_feature":' /root/vibex-workbench/generators/validate_specs.py
# 期望: "4_feature": "3_module"
```

### Step 3: 如果是模式A（validator 未更新），修复 validate_specs.py
```python
# 文件: /root/vibex-workbench/generators/validate_specs.py

# 修改1: LEVEL_PARENT
LEVEL_PARENT = {
    "2_architecture": "1_project_goal",
    "3_module": "2_architecture",
    "4_feature": "3_module",     # ← 从 "2_architecture" 改为 "3_module"
    "5a_uiux": "4_feature",
    "5b_service": "4_feature",
    "5c_data": "4_feature",
    "5d_test": "4_feature",
}

# 修改2: get_spec_path() 新增 L3 module + MOD-* 解析
def get_spec_path(level: str, name: str) -> Path:
    ...
    elif level == "3_module":
        # MOD-* module specs live in specs/module/
        if name.startswith("MOD-"):
            return SPEC_DIR / "module" / f"{name}_module.yaml"
        return SPEC_DIR / "module" / f"{name}_module.yaml"
    elif level.startswith("4_") or level == "4_feature":
        # MOD-* parent 映射到 L3 module 目录
        if name.startswith("MOD-"):
            return SPEC_DIR / "module" / f"{name}_module.yaml"
        return SPEC_DIR / "feature" / name / f"{name}_feature.yaml"
    ...
```

### Step 4: 验证修复
```bash
cd /root/vibex-workbench
python3 generators/validate_specs.py specs
# 期望: ✅ 所有 Spec 验证通过！（32 个）
make validate
# 期望: ✅ 深度验证通过
```

## 组件骨架问题

### 区分 stub vs 真实组件
```bash
cd /root/vibex-workbench/frontend/src/lib/generated/components
for f in *.svelte; do
  lines=$(wc -l < "$f")
  if [ "$lines" -lt 50 ]; then
    echo "⚠️  STUB: $f ($lines 行)"
  fi
done
```

**当前组件状态**（2026-04-20 心跳 + 新实现）：
```
generated/components/ 骨架（14个）:
  CanvasToolbar.svelte        246b   → stub
  DiffViewer.svelte           227b   → stub
  GenerationTimeline.svelte    288b   → stub
  MermaidNode.svelte          252b   → stub
  SpecRecommendationList.svelte 275b  → stub
  StatusBar.svelte            230b   → stub
  TabBar.svelte               218b   → stub
  ValidationBadge.svelte       262b   → stub
  WorkbenchLayout.svelte      247b   → stub
  SpecEditor.svelte           916b   → stub/占位

workbench/ 真实组件（开发者维护，不被gen.py覆盖，6个）:
  WorkbenchShell.svelte       1222b  — 三栏布局外壳
  Composer.svelte             2402b  — 多模态输入框
  ArtifactPanel.svelte        1994b  — AI产出面板
  ThreadList.svelte           2434b  — 线程列表
  (需确认其他组件)

⚠️ 关键：types.ts + api.ts + 5个 stores 由 spec-to-sveltekit 生成，修改需先更新 YAML 再重新 generate
⚠️ 已知：生成文件常缺类型定义，详见上方 "TypeScript 编译失败" 章节
```

### 实现新组件的方法

**Step 1: 如需第三方库**（在 frontend 目录执行）
```bash
cd /root/vibex-workbench/frontend && npm install package-name
```

**Step 2: 写 Svelte 组件**
⚠️ 不要用 `write_file` / `execute_code` — Svelte 的 `<script module>` 块会触发 heredoc 解析错误。
**用 terminal heredoc**:
```bash
cat > /root/vibex-workbench/frontend/src/lib/generated/components/ComponentName.svelte << 'ENDOFFILE'
<script lang="ts">
  import { onMount } from 'svelte';
  // content
</script>
<!-- template -->
<style>
  /* styles */
</style>
ENDOFFILE
```

**Step 3: 验证 build**
```bash
cd /root/vibex-workbench/frontend && npm run build 2>&1 | tail -5
```

**Step 4: 检查文件大小**
```bash
wc -c /root/vibex-workbench/frontend/src/lib/generated/components/ComponentName.svelte
# 真实实现 > 500 字节
```
### 快速判断 Generator 是否可用
```bash
cd /root/vibex-workbench
for f in generators/spec-to-*/gen.py; do
  size=$(wc -c < "$f")
  echo "$(basename $(dirname $f)): $size bytes"
done
# 真实实现: > 5000 bytes
# 骨架/空实现: < 1000 bytes
```

## 架构设计洞察

### 为什么模块内联在 L2？
`vibex-workbench-skeleton.yaml` 定义了 5 个模块作为内联列表（不是独立文件），这是设计选择：
- 模块数量少（5 个），不需要独立文件
- 但 validator 只认文件路径，不理解内联定义
- 修复：L4 feature 直接 parent 到 L2 文件名

### L3 module 层去哪了？
- 在架构文档中是 "3_module" level
- 但实际没有 `specs/module/` 目录
- validator 能识别 level `"3_module"` 但找不到独立文件
- 当前所有 L4 feature 都绕过 L3，直接 parent 到 L2

## TypeScript 编译失败（spec-to-sveltekit 生成文件常见问题）

### 症状：`npx tsc --noEmit` 报错，build 失败

**根因**：`spec-to-sveltekit` 生成的 `api.ts`、`stores/*.ts`、`types.ts` 引用了 YAML spec 中声明但本地 `types.ts` 未定义的类型。

**诊断**：
```bash
cd /root/vibex-workbench/frontend
npx tsc --noEmit 2>&1 | grep 'error TS' | head -20
```

**已知缺失类型及修复**：

| 缺失类型 | 修复位置 | 内容 |
|----------|----------|------|
| `DependencyGraph`, `Change` | `types.ts` | `export interface DependencyGraph { nodes: SpecNode[]; edges: SpecEdge[] }` |
| `GenerationJobCreate/Update` | `types.ts` | 见下方完整类型块 |
| `FileDiff`, `ImpactReport`, `SpecAST`, `Viewport` | `types.ts` | 见下方完整类型块 |
| `ViolationCreate`, `StepLog`, `Step` | `types.ts` | 添加缺失 interface |
| `DBArtifact`, `threads` table | `db.ts` | 添加 Dexie table + type export |
| `Tab` | `types.ts` | `export interface Tab { id?: string; specFileId?: string; label?: string; isActive?: boolean; }` |

**修复 types.ts（追加到文件末尾）**：
```typescript
// types.ts 缺失类型补充块
export interface Step {
  id?: string; name?: string;
  status?: 'pending' | 'running' | 'done' | 'error';
  log?: string; startedAt?: Date; completedAt?: Date | null;
}
export interface StepLog {
  step: string; status: 'pending' | 'running' | 'done' | 'error';
  log: string; timestamp?: Date;
}
export interface GenerationJobCreate { goal?: string; threadId?: string; }
export interface GenerationJobUpdate { id: string; status?: 'running' | 'done' | 'error'; steps?: Step[]; }
export interface FileDiff { id?: string; filePath?: string; oldContent?: string | null; newContent?: string | null; status?: FileDiffStatus; hunks?: DiffHunk[]; }
export type FileDiffStatus = 'added' | 'modified' | 'deleted' | 'unchanged';
export interface DiffHunk { oldStart: number; oldLines: number; newStart: number; newLines: number; lines: string[]; }
export interface FileDiffCreate { filePath: string; oldContent?: string; newContent?: string; status?: FileDiffStatus; }
export interface FileDiffUpdate { id: string; oldContent?: string; newContent?: string; status?: FileDiffStatus; }
export interface ImpactReport { id?: string; specId?: string; affectedFiles?: string[]; breakingChanges?: BreakingChange[]; createdAt?: Date; }
export interface BreakingChange { file: string; type: 'api' | 'ui' | 'data' | 'behavior'; description: string; severity?: 'low' | 'medium' | 'high'; }
export interface ImpactReportCreate { specId: string; affectedFiles?: string[]; }
export interface ImpactReportUpdate { id: string; affectedFiles?: string[]; breakingChanges?: BreakingChange[]; }
export interface SpecAST { id?: string; specId?: string; content?: string; parsed?: unknown; level?: number; parentChain?: string[]; }
export interface SpecASTCreate { specId: string; content: string; level?: number; }
export interface SpecASTUpdate { id: string; content?: string; }
export interface SpecFileCreate { path: string; content?: string; level?: number; name?: string; parent?: string; }
export interface SpecFileUpdate { id: string; content?: string; name?: string; status?: 'active' | 'draft' | 'deprecated'; }
export interface SpecLocationCreate { specId: string; level?: number; confidence?: number; reason?: string; }
export interface SpecLocationUpdate { id: string; confidence?: number; }
export interface SpecNodeCreate { specId: string; level?: number; name?: string; x?: number; y?: number; color?: string; }
export interface SpecNodeUpdate { id: string; x?: number; y?: number; name?: string; color?: string; }
export interface Viewport { x?: number; y?: number; zoom?: number; width?: number; height?: number; }
export interface ViewportCreate { x?: number; y?: number; zoom?: number; width?: number; height?: number; }
export interface ViewportUpdate { id?: string; x?: number; y?: number; zoom?: number; }
export interface ViolationCreate { specId: string; type: 'missing-parent' | 'circular-ref' | 'invalid-level'; message: string; line?: number; }
export interface ViolationUpdate { id: string; resolved?: boolean; }
export interface DependencyGraph { nodes: SpecNode[]; edges: SpecEdge[]; }
```

**修复 db.ts（添加缺失表）**：
```typescript
// 添加到 VibexDB class 中
artifacts!: Dexie.Table<{
  id: string; name: string; type: string; content: string;
  mime_type?: string; tags: string[]; thread_id?: string; run_id?: string;
  created_at: string; updated_at?: string; is_deleted: number;
}>;
threads!: Dexie.Table<{
  id: string; title: string; goal?: string; status?: string;
  created_at: string; updated_at: string; is_deleted: number; deleted_at?: string | null;
}>;

// 添加到 schema stores() 中
artifacts: 'id, type, created_at, is_deleted',
threads: 'id, updated_at, is_deleted',

// 添加类型导出
export type DBArtifact = { id: string; name: string; type: string; content: string;
  mime_type?: string; tags: string[]; thread_id?: string; run_id?: string;
  created_at: string; updated_at?: string; is_deleted: number; };
export type DBThread = { id: string; title: string; goal?: string; status?: string;
  created_at: string; updated_at: string; is_deleted: number; deleted_at?: string | null; };
```

**修复 artifact-store.ts**：
- 移除 `import { db, type DBArtifact }` → 改为 `import { db }`
- `db.artifacts.put()` 需要包含 `is_deleted: 0` 字段

**修复 thread-store.ts**：
- DB 字段名用 snake_case：`created_at`, `updated_at`, `is_deleted`, `deleted_at`
- Thread 业务类型用 camelCase：`createdAt`, `updatedAt`, `deletedAt`
- `db.threads.put()` 写 snake_case，`loadFromDB()` 映射到 camelCase

**修复 stores 类型导入**：
```typescript
// codeGen.ts: 添加 StepLog import
import type { GenerationJob, StepLog } from '../types';

// dslCanvas.ts: 修复 null 初始值
panOffset: { x: 0, y: 0 },
viewMode: 'mermaid' as const,

// routing.ts: 添加 SpecLocation import
import type { SpecLocation } from '../types';

// specEditor.ts: 添加 Tab, Violation import
import type { Tab, Violation } from '../types';
```

**修复 vitest 测试文件**（缺 `vi` 导入）：
```typescript
// 每个使用 vi.resetModules() 的测试文件都需要:
import { describe, it, expect, beforeEach, vi } from 'vitest';
```

**修复 Playwright e2e 测试**：
```typescript
// verbatimModuleSyntax 需要 type-only import
import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';  // ← 必须分开写

// 修复 areas! 断言语法（definite assignment assertion 错误）
const _areas = await page.evaluate(() => ...);
expect(_areas!).toBeTruthy();  // ← 用变量替代 ! 后缀
```

**验证**：
```bash
cd /root/vibex-workbench/frontend
npx tsc --noEmit  # 必须 0 errors
npm run build     # 必须 exit 0
```

## YAML `|` 块标量指示符陷阱（2026-04-20 新增）

**症状：** `make validate` 报错 `expected <block end>, but found '<scalar>'`，指向看起来完全正常的嵌套映射行。

**根因：** YAML 中 `type: "'mermaid' | 'canvas'"` 的 `|` 在双引号字符串内仍被解析器识别为**块标量指示符**，导致该字段的 description 内容被吞掉，并污染后续内容。

**诊断方法：**
```bash
cd /root/vibex-workbench && make validate 2>&1
# 找 "expected <block end>" 错误，往上找第一个 "name:" 或 "description:" 行
```

**修复方案（按优先级）：**
1. **类型值不用 `|`**：改为描述性词如 `mermaid_or_canvas`、`route_type`
2. **description 独立一行**：不要在 field 内把 description 写在 description 字段外面
3. **避免 `|` 作为文字**：union 类型用 `/` 或 ` or `

**示例对比：**
```yaml
# ❌ 错误（`|` 被误判为块标量，description 被吞）
- name: "viewMode"
  type: "'mermaid' | 'canvas'"
  description: "dsl-canvas 的视图模式

# ✅ 正确（类型值无 `|`，description 独立）
- name: "viewMode"
  type: "mermaid_or_canvas"
  description: "dsl-canvas 的视图模式，值: mermaid | canvas"

# ✅ 正确（type 用双引号包裹 union，但 description 不内嵌进被吞范围）
- name: "viewMode"
  type: "'mermaid' | 'canvas'"
```

**验证：** 修复后运行 `make validate`，确认 0 个 YAML 错误。

## Clarification Flow 已知状态（2026-04-22）

**✅ 已验证正常工作：**
- 后端 API 全链路：start → qa → draft → confirm → 文件持久化到 `.memlace/clarifications/*.clf.json`
- ClarificationPanel.svelte UI 代码完整（Q&A轮次、草稿编辑、Confirm/Reject 按钮）
- GoalSpecCanvas.svelte L2 圆盘卡代码完整（⚙️技术选型、🎨MVP原型、🏗️前后端分层、📋功能用户故事）
- Vite proxy `/api` → `localhost:33338` 已配置

**⚠️ 待排查：**
- Workbench 打开后左侧文件树显示"加载中…"，Canvas 没有渲染出来
- ClarificationPanel 在 UI 层还未被实际触发过（点击 L2 卡 → 打开抽屉 的完整用户流程未端到端验证）
- `confirm` 后 draft 内容还未写入 `.memlace/clarifications/` 之外的 spec YAML 文件

---

### 🔴 P1: memlace singleton 每次请求重建

**症状**：`POST /api/clarifications/test-spec {"action":"start"}` 返回成功，但 `GET /api/clarifications` 永远返回空列表。

**根因**：`getMemLaceMgr()` 在每次 HTTP 请求时都调用 `NewSessionManager()`，每次都是新内存实例，创建的 session 全丢。

**修复**（`agent/cmd/web/server.go`）：
```go
var (
    _memLaceMgr     *memlace.SessionManager
    _memLaceMgrOnce bool
)

func getMemLaceMgr() *memlace.SessionManager {
    if _memLaceMgrOnce {
        return _memLaceMgr
    }
    _memLaceMgrOnce = true
    mCfg := memlace.DefaultConfig(cfg.WorkspaceDir)
    mgr, err := memlace.NewSessionManager(mCfg)
    if err != nil {
        return nil
    }
    _memLaceMgr = mgr
    return mgr
}
```

**验证**：
```bash
# 启动 agent
cd /root/vibex-workbench/agent
go build -o /tmp/vw ./cmd/web/
WORKSPACE_ROOT=/root/vibex-workbench /tmp/vw > /tmp/vw.log 2>&1 &

# 测试单例
curl -X POST http://localhost:33338/api/clarifications/test-spec \
  -H "Content-Type: application/json" \
  -d '{"action":"start","phase":"mvp_prototype"}'
curl -s http://localhost:33338/api/clarifications
# 应该看到 session，不为空

# 验证文件落地
ls /root/vibex-workbench/.memlace/clarifications/
cat /root/vibex-workbench/.memlace/clarifications/test-spec.clf.json
```

---

### 🔴 P2: Go agent API 路由不达（vite proxy 缺失）

**症状**：`ClarificationPanel` 发起 `fetch('/api/clarifications/xxx')` 返回 404。Go agent 在 `localhost:33338`，SvelteKit dev 在 `localhost:5173`，路由不达。

**修复**（`frontend/vite.config.ts`）：
```ts
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

const SSE_PORT = process.env.VITE_SSE_PORT || '33338';

export default defineConfig({
    plugins: [sveltekit()],
    server: {
        port: 5173,
        proxy: {
            '/api': {
                target: `http://localhost:${SSE_PORT}`,
                changeOrigin: true,
            },
        },
    },
});
```

**验证**：
```bash
curl -s http://localhost:5173/api/clarifications
# 应该和下面一致
curl -s http://localhost:33338/api/clarifications
```

---

### 🔴 P3: clarificationHandler 缺少 qa/draft action

**症状**：`ClarificationPanel` 的"添加轮次"按钮报错 `unknown action: qa`。

**根因**：Go handler 只有 `start/confirm/reject` 三个 action，前端需要 `qa`（新增轮次）和 `draft`（保存草稿）。

**修复**（`agent/cmd/web/server.go`）：
```go
var req struct {
    Action   string `json:"action"` // "start"|"qa"|"draft"|"confirm"|"reject"
    Phase    string `json:"phase"`
    Draft    string `json:"draft"`
    Question string `json:"question"`
    Answer   string `json:"answer"`
}

case "qa":
    session, err := mgr.AddRound(specName, req.Question, req.Answer)
    // 返回: ok=true, round=N, rounds=[...], draft="...", status="in_progress"
case "draft":
    if err := mgr.SetDraft(specName, req.Draft); err != nil { return }
    // 返回: ok=true, status="draft_saved"
case "confirm":
    // body 带 draft 时先更新
    if req.Draft != "" {
        mgr.SetDraft(specName, req.Draft)
    }
    // ...
```

**前端对应**：
```typescript
// qa action
fetch(`/api/clarifications/${specName}`, {
    method: 'POST',
    body: JSON.stringify({ action: 'qa', question, answer })
})

// draft action
fetch(`/api/clarifications/${specName}`, {
    method: 'POST',
    body: JSON.stringify({ action: 'draft', draft: draftText })
})

// confirm action（带 draft）
fetch(`/api/clarifications/${specName}`, {
    method: 'POST',
    body: JSON.stringify({ action: 'confirm', draft: draftText })
})
```

---

### 🔴 P5: mvm_prototype 拼写 bug

**症状**：`ClarificationPanel` 发出 `start` action，但返回的 phase 是 `"mvm_prototype"` 而不是 `"mvp_prototype"`。

**根因**：Go 代码里 `"mvm_prototype"` 写错了，两处：
- `generators/memlace/clarification.go:97`
- `agent/cmd/web/server.go:497`

**修复**（两处）：
```go
// 追加正确的 key
"mvp_prototype": {
    phase:       "mvp_prototype",
    label:       "MVP Prototype",
    description: "可交互 HTML 原型验证",
    color:       "#c084fc",
    icon:        "🎯",
},
_ = mvm_prototype // mvp_prototype was misspelled
```

**验证**：
```bash
# API 测试
curl -s -X POST http://localhost:33338/api/clarifications/test \
  -H "Content-Type: application/json" \
  -d '{"action":"start","phase":"mvp_prototype"}' | \
  python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('phase'))"
# 期望: mvp_prototype（不是 mvm_prototype）
```

---

### 🔴 P6: SvelteKit server routes 被 Vite proxy 404（2026-04-22 新增）

**症状**：左侧文件树一直显示"加载中…"，`fetch('/api/workspace/specs/list')` 和 `fetch('/api/workspace/specs/convention')` 返回 404。

**根因**：Vite proxy 的 `/api` → Go backend 优先级高于 SvelteKit server routes。所有 `/api` 请求被代理到 Go 后端，SvelteKit 的 `src/routes/api/workspace/specs/` 下的 server-side routes 永远走不到。

**诊断**：
```bash
# 浏览器 console 查 404
window.performance.getEntriesByType('resource')
  .filter(r => r.responseStatus >= 400)

# 确认 SvelteKit routes 存在
ls /root/vibex-workbench/frontend/src/routes/api/workspace/specs/

# 确认 SvelteKit 能直接 serve（不走 proxy）
curl http://localhost:5173/api/workspace/specs/list
# 应返回 {"paths": [...]}，而不是 404
```

**修复**（`frontend/vite.config.ts`）：
```ts
server: {
  proxy: {
    '/api': {
      target: `http://localhost:${SSE_PORT}`,
      changeOrigin: true,
      bypass(req) {
        const p = req.url ?? '';
        // SvelteKit server-side routes — do NOT proxy
        if (p.startsWith('/api/workspace/specs/list') ||
            p.startsWith('/api/workspace/specs/convention')) {
          return p; // bypass → serve via SvelteKit
        }
        return undefined; // proxy to backend
      },
    },
  },
},
```

**验证**：
```bash
# 确认无 404
curl "http://localhost:5173/api/workspace/specs/list"
# 应返回 {"paths": [...]}，而不是 404
```

---

### 🟡 P4: git commit 混入 generated 文件

**症状**：`.pyc` / `agent-queue/*.json` / `__pycache__/` 被 commit 到仓库。

**修复**：commit 前精确选择文件，不要 `git add -A`：
```bash
# 检查待提交文件
git status -s

# 排除 generated 文件
git reset HEAD -- generators/__pycache__/ generators/.manifest.json generators/agent-queue/ agent/web

# 只加需要的
git add frontend/vite.config.ts
git add frontend/src/lib/components/workbench/ClarificationPanel.svelte
git add agent/cmd/web/server.go
git add generators/memlace/
git add skills/spec-first-workflow/
```

**常用 .gitignore 规则**：
```
generators/__pycache__/
generators/.criteria_report.json
generators/.manifest.json
generators/agent-queue/
agent/web
*.pyc
```

---

## Wails Dev 访问 404 调试（2026-04-25 新增）

### 症状
`make wails-dev` 启动后，浏览器访问 `http://wails.localhost:34115/` 返回"无法访问此页面"。

### 诊断流程

**Step 1: 确认是哪个端口/路由问题**
```
# 能加载空白页（HTML shell 到了，但 JS/API 404）→ 前端 JS 能加载，但 API 调用失败
# 完全白屏（连 HTML 都没到）→ Wails embed 的 build 目录缺失或路径错误
```

**Step 2: 检查 wails.json 的 devserver 配置**
```bash
cat /root/v-test/wails.json | grep -E "devserver|serverUrl|watcher"
```
- ❌ 如果有 `"frontend:dev:serverUrl"` 或 `"frontend:dev:watcher"` → **删掉**。这些会让 Wails 在 dev 模式代理到 Vite dev server，而不是直接 serve embed 的 `frontend/build/` 目录。在 embed 模式下 Vite dev server 路由不完整。
- ✅ 正确配置：只有 `devserver: "localhost:34115"`，无 `serverUrl`/`watcher`

**Step 3: 确认前端 build 存在**
```bash
ls /root/v-test/frontend/build/index.html   # 必须存在
```

**Step 4: 检查端口一致性（最常见根因）**

agent/cmd/web 硬编码 `:33338`，但前端所有组件必须对齐到这个端口：

| 文件 | 需要检查的值 | 正确值 |
|------|------------|--------|
| `frontend/vite.config.ts` | `BACKEND_PORT` | `'33338'` |
| `frontend/src/routes/workbench/+page.svelte` | `SSE_URL` | `'http://localhost:33338'` |
| `frontend/src/lib/sse.ts` | `SSEConsumer.url` | `'http://localhost:33338'` |
| `main.go` | `backendPort` 默认值 | `33338` |

```bash
# 快速验证端口一致性
grep -rn "33335\|33338" /root/v-test/frontend/src/ --include="*.ts" --include="*.svelte" | grep -v "test\|\.d\.ts"
grep "backendPort.*33335\|= 33335" /root/v-test/main.go
```

**Step 5: SvelteKit server routes 在 embed 模式下 404**

`adapter-static` 构建的 `frontend/build/` **不包含** SvelteKit server-side routes（`+server.ts` 文件在构建时被忽略）。所有 `/api/workspace/specs/list` 等调用在 embed 模式下会 404。

**诊断**：
```bash
# 列出 SvelteKit server routes
find /root/v-test/frontend/src/routes/api -name "+server.ts"

# 检查 build 目录（应该没有这些路由）
find /root/v-test/frontend/build -name "+server*"   # 应为空
```

**修复**：在 Go backend（agent/cmd/web）补全这些 endpoint，然后注册到 main.go：

```go
// agent/cmd/web/workspace_handlers.go — 添加缺失的 handlers
// 注意：这些函数使用包级变量 cfg（来自 server.go 的 package-level var）

// workspaceSpecsListHandler GET /api/workspace/specs/list
func workspaceSpecsListHandler(w http.ResponseWriter, r *http.Request) {
    wsRoot := r.URL.Query().Get("workspaceRoot")
    if wsRoot == "" {
        wsRoot = cfg.WorkspaceDir
    }
    // ... 遍历 specs/ 目录找 .yaml/.yml 文件
}

// workspaceSpecsConventionHandler GET /api/workspace/specs/convention
func workspaceSpecsConventionHandler(w http.ResponseWriter, r *http.Request) {
    // ... 返回 spec 约定信息
}
```

注册到 `agent/cmd/web/main.go`：
```go
http.HandleFunc("/api/workspace/specs/list", withCORS(workspaceSpecsListHandler))
http.HandleFunc("/api/workspace/specs/convention", withCORS(workspaceSpecsConventionHandler))
```

**Step 6: Vite proxy bypass 逻辑**

Vite dev 模式 proxy 的 `bypass` 函数如果把太多路径绕过，后端也收不到请求。当前推荐把所有 `/api` 统一代理到 Go backend：

```ts
// vite.config.ts — 不要用 bypass 绕过 workspace/specs routes
proxy: {
    '/api': {
        target: `http://localhost:${BACKEND_PORT}`,
        changeOrigin: true,
        // ❌ 不要加 bypass 逻辑让 /api/workspace/specs/* 绕回 SvelteKit
        // ✅ Go backend 已实现这些 endpoint，直接 proxy 过去即可
    },
},
```

**Step 7: 验证完整链路**

```bash
# 1. 构建确认
cd /root/v-test && go build -o /tmp/vw . && echo "MAIN OK"
cd /root/v-test/agent/cmd/web && go build -o /tmp/agent . && echo "AGENT OK"

# 2. 启动 agent（后台）
cp /tmp/agent /root/v-test/backend/vibex-backend
WORKSPACE_ROOT=/root/v-test /tmp/agent > /tmp/agent.log 2>&1 &

# 3. 验证 agent 端口
curl -s http://localhost:33338/health

# 4. 验证 workspace API
curl -s "http://localhost:33338/api/workspace/specs/list?workspaceRoot=/root/v-test"

# 5. 启动 Wails（手动测试）
# cd /root/v-test && ./tmp/vw
```

---

## SSE Chat Message Ordering（2026-04-23 新增）

### 症状
多轮对话中，用户第二次发的消息不展示，或出现在 agent 回复下方而不是中间。

### 根因：两条创建路径竞争

```
路径A（handleSubmit 本地创建）:
  handleSubmit() → appendMessage(msgA, UUID_A) → 加入 messagesByThread

路径B（SSE echo 回显）:
  Server: message.delta(role='user', content) → SSE
  SSE bridge: appendDelta(role='user') → 创建 msgB(UUID_B ≠ UUID_A) → 加入 messagesByThread

结果：
- 如果 msgB 先到达：messagesByThread = [msgB]
- handleSubmit 的 appendMessage 到达 → UUID_A 不在现有 messages → 去重跳过 → msgA 消失
- 或 UUID_A 被排进 queuedUserMessages → 永远无法 flush（因为 UUID_B 已存在）
```

### 修复原则

**单一数据源**：用户消息的创建和 SSE 回显不能同时存在。选择其一：

| 方案 | 做法 | 缺点 |
|------|------|------|
| ✅ 采用 | 消息完全由 SSE 回显，`handleSubmit` 只发 POST | 有微小延迟 |
| ❌ 不用 | handleSubmit 本地创建 + SSE echo 去重 | ID 不同无法去重 |

### 最终架构（thread-store.ts + sse.ts）

```typescript
// thread-store.ts — 简化版，无排队
appendDelta(threadId, { role, delta, is_final }) {
  if (role === 'user') {
    // 用户消息：SSE echo 直接追加（无本地预创建）
    messagesByThread[tid].push({ id: uuid, role: 'user', content: delta });
  }
  if (role === 'assistant' && !is_final) {
    // 流式累积到 pending 气泡
  }
  if (role === 'assistant' && is_final) {
    // 替换 pending 气泡
    pendingAssistantIdByThread[tid] = '';
  }
}

// run.completed / run.failed 时清状态（sse.ts + workbench-message-sse-bridge.ts）
'run.completed': (data) => {
  runStore.updateRunStatus(data.run_id, 'completed');
  const tid = data.thread_id ?? data.threadId;
  if (tid) threadStore.clearPendingAssistant(tid);
},
```

### 关键教训

1. **SSE echo 的消息 ID ≠ handleSubmit 预创建的 ID** → 永远无法靠 ID 匹配去重
2. **排队机制（queuedUserMessages）引入状态交叉**：pendingAssistantIdByThread + 排队 + 两次 appendDelta 的交互难以维护
3. **`run.completed` / `run.failed` 是可靠的 agent 轮次边界信号** → 优先用它们清状态，不要依赖 `is_final=true`
4. **SSE 事件顺序基本可靠** → 除非有反向代理/hijack，不强依赖队列排序

---

## agent/vibex/ 残留导致 Go Build 重复声明（2026-04-27 新增）

### 症状
`go build ./cmd/web/` 报错：
```
MakeWorkspaceDetectStateHandler redeclared in this block
  other declaration of MakeWorkspaceDetectStateHandler
```

### 根因
`agent/vibex/domain/spec/handlers_workspace.go` 是旧 antch backend 残留文件（与 `agent/vibex/domain/spec/handlers.go` 同一 Go package），包含完全重复的函数声明。当前 agent module 是 `vibex/agent`，但 `agent/vibex/` 子目录不属于任何被引用路径，仍被 Go 编译器扫描到并触发包内重复声明。

### 诊断
```bash
# 找所有 MakeWorkspaceDetectStateHandler 声明位置
grep -rn 'MakeWorkspaceDetectStateHandler' /root/vibex-workbench/agent/ --include='*.go'

# 确认是哪个文件属于旧 antch
grep -rn 'vibex/domain/spec\|vibex/domain' /root/vibex-workbench/agent/ --include='*.go' | grep -v 'cmd/web/server.go'
```

### 修复
```bash
# 确认残留目录不属于任何被引用路径
grep -r 'vibex/domain/spec' /root/vibex-workbench/agent/ --include='*.go' | grep -v 'handlers.go:513'
# 如果只有 handlers.go 引用 → 可以安全删除

# 删除残留目录
rm /root/vibex-workbench/agent/vibex/domain/spec/handlers_workspace.go

# 验证 build
cd /root/vibex-workbench/agent && go build ./cmd/web/
```

### 教训
- `agent/vibex/` 是 antch → vibex-workbench 迁移历史遗留，不是当前架构一部分
- Go package 内函数不能重复声明，无论文件多少
- `go vet` 单独跑会报 `undefined: cfg`（预期，包级变量在 server.go），完整 build 才是准的

---

## Git Push 调试（已知问题模式）

### 症状：`git push` 挂起超时，无错误信息

**诊断命令**：
```bash
# 使用 GIT_TRACE_PACKET 看到实际服务器响应（需要较长超时）
cd /root/vibex-workbench
GIT_TRACE_PACKET=1 timeout 60 git push origin master 2>&1 | tail -20
```

**已知根因**：
- GitHub 限制单文件 100MB；playwright chromium binary ~260MB
- playwright cache 目录是 `~/`（home 目录下的路径），`.gitignore` 中的 `~/*` **不生效**（git 不追踪 `~/` 路径）
- `.gitignore` 实际只覆盖了 `test-results/` 但没覆盖 `frontend/~/.cache/ms-playwright/`

**快速检查是否有大文件**：
```bash
# 方法1：直接 push 看错误（需要较长超时）
timeout 60 git push origin master 2>&1 | grep -E "error|Large|exceeds"

# 方法2：检查 git bundle 大小
git bundle create /tmp/push.bundle HEAD 2>/dev/null && \
  ls -lh /tmp/push.bundle && rm /tmp/push.bundle
# 如果 > 200MB，很可能有 binary 大文件

# 方法3：检查 git 对象中的大 blob
git rev-list --objects --all 2>/dev/null | \
  git cat-file --batch-check --batch-all-objects 2>/dev/null | \
  grep -E "blob" | awk '{print $3, $1}' | sort -k1 -n -r | head -5
```

**修复方法（已验证）**：
```bash
# Step 1: 从 origin/master 重建干净分支
git checkout -b clean origin/master

# Step 2: 提取目标 commit 的干净源码文件（排除 binary）
for f in \
  frontend/src/lib/components/workbench/ArtifactPanel.svelte \
  frontend/src/lib/components/workbench/ArtifactPreviewModal.svelte \
  frontend/src/lib/components/workbench/Composer.svelte \
  frontend/src/lib/stores/artifact-store.ts \
  frontend/src/lib/stores/run-store.test.ts \
  frontend/tests/e2e/run-engine.spec.ts; do
  git show BAD_COMMIT:"$f" > "$f" 2>/dev/null && echo "✅ $f"
done

# Step 3: 提交
git add <clean files>
git commit -m "feat(E4): clean commit (no binary cache)"

# Step 4: push
timeout 60 git push origin clean:master
```

**根本预防**：在 `frontend/.gitignore` 添加：
```
# playwright cache (uses ~/ not ./ — MUST use this pattern)
~/
```

### 验证命令汇总
```bash
cd /root/vibex-workbench

# 1. lint-specs（必须先通过）
make lint-specs

# 2. 组件数量
echo "组件: $(ls frontend/src/lib/generated/components/*.svelte | wc -l) 个"

# 3. stub 检查
for f in frontend/src/lib/generated/components/*.svelte; do
  [ $(wc -l < "$f") -lt 50 ] && echo "STUB: $f"
done

# 4. spec 文件数量
echo "spec文件: $(find specs -name '*.yaml' | wc -l) 个"
```

## WebView2 缓存不刷新（2026-04-26 新增）

### 症状
`make wails-dev` 后，前端仍显示旧代码（如 `$state is not defined` at `4.C7jGqSvZ.js`），即使已重新 build。

### 根因：三层缓存叠加
1. `frontend/build/` 里旧 chunk（如 `C7jGqSvZ.js`）在增量 rebuild 时残留
2. WebView2 有独立 disk cache，`Ctrl+Shift+R` / `Cmd+Shift+R` **完全无效**（不走浏览器刷新机制）
3. 新 `app.Vxxx.js` 里的动态 import 引用了不存在的旧 chunk → `$state not defined`

**关键**：WebView2 的 HTTP 缓存完全独立于浏览器，DevTools 的 "Disable cache" 也只影响 DevTools 会话自身。

### 正确解法：四步全部需要

**Step 1：构建前清空 build 目录**
```makefile
# Makefile — frontend-build 目标必须先 rm -rf
frontend-build:
	@cd $(FRONTEND_DIR) && npm install
	@rm -rf $(FRONTEND_DIR)/build          # ← 关键：删旧文件
	@cd $(FRONTEND_DIR) && npm run build
	@$(PYTHON) $(ROOT)/scripts/cache_bust.py $(FRONTEND_DIR)/build/index.html
```

**Step 2：cache-busting 注入时间戳**
```python
# scripts/cache_bust.py（独立脚本，避免 Makefile tab 问题）
import sys, re, pathlib, time

html_path = pathlib.Path(sys.argv[1])
html = html_path.read_text(encoding="utf-8")
ts = str(int(time.time()))

def buster(m):
    url = m.group(1)
    sep = "&" if "?" in url else "?"
    return m.group(0).replace(m.group(1), url + sep + "v=" + ts)

html = re.sub(r'(src|href)="(/[^"]+)"', buster, html)
html_path.write_text(html, encoding="utf-8")
print(f"[cache-bust] v={ts}")
```

**Step 3：OnDomReady 自动 WindowReload**
```go
// main.go — 启动 500ms 后强制重新拉取所有资源
OnDomReady: func(ctx context.Context) {
    go func() {
        time.Sleep(500 * time.Millisecond)
        runtime.WindowReload(ctx)  // 强制 WebView2 重新请求所有资源
    }()
    // ... spawn backend ...
},
```

**Step 4：菜单加手动刷新入口**
```go
// main.go — 视图菜单
viewMenu.AddText("清除缓存并刷新", nil, func(_ *menu.CallbackData) {
    runtime.WindowReload(ctx)
})
```

### ❌ 无效的方法（不要试）
- ❌ Ctrl+Shift+R / Cmd+Shift+R（WebView2 不走浏览器刷新）
- ❌ DevTools → Network → Disable cache（仅影响 DevTools 会话）
- ❌ 清除 Chrome/Edge 浏览器历史记录（清除的是 Chrome 的缓存，不是 WebView2 的）
- ❌ 只做 cache-busting 时间戳（不 rm -rf build 的话，旧的 chunk 文件仍然存在于磁盘，动态 import 仍能找到旧文件）

### Makefile Tab 陷阱（2026-04-26 新增）
**症状**：`Makefile:190: *** missing separator. Stop.`
**原因**：Python heredoc 作为 Make recipe 时，Python 代码的缩进是空格，但 Make 要求 recipe lines 必须以 Tab 开头。
```makefile
# ❌ 错误（Python 用空格缩进，Make 解析器看到非 Tab 报错）
	@cd $(ROOT) && $(PYTHON) -c "
import time, re, pathlib   # ← 这里是空格，Make 把它当普通行处理
html = pathlib.Path('...')  # ← 第二个普通行，缺 recipe 前的 Tab
"
# ✅ 正确：提取到独立脚本
	@$(PYTHON) $(ROOT)/scripts/cache_bust.py $(FRONTEND_DIR)/build/index.html
```

### 验证修复
```bash
cd /root/vibex-workbench && git pull && make wails-dev
# 看 Makefile 输出：
#   [frontend-build] Cleaning old build artifacts...
#   [cache-bust] v=XXXXXXXX
# 看 index.html：
#   grep 'v=' frontend/build/index.html | head -1
#   应输出 href="?v=XXXXXXXX=/_app/..."
# WebView2 启动后旧 chunk 不再出现
```

## Spec 写作规范（2026-04-27 新增）

### `dependencies` 必须是顶层 key

VibeX spec 文件结构中，`dependencies` 是**顶层列表**，不能放在 `constraints`、`acceptance_criteria` 等 list item 内部。

```yaml
# ❌ 错误 — dependencies 作为 list item 嵌套在 constraints 下
constraints:
  - id: "C1"
    rule: "..."
    validation: "..."
  dependencies:          # ← YAML parse 失败！
    - file: specs/...
      reason: "..."

# ✅ 正确 — dependencies 是顶层 key
constraints:
  - id: "C1"
    rule: "..."
    validation: "..."

dependencies:
  - file: specs/...
    reason: "..."
```

**症状**：`make validate` 报错 `expected <block end>, but found '<scalar>'`，指向 `acceptance_criteria` 或 `constraints` 列表内某行。YAML 解析器把 list item 内的 map key 当成 list item 继续解析，找不到 `:` 分隔符就爆。

**验证**：
```bash
cd /root/vibex-workbench && python3 -c "
import yaml, sys
for f in sys.argv[1:]:
    try:
        yaml.safe_load(open(f))
        print(f'OK: {f}')
    except Exception as e:
        print(f'FAIL: {f}: {e}')
" specs/**/*.yaml
```

---

## Git Stash 残留文件污染（2026-04-27 新增）

### 症状
`git stash pop` 后，某个应该删除的旧文件出现在 `git status` 里，被 `git add -A` 误提交。

### 根因
`git stash` 只 stash 已追踪文件的修改，不 stash untracked 文件。`git stash pop` 后 untracked 文件仍保留在磁盘上。

### 诊断
```bash
git status --short  # 看是否有意外新文件
git log --oneline -3  # 看是否有多余 commit
```

### 修复
```bash
# 删掉残留文件并 amend
rm agent/vibex/domain/spec/handlers_workspace.go  # 旧 antch 残留
git add -A
git commit --amend --no-edit
git push --force
```

### 预防
commit 前用 `git status -s` 精确审查，不要 `git add -A`。

---

## Wails Event System：两条独立 IPC Channel（2026-04-29 新增）

### 核心约束（必须记住）

Wails 的事件系统有**两条完全独立的 channel**，不得混用：

| Channel | 方向 | API | 典型用途 |
|---|---|---|---|
| **EventsEmit** | frontend → backend | `runtime.EventsEmit(ctx, "event", data)` | Go backend 监听：native menu callbacks、`buildAppMenu` 里的 `AddText` 回调 |
| **EventsOn** | backend → frontend | `rt.EventsOn('event', callback)` | frontend 监听：Go backend 主动发来的事件 |

```
❌ 错误理解：
  frontend: eventsEmit('menu:open-project')
  +layout: eventsOn('menu:open-project', handler)
  → 这两个不在同一 channel，永远不通！

✅ 正确理解：
  Channel A (frontend→backend)：titlebar onclick → runtime.EventsEmit('menu:open-project')
                      → Go backend 的 menu callback 收到

  Channel B (backend→frontend)：Go backend runtime.EventsEmit('menu:open-project')
                      → frontend eventsOn('menu:open-project') 收到
```

### 症状：titlebar 点「打开项目」没反应

**根因**：titlebar 调用 `eventsEmit('menu:open-project')`，但 `+layout` 的 `eventsOn('menu:open-project')` 监听的是另一条 channel。

**修复原则**：

1. **titlebar UI 交互 → 直接调用 runtime API**（如 `openDirectoryDialog()`），禁止 `eventsEmit` 绕转
2. **`eventsOn` 只用于监听 Go backend 主动发来的事件**（native menu、Wails 系统事件）
3. **所有入口的最终行为保持一致**：无论哪条路径触发，最终都弹目录选择器 → 保存 localStorage → `goto('/workbench')`

### 正确 wiring 示例

```typescript
// WorkbenchTitlebar.svelte — UI 交互直接调 runtime，不走 eventsEmit
import { openDirectoryDialog } from '$lib/wails-runtime';

async function openProject() {
  fileMenuOpen = false;
  const dir = await openDirectoryDialog();  // ✅ 直接调
  if (!dir) return;
  localStorage.setItem('vibex-workspace-root', dir);
  goto('/workbench');
}

// +layout.svelte — eventsOn 只接收 Go backend 的 native menu 事件
import { eventsOn, openDirectoryDialog } from '$lib/wails-runtime';
import { goto } from '$app/navigation';

async function handleOpenProject() {
  const dir = await openDirectoryDialog();  // ✅ Go native menu 触发时走这里
  if (!dir) return;
  localStorage.setItem('vibex-workspace-root', dir);
  goto('/workbench');
}

onMount(() => {
  eventsOn('menu:open-project', handleOpenProject);  // ✅ 监听 Go → frontend 事件
});
```

### Runtime Unavailable 时的 Fallback

浏览器（非 Wails 环境）`window.runtime` 不存在。所有 `wails-runtime.ts` 函数必须 guard：

```typescript
// ✅ 正确：runtime 不可用时降级，不崩溃
export async function openDirectoryDialog(fallbackPrompt = '请输入目录路径:') {
  const rt = getRuntime();
  if (!rt) {
    return window.prompt(fallbackPrompt) ?? '';  // 降级为浏览器 prompt
  }
  return (await rt.OpenDirectoryDialog()) ?? '';
}

// ✅ 正确：其他 runtime API 静默 no-op
export async function windowMinimize() {
  const rt = getRuntime();
  if (!rt) return;
  await rt.WindowMinimise();
}
```

**禁止**：
- `console.warn` 后 return 空字符串（用户没有反馈，不知道发生了什么）
- 直接 `throw new Error`（导致页面崩溃）

### 在 SLICE-ide-titlebar-component.yaml 中的体现

每条涉及 Wails event 的 L5 spec，都必须在 `generation_rules` 里写明 event channel 约束：

```yaml
  - rule: Wails event channel constraint ⚠
    detail: |
      EventsEmit（frontend→backend）和 EventsOn（backend→frontend）是两条独立 IPC channel。
      titlebar UI 交互必须直接调用 runtime API，禁止 eventsEmit 绕转。
      eventsOn 只用于监听 Go backend 主动发来的事件。
      Runtime unavailable 时：openDirectoryDialog → window.prompt()；
      其他 runtime API → 静默 no-op，不得崩溃。
```

### 验证方法

```bash
# 确认 build 通过
cd /root/vibex-workbench && npm --prefix frontend run build

# 确认无重复事件监听（同一个事件名最多在一个文件里注册）
grep -rn "eventsOn('menu:" frontend/src/
# 期望：每个事件名最多在 1 个文件里注册
```

---

## 大重构后残留变量引用导致 Go Build 失败（2026-04-27 新增）

### 症状
patch 大文件后，`go build` 失败：`undefined: wsRoot`，但代码里明明有 `wsRoot = ...`。

### 根因
重构时将变量改名（如 `wsRoot` → `wsRootNorm`），但某处残留了旧变量名的引用。Go vet 单文件报错 `undefined: cfg` 是正常的（包级变量在其他文件），但完整 build 时如果残留变量名，实际错误可能被遮盖。

### 诊断
```bash
cd /root/vibex-workbench/agent && go build ./cmd/web/ 2>&1
# 找 "undefined:" 错误 — 这是残留变量引用
```

### 修复
patch 后搜索残留变量名：
```bash
# 在改过的文件里搜旧变量名
grep -n 'wsRoot[^N]\|\bwsRoot$\|wsRoot,' agent/cmd/web/workspace_handlers.go
# 确认每处引用都已更新
```

### 教训
重构变量名时，用 `grep -n '旧变量名'` 全文扫描，而不只是 `go build`。

---

## Windows wails.localhost 解析失败（2026-04-26 新增）

### 症状
`Unsafe attempt to load URL http://wails.localhost/workbench from frame with URL chrome-error://chromewebdata/`

### 根因
Windows 默认不带 mDNS/Bonjour，`wails.localhost` 无法解析。macOS 和 Linux 自带 Bonjour/Avahi 所以没问题。

### 解法：两步都要做

**Step 1：wails.json 加 devserverurl**
```json
// wails.json
{
  "devserver": "localhost:34115",
  "devserverurl": "http://localhost:34115"  // ← 加这一行
}
```

**Step 2：make wails-dev 自动写 hosts（幂等）**
```makefile
# Makefile
IS_WINDOWS := $(shell go env GOOS 2>/dev/null | grep -q windows && echo 1 || echo 0)

wails-hosts-setup:
ifneq ($(strip $(IS_WINDOWS)),0)
	@powershell -Command " \
		$$h='C:\\Windows\\System32\\drivers\\etc\\hosts'; \
		$$l='127.0.0.1  wails.localhost'; \
		if(!(Select-String -Path $$h -Pattern 'wails.localhost' -Quiet)) { \
			$$a=@(); Get-Content $$h | ForEach-Object { $$a+=$_ }; \
			$$a+=''; $$a+='# Added by VibeX Workbench'; $$a+=$$l; \
			Set-Content -Path $$h -Value ($$a -join \"`r`n\"); \
			Write-Host '[wails-hosts] Added.'; \
		} else { \
			Write-Host '[wails-hosts] Already present.'; \
		}"
endif

wails-dev: agent-build frontend-build wails-hosts-setup
	# ...
```

**效果**：首次运行提示需要管理员权限写 hosts，之后幂等跳过。

## spec_coverage.py 对 5_implementation 漏计

**症状**：coverage report 显示某 L4 无 L5，但实际存在同名 `_L5.yaml` 文件。

**根因**：`scripts/spec_coverage.py` 只识别 `level: 5_slice`，不识别 `level: 5_implementation`。

**修复**：两处改动
```python
# 1. l4_to_l5 映射构建
elif info['level'] in ('5_slice', '5_implementation'):

# 2. 统计计数
l5_n = sum(1 for i in specs.values() if i['level'] in ('5_slice', '5_implementation'))
```

**验证**：`python3 scripts/spec_coverage.py` 后该 L4 应显示 `OK L5xN`。

## WORKSPACE_DIR env vs workspace_root 请求体（架构区分）

| 变量 | 作用域 | 作用 |
|---|---|---|
| `WORKSPACE_DIR` (shell env) | Go backend 进程启动时 | 决定 backend **自己的**代码从哪里跑 |
| `workspace_root` (HTTP body) | 每个 HTTP 请求 | 用户想操作的项目目录，handler 直接用 |

**关键**：前端 `+page.svelte` 的所有 API 调用都是 `POST body { "workspace_root": "/path" }` ——直接传给 handler，不依赖 `WORKSPACE_DIR`。

**调试要点**：当发现 `workspace_dir` 在 health 响应里值不对时，先确认是否影响实际功能——API 用的是请求体里的 workspace_root，health 里的值只是默认值。

## Go Build Errors

### `MakeWorkspaceDetectStateHandler redeclared`
- `handlers.go` 和 `handlers_workspace.go` 都声明了同名函数
- 删 `handlers.go` 里的版本，保留 `handlers_workspace.go` 的原生实现

### `normalizeWorkspaceRoot undefined`
- `spec_write_protocol.go` 调用了但无定义
- 补充实现：验证目录存在且是目录，返回 (path, errorCode)

## Terminal Deadlock（后台进程导致工具链卡死）

**症状**：启动 `vibex-agent-web &` 后，所有 terminal/execute_code/browser 工具调用立即返回 `[Command interrupted] (exit code 130)`，0.02秒内中断。工具链完全失效。

**根因**：后台进程启动后，即使 pkill 也可能残留 port/FD 状态，导致 shell 环境异常。

**恢复步骤（逐级尝试）：**
1. `killall -9 vibex-agent-web; sleep 1` — 先在自己的终端执行
2. `killall -9 go` — 杀残留 go 进程
3. 确认 port 释放：`ss -tlnp | grep 33338`（应返回空）
4. 若仍卡：重启当前会话

**验证 terminal 已恢复**：运行 `pwd` 或 `ls` — 若立即返回而非卡住，说明恢复正常。

**预防**：每次启动 server 后立即验证 health，再继续后续操作。
