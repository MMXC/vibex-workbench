---
name: specpilot-agent
description: |
  SpecPilot — file-based prototype workspace manager (simplified, no backend services).
  触发条件：用户提到"原型"、"preview"、"生成原型"、"看原型"、"做界面"。
  触发词：prototype、preview、generate、sketch、做、界面。
---

# SpecPilot — File-Based Prototype Workspace

**设计原则：文件系统即架构，无后台服务。**

## 目录结构

```
{workspace}/
  .specpilot/
    prototypes/              # Agent 生成的 HTML 原型，按 spec-id 命名
      {spec-id}.html        # 绑定 spec YAML 的 HTML 原型
    components/             # 可复用 HTML 片段
    specs/                  # SPEC YAML 副本（可选，用于归档）
```

## 核心约束

- **无 Python 服务**：不需要 DataCenter/EC/MF shell 后台进程
- **无端口依赖**：不需要 7890/5177 等端口
- **文件系统是唯一状态**：原型就是 HTML 文件，绑定关系在文件名
- **Go backend 负责文件读写**：Agent 通过 Go tool handler 写文件，前端通过 Go HTTP API 读文件

## 端到端链路

```
用户说意图
  → spec-designer 写 spec YAML (claude-design)
  → claude-design / sketch 生成 HTML 原型
  → Agent 写文件 → {workspace}/.specpilot/prototypes/{spec-id}.html
  → SpecSlot 原型区 iframe 加载
  → done
```

## Agent 工作流

### 流程 1：生成新原型

```
1. 理解用户意图（clarify/grill-me）
2. spec-designer 生成 SPEC.yaml（写入 workspace）
3. 决定 spec-id（通常取自 spec.name 或 L5 的 slug）
4. claude-design 生成 HTML 原型（单文件 HTML，含内联 CSS/JS）
5. Agent 写文件：
   write_file(
     path="{workspace}/.specpilot/prototypes/{spec-id}.html",
     content="<html>..."
   )
6. 前端自动检测到文件 → iframe 加载
```

### 流程 2：修改原型

```
1. 用户说"把按钮改成红色"
2. Agent 读文件 → read_file("{workspace}/.specpilot/prototypes/{spec-id}.html")
3. 修改 HTML
4. 写回 → write_file(..., content=修改后内容)
5. 前端 iframe 自动热刷新（location.reload()）
```

### 流程 3：绑定 spec → 原型

```
spec YAML 文件中加：
  prototype:
    type: file
    path: .specpilot/prototypes/{spec-id}.html

前端加载 spec 时，读取 prototype.path → 拼接 workspaceRoot → 加载 iframe
```

## 路由约定

| 操作 | 工具 | 路径 |
|------|------|------|
| 写原型 | `write_file` | `{workspace}/.specpilot/prototypes/{spec-id}.html` |
| 读原型 | `read_file` | `{workspace}/.specpilot/prototypes/{spec-id}.html` |
| 列表原型 | `bash` | `ls {workspace}/.specpilot/prototypes/` |
| 删除原型 | `bash` | `rm {workspace}/.specpilot/prototypes/{spec-id}.html` |

## 前端集成

### 预览加载

SpecSlot 原型区通过 Go backend HTTP API 加载 HTML：

```
GET /api/specpilot/prototype/{spec-id}
→ 返回 {exists: bool, html?: string, path?: string, updatedAt?: string}
```

iframe 加载：
```
/api/specpilot/prototype/{spec-id} → 返回完整 HTML（Content-Type: text/html）
```

### 原型列表

```
GET /api/specpilot/prototypes
→ 返回 {prototypes: [{specId, path, size, updatedAt}, ...]}
```

### spec → prototype 绑定

spec YAML 的 `prototype.type = file` 时，前端用 `prototype.path` 字段拼接 workspaceRoot。

## Go Backend Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/specpilot/prototype/{specId}` | 获取单个原型（HTML 或元信息） |
| GET | `/api/specpilot/prototypes` | 列出所有原型 |
| GET | `/api/specpilot/status` | 工作区状态（.specpilot 目录是否存在） |

## 与旧版 Python Service 的区别

| 维度 | 旧版（Python Services） | 新版（File-Based） |
|------|----------------------|-------------------|
| DC 状态 | Python Flask 服务 (7890) | spec YAML 中的 `data: {}` |
| EC 事件 | Python SSE 服务 | HTML 中的 JS 事件 |
| MF 组件 | Python http.server (5177) | 内联 HTML 片段 |
| 热更新 | WebSocket reload | location.reload() |
| 启动 | specpilot init + start | 无需启动 |
| 端口依赖 | 需要 7890/5177 | 无 |

## 适用场景

✅ 纯 UI 原型（HTML + CSS + JS）  
✅ 单页面交互原型  
✅ 快速设计验证  

❌ 需要实时数据流（用实时数据改 spec YAML）  
❌ 需要跨组件状态共享（改用 sessionStorage/localStorage）  
❌ 需要多人协作（文件系统不适合）  
