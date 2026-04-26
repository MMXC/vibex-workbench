---
name: vibex-agent-ops
description: Vibex 内置 Agent 部署、调试与重构验证流程
category: devops
tags: [vibex, go, agent, deployment, sse, skills]
version: 1.1.0
created: 2026-04-20
---
repo_tracked: true


# Vibex Agent Ops

## Trigger
- 部署 vibex-agent-web 二进制
- 调试 agent 连接问题、SSE 流、skills 加载
- 重构后需要验证 build 是否正常

## 前置条件
- Go 1.22+
- /root/vibex-workbench/agent/ 已构建
- API key（OpenAI 或 OpenAI-compatible）

## 快速启动

```bash
cd /root/vibex-workbench/agent

# 用 env 启动（调试用，不依赖 .env 文件）
OPENAI_API_KEY=sk-xxx OPENAI_BASE_URL=https://api.openai.com/v1 \
OPENAI_MODEL=gpt-4o SKILLS_DIR=/root/.hermes/skills \
WORKSPACE_DIR=/root/vibex-workbench \
./vibex-agent-web &

# 用 .env 文件（生产用）
nohup ./vibex-agent-web > /tmp/agent.log 2>&1 &
```

## 验证步骤

1. 健康检查（不需要真实 key 就能通）:
   curl http://localhost:33338/health

2. SSE 连接:
   curl -N http://localhost:33338/api/sse/test-thread

3. Skills 数量:
   curl http://localhost:33338/api/skills

4. Chat 测试（需真实 key 才有 SSE 流）:
   curl -X POST http://localhost:33338/api/chat \
     -H "Content-Type: application/json" \
     -d '{"threadId":"test","input":"hi"}'

## 常见问题

- Server 启动后立即退出: OPENAI_API_KEY 为空或 key 无效
- 端口被占用: fuser -k 33338/tcp
- Health 正常但 chat pending: key 是假的（需要真实 key）
- Skills 加载为 0: SKILLS_DIR 路径不对，检查 /root/.hermes/skills 是否存在

## Agent vs Direct 写文件决策

spawn Go agent 写 L5 spec 时发现：context 紧张（31K token 读完 spec 就耗尽）导致 agent 读了很多但写不出任何文件。**决策原则**：

| 任务类型 | 推荐方式 | 原因 |
|---------|---------|------|
| 写结构化文件（spec/代码/配置）| **直接 write_file** | 结构已知，输入明确，直接写最快 |
| 复杂推理 + 多步工具调用 | spawn agent | 需要 LLM 推理和状态管理 |
| 需要先读很多上下文再写 | 读完→直接写 | 避免 agent 冷启动时 token 耗尽 |
| 调试/探索性任务 | spawn agent | 可能有多次迭代 |

**实战经验**：Go agent 在「冷启动 + 长 prompt + 写文件」场景下不可靠——token 被 reading 阶段耗尽，writing 阶段饿死。解法：主 agent 读完所有输入，**直接写文件**，不 spawn 子 agent 写。

agent/
  cmd/web/
    main.go      入口（30行）
    server.go    HTTP处理器+tool loop（~300行）
    broadcast.go SSE广播（~90行）
    thread.go    线程状态（~30行）
  vibex/domain/
    registry.go  统一注册器（注入 Broadcaster）
    spec/       8个 spec 工具（factory 模式）
    tdd/        3个 TDD 工具（依赖注入 bc）
  agents/runtime/tools/  nanoClaudeCode base tools（保持纯净）

## 架构陷阱（关键）

### ⚠️ server.go vs runtime.go 双 runToolLoop

`cmd/web/server.go` 和 `agents/runtime/runtime.go` 各有独立的 `runToolLoop` 实现。

- `server.go` → 33338 SSE/HTTP 模式（前端实际调用的那个）
- `runtime.go` → stdin/CLI 交互模式

**任何改动都要同时改两个文件**，否则 33338 不生效。

修法：
1. `runToolLoop` 签名改为返回 `(answer, turnItems, error)` — 三个值都要改
2. 内部所有 `return "", err` → `return "", inputItems, err`
3. `runAgentTurn` 里解构 `answer, turnItems, err := runToolLoop(...)` 并调用 reflection

### ⚠️ go build 路径

lint 工具的路径解析有偏差（用 `/usr/lib/go-1.22/src/`），但真实 build 用 `go build ./...` 在 agent 目录执行。以 `go build` 输出为准。

### ⚠️ responses 类型转换

```go
// 错误：Output 是 union type，不能直接 string()
calls[0].OutputLen = len(string(item.OfFunctionCallOutput.Output))

// 正确
calls[0].OutputLen = len(fmt.Sprintf("%v", item.OfFunctionCallOutput.Output))
```

### ⚠️ patch 自改代码时的保守策略

reflection 尝试给 handler 加 SSE broadcast 时，先检查目标文件是否已有 `sse` 包 import：
```go
hasSSEImport := strings.Contains(s, `"sse"`) || strings.Contains(s, `sse `)
if !hasSSEImport { return false, nil } // 跳过，避免破坏编译
```

### ⚠️ verify 自改代码后编译通过

```bash
cd /root/vibex-workbench/agent && go build ./...
```

## 重构检查清单

1. go build ./... exit 0
2. go build -o vibex-agent-web ./cmd/web/
3. Health 返回 status ok
4. SSE connected 事件正常
5. Skills count 大于 0
6. 真实 key 填入后 chat 有 SSE 流输出
