# VibeX Workbench Phase 1 — Implementation Plan

**项目**: vibex-workbench-integration
**阶段**: design-architecture
**日期**: 2026-04-20
**Agent**: architect
**来源**: architecture.md

---

## Unit Index

| Epic | Units | Status | Next |
|------|-------|--------|------|
| E1: SSE Backend Integration | U1-U2 | 0/2 | U1 |
| E2: Thread Management | U3-U5 | 0/3 | U3 |
| E3: Run Engine | U6-U7 | 0/2 | U6 |
| E4: Artifact Registry | U8-U10 | 0/3 | U8 |
| E5: Canvas Orchestration | U11-U14 | 0/4 | U11 |
| E6: Workbench Shell | U15-U17 | 0/3 | U15 |

**总进度**: 0/17

---

## E1: SSE Backend Integration

| ID | Name | Status | Depends On | Acceptance Criteria |
|----|------|--------|-----------|---------------------|
| E1-U1 | SSE URL 环境变量化 | ⬜ | — | `import.meta.env.VITE_SSE_URL` 替代所有硬编码 URL |
| E1-U2 | SSE 指数退避重连 | ⬜ | E1-U1 | 断连后 3s→6s→12s→24s→48s 退避，最多 5 次 |

### E1-U1 详细说明

**文件变更**：
- `frontend/src/lib/sse.ts` — `SSEConsumer` 构造函数取 `VITE_SSE_URL` + 新增 `disconnect()` 方法
- `frontend/src/routes/workbench/+page.svelte` — SSE URL 改为 `import.meta.env.VITE_SSE_URL`，`onDestroy` 时调用 `sseConsumer.disconnect()`
- `frontend/.env` — 新建，设置 `VITE_SSE_URL=http://localhost:33335`
- `frontend/.env.example` — 新建模板

**实现步骤**：
1. 创建 `frontend/.env` 和 `.env.example`
2. 修改 `SSEConsumer.connect()` 默认 URL
3. `SSEConsumer` 新增 `disconnect()` 方法：`this.es?.close(); this.es = null;`
4. `+page.svelte` 添加 `import { onDestroy } from 'svelte'`，`onDestroy(() => sseConsumer.disconnect())`
5. 搜索全项目硬编码 `localhost:33335`，全部替换为环境变量引用

**风险**：低

**Note**: SSEConsumer 是 singleton，`disconnect()` 方法是组件卸载时清理资源的必要出口，**禁止省略此步骤**。

---

### E1-U2 详细说明

**文件变更**：
- `frontend/src/lib/sse.ts` — 重写 `SSEConsumer.onerror` 重连逻辑

**实现步骤**：
1. 添加 `retryCount` 和 `maxRetries = 5`
2. `onerror` 时计算 delay = `3000 * 2^retryCount`
3. 超过 maxRetries 后停止重连，emit `sse.disconnected` 事件
4. `connect()` 时重置 retryCount

**风险**：低

---

## E2: Thread Management

| ID | Name | Status | Depends On | Acceptance Criteria |
|----|------|--------|-----------|---------------------|
| E2-U1 | Thread IndexedDB 持久化 | ⬜ | E1-U1 | 新建/删除 Thread 后刷新页面，列表恢复 |
| E2-U2 | Thread 列表四态 UI | ⬜ | E2-U1 | 骨架屏/空态/正常/错误重试均正常 |
| E2-U3 | Thread 切换 SSE 重连 | ⬜ | E1-U1 | 切换 Thread 时 SSE 重连到新 threadId |

### E2-U1 详细说明

**文件变更**：
- `frontend/src/lib/db.ts` — 新建，Dexie 数据库定义
- `frontend/src/lib/stores/thread-store.ts` — 增加 loadFromDB / persistToDB

