---
name: team-tasks-json-debugging
description: team-tasks-json-debugging — skill for openclaw category
category: openclaw
triggers:
- openclaw
- team-tasks
- gateway hook
- custom hook
- coord decision
- openclaw hook
- team tasks json debugging
related_skills:
- archon-workflow-engine
- openclaw-internals-reversed
- darwin-skill-execution
---
repo_tracked: true


# Team-Tasks JSON 调试方法

## 背景

team-tasks DAG 模式的 JSON 结构中，`stages` 里的 log 条目格式与预期不同。

## 关键发现：logs 条目的实际格式

**预期（错误）**：`logs[].action` + `logs[].reason` 字段  
**实际（正确）**：`logs[].event` 字符串，格式如 `"status: ready → blocked (reason: ...)"`

```python
import json
data = json.load(open('/root/.openclaw/workspace-coord/team-tasks/<project>.json'))
stages = data.get('stages', {})

# 找 done / blocked / rejected 事件
for sid, s in stages.items():
    for log in s.get('logs', []):
        event = log.get('event', '')
        if not event:
            continue
        # 解析 event 字符串
        if 'done' in event.lower():
            print(f"[DONE] {sid}: {event[:200]}")
        if 'blocked' in event.lower():
            print(f"[BLOCKED] {sid}: {event[:200]}")
        if 'rejected' in event.lower():
            print(f"[REJECTED] {sid}: {event[:200]}")
```

## 快速查看某个项目的 reject 概览

```python
import json
data = json.load(open('/root/.openclaw/workspace-coord/team-tasks/<project>.json'))
for sid, s in data['stages'].items():
    for l in s.get('logs', []):
        evt = l.get('event', '')
        if 'rejected' in evt.lower():
            print(f"{sid}: {evt[:250]}")
```

## 常见 event 类型

| event 内容 | 含义 |
|-----------|------|
| `status: pending → ready` | 任务被派发 |
| `status: ready → blocked` | Reviewer/coord 因上游问题阻塞 |
| `status: ready → done` | 任务完成 |
| `status: done → rejected` | Coord/Reviewer 驳回（reason 在括号内） |
| `status: blocked → pending` | 阻塞解除，重新等待 |
