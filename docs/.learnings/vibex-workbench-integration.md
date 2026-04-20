# Project Learnings: vibex-workbench-integration

**项目**: VibeX Workbench 前后端集成落地
**完成日期**: 2026-04-20
**Agent**: coord (收口)

## 项目概述

前后端集成落地项目，完成 6 个 Epic 的开发与集成：
- E1: SSE Backend Integration
- E2: Thread Management
- E3: Run Engine
- E4: Artifact Registry
- E5: Canvas Orchestration
- E6: Workbench Shell

## 关键经验

### 1. Monorepo 结构下测试策略
- `npm test` 需在 frontend 子目录执行（根目录无 package.json）
- 测试通过率：139/139 (100%)
- 工具链：vitest

### 2. SSE 集成要点
- `import.meta.env.VITE_SSE_URL` 替代硬编码 URL
- 需要处理 SSE 内存泄漏（dev 阶段修复过一次）

### 3. Canvas 集成
- 使用 @xyflow/svelte + dagre 做节点布局
- addEdge() 需补充 id 字段（常见遗漏）

### 4. 持久化策略
- Thread: IndexedDB 四态 UI
- Artifact: 持久化 + 预览 + 拖入 Composer

### 5. CI/CD
- gitignore 需排除 test-results 和 playwright cache
- Reviewer 驳回后需主动修复 TS 错误

## 统计数据

| 指标 | 数值 |
|------|------|
| Epic 总数 | 6 |
| Task 总数 | 29 |
| 测试用例 | 139 |
| 测试通过率 | 100% |
| Commit 数 | 25+ |

## 可复用模式

1. **Monorepo 测试**: 在子目录运行 `npm test`，根目录做健康检查
2. **SSE 集成**: 环境变量化 URL + 内存泄漏防护
3. **Canvas 节点**: dagre 布局 + id 字段必填
4. **Artifact 持久化**: IndexedDB + 预览组件 + drag-drop

---
_经验沉淀自 coord 收口阶段_
