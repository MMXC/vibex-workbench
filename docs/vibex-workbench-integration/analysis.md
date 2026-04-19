# VibeX Workbench — 需求分析报告

**项目**: vibex-workbench-integration
**任务**: analyze-requirements
**日期**: 2026-04-20
**分析**: analyst

---

## 执行决策

- **决策**: Conditional（技术可行性存在，附带关键风险项）
- **执行项目**: vibex-workbench-integration
- **执行日期**: 待定（需先解决 R1/R2/R3 风险）

---

## 一、技术可行性总评

**结论：有条件通过。**

| 模块 | 状态 | 说明 |
|------|------|------|
| workbench-shell | ✅ 可行 | 三栏布局已有，Composer 已有，缺 sidebar/panel 真实内容 |
| thread-manager | ✅ 可行 | Thread CRUD store 已就绪，缺 localStorage/Persistence 层 |
| run-engine | ✅ 可行 | SSE mock backend 已通，frontend store 已就绪 |
| artifact-registry | ⚠️ 风险 | artifactStore 缺少真实 IndexedDB 持久化实现 |
| canvas-workbench | ❌ 高风险 | Canvas 渲染层完全缺失（只有 placeholder） |

---

## 二、技术风险矩阵

### R1: Canvas 渲染层缺失（高）

**描述**: canvas-workbench 模块只有 store 定义和 SSE 事件映射，渲染层（ReactFlow / SvelteFlow）完全缺失。

**影响**: 核心卖点（画布即 Orchestration 层）无法落地，整个项目价值大幅缩水。

**缓解**: 
- 方案A：接入 `@xyflow/svelte`（SvelteKit 适配层）或 `svelte-flow`
- 方案B：iframe 嵌入现有 VibeX canvas 页面，通过 postMessage 通信
- 方案C：简化降级，先用 DOM 实现基础节点渲染

**推荐**: 方案A。成本最低，与项目架构一致。

**估计工时**: 3–5 PD

### R2: Backend 是 Mock，无法支撑真实 Run（高）

**描述**: `sse_server.py` 是纯模拟事件生成器，每次连接自动触发固定 3-tool mock run。真实 Run 需要：
- LLM 调用（OpenAI/Anthropic）
- 工具执行沙箱
- 真实状态管理

**影响**: `run-engine` 模块的完整验证依赖真实 backend。目前 S1/S2/S3 阶段任务均基于 mock，集成测试意义有限。

**缓解**: 
- Phase 1（当前）: 接受 mock，完成前端集成
- Phase 2: 接入真实 backend 或 stub server
- 验收标准：前端能完整走完 SSE 事件流，不要求后端真实执行

**估计工时**: Phase 1 无额外成本，Phase 2 另需 5–10 PD

### R3: SSE URL 硬编码 `http://localhost:33335`（中）

**描述**: `sse.ts` 和 `workbench/+page.svelte` 中 SSE URL 均硬编码。跨环境（CI/生产）无法工作。

**缓解**: 
- 开发阶段：可通过 env var `VITE_SSE_URL` 配置
- SvelteKit 支持 `import.meta.env.VITE_SSE_URL`
- S2 任务已计划修复此问题

**估计工时**: 0.5 PD

### R4: Artifact 持久化层缺失（中）

**描述**: `artifactStore` 定义了接口，但 IndexedDB 持久化层未实现。当前 artifact 存活在内存中，页面刷新即丢失。

**缓解**: 接入 Dexie.js（项目已规划 local-first），2 PD

### R5: Canvas 与 runStore 双向同步缺失（中）

**描述**: SSE 事件能驱动 canvasStore 创建节点，但节点位置/连线/布局全随机（`Math.random()` 硬编码）。没有自动布局算法。

**缓解**: 初期接受随机位置，后期引入 dagre 自动布局

**估计工时**: 1 PD

### R6: Thread 上下文注入协议未实现（低）

