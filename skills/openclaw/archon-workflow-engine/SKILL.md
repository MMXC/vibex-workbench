---
name: archon-workflow-engine
description: archon-workflow-engine — skill for openclaw category
category: openclaw
triggers:
- openclaw
- team-tasks
- gateway hook
- custom hook
- coord decision
- openclaw hook
- archon workflow engine
related_skills:
- archon-workflow-engine
- openclaw-internals-reversed
- darwin-skill-execution
---
repo_tracked: true


# Archon-Workflow-Engine

从 [Archon](https://github.com/coleam00/Archon) 汲取灵感的下一代 team-tasks 工作流引擎。

## 核心设计理念

Archon 的本质：把 AI 编码流程变成**确定性、可复用、有审计**的 DAG workflow。

类比：
- Docker → 容器化基础设施
- GitHub Actions → CI/CD 自动化
- **Archon → AI 编码的 workflow 自动化**

两层分离是 Archon 最精妙的设计：

| 层 | 文件格式 | 作用 |
|---|---|---|
| **Workflow** | `workflows/*.yaml` | DAG 拓扑：节点顺序、依赖、循环、条件分支 |
| **Command** | `commands/*.md` | 原子 AI 任务：具体 prompt + 输出规范 |

---

## 现状 vs 目标

### 现状（team-tasks）
- Phase 序列：phase1 → coord → phase2 → reviewer → tester → coord
- 验证门：全 AI 判断（贵、慢、reviewer 误判）
- 状态传递：task desc 文本塞一坨
- 无条件分支、无循环、无人工 gate

### 目标（Archon-inspired）
- DAG 拓扑：depends_on + when 条件 + loop 循环
- 验证门：bash 确定性校验 + AI 语义审查
- 状态传递：artifact 文件 + `$node.output` 变量替换
- 人工 gate：interactive loop + approval node
- 可扩展：17 种 workflow 类型，按场景路由

---

## 项目结构

```
~/.openclaw/skills/archon-workflow-engine/
├── SKILL.md                    ← 本文件
├── OPENAPI.md                  ← API 契约文档（基于 Archon server/routes/api.ts）
├── workflows/                  ← Workflow YAML（DAG 定义）
│   ├── bug-fix.yaml            ← 快速通道
│   ├── feature-dev.yaml        ← 标准流程
│   ├── refactor.yaml           ← 安全优先
│   └── docs.yaml               ← 简化流程
├── commands/                   ← Command md 文件（原子 AI 任务）
│   ├── analyze.md              ← 分析问题
│   ├── plan.md                 ← 制定计划
│   ├── implement.md            ← 实施代码
│   ├── review.md               ← 代码审查
│   ├── test.md                 ← 测试验证
│   └── approve.md              ← 人工审批
├── engine/                     ← Python 工作流引擎
│   ├── __init__.py
│   ├── dag.py                  ← DAG 拓扑（Kahn 算法）
│   ├── executor.py             ← 执行器（支持 event_emitter）
│   ├── loader.py               ← YAML loader + Zod schema 验证
│   ├── variables.py            ← $node.output 变量替换
│   ├── conditions.py           ← when 条件求值
│   ├── artifacts.py            ← artifact 文件读写
│   ├── bash_gate.py            ← bash 验证门
│   ├── events.py               ← SSE 结构化事件系统
│   ├── ai_gate.py              ← MiniMax/Claude AI gate
│   └── subagent.py             ← subagent factory
├── ui/                         ← FastAPI + Svelte UI
│   ├── backend/
│   │   ├── main.py             ← FastAPI app（lifespan 管理 EventQueue）
│   │   ├── workflow_api.py     ← workflow CRUD
│   │   ├── engine_api.py       ← run 启动 + in-memory run store
│   │   └── stream_api.py       ← SSE /api/stream/{run_id}
│   └── frontend/               ← Svelte 4 SPA
│       ├── src/
│       │   ├── lib/
│       │   │   ├── api.ts      ← REST client + connectSSE()
│       │   │   └── types.ts    ← WorkflowEvent, NodeLiveStatus
│       │   └── routes/
│       │       ├── +page.svelte   ← workflow 列表
│       │       └── runs/+page.svelte ← run 历史 + 实时 SSE 日志
│       └── package.json
└── scripts/
    ├── init.py                 ← 初始化项目 workflow
    └── test-engine.py          ← 引擎单元测试
```

---

## DAG Node 类型（7 种）

### 1. `prompt` — 内联 AI prompt
```yaml
- id: classify
  prompt: |
    分析问题类型：bug/feature/enhancement
    
    输出 JSON：{"type": "bug|feature|..."}
  model: haiku
  output_format:
    type: object
    properties:
      type: { enum: ["bug", "feature", "enhancement"] }
```

### 2. `command` — 调用命令文件
```yaml
- id: plan
  command: team-tasks-plan
  context: fresh
  model: sonnet
```

### 3. `bash` — 确定性验证门（最重要！）
```yaml
- id: typecheck
  bash: |
    cd $ARTIFACTS_DIR/..
    tsc --noEmit 2>&1 | head -20
  timeout: 60000
```

### 4. `loop` — AI 循环直到完成
```yaml
- id: implement
  command: team-tasks-implement
  loop:
    prompt: "实现下一项任务，运行验证"
    until: ALL_TASKS_COMPLETE
    fresh_context: true
    max_iterations: 20
```

### 5. `approval` — 人工审批门
```yaml
- id: review-approve
  approval:
    message: "代码已就绪，请审核"
    on_reject:
      prompt: "根据反馈修复问题"
      max_attempts: 3
```

### 6. `cancel` — 条件取消
```yaml
- id: cancel-if-trivial
  cancel: "问题太简单，无需 workflow"
  when: "$classify.output.type == 'chore'"
```

### 7. `script` — Bun/UV 脚本
```yaml
- id: gen-report
  script: |
    import { generateReport } from './report.ts'
    await generateReport(context)
  runtime: bun
  deps: [zod]
```

---

## SSE 事件流架构

Executor 执行时实时推送事件 → FastAPI SSE → 前端 EventSource：

```
WorkflowExecutor._emit_node(type, node_id, data)
  → WorkflowEventQueue.emit(run_id, event)
    → GET /api/stream/{run_id}  (FastAPI StreamingResponse)
      → Svelte EventSource('/api/stream/' + runId)
        → UI 实时更新节点状态 + 事件日志
```

### 事件类型（engine/events.py）
| event_type | 说明 |
|---|---|
| `workflow_started` | workflow 开始 |
| `workflow_completed` | workflow 成功完成 |
| `workflow_failed` | workflow 异常结束 |
| `node_started` | 节点开始执行 |
| `node_completed` | 节点成功完成 |
| `node_failed` | 节点失败 |
| `node_skipped` | 节点跳过（condition/trigger_rule） |
| `approval_requested` | 等待人工审批 |
| `tool_started` | 工具调用开始 |
| `tool_completed` | 工具调用完成 |
| `log` | 原始日志行 |

### Executor event_emitter 用法
```python
from engine.events import get_event_queue

async def event_emitter(event):
    await get_event_queue().emit(run_id, event)

executor = WorkflowExecutor(
    workflow_path="workflows/bug-fix.yaml",
    commands_dir="commands",
    workspace_dir="/tmp/workspace",
    run_id="run-abc123",
    event_emitter=event_emitter,  # 替换 platform_sender
    ...
)
```

### SSE 端点
- `GET /api/stream/{run_id}` — SSE stream，每 30s heartbeat
- `GET /api/runs/{run_id}/state` — run 状态快照

### 前端接入
```typescript
import { connectSSE, eventsToNodeStatus } from '$lib/api';

let events: WorkflowEvent[] = [];
const disconnect = connectSSE(runId, (evt) => {
  events = [...events, evt];
  const nodeStatus = eventsToNodeStatus(events, nodeIds);
});
// 清理
onDestroy(disconnect);
```

---

## 变量替换系统

### 标准变量（workflow 全局）
| 变量 | 说明 |
|---|---|
| `$WORKFLOW_ID` | 当前 workflow run ID |
| `$ARGUMENTS` | 用户原始输入 |
| `$ARTIFACTS_DIR` | artifact 文件目录 |
| `$BASE_BRANCH` | 基准分支 |
| `$CWD` | 工作目录 |
| `$PROVIDER` | 当前 AI provider |
| `$MODEL` | 当前模型 |

### 节点输出引用（跨节点传递）
```yaml
# 从 classify 节点取输出
when: "$classify.output.type == 'bug'"
bash: "gh issue view $classify.output.issue_number"

# 从 plan 节点取文件内容
prompt: |
  实施计划：
  $plan.output
  
  # plan.md 内容注入
```

### Bash 输出引用（shell quote 安全）
```yaml
- id: fetch-issue
  bash: "gh issue view $classify.output.issue_number"
  
# bash 节点中引用上游输出，自动 shell-quote：
# bash -c 'gh issue view '\''#42'\'''
```

---

## Trigger Rule（节点触发规则）

控制节点何时执行：
```yaml
- id: fast-track
  depends_on: [classify]
  trigger_rule: one_success    # 任一上游完成即执行
  when: "$classify.output.priority == 'critical'"
```

| 规则 | 含义 |
|---|---|
| `all_success` | 所有上游完成（默认） |
| `one_success` | 任一上游完成 |
| `none_failed_min_one_success` | 无失败且至少一个完成 |
| `all_done` | 所有上游非 pending 状态 |

---

## Example: Bug-Fix Workflow

```yaml
# workflows/bug-fix.yaml
name: bug-fix
description: 快速修复 GitHub issue

provider: claude
model: sonnet

nodes:
  # ── PHASE 0: 分类 ─────────────────────────────────
  - id: classify
    prompt: |
      解析 issue 信息，输出类型和优先级
      
      输入: $ARGUMENTS
      输出 JSON: {"type": "bug", "issue_number": "42", "priority": "high"}
    output_format:
      type: object
      properties:
        type: { enum: ["bug","feature"] }
        issue_number: { type: string }
        priority: { enum: ["low","medium","high","critical"] }
      required: [type, issue_number]

  # ── PHASE 1: 研究 ─────────────────────────────────
  - id: fetch-issue
    bash: |
      gh issue view $classify.output.issue_number \
        --json title,body,labels,state,url
    depends_on: [classify]

  - id: investigate
    command: team-tasks-investigate
    depends_on: [fetch-issue]
    context: fresh
    when: "$classify.output.type == 'bug'"

  # ── PHASE 2: 实施 + 循环验证 ─────────────────────────
  - id: implement
    command: team-tasks-implement
    depends_on: [investigate]
    loop:
      prompt: "实现下一项修复，运行测试验证"
      until: ALL_TASKS_COMPLETE
      fresh_context: true
      max_iterations: 10

  # ── PHASE 3: 确定性验证门 ────────────────────────────
  - id: typecheck
    bash: |
      cd $ARTIFACTS_DIR/..
      tsc --noEmit 2>&1 | head -30
    depends_on: [implement]
    timeout: 60000

  - id: test
    bash: |
      cd $ARTIFACTS_DIR/..
      npm test 2>&1 | tail -20
    depends_on: [typecheck]
    timeout: 120000

  # ── PHASE 4: AI 语义审查 ────────────────────────────
  - id: review
    command: team-tasks-review
    depends_on: [test]
    context: shared
    output_format:
      type: object
      properties:
        issues_found: { type: boolean }
        severity: { enum: ["none","low","medium","high"] }
      required: [issues_found, severity]

  # ── PHASE 5: 条件分支 ──────────────────────────────
  - id: fix-review-issues
    command: team-tasks-fix-review
    depends_on: [review]
    when: "$review.output.issues_found == true"
    loop:
      prompt: "修复审查问题，重新运行测试"
      until: $review.output.severity == 'none'
      fresh_context: false
      max_iterations: 3

  # ── PHASE 6: 人工审批 ──────────────────────────────
  - id: approve
    approval:
      message: "修复已完成，请最终审核"
    depends_on: [review]
    trigger_rule: all_done

  # ── PHASE 7: 创建 PR ──────────────────────────────
  - id: create-pr
    bash: |
      cd $ARTIFACTS_DIR/..
      git add -A && git commit -m "fix: $classify.output.issue_number" \
        && gh pr create --fill
    depends_on: [approve]
```

---

## Bash 验证门设计原则

**核心原则：确定性检查全踢给 bash，AI 只做语义审查**

| 检查类型 | 方案 | 理由 |
|---|---|---|
| TypeScript 编译错误 | `tsc --noEmit` | 确定性，毫秒级 |
| import 缺失 | `ruff check .` / `tsc --noEmit` | 确定性 |
| 测试通过 | `npm test` / `bun run validate` | 确定性 |
| 文件存在性 | `test -f path/file.ts` | 确定性 |
| 语法风格 | `ruff check .` / `eslint` | 确定性 |
| **代码逻辑错误** | AI review | 需要语义理解 |
| **架构合理性** | AI review | 需要语义理解 |
| **安全性审查** | AI review | 需要语义理解 |

### Bash gate 失败处理
```yaml
- id: typecheck
  bash: |
    tsc --noEmit 2>&1 || {
      echo "TYPE_ERRORS"
      tsc --noEmit 2>&1
      exit 1
    }
  retry:
    max_attempts: 2
    delay_ms: 5000
    on_error: transient   # transient 才重试，fatal 不重试
```

---

## 与现有 team-tasks 的关系

### 迁移策略：共存 → 渐进替换

1. **Phase 1（完全独立）**
   - 新引擎放在 `~/.openclaw/skills/archon-workflow-engine/`
   - 数据文件：`~/.openclaw/workspace-archon/`
   - 与 `~/.openclaw/workspace-coord/` 完全隔离
   - coord 不感知新引擎，直到明确激活

2. **Phase 2（新项目用新引擎）**
   - 新项目初始化：`archon-workflow-engine init <project>`
   - 提案通过新引擎派发
   - 老项目继续用现有 team-tasks

3. **Phase 3（数据迁移）**
   - 稳定后迁移老项目的 task 数据
   - 迁移工具：`scripts/migrate.py`

### 激活条件
新引擎激活后，通过 `~/.openclaw/config.yaml` 切换：
```yaml
coord:
  workflow_engine: archon-workflow-engine  # 新引擎
  # workflow_engine: team-tasks           # 旧引擎
```

---

## 实现优先级

### P0 — 核心引擎（MVP）✅ 已完成
- [x] `engine/loader.py` — YAML 加载 + schema 验证
- [x] `engine/dag.py` — Kahn 拓扑排序 + 循环检测
- [x] `engine/variables.py` — 变量替换（$node.output 等）
- [x] `engine/executor.py` — 单节点执行（prompt/bash/command/loop/approval）
- [x] `engine/bash_gate.py` — bash 验证门
- [x] `engine/ai_gate.py` — MiniMax claude --print 调用
- [x] `engine/subagent.py` — subagent factory（claude --print）
- [x] `engine/events.py` — SSE 事件流系统
- [x] 最小 Command 文件：analyze.md, plan.md, implement.md

### P0 — UI 层 ✅ 主要完成
- [x] FastAPI backend — workflow CRUD + run 执行
- [x] SSE streaming — `GET /api/stream/{run_id}` + heartbeat
- [x] Svelte 4 frontend — DAG 可视化 + run 列表
- [x] 前端 SSE 接入 — `connectSSE()` + 实时事件日志面板

### P1 — 完整流程
- [ ] `engine/conditions.py` — when 条件求值（已在 executor 内联）
- [ ] `engine/artifacts.py` — artifact 文件管理
- [ ] JSON 文件存储替代 in-memory run store（runs/ 目录）
- [ ] 4 个 workflow 文件：bug-fix, feature-dev, refactor, docs

### P2 — 集成
- [ ] `GET /api/dashboard/runs` — enriched runs + counts
- [ ] `POST /api/workflows/runs/{runId}/approve|reject` — approval 节点
- [ ] `GET /api/artifacts/{runId}/*` — 静态文件服务
- [ ] workflow discovery（从项目 .archon/workflows/ 加载）
- [ ] 项目级 workflow 覆盖

### P3 — 高级功能
- [ ] script 节点（bun/uv runtime）
- [ ] cancel 节点
- [ ] trigger_rule 全部支持
- [ ] 多 AI provider（Claude/Codex）
- [ ] Git worktree 隔离

---

## 已知 Bug 与 Workaround

### dag.py dict vs DagNode 兼容性（已修复 2026-04-18）

**问题**：`build_topological_layers()` 期望 `DagNode` 对象（dataclass），但 `load_workflow()` 返回的是 dict。导致所有节点执行时 0ms 失败，events.jsonl 为空。

**症状**：
```json
{"node_id": "parse-requirement", "state": "failed", "duration_ms": 0}
{"node_id": "backend-gen", "state": "failed", "duration_ms": 0}
```
且 `~/.archon/runs/<run_id>/events.jsonl` 为空。

**根因**：`engine/dag.py` 多处直接访问 `node.depends_on`、`node.id` 属性，未处理 dict 格式。

**修复**（已应用）：在 `build_topological_layers()` 的所有节点访问点加 hasattr 兼容：
```python
node_id = node.id if hasattr(node, 'id') else node['id']
node_deps = node.depends_on if hasattr(node, 'depends_on') else (node.get('depends_on') or [])
```

**验证**：修复后正常输出 10 层拓扑：
```
Layers: 10
  L0: ['parse-requirement']
  L1: ['backend-gen']
  ...
```

**调试三步法**（workflow 不执行时）：
1. `curl http://localhost:33335/api/workflows` — 确认 workflow 加载成功
2. `cat ~/.archon/runs/<run_id>/events.jsonl` — 确认有事件写入（空 = executor 启动时崩）
3. `cat ~/.archon/runs/<run_id>/state.json` — 查看 error 字段

---

## 关键参考（Archon 源码）

Archon 的实现细节，参考 `/root/archon/`：

| 文件 | 用途 |
|---|---|
| `packages/server/src/routes/api.ts` | **核心 API 契约**（Hono routes，见 OPENAPI.md） |
| `packages/web/` | React 19 前端（参考 UI 设计） |
| `packages/workflows/src/dag-executor.ts` | 核心引擎，最重要 |
| `packages/workflows/src/schemas/dag-node.ts` | Node schema 定义 |
| `packages/workflows/src/loader.ts` | YAML loader |
| `packages/workflows/src/condition-evaluator.ts` | when 条件 |
| `.archon/workflows/defaults/archon-fix-github-issue.yaml` | 完整 workflow 模板 |
| `.archon/commands/defaults/archon-create-plan.md` | Command 文件格式 |

### 技术栈差异（Archon → Skill）

| 组件 | Archon | Skill |
|---|---|---|
| 前端 | React 19 + Vite | **Svelte 4 + Vite** |
| HTTP 框架 | Hono + Bun | **FastAPI** |
| Run 存储 | PostgreSQL | **JSON 文件**（待实现） |
| SSE | Hono SSE + Postgres 订阅 | **FastAPI StreamingResponse + asyncio.Queue** |
| 执行引擎 | TypeScript | **Python** |
| 部署依赖 | 需 PostgreSQL | **零依赖** |

**用户偏好**：文件化存储 > 数据库。JSON 文件路线更符合口味。