**实现步骤**：
1. `npm install dexie` — 添加依赖
2. 创建 `db.ts` 定义 `threads` / `artifacts` 表
3. `threadStore` 增加 `loadFromDB()`: 页面加载时从 IndexedDB 恢复
4. `threadStore.addThread()` → 同时写入 IndexedDB
5. `threadStore.removeThread()` → 软删除（设置 `deletedAt`）

**Patterns to follow**: `stores/artifact-store.ts` 的现有 CRUD 结构

**风险**：低

---

### E2-U2 详细说明

**文件变更**：
- `frontend/src/lib/components/workbench/ThreadList.svelte` — 四态逻辑

**实现步骤**：
1. 导入 `Skeleton.svelte` 组件
2. `$threadStore.loading` → 显示骨架屏
3. `$threadStore.error` → 显示错误态 + 重试按钮
4. `$threadStore.threads.length === 0` → 显示空态引导文案 + "新建 Thread" 按钮
5. 正常态 → 显示 Thread 列表

**风险**：低

---

### E2-U3 详细说明

**文件变更**：
- `frontend/src/routes/workbench/+page.svelte` — Thread 切换逻辑

**实现步骤**：
1. `$effect` 中监听 `$currentThread` 变化
2. 变化时：disconnect 旧 SSE → connect 新 SSE
3. 加载新 Thread 的历史消息（从 IndexedDB + SSE backlog）

**风险**：中（需确保 SSE 不会串台）

---

## E3: Run Engine

| ID | Name | Status | Depends On | Acceptance Criteria |
|----|------|--------|-----------|---------------------|
| E3-U1 | Run 状态追踪 | ⬜ | E1-U2 | `runStore.toolInvocations` 数组在 run 期间持续更新 |
| E3-U2 | Run 结果展示 | ⬜ | E3-U1 | Canvas 节点状态更新 + Composer 底部显示完成摘要 |

### E3-U1 详细说明

**文件变更**：
- `frontend/src/lib/stores/run-store.ts` — 增加 toolInvocations 数组

**实现步骤**：
1. `runStore` 增加 `toolInvocations: ToolInvocation[]` 状态
2. SSE `tool.called` → `runStore.addToolInvocation()`
3. SSE `tool.completed` / `tool.failed` → `runStore.updateToolInvocation()`
4. `Composer.svelte` 底部监听 `runStore` 显示运行进度

**风险**：低

---

### E3-U2 详细说明

**文件变更**：
- `frontend/src/lib/components/workbench/Composer.svelte` — 底部摘要区

**实现步骤**：
1. `Composer.svelte` 底部添加 `<RunStatusBar>` 区域
2. `run.started` → 显示 "运行中..." + spinner
3. `run.completed` → 显示摘要文本 + 绿色勾
4. `run.failed` → 显示错误信息 + 红色叉
5. 全部完成后 5s 自动隐藏

**风险**：低

---

## E4: Artifact Registry

| ID | Name | Status | Depends On | Acceptance Criteria |
|----|------|--------|-----------|---------------------|
| E4-U1 | Artifact IndexedDB 持久化 | ⬜ | E2-U1 | 上传文件后刷新页面，Artifact 列表恢复 |
| E4-U2 | Artifact 预览 | ⬜ | E4-U1 | 点击 Artifact 弹出 modal，支持图片/代码高亮 |
| E4-U3 | Artifact 拖入 Composer | ⬜ | E4-U2 | 拖拽 Artifact 到 Composer，注入 `@{artifactId}` |

### E4-U1 详细说明

**文件变更**：
- `frontend/src/lib/db.ts` — artifacts 表已定义（E2-U1）
- `frontend/src/lib/stores/artifact-store.ts` — 增加 loadFromDB / persistToDB

**实现步骤**：
1. `artifactStore` 增加 `loadFromDB()`: 页面加载时恢复
2. `artifactStore.create()` → 同时写入 IndexedDB
3. SSE `artifact.created` 事件 → 写入 IndexedDB

**风险**：低

---

### E4-U2 详细说明

**文件变更**：
- `frontend/src/lib/components/workbench/ArtifactPanel.svelte` — 预览功能