**描述**: `assembleContext()` 方法在 spec 中定义，但 store 中只有空方法。

**缓解**: 低优先级，可作为迭代优化

**估计工时**: 2 PD

### R7: Three-layer 工具可见性未实现（低）

**描述**: `visibilityLayer` 在 spec 中定义（1=自然语言/2=结构化/3=开发者日志），但 UI 无切换控件，渲染也无分层逻辑。

**缓解**: 迭代 2 实现

---

## 三、工期估算

| 阶段 | 任务 | 估算 | 依赖 |
|------|------|------|------|
| S1 | 清理旧进程 + 重启 SSE backend | 0.5 PD | 无 |
| S2 | 修改 SSE URL 从空字符串到 http://localhost:33335 | 0.5 PD | S1 |
| S3 | gstack QA 端到端验证 | 1 PD | S2 |
| Phase 2 | Canvas 渲染层（ReactFlow/SvelteFlow） | 3–5 PD | S3 |
| Phase 2 | Artifact IndexedDB 持久化 | 2 PD | S3 |
| Phase 2 | Backend stub（可测试的 mock） | 3–5 PD | S3 |

---

## 四、依赖分析

### 外部依赖

| 依赖项 | 当前状态 | 风险 |
|--------|----------|------|
| Python `sse_server.py` | 可独立运行 | 无 |
| SvelteKit `npm run dev` | 可启动 | 无 |
| `@xyflow/svelte` / `svelte-flow` | 未安装 | 中（需选型决策） |
| Dexie.js | 未安装 | 低 |
| VibeX 现有 canvas 节点渲染 | spec 引用，代码不存在 | 高 |

### 内部模块依赖

```
workbench-shell (M1)
    ↓
thread-manager (M2) ←→ localStorage
    ↓
run-engine (M3) ←→ sse_server.py (mock)
    ↓
canvas-workbench (M5) ← 高风险：渲染层缺失
    ↑
artifact-registry (M4)
```

### 阻塞项

- M5（canvas-workbench）的真实渲染依赖 M3（run-engine）SSE 流打通
- Artifact 拖入 Composer 依赖 artifactStore 持久化完成

---

## 五、每个需求实现方案

### 5.1 workbench-shell（三栏布局 + Composer）

**现状**: shell 布局 + Composer 组件已有，sidebar/panel 为空占位

**实现方案**:
1. `ThreadList.svelte` → 渲染 `threadStore.threads`，订阅 `currentThread`
2. `ArtifactPanel.svelte` → 渲染 `artifactStore`，支持拖拽排序
3. 三栏宽度可拖拽调整 → `uiStore` 添加 `leftSidebarWidth` / `rightSidebarWidth`

**验收标准**:
- [ ] Thread 切换后左侧 sidebar 刷新
- [ ] 三栏布局在 1440px / 1024px / 768px 下均正常
- [ ] Composer Ctrl+Enter 提交触发 handleSubmit

### 5.2 thread-manager（Thread CRUD）

**现状**: store 有 add/set/update/remove，缺 IndexedDB 持久化

**实现方案**:
1. 引入 `Dexie.js` 包装 IndexedDB
2. `ThreadManager` 类封装 CRUD → 自动同步 Dexie
3. 页面加载时从 IndexedDB 恢复 threads

**验收标准**:
- [ ] 新建 Thread → IndexedDB 持久化
- [ ] 刷新页面 → Thread 列表恢复
- [ ] 删除 Thread → 软删除（设置 `deletedAt`）

### 5.3 run-engine（SSE 事件流）

**现状**: mock backend 已通，SSE consumer 已就绪

**实现方案**:
1. SSE URL 环境变量化：`const SSE_URL = import.meta.env.VITE_SSE_URL || 'http://localhost:33335'`
2. `runStore` 增加 `toolInvocations` 数组追踪
3. `message.delta` 事件追加到 `runStore.messageStream`
4. 连接断开/重连逻辑（当前 `onerror` 已有 3s 重试）

