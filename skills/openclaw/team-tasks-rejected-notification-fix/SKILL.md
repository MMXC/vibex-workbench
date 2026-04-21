---
name: team-tasks-rejected-notification-fix
description: team-tasks-rejected-notification-fix — skill for openclaw category
category: openclaw
triggers:
- openclaw
- team-tasks
- gateway hook
- custom hook
- coord decision
- openclaw hook
- team tasks rejected notification fix
related_skills:
- archon-workflow-engine
- openclaw-internals-reversed
- darwin-skill-execution
---
repo_tracked: true


# Team-Tasks Rejected 通知修复 — 特殊字符级联问题

## 触发场景

当 `cmd_update` 处理 `rejected` 状态时，`title` 变量拼接了 `（被驳回）` 后缀：

```python
# ❌ 错误：title 含特殊字符
title = f"{args.project}/{stage_id}（被驳回）"
```

这个 `title` 作为 `project_task` 传给 `make_rejected_message` → `heartbeat_guide_for()` → 嵌入 `task update` 命令示例。

**结果：** Slack 消息里的 `task update` 命令带 `（被驳回）`，CLI 无法解析，agent 收到通知但无法执行。

## 修复方案

### 1. `slack_notify_templates.py` — 新增 strip 辅助函数

```python
def _strip_status_suffix(pt: str) -> str:
    """Strip status suffix like （被驳回） from task identifier for CLI commands."""
    return pt.replace("（被驳回）", "").replace("(被驳回)", "").strip()
```

应用到 `heartbeat_guide_for` 和 `cli_warning_for`：

```python
def heartbeat_guide_for(agent: str, project_task: str = "") -> str:
    pt = _strip_status_suffix(project_task.replace("/", " "))
    # ... CLI 命令示例用 clean 的 pt

def cli_warning_for(project_task: str = "") -> str:
    pt = _strip_status_suffix(project_task.replace("/", " "))
    # ... CLI 命令示例用 clean 的 pt
```

### 2. `task_manager.py` — 移除 title 的冗余后缀

```python
# ✅ 正确：title 干净，消息模板已有"任务被驳回"
title = f"{args.project}/{stage_id}"
```

## 修复前后对比

```
# 修复前（CLI 解析失败）
task update vibex-proposals-20260416 coord-decision（被驳回） done

# 修复后（CLI 正常）
task update vibex-proposals-20260416 coord-decision done
```

## 根因

`make_rejected_message` 模板第一条就是 `🔴 <@{user}> 任务被驳回: {project_task}`，再加 `（被驳回）` 冗余且破坏下游。

## 文件位置

- `task_manager.py` line ~2503: `title = f"{args.project}/{stage_id}"`
- `slack_notify_templates.py` line ~130-148: `_strip_status_suffix` + `heartbeat_guide_for` + `cli_warning_for`