**实现步骤**：
1. 点击 Artifact 项 → 设置 `selected_artifact_id`
2. 派生 `ArtifactPreviewModal.svelte` 组件
3. 图片类型 → `<img src={blobUrl} />`
4. 代码类型 → `<pre><code>{content}</code></pre>` + highlight.js
5. Modal 显示时生成 blob URL，关闭时 revoke

**风险**：中（blob URL 内存泄漏需注意）

---

### E4-U3 详细说明

**文件变更**：
- `frontend/src/lib/components/workbench/Composer.svelte` — 拖放区
- `frontend/src/lib/components/workbench/ArtifactPanel.svelte` — draggable 属性

**实现步骤**：
1. `ArtifactPanel` 每个 Artifact item 添加 `draggable="true"`
2. `Composer` 添加 `ondragover` / `ondrop` 事件
3. `drop` 时获取 artifact id，注入到 composer 文本：`@{artifactId}`
4. 注入位置 = 当前光标位置

**风险**：中（跨组件事件通信）

---

## E5: Canvas Orchestration

| ID | Name | Status | Depends On | Acceptance Criteria |
|----|------|--------|-----------|---------------------|
| E5-U1 | Canvas 渲染层集成 | ⬜ | E3-U1 | `@xyflow/svelte` 安装，Canvas 渲染 nodes/edges |
| E5-U2 | Canvas 自动布局 | ⬜ | E5-U1 | 初始节点使用 dagre 布局，用户拖拽后覆盖 |
| E5-U3 | Canvas 节点交互 | ⬜ | E5-U2 | 节点可拖拽、展开详情、连线高亮 |
| E5-U4 | Canvas ↔ SSE 同步 | ⬜ | E5-U1 | SSE 事件驱动 canvasStore → 渲染层同步 |

### E5-U1 详细说明

**文件变更**：
- `frontend/package.json` — 添加 `@xyflow/svelte` 依赖
- `frontend/src/lib/components/workbench/CanvasRenderer.svelte` — 新建

**实现步骤**：
1. `npm install @xyflow/svelte`
2. 创建 `CanvasRenderer.svelte`，使用 `<SvelteFlow>` 组件
3. 订阅 `canvasStore`，同步 nodes/edges 到 xyflow store
4. `CanvasRenderer` 替换 `+page.svelte` 中的 `<div class="canvas-area">`

**风险**：高（Svelte 5 runes 与 @xyflow/svelte 兼容性需验证）

**Verification**: 启动 dev server，发送消息，观察 Canvas 区域出现节点

---

### E5-U2 详细说明

**文件变更**：
- `frontend/src/lib/canvas-layout.ts` — 新建，dagre 布局封装
- `frontend/src/lib/stores/canvas-store.ts` — 布局触发逻辑

**实现步骤**：
1. `npm install dagre @types/dagre`
2. 创建 `canvas-layout.ts`，封装 dagre `graphlib`
3. `canvasStore.addNode()` 时，如果是首个节点，触发 dagre 布局
4. 后续节点按依赖顺序追加布局

**风险**：中（dagre 布局方向参数需调优）

---

### E5-U3 详细说明

**文件变更**：
- `frontend/src/lib/components/workbench/CanvasRenderer.svelte` — 交互增强

**实现步骤**：
1. 节点双击 → 展开详情面板（显示 args/result/error）
2. 边点击 → 高亮（CSS class 切换）
3. 节点拖拽后 → 保存手动位置到 canvasStore（`node.position` 覆盖）
4. Tool 节点显示 toolName + 状态图标

**风险**：低

---

### E5-U4 详细说明

**文件变更**：
- `frontend/src/lib/stores/canvas-store.ts` — 边创建逻辑
- `frontend/src/lib/sse.ts` — `tool.called` handler 增加 edge 创建