**验收标准**:
- [ ] 发送消息 → `run.started` 事件触发
- [ ] Tool 调用序列 → `tool.called` → `tool.completed` 依次触发
- [ ] Run 完成 → `run.completed` + Canvas 节点状态更新
- [ ] SSE 断连 → 3s 后自动重连

### 5.4 artifact-registry（上传/预览/复用）

**现状**: artifactStore 有 create/update，缺 IndexedDB

**实现方案**:
1. `Dexie.js` 表：`artifacts` / `artifact_versions`
2. `createArtifact()` → 持久化 + 触发 `artifact.created` SSE 事件
3. `ArtifactPanel` 预览：图片用 blob URL，代码用 `<pre>` 高亮
4. 拖拽 artifact 到 Composer → `composerText` 注入 `@{artifactId}`

**验收标准**:
- [ ] 上传文件 → IndexedDB + 右侧 panel 显示
- [ ] 刷新页面 → Artifact 列表恢复
- [ ] 点击 Artifact → 弹出预览 modal

### 5.5 canvas-workbench（Orchestration 可视化）

**现状**: canvasStore + SSE 事件映射已有，渲染层完全缺失

**实现方案**:
1. 安装 `@xyflow/svelte` 或 `svelte-flow`
2. `CanvasRenderer.svelte` 包装 ReactFlow/SvelteFlow
3. `nodes` / `edges` 来自 `canvasStore`
4. Tool 节点展开显示 args/result（layer=3）
5. 自动布局：初始 `dagre`，用户拖拽覆盖

**验收标准**:
- [ ] `run.started` 事件 → Canvas 出现 Run 节点
- [ ] `tool.called` 事件 → Canvas 出现 Tool 节点（自动连线到父 Run）
- [ ] 节点可拖拽、可展开查看详情
- [ ] `artifact.created` 事件 → Artifact 节点出现

---

## 六、Spec 三层分离验证（SPEC-QA）

| spec 层 | 状态 | 说明 |
|---------|------|------|
| L1 project-goal | ✅ | vibex-workbench-goal.yaml 完整 |
| L2 architecture | ✅ | 5 模块 + 数据流 + store 架构清晰 |
| L3 feature | ⚠️ | 5 个 feature spec draft 状态，细节不足 |
| L4 feature yaml | ✅ | feature.yaml 覆盖 5 feature |
| L5 service contract | ✅ | service.yaml 接口契约完整 |
| L6 data schema | ✅ | *_data.yaml 类型完整 |

**问题**: feature spec 均为 draft，缺少验收标准（acceptance criteria）。

---

## 七、gstack QA 验证策略

**前提**: `sse_server.py` 已在 port 33335 运行

1. **SSE 连接验证**（/browse）
   - 启动 `frontend npm run dev`
   - 打开 `http://localhost:5173/workbench`
   - 观察 Network 面板 SSE 连接建立

2. **Run 流程端到端**（/qa）
   - Composer 输入 "test" → 发送
   - 预期：run.started → tool×3 → artifact.created → run.completed
   - 检查 Canvas 区域节点出现

3. **Canvas 节点渲染**（/qa-only）
   - 跳过（渲染层缺失，暂缓）

4. **Artifact 持久化**（/qa）
   - 上传文件 → 刷新页面 → 检查列表恢复

---

## 八、总结

**推荐：有条件推进。**

核心风险是 Canvas 渲染层缺失（R1）和 Backend mock（R2）。两者都是"可以先跑起来，后续补全"的类型，不阻塞当前 Phase。

**立即可执行**: S1 → S2 → S3（GStack QA 验证 SSE 事件流）可立即开始。

**必须决策**: Canvas 渲染层技术选型（@xyflow/svelte vs iframe embed）需 Architect 确认。

**后续任务队列**: create-prd → design-architecture → coord-decision
