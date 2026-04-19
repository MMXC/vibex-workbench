# AGENTS.md — VibeX Workbench Phase 1

**项目**: vibex-workbench-integration
**日期**: 2026-04-20
**Agent**: architect
**审查状态**: ✅ /plan-eng-review 通过，4 项 Critical Findings 已记录

---

## 开发约束

### 1. 文件覆盖规则

> ⚠️ `WorkbenchShell.svelte` 和 `sse.ts` 是 spec-to-code 生成文件，gen.py 会覆盖。**禁止直接编辑这些文件**，只允许通过以下方式修改：
>
> - 在 `frontend/src/routes/workbench/+page.svelte`（标注了 `// 开发者维护，gen.py 永不覆盖`）中添加 wrapper 逻辑
> - 若 gen.py 确实需要更新这些文件，先同步修改 spec 文件，再重新生成

**可直接安全编辑的文件**（gen.py 永不覆盖）：
- `frontend/src/routes/workbench/+page.svelte` ✅
- `frontend/src/lib/db.ts` ✅（新建）
- `frontend/src/lib/canvas-layout.ts` ✅（新建）
- `frontend/src/lib/components/workbench/ThreadList.svelte` ✅
- `frontend/src/lib/components/workbench/Composer.svelte` ✅
- `frontend/src/lib/components/workbench/ArtifactPanel.svelte` ✅
- `frontend/src/lib/components/workbench/CanvasRenderer.svelte` ✅（新建）
- `frontend/src/lib/components/workbench/ArtifactPreviewModal.svelte` ✅（新建）
- `frontend/src/lib/stores/thread-store.ts` ✅
- `frontend/src/lib/stores/artifact-store.ts` ✅
- `frontend/src/lib/stores/run-store.ts` ✅
- `frontend/src/lib/stores/canvas-store.ts` ✅

### 2. 依赖引入规则

**必须引入的新依赖**（Phase 1 核心依赖）：
- `@xyflow/svelte` — Canvas 渲染
- `dexie` — IndexedDB ORM
- `dagre` + `@types/dagre` — 自动布局
- `highlight.js` — 代码高亮预览
- `vitest` — 单元测试
- `@testing-library/svelte` — Svelte 组件测试
- `@playwright/test` — E2E 测试

**禁止引入**：
- 任何状态管理库（Svelte store 够用）
- 任何 CSS 框架（保持轻量，用 CSS 变量）
- 任何 Router 库（SvelteKit 自带）
- `axios` / `fetch` 以外的 HTTP 客户端

### 3. Store 修改规则

- **只允许扩展，不允许破坏现有 API**：threadStore/runStore/canvasStore/artifactStore 已有公开方法不能删除
- **新增方法必须向后兼容**：新加的 `loadFromDB` / `persistToDB` 不得影响现有 `addThread` / `create` 等方法签名
- **Svelte 5 runes**：新增 store 逻辑使用 `$state()` / `$derived()` / `$effect()`，不混用旧版 `writable`

### 4. SSE 事件处理规则

- SSE handler 只能写入对应 Store，不得在 handler 中直接操作 DOM
- 所有 SSE URL 必须通过 `import.meta.env.VITE_SSE_URL` 读取，**禁止硬编码**
- `SSEConsumer` 必须在组件 `onMount` 时 connect，`onDestroy` 时 disconnect
- `tool.called` handler 必须同时创建 tool node 和 edge（run→tool）

### 5. Canvas 规则

- **禁止硬编码节点坐标**：`x: Math.random()` 必须替换为 dagre 自动布局
- **Canvas 节点类型**：`run` / `tool` / `artifact` 三种，禁止新增类型除非更新本文件
- **拖拽保存**：用户拖拽后的位置必须保存到 canvasStore，不自动重排
- **@xyflow/svelte 兼容性**：先安装包验证 `vite dev` 成功，再集成

### 6. 测试规则

- 每个 Epic 至少有一个 Playwright E2E 测试验证核心用户流程
- Store 层使用 Vitest 单元测试
- **禁止 `setTimeout` hack 等待异步操作**：使用 Playwright `waitForSelector`
- Vitest 配置文件：`frontend/vitest.config.ts` 需新建

### 7. 响应式布局规则

- **三栏宽度**：`280px | 1fr | 320px`（1440px+），不得在代码中硬编码 `0px`
- **Composer footer**：所有断点下始终固定在底部
- **降级策略**：`<768px` 侧栏变 drawer，主区域优先

### 8. 环境变量规则

```bash
# frontend/.env（git ignore）
VITE_SSE_URL=http://localhost:33335

# frontend/.env.example（git track，分发模板）
VITE_SSE_URL=http://localhost:33335
```

- 所有环境变量必须以 `VITE_` 开头（Vite 构建时注入）
- `.env` 文件必须加入 `.gitignore`

### 9. 代码风格

- TypeScript strict mode
- Svelte 5 runes 语法（`$state` / `$derived` / `$effect`）
- 组件文件名：`PascalCase.svelte`
- Store 文件名：`kebab-case.ts`
- CSS 变量前缀：`--wb-`（workbench）

### 10. git commit 约定

```
feat(CF): 安装 Phase 1 核心依赖（dexie, @xyflow/svelte, dagre）
fix(CF): 修复 WorkbenchShell.svelte 右栏 0px → 320px
feat(CF): 添加 SSE disconnect() 和 onDestroy 生命周期绑定
feat(CF): 创建 .env 环境变量文件
feat(E1): 环境变量化 SSE URL + 指数退避重连
feat(E2): Thread IndexedDB 持久化
feat(E5): Canvas 渲染层集成（@xyflow/svelte）
fix(E6): 修复右栏宽度 0px 问题
test(E1): Vitest SSE 重连逻辑测试
e2e(E6): Playwright 三栏布局响应式测试
```

---

## Critical Findings（Day 1 必完成）

| ID | 描述 | 文件 | 修复方法 |
|----|------|------|----------|
| CF-1 | package.json 缺少 7 个依赖 | `package.json` | `npm install ...` |
| CF-2 | 右栏 0px | `WorkbenchShell.svelte:25` | `0px` → `320px` |
| CF-3 | SSE 无 disconnect() 调用 | `+page.svelte` | 添加 `onDestroy` |
| CF-4 | 无 .env 文件 | `frontend/.env` | 创建文件 |

---

## Gen 文件待修复清单

| 文件 | 问题 | 修复方式 |
|------|------|----------|
| `WorkbenchShell.svelte` | `grid-template-columns: ... 0px` | E6-U1 覆盖此样式 |
| `sse.ts` | 无指数退避重连 | E1-U2 重写 `onerror` |
| `sse.ts` | SSE URL 硬编码 | E1-U1 替换为环境变量 |
| `sse.ts` | `tool.called` 不创建 edge | E5-U4 补充 edge 创建 |
| `+page.svelte` | 无 `onDestroy` | E1-U1 添加 |

> 注意：以上修复通过 E1-U1/E1-U2/E6-U1 直接修改对应文件实现，不需要通过 gen.py
