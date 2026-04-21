---
name: vibex-backend-ts-debt-workarounds
description: vibex-backend-ts-debt-workarounds — skill for openclaw category
category: openclaw
triggers:
- openclaw
- team-tasks
- gateway hook
- custom hook
- coord decision
- openclaw hook
- vibex backend ts debt workarounds
related_skills:
- archon-workflow-engine
- openclaw-internals-reversed
- darwin-skill-execution
---
repo_tracked: true


# vibex-backend TypeScript 工作目录和工具链

## 关键路径

| 用途 | 路径 |
|------|------|
| Backend 代码 | `/root/.openclaw/vibex/vibex-backend/` |
| Frontend 代码 | `/root/.openclaw/vibex/vibex-fronted/` |
| 文档目录 | `/root/.openclaw/vibex/vibex-backend/docs/` |
| Git remote | `git@github.com:MMXC/vibex.git` |

**注意**: 不是 `~/vibex/` ——那是空目录占位符。

## TypeScript 检查

```bash
cd /root/.openclaw/vibex/vibex-backend
./node_modules/.bin/tsc --noEmit 2>&1   # 完整输出
./node_modules/.bin/tsc --noEmit 2>&1 | grep "^src/" | wc -l  # 仅 src/ 错误数
./node_modules/.bin/tsc --noEmit 2>&1 | grep "^src/" | cut -d: -f1 | sort -u  # 受影响文件
./node_modules/.bin/tsc --noEmit 2>&1 | grep "error TS" | sed 's/.*error TS[0-9]*: //' | sort | uniq -c | sort -rn  # 按错误类型统计
```

**常见陷阱**:
- `npx tsc` — 不工作（显示 "not the tsc command you are looking for"）
- `npx --yes typescript tsc` — 不工作（"could not determine executable"）
- `pnpm run type-check` — 不存在（scripts 里没有）
- **正确方式**: 直接调用 `./node_modules/.bin/tsc`

## 已知 TS 错误基线（2026-04-14）

- 总错误数: 173
- 受影响文件: 40
- 5大类别:
  1. `Expected 2 arguments, but got 1` (~18处) — auth() 签名变更
  2. `Object literal may only specify known properties` (~65处) — openapi.ts schema 格式
  3. `'auth' is possibly 'null'` (~20处) — auth null 检查
  4. `'z' only refers to a type` (~6处) — zod 导入
  5. 其他杂项 (~64处)

## CI 相关

- E1 Epic 已将 frontend `tsc --noEmit` 加入 CI
- Backend TS 债务是历史债务，不在 E1 scope 内
- package.json 无 `type-check` script，需直接调 tsc
