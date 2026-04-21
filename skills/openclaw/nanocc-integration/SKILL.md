---
name: nanocc-integration
description: 将 neyuki778/nanoClaudeCode (Go coding agent) 集成到项目中的完整流程——模块名修正、自定义工具扩展、Web SSE 桥接、build 验证。
category: openclaw
---
repo_tracked: true


# nanoClaudeCode 集成指南

nanoClaudeCode (neyuki778/nanoClaudeCode) 是一个纯 Go 实现的 Coding Agent Harness（S01-S08：agent loop + tool use + TODO + subagent + skills + 上下文压缩 + 会话持久化 + 后台任务）。

## 集成流程

### 1. Clone

```bash
git clone https://github.com/neyuki778/nanoClaudeCode.git /path/to/your-agent
```

### 2. 修正模块名

原始项目模块名是 `nanocc`，集成到你的项目时必须改为你的模块路径：

```bash
cd your-agent
find . -name '*.go' -not -path './.git/*' | \
  xargs sed -i 's|"nanocc|"your-module|g'
```

然后更新 `go.mod` 第一行：
```go
module your-module  // 原来是 module nanocc
```

### 3. 下载依赖

```bash
export GOPROXY=https://goproxy.cn,direct
go mod tidy
```

注意：直接 `go build` 会因网络超时下载失败，`goproxy.cn` 通常有效。

### 4. 验证基础 build

```bash
go build ./...  # 必须 clean 才能继续
```

### 5. 添加自定义工具

在 `agents/runtime/tools/` 下新增文件：

- `handlers_xxx.go` — 工具实现（Handler 函数）
- `specs_xxx.go` — 工具规格定义（Spec struct + 参数 schema）

工具注册在 `VibexSpecs()` / `ParentSpecs()` 中。

### 6. Web SSE 桥接（可选）

如果需要 HTTP 接口，在 `cmd/web/main.go` 中实现：

```go
// 关键：设置 canvas 广播函数
rtools.SetBroadcastCanvas(broadcastSSE)

// 然后注册路由
http.HandleFunc("/api/chat", chatHandler)
http.HandleFunc("/api/sse/", sseHandler)
```

工具 handler 中通过 `BroadcastCanvas` 全局变量发送 SSE 事件到前端。

## 关键坑点

### 模块名污染

如果 `find+sed` 后仍有 `nanocc` 残留（某些 import 格式），单独 patch：

```go
// 原来
import "nanocc/agents/runtime"
// 改为
import "your-module/agents/runtime"
```

### Build 顺序

`go mod tidy` 必须先跑，否则 `go build` 报 `package not in std`。

### 编译产物

```bash
go build -o your-agent-web ./cmd/web/
# 输出约 13MB 单二进制
```

## 文件清单（VibeX 集成产出）

```
your-agent/
├── cmd/web/main.go              # SSE+HTTP web 入口
├── agents/runtime/tools/
│   ├── specs_vibex.go          # 8个 VibeX 专用工具规格
│   └── handlers_vibex.go       # 工具实现 + BroadcastCanvas 全局变量
├── internal/common/config.go    # 支持 SKILLS_DIR / WORKSPACE_DIR 环境变量
└── .env                        # OPENAI_API_KEY 等配置
```

## 工具扩展模板

参考 `handlers_vibex.go`，每个工具包含：

1. `Spec` 定义（Name/Description/Parameters/Handler）
2. `Handler` 函数（返回 string，即工具输出）
3. SSE 广播（如需前端响应）
4. `optString`/`reqInteger`/`reqBool` 辅助函数（见 `specs_vibex.go`）

### BroadcastCanvas 全局变量

在 `handlers_xxx.go` 顶部声明：

```go
var BroadcastCanvas func(threadID, event string, data interface{})
```

在 `cmd/web/main.go` 中通过 `rtools.SetBroadcastCanvas(fn)` 注入。
