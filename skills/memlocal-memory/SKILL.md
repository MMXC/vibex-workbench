---
name: memlocal-memory
description: Vibex Workbench 本地偏好记忆系统 — Go 实现的轻量 memlace，存储在项目 .memlace/ 目录。包含 SessionManager 单例模式、HTTP API 集成、最佳实践。
category: openclaw
title: MemLocal Memory
triggers:
- memlace
- 偏好存储
- 本地记忆
- clarification session
- Go singleton
related_skills:
- spec-first-workflow
- memlocal-team-tasks-integration
---

# MemLocal Memory

Vibex Workbench 的本地偏好记忆系统，纯 Go stdlib 实现，零外部依赖。

## 核心架构

```
.memlace/config.json          ← 用户配置（memory_store 路径等）
.memlace/clarifications/      ← 澄清会话历史（*.clf.json）
.memlace/local/preferences.json ← 本地独立偏好存储
generators/memlace/           ← Go 包（config/store/search/clarification/tools）
agent/cmd/web/server.go       ← HTTP API 集成（/api/clarifications）
```

## 配置加载

配置文件：`.memlace/config.json`（项目根目录下）

```json
{
  "memory_store": "hermes",
  "clarifications": ".memlace/clarifications",
  "work_dir": "",
  "hermes_link": true
}
```

| 字段 | 说明 |
|------|------|
| `memory_store` | `hermes`（读 `~/.hermes/memories/*.md`）/ `local`（读 `.memlace/local/`）/ 绝对路径 |
| `clarifications` | 澄清会话目录 |
| `hermes_link` | true 时同时读 Hermes 偏好 |

## 踩坑教训：Go HTTP Handler 单例模式

在 HTTP handler 中使用模块级单例，避免每次请求重建内存：

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

**踩坑教训**：如果不加 `_memLaceMgrOnce`，每次 HTTP 请求都会 `NewSessionManager`，内存中的 sessions map 是新创建的，**文件写入结果无法共享**，看起来像 bug 实际上只是内存没复用。

**关键点**：
- `Once` 布尔标志控制只初始化一次
- 所有 HTTP handler 调用 `getMemLaceMgr()` 获取单例
- 初始化失败返回 nil，handler 负责处理降级

## HTTP API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/clarifications` | GET | 列出所有会话（摘要） |
| `/api/clarifications/:spec` | GET | 单个会话详情 |
| `/api/clarifications/:spec` | POST | start/qa/draft/confirm 四种 action |

## Clarification 会话状态机

```
draft → in_progress → confirmed
                   ↳ draft_amend（重新修改）
```

阶段顺序：tech_stack → mvp_prototype → frontend_split → user_stories（可跳阶段）

## 与 spec-first-workflow 的关系

spec-first workflow 的 Step 4（四阶段澄清）使用 memlace API：
- `clarification_start` — 创建澄清会话
- `clarification_qa` — 每轮 Q&A 追加
- `clarification_draft` — 出草稿
- `clarification_confirm` — 锁定会话 → spec confirmed 状态
