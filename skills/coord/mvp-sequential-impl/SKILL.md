---
name: mvp-sequential-impl
version: "0.1"
level: coord-skill
description: "顺序实现 spec 列表的 cron job 模式 — 状态文件游标 + 每次 Slack 汇报"
triggers:
  - "批量实现 implementation_phases 中的多个 spec"
  - "每次 20min，顺序执行，Slack 汇报"
created: "2026-04-27"
---

# mvp-sequential-impl — 顺序实现 spec 列表的 cron job 模式

## 适用场景
批量实现 implementation_phases 中的多个 spec，每次一个，20min 间隔，Slack 汇报进度。

## 核心设计：状态文件控制游标

```json
// /root/.hermes/mvp-impl-state.json
{
  "current": 0,
  "specs": [
    {"name": "FEAT-xxx", "phase": "P0", "file": "path/to/file.py", "status": "pending"},
    ...
  ]
}
```

- `current` 是游标，每次 +1
- `status`: pending → done | blocked | skip
- 依赖未满足时标记 `blocked` + 原因，继续下一个

## cron job 创建

```
schedule: every 20m    （不能用 "once in 20m" — 会报错）
repeat: 50
deliver: slack:C0ARQQ7NE7M
```

prompt 核心逻辑：
```
1. 读 /root/.hermes/mvp-impl-state.json，取 current 对应 spec
2. 如果 current >= len(specs) → 输出「全部完成」+ pause 自己
3. 读 spec YAML（L4-feature/ 或 L5-slice/）
4. 读目标代码文件
5. patch 实现
6. 验证（见下表）
7. git add + commit + push
8. 更新 L1 spec_status（patch vibex-workbench-mvp.yaml）
9. 更新状态文件 current++
10. Slack 汇报
```

## 验证命令速查

| 文件类型 | 验证命令 |
|---|---|
| Python | `python3 -m py_compile {file}` |
| Svelte/TS | `cd frontend && npx svelte-check --threshold error` |
| Spec 一致性 | `python3 scripts/validate_specs.py` |
| Go | `go build ./...` |

## Slack 汇报格式

```
✅ MVP 实现进度 [{current}/N]
**Spec**: {spec_name}
**文件**: {file}
**Commit**: {commit_hash}
**下一步**: {next_name} 或 "全部完成！🎉"
```

blocked 时：
```
⏸ [{current}/N] {spec_name} 被 block：{原因}
继续下一个：{next_name}
```

## L1 spec_status 更新格式

在 `specs/L1-goal/vibex-workbench-mvp.yaml` 的 `spec_status` 下：

```yaml
  {spec_name}: {status: done, note: "实现完成 YYYY-MM-DD"}
```

同时追加 changelog 条目。

## 注意事项
- 每次 commit 尽量小，单 spec 单一 commit
- 不要大改，只修 spec 描述的核心问题
- blocked 时继续下一个，不要卡住整个流程
- 完成全部后 cron job 自动 pause
