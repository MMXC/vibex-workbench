# SPEC-QA 验证报告

**日期**: 2026-04-20
**Frontend**: http://localhost:5173 ✅
**Backend SSE**: http://localhost:33335 ✅
**整体状态**: 5/5 通过（mock 路径）⚠️ Agent（33338）payload 未对齐，见「已知问题」P0

---

## 概览

| Feature | Spec 三层 | E2E 验证 | 状态 |
|---|---|---|---|
| thread-manager | ✓✓✓ | ✓ | ✅ |
| run-engine | ✓✓✓ | ✓ | ✅ |
| canvas-workbench | ✓✓✓ | ✓ | ✅ |
| artifact-registry | ✓✓✓ | ✓ | ✅ |
| workbench-shell | ✓✓✓ | ✓ | ✅ |

---

## Feature 验证结果

### 1. thread-manager

| 检查项 | 结果 |
|---|---|
| specs/feature/thread-manager/service.yaml | ✓ |
| specs/feature/thread-manager/service.frontend.yaml | ✓ |
| specs/feature/thread-manager/service.backend.yaml | ✓ |
| 新建线程（点击 "+ 新建"） | ✓ — 出现 "新线程 draft" |
| 线程列表显示 | ✓ — sidebar 显示线程项 |
| 线程切换 | ✓ — 点击线程项切换 |

**E2E 截图**: `/tmp/qa_final_02.png`

---

### 2. run-engine

| 检查项 | 结果 |
|---|---|
| specs/feature/run-engine/service.yaml | ✓ |
| specs/feature/run-engine/service.frontend.yaml | ✓ |
| specs/feature/run-engine/service.backend.yaml | ✓ |
| POST /api/runs | ✓ — 返回 200，runId 已生成 |
| 后端 mock run 事件（33335）| ✓ — mock 路径符合预期事件序列 |
| SSE 事件流 | ✓ — 8 个节点正确推送 |
| Console 无 error | ✓ — 无错误输出 |

**E2E 截图**: `/tmp/qa_final_03.png`

---

### 3. canvas-workbench

| 检查项 | 结果 |
|---|---|
| specs/feature/canvas-workbench/service.yaml | ✓ |
| specs/feature/canvas-workbench/service.frontend.yaml | ✓ |
| specs/feature/canvas-workbench/service.backend.yaml | ✓ |
| Canvas 区域可见 | ✓ — `<div class="canvas-area">` 渲染 |
| canvasStore 订阅 | ✓ — `+page.svelte` 订阅 `canvasStore`，`canvasNodes` 响应式更新 |
| 节点渲染（8 nodes，mock） | ✓ — placeholder 显示 "(8 nodes)"，节点来自 SSE `run.started` + `tool.called` 事件（mock 路径） |
| Console 无 error | ✓ |

**根因修复**: `backend/sse_server.py` 的 `/api/sse/threads/<threadId>` 端点之前缺少 `send_sse_headers()` 调用，导致浏览器收到 `HTTP/0.9 invalid response` 而拒绝连接。已在 S1 修复。

---

### 4. artifact-registry

| 检查项 | 结果 |
|---|---|
| specs/feature/artifact-registry/service.yaml | ✓ |
| specs/feature/artifact-registry/service.frontend.yaml | ✓ |
| specs/feature/artifact-registry/service.backend.yaml | ✓ |
| Artifact 创建 | ✓ — 后端推送 `artifact.created`，事件含 `name: "output.py"`, `type: "code"` |
| Artifact 面板 | ✓ — `[code] output.py` 按钮出现在 composer 区域上方 |
| Console 无 error | ✓ |

---

### 5. workbench-shell

| 检查项 | 结果 |
|---|---|
| specs/feature/workbench-shell/service.yaml | ✓ |
| specs/feature/workbench-shell/service.frontend.yaml | ✓ |
| specs/feature/workbench-shell/service.backend.yaml | ✓ |
| Shell 布局渲染 | ✓ — WorkbenchShell 组件正常渲染 |
| Sidebar（ThreadList） | ✓ — 线程列表显示 |
| Composer | ✓ — 文本/图片/文件/URL tabs，输入框，"发送 ⌘↵" 按钮 |
| ArtifactPanel | ✓ — 面板组件存在并响应 artifactStore |
| Console 无 error | ✓ |

---

## 已修复的 Bug

### S1 — SSE server 无法启动
- **问题**: `BaseHTTPRequestHandler` 未直接 import，`HTTPServer` 未定义
- **修复**: 添加 `from http.server import BaseHTTPRequestHandler, HTTPServer`
- **验证**: `curl localhost:33335/api/health` → `{"status":"ok"}` ✅

### S1 — SSE 端点缺少 HTTP 响应头
- **问题**: `/api/sse/threads/<threadId>` 在发送事件前没有发送 HTTP 200 + SSE headers
- **修复**: 在 `do_GET()` SSE 分支中添加 `self.send_sse_headers()` 调用
- **验证**: `curl -N localhost:33335/api/sse/threads/test` 返回正确 SSE 事件流 ✅

---

## 已知问题

### P0 — Agent（33338）SSE payload 与前端契约未对齐

报告测试基于 mock 后端（33335），但 Agent（33338）存在以下不一致：

**1. `tool.called` 字段名完全不匹配**
- **Agent 发送**（`agent/cmd/web/server.go:129-131`）：`{ tool, call_id, args }`
- **sse.ts 期望**（`frontend/src/lib/sse.ts:61-70`）：`{ invocationId, runId, toolName, args }`
- **影响**：Canvas 上不会创建 tool 节点，run→tool edge 也不会生成

**2. `run.started` 字段名不一致**
- **Agent 发送**（`server.go:229-234`）：`run_id`（下划线）
- **sse.ts 读取**（`sse.ts:29-30`）：`data.runId`（驼峰）
- **影响**：runStore 无法追踪活跃 run，但 run 节点仍可通过其他路径创建

**3. 缺少 `canvas.tdd_nodes` / `canvas.tdd_cycle` listener**
- Agent TDD handler（`agent/vibex/domain/tdd/handlers.go:572,594`）广播这两个事件
- `sse.ts` HANDLERS 对象中没有对应条目
- **影响**：Canvas 上 TDD 循环节点不会渲染

**优先级建议**：
- 若以真实 Agent 演示为准 → 先修 payload（Point 1），再修 `canvas.tdd_*`（Point 3）
- 若以 spec 澄清链叙事为准 → S4-C3 dialog 扩展仍合理

### ⚠️ 测试环境声明

本报告验证基于 mock backend（`sse_server.py` / `main.go`，端口 33335）。
Agent（33338）行为以上述代码审查为准，未做完整 E2E 验证。

**无其他已知问题。**

---

## 结论

**通过率: 5/5**（基于 mock backend 33335）

- 5 个 feature 均有完整三层分离（service.yaml / service.frontend.yaml / service.backend.yaml）✅
- mock SSE 竖切完整，事件序列符合预期 ✅
- 端到端流程（mock 路径）：新建线程 → 输入消息 → 触发 run → SSE 事件 → canvas 节点 → artifact 创建 ✅
- 无 console error ✅

**⚠️ Agent（33338）路径不在本报告验证范围内**，见「已知问题」P0 条目。

