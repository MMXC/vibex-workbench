# Feature List — vibex-workbench Phase 1

> Planning 输出物，基于 `analysis.md` 技术方案拆解
> 日期: 2026-04-20
> Agent: pm

## Feature List 表格

| ID | 功能名 | 描述 | 根因关联 | 工时 | Epic |
|----|--------|------|----------|------|------|
| F1.1 | SSE URL 环境变量化 | `VITE_SSE_URL` 环境变量替代硬编码 `http://localhost:33335` | R3 | 0.5d | E1 |
| F1.2 | SSE 重连逻辑增强 | 指数退避重连（3s → 6s → 12s），最多 5 次 | R2 | 0.5d | E1 |
| F2.1 | Thread CRUD 持久化 | Thread 新建/删除通过 Dexie.js 持久化到 IndexedDB，刷新恢复 | — | 1d | E2 |
| F2.2 | Thread 列表四态 | 加载骨架屏、空状态引导、新建按钮、错误态重试 | — | 0.5d | E2 |
| F3.1 | Run 状态追踪 | runStore 增加 `toolInvocations` 数组，Composer 显示 Run 进度 | — | 0.5d | E3 |
| F3.2 | Run 结果展示 | completed/failed 状态在 Canvas 和 Composer 底部显示摘要 | — | 0.5d | E3 |
| F4.1 | Artifact 持久化 | Artifact 通过 Dexie.js 持久化，刷新恢复 | R4 | 1d | E4 |
| F4.2 | Artifact 预览 | 点击 Artifact 弹出预览 modal（图片/代码高亮） | — | 0.5d | E4 |
| F4.3 | Artifact 拖入 Composer | 拖拽 Artifact 到 Composer，注入 `@{artifactId}` 引用 | — | 0.5d | E4 |
| F5.1 | Canvas 渲染层集成 | 安装 `@xyflow/svelte`，`CanvasRenderer.svelte` 包装 ReactFlow/SvelteFlow | R1 | 2d | E5 |
| F5.2 | Canvas 自动布局 | 使用 dagre 对初始节点进行自动布局 | R5 | 1d | E5 |
| F5.3 | Canvas 节点交互 | 节点可拖拽、展开详情、连线高亮 | — | 1d | E5 |
| F6.1 | 右栏宽度激活 | `grid-template-columns: 280px 1fr 320px` 替代 `0px` | — | 0.5d | E6 |
| F6.2 | 三栏响应式 | 1440px/1024px/768px 三档断点，Composer 始终可见 | — | 1d | E6 |

## Epic/Story 映射

### Epic 1: SSE Backend Integration（E1）
- Story 1.1: 环境变量化 SSE URL → F1.1
- Story 1.2: 指数退避重连 → F1.2

### Epic 2: Thread Management（E2）
- Story 2.1: Thread IndexedDB 持久化 → F2.1
- Story 2.2: Thread 列表四态 → F2.2

### Epic 3: Run Engine（E3）
- Story 3.1: Run 状态追踪 → F3.1
- Story 3.2: Run 结果展示 → F3.2

### Epic 4: Artifact Registry（E4）
- Story 4.1: Artifact 持久化 → F4.1
- Story 4.2: Artifact 预览 → F4.2
- Story 4.3: Artifact 引用注入 → F4.3

### Epic 5: Canvas Orchestration（E5）
- Story 5.1: Canvas 渲染层 → F5.1
- Story 5.2: 自动布局 → F5.2
- Story 5.3: 节点交互 → F5.3

### Epic 6: Workbench Shell（E6）
- Story 6.1: 右栏宽度激活 → F6.1
- Story 6.2: 三栏响应式 → F6.2

## 工期汇总

| Epic | 名称 | 工时合计 |
|------|------|----------|
| E1 | SSE Backend Integration | 1d |
| E2 | Thread Management | 1.5d |
| E3 | Run Engine | 1d |
| E4 | Artifact Registry | 2d |
| E5 | Canvas Orchestration | 4d |
| E6 | Workbench Shell | 1.5d |
| **合计** | | **11d** |

> 注: Phase 1 核心交付（E1+E2+E3+E4+E6）约 6.5d，E5（Canvas）高风险，可独立交付。