**实现步骤**：
1. SSE `tool.called` → 创建 tool node + 自动创建 edge (run→tool)
2. SSE `artifact.created` → 创建 artifact node + edge (last tool→artifact)
3. SSE `run.completed` → 更新 run node 状态
4. `canvasStore` 增加 `syncToRenderer()` 方法

**风险**：低

---

## E6: Workbench Shell

| ID | Name | Status | Depends On | Acceptance Criteria |
|----|------|--------|-----------|---------------------|
| E6-U1 | 右栏宽度激活 | ✅ | — | `grid-template-columns: 280px 1fr 320px` |
| E6-U2 | 三栏响应式断点 | ✅ | E6-U1 | 1440px/1024px/768px 三档正常显示 |
| E6-U3 | 布局降级 | ✅ | E6-U2 | 768px 以下侧栏折叠，Composer 始终可见 |

### E6-U1 详细说明

**文件变更**：
- `frontend/src/lib/components/workbench/WorkbenchShell.svelte` — grid 修复

**实现步骤**：
1. 将 `grid-template-columns: 280px 1fr 0px` 改为 `280px 1fr 320px`
2. 验证 `.sidebar-right` 实际渲染宽度

**风险**：低

---

### E6-U2 详细说明

**文件变更**：
- `frontend/src/lib/components/workbench/WorkbenchShell.svelte` — 媒体查询

**实现步骤**：
1. 1440px+：三栏 `280px 1fr 320px`
2. 1024px-1439px：三栏 `240px 1fr 280px`
3. 768px-1023px：左栏折叠为图标模式，右栏可隐藏
4. `<768px：单栏（主区域 + 底部 Composer，侧栏抽屉）

**风险**：中（响应式 CSS 需要在多档断点实际测试）

---

### E6-U3 详细说明

**文件变更**：
- `frontend/src/lib/components/workbench/WorkbenchShell.svelte`

**实现步骤**：
1. `<768px` 时左栏 / 右栏变为 drawer/drawer-trigger 模式
2. Composer footer 固定在底部，始终可见
3. 主区域可滚动

**风险**：低

---

## 依赖关系图（执行顺序）

```
E1-U1 (SSE URL)  ──┬──→ E1-U2 (重连)  ──┬──→ E2-U3 (Thread 切换)
                    │                    │
E2-U1 (Thread DB)  ─┴──→ E2-U2 (四态)  ──┴──→ E3-U1 (Run 追踪)
                                                            ↓
E4-U1 (Artifact DB) ───→ E4-U2 (预览)  ───→ E4-U3 (拖入 Composer)
                                                            ↓
                                            E5-U1 (Canvas 渲染层)  ← E3-U2 (结果展示)
                                                     ↓
                                            E5-U2 (自动布局)
                                                     ↓
                                            E5-U3 (节点交互)
                                                     ↓
                                            E5-U4 (SSE 同步)
                                                     ↓
E6-U1 (右栏宽度)  ───→ E6-U2 (响应式)  ───→ E6-U3 (降级)
```

**派发优先级**：
1. E1-U1 → E1-U2（基础设施，所有 Epic 依赖）
2. E2-U1 → E2-U2 → E2-U3（左栏功能）
3. E3-U1 → E3-U2（Run 核心）
4. E4-U1 → E4-U2 → E4-U3（右栏功能）
5. E5-U1 → E5-U2 → E5-U3 → E5-U4（Canvas，与 E3 并行）
6. E6-U1 → E6-U2 → E6-U3（最终布局，独立可并行了）

---

## 工期汇总

| Epic | Units | 估算工时 |
|------|-------|----------|
| E1: SSE Backend Integration | U1-U2 | 1d |
| E2: Thread Management | U3-U5 | 1.5d |
| E3: Run Engine | U6-U7 | 1d |
| E4: Artifact Registry | U8-U10 | 2d |
| E5: Canvas Orchestration | U11-U14 | 4d |
| E6: Workbench Shell | U15-U17 | 1.5d |
| **合计** | **17 units** | **11d** |
