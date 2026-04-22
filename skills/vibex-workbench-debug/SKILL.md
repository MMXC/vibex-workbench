---
name: vibex-workbench-debug
description: vibex-workbench 本地桌面工作台常见问题调试技能
category: devops
---
repo_tracked: true


# vibex-workbench-debug

## 触发条件

遇到以下任一问题时，调用此技能：
- `npx tsc --noEmit` 有 TypeScript 错误（31个常见错误模式）
- `npm run build` 失败
- `make lint-specs` 报错 "parent 'MOD-xxx' 未找到"
- spec L4 feature 的 parent 引用无法通过验证
- `make generate` 失败或产出空文件
- 前端组件 stub（13 行骨架）无实际功能

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

**Step 1: 安装 npm 依赖**（如需第三方库）
```bash
cd /root/vibex-workbench/frontend
npm install <package>
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

## Clarification Flow 调试（2026-04-22 新增）

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
