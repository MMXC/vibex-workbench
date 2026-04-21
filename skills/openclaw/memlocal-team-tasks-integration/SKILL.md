---
name: memlocal-team-tasks-integration
description: memlocal-team-tasks-integration — skill for openclaw category
category: openclaw
triggers:
- openclaw
- team-tasks
- gateway hook
- custom hook
- coord decision
- openclaw hook
- memlocal team tasks integration
related_skills:
- archon-workflow-engine
- openclaw-internals-reversed
- darwin-skill-execution
---
repo_tracked: true


# MemLocal 接入 team-tasks 模板工作流

## 核心原则

每次 team-tasks 阶段任务（phase1/phase2）都应引用 `memlocal-memory` skill，格式固定：

```
# `memlocal-memory` — ★ 本地记忆系统
#   - 开始前：`memlocal search "<任务关键词>"`
#   - 完成后：`memlocal mine <work_dir> --wing <project>`
```

## ⚠️ 命令陷阱（必须记住）

### ❌ `--room` 参数不存在
MemPalace `mine` 命令的 Room 是**自动从文件路径检测**的，不接受 `--room` 参数：
```bash
# 错误 —— 会报 "unrecognized arguments: --room"
memlocal mine <dir> --wing <project> --room src

# 正确 —— room 自动检测
memlocal mine <dir> --wing <project>
```

### ❌ 旧版 python3 脚本已废弃
不要用 `python3 /root/.local/lib/memlocal.py`，那是旧版简单 JSON 实现，和 MemPalace 协议不兼容。

### ✅ 正确入口
```bash
memlocal search "<query>"     # BM25 全文搜索
memlocal mine <dir> --wing <name>  # 增量写入记忆
memlocal status               # 查看 drawer 数量
memlocal wake-up             # L0 + L1 冷启动上下文
```

## Wing/Room 结构约定

| Wing | 用途 | Room（自动） |
|------|------|-------------|
| `coord` | 我的协调经验 | `general` |
| `openclaw` | OpenClaw 团队知识 | `src`, `testing`, `general` |
| `<project>` | 具体项目 | `general`（自动） |
| `mine` | 我的会话记录 | `sessions`（自动） |

## mine 增量策略

大目录直接 mine 会超时，用 `--limit` 分批：
```bash
memlocal mine <big_dir> --wing <project> --limit 50   # 先试5个
# 确认无误后，去掉 --limit 全量跑
memlocal mine <big_dir> --wing <project>
```

dry-run 预览（不写入）：
```bash
memlocal mine <dir> --wing <project> --dry-run
```

## 更新任何模板时的检查清单

更新 team-tasks 模板时，grep 检查是否有旧路径残留：
```bash
grep -rn "python3 /root/.local/lib" /root/.openclaw/skills/team-tasks/scripts/templates/
grep -rn "--room" /root/.openclaw/skills/team-tasks/scripts/templates/
grep -rn "room match" /root/.openclaw/skills/team-tasks/scripts/templates/
```

## 模板文件位置

所有 team-tasks 模板在：
`/root/.openclaw/skills/team-tasks/scripts/templates/`

包含 memlocal 引用的模板（10个）：
- phase1-analyst-analyze-requirements.md
- phase1-pm-create-prd.md
- phase1-architect-design-architecture.md
- phase1-coord-decision.md
- phase2-dev.md
- phase2-tester.md
- phase2-reviewer.md
- phase2-reviewer-push.md
- phase2-coord-completed.md
- agent-submit.md
