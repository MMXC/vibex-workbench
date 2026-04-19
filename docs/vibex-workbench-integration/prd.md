# PRD — VibeX Workbench Phase 1

**项目**: vibex-workbench-integration
**阶段**: create-prd
**日期**: 2026-04-20
**Agent**: pm
**上游**: analysis.md (analyst, 2026-04-20)
**Planning**: feature-list.md

---

## 执行摘要

### 背景
VibeX Workbench 是 AI Coding Agent 的可视化工作台，前端（SvelteKit）+ 后端（Python SSE）已完成基础框架搭建，存在以下问题：
- SSE URL 硬编码 `http://localhost:33335`，跨环境无法工作
- Thread/Artifact 数据存内存，刷新页面即丢失
- Canvas 区域只有占位文字，Orchestration 可视化完全缺失
- 右栏宽度为 `0px`，ArtifactPanel 不可见

### 目标
Phase 1 完成前后端集成落地：
1. SSE 事件流打通 + 环境变量化
2. Thread/Artifact IndexedDB 持久化
3. Canvas Orchestration 渲染层集成
4. 三栏布局完整可用

### 成功指标
- [ ] `sse_server.py` + `npm run dev` 启动后，Workbench 页面 SSE 连接建立
- [ ] 发送消息触发 Run，Canvas 出现 Run 节点 + Tool 节点序列
- [ ] Thread 列表刷新后恢复，Artifact 刷新后恢复
- [ ] Canvas 节点可拖拽、展开
- [ ] 三栏布局在 1440p / 1024p / 768p 下均正常

---

## Epic 拆分

### E1: SSE Backend Integration

| Story | 描述 | 工时 | 验收标准 |
|-------|------|------|----------|
| 1.1 | SSE URL 环境变量化 | 0.5d | `import.meta.env.VITE_SSE_URL` 替代硬编码 |
| 1.2 | 指数退避重连 | 0.5d | SSE 断连后 3s → 6s → 12s 退避，最多 5 次 |

#### 2a. 本质需求穿透（剥洋葱）
- **用户的底层动机**：确保 Workbench 在任何环境下都能与 Backend 通信，不因网络波动丢失 Run 状态
- **去掉现有方案**：硬编码 URL + 无重试 → 不同机器/CI 环境全部失效
- **本质问题**：前后端通信路径的可配置性与韧性

#### 2b. 最小可行范围
- **本期必做**: F1.1（环境变量化）— 无此则跨环境不可用
- **本期不做**: 断连告警 UI（用户可感知重连行为）
- **暂缓**: SSE 健康检查心跳

#### 2c. 用户情绪地图
- **进入时**: 无感知（SSE 在后台连接）
- **迷路时**: 无明显标识，需靠 Network 面板排查
- **出错时**: Console 报 `[SSE] Connection error, retrying...`，用户无感知

#### 2d. UI状态规范
> 详见 `specs/e1-sse-integration.md`

---

### E2: Thread Management

| Story | 描述 | 工时 | 验收标准 |
|-------|------|------|----------|
| 2.1 | Thread IndexedDB 持久化 | 1d | 新建/删除 Thread 后刷新页面，列表恢复 |
| 2.2 | Thread 列表四态 | 0.5d | 加载骨架屏 / 空状态引导 / 正常态 / 错误态重试 |

#### 2a. 本质需求穿透
- **用户的底层动机**：保留对话上下文，随时回到之前的 Run 历史
- **去掉现有方案**：无持久化 → 刷新 = 对话历史全失，用户必须重新输入
- **本质问题**：对话上下文的长久存储与快速恢复

#### 2b. 最小可行范围
- **本期必做**: F2.1（持久化）— 无此则 Workbench 无可用性
- **本期不做**: Thread 搜索/过滤（已有 search_query store，但无 UI）
- **暂缓**: Thread 导出/分享，多端同步

#### 2c. 用户情绪地图
- **进入时**: 期待看到之前的对话列表（如果用过）
- **迷路时**: 空列表不知道该做什么 → 空状态必须引导
- **出错时**: "加载失败，请重试" + 重试按钮，禁止只写"加载错误"

#### 2d. UI状态规范
> 详见 `specs/e2-thread-management.md`

---

### E3: Run Engine

| Story | 描述 | 工时 | 验收标准 |
|-------|------|------|----------|
| 3.1 | Run 状态追踪 | 0.5d | `runStore.toolInvocations` 数组在 run.started → run.completed 期间持续更新 |
| 3.2 | Run 结果展示 | 0.5d | Canvas 节点状态更新 + Composer 底部显示完成摘要 |

#### 2a. 本质需求穿透
- **用户的底层动机**：看到 AI 执行了哪些工具，是否成功
- **去掉现有方案**：无 Run 状态 → 用户不知道 Run 在做什么、是否卡住
- **本质问题**：AI 执行过程的可观测性

#### 2b. 最小可行范围
- **本期必做**: F3.1 + F3.2 — 无 Run 状态则 Workbench 无实质价值
- **本期不做**: Run 取消、Run 步骤回放
- **暂缓**: Run 实时流式输出展开

#### 2c. 用户情绪地图
- **进入时**: 期待发送后立即看到 Run 启动
- **迷路时**: 长时间无响应不知道是卡住了还是运行中 → Composer 显示 "运行中..."
- **出错时**: Canvas 节点变红 + Composer 显示错误信息

#### 2d. UI状态规范
> 详见 `specs/e3-run-engine.md`

---

### E4: Artifact Registry

| Story | 描述 | 工时 | 验收标准 |
|-------|------|------|----------|
| 4.1 | Artifact 持久化 | 1d | 上传文件后刷新页面，Artifact 列表恢复 |
| 4.2 | Artifact 预览 | 0.5d | 点击 Artifact 弹出 modal，支持图片/代码高亮 |
| 4.3 | Artifact 引用注入 | 0.5d | 拖拽 Artifact 到 Composer，注入 `@{artifactId}` |

#### 2a. 本质需求穿透
- **用户的底层动机**：复用之前生成的代码/图片，无需重新生成
- **去掉现有方案**：无持久化 → 每次都要重新生成
- **本质问题**：AI 生成内容的积累与复用

#### 2b. 最小可行范围
- **本期必做**: F4.1 + F4.2 — 无持久化则 Artifact 无积累，无预览则无法确认内容
- **本期不做**: Artifact 版本历史、Artifact 分类标签
- **暂缓**: Artifact 跨 Thread 引用

#### 2c. 用户情绪地图
- **进入时**: 期待看到之前生成的文件列表
- **迷路时**: 不知道有哪些 Artifact → 右栏显示数量 + 搜索
- **出错时**: 上传失败显示具体原因（文件过大/类型不支持）

#### 2d. UI状态规范
> 详见 `specs/e4-artifact-registry.md`

---

### E5: Canvas Orchestration

| Story | 描述 | 工时 | 验收标准 |
|-------|------|------|----------|
| 5.1 | Canvas 渲染层集成 | 2d | `@xyflow/svelte` 安装，`CanvasRenderer.svelte` 可渲染 nodes/edges |
| 5.2 | 自动布局 | 1d | 初始节点使用 dagre 自动布局，用户拖拽覆盖 |
| 5.3 | 节点交互 | 1d | 节点可拖拽、展开显示 Tool args/result、连线高亮 |

#### 2a. 本质需求穿透
- **用户的底层动机**：直观看到 Run 的执行路径和工具调用链
- **去掉现有方案**：无 Canvas → Run 执行黑盒，用户无法理解执行过程
- **本质问题**：AI 执行过程的可视化表达

#### 2b. 最小可行范围
- **本期必做**: F5.1 — 无渲染层则 Canvas 价值为零
- **本期不做**: 节点编辑（拖拽后保存位置）、多选、节点复制
- **暂缓**: Canvas 截图导出、节点分组

#### 2c. 用户情绪地图
- **进入时**: 期待看到动态的节点图（如果已有 Run）
- **迷路时**: 不知道节点代表什么 → 节点默认显示 toolName + 状态
- **出错时**: 节点显示红色 + tooltip 显示 error message

#### 2d. UI状态规范
> 详见 `specs/e5-canvas-orchestration.md`

---

### E6: Workbench Shell

| Story | 描述 | 工时 | 验收标准 |
|-------|------|------|----------|
| 6.1 | 右栏宽度激活 | 0.5d | `grid-template-columns: 280px 1fr 320px` |
| 6.2 | 三栏响应式 | 1d | 1440p / 1024p / 768p 三档断点正常显示 |

#### 2a. 本质需求穿透
- **用户的底层动机**：需要同时看到线程列表、主工作区、产物面板，三者缺一不可
- **去掉现有方案**：右栏 `0px` → 产物面板不可见，用户无法操作 Artifact
- **本质问题**：多任务并行场景下的全功能可见性

#### 2b. 最小可行范围
- **本期必做**: F6.1 — 右栏 320px 是 Artifact 操作空间的最小值
- **本期不做**: 可拖拽调整栏宽、保存栏宽偏好
- **暂缓**: 深色/浅色主题切换

#### 2c. 用户情绪地图
- **进入时**: 期待看到完整三栏布局
- **迷路时**: 三栏各司其职，通过视觉分隔符区分
- **出错时**: 布局降级为单栏（主区域优先），侧栏折叠

#### 2d. UI状态规范
> 详见 `specs/e6-workbench-shell.md`

---

## 功能点汇总

| ID | 功能点 | 描述 | 验收标准 | 页面集成 |
|----|--------|------|----------|----------|
| F1.1 | SSE URL 环境变量化 | `VITE_SSE_URL` 替代硬编码 | expect(`import.meta.env.VITE_SSE_URL`).toBeDefined() | 无 |
| F1.2 | 指数退避重连 | 3s→6s→12s 退避 | SSE 断连 5 次内重连成功 | 无 |
| F2.1 | Thread 持久化 | Dexie.js IndexedDB | 刷新后 expect(threadCount).toBeGreaterThan(0) | 【需页面集成】 |
| F2.2 | Thread 列表四态 | 骨架屏/空态/正常/错误 | expect(screen.getByRole('button', {name: '重试'})).toBeVisible() | 【需页面集成】 |
| F3.1 | Run 状态追踪 | toolInvocations 数组 | run.completed 后 expect(toolInvocations.length).toBe(3) | 【需页面集成】 |
| F3.2 | Run 结果展示 | Canvas 节点状态 + 摘要 | expect(canvasNode.data.status).toBe('completed') | 【需页面集成】 |
| F4.1 | Artifact 持久化 | Dexie.js IndexedDB | 刷新后 expect(artifactCount).toBeGreaterThan(0) | 【需页面集成】 |
| F4.2 | Artifact 预览 | 弹出 modal | expect(modal).toBeVisible() on artifact click | 【需页面集成】 |
| F4.3 | Artifact 引用注入 | 拖拽注入 `@{id}` | expect(composerText).toContain('@{artifactId}') | 【需页面集成】 |
| F5.1 | Canvas 渲染层 | @xyflow/svelte | Canvas 渲染 nodes.length > 0 | 【需页面集成】 |
| F5.2 | 自动布局 | dagre | 节点无硬编码 x,y | 【需页面集成】 |
| F5.3 | 节点交互 | 拖拽/展开/高亮 | 节点可拖拽，展开显示 args | 【需页面集成】 |
| F6.1 | 右栏宽度激活 | 320px 实际宽度 | rightPanel.offsetWidth === 320 | 【需页面集成】 |
| F6.2 | 三栏响应式 | 1440/1024/768 | 三档断点下 expect(layout).not.toBeBroken() | 【需页面集成】 |

---

## 验收标准（expect() 条目）

### E1 — SSE
- `expect(import.meta.env.VITE_SSE_URL ?? 'http://localhost:33335').toBeTruthy()`
- SSE 断连后 3 次重连内成功（mock 环境下可验证重连次数）

### E2 — Thread
- `expect(ThreadList).toBeVisible()` 在页面加载后
- `expect(threadCount).toBeGreaterThan(0)` 在新建 Thread 后
- 刷新页面 `expect(threadCount).toBeGreaterThan(0)` (持久化)
- `expect(emptyStateMessage).toBeVisible()` 在 threads=[] 时
- 加载失败 `expect(retryButton).toBeVisible()` 在 error 时

### E3 — Run
- `expect(run.started event triggers) → runStore.activeRun !== null`
- `expect(toolInvocations.length).toBe(3)` 在 mock run 完成时
- `expect(run.completed event → canvasNode.status === 'completed')`

### E4 — Artifact
- `expect(artifactPanel.offsetWidth).toBeGreaterThan(0)` 右栏激活
- `expect(uploadFile → artifactCount++) → artifactStore.persist()`
- 刷新 `expect(artifactCount).toBeGreaterThan(0)`
- `expect(previewModal).toBeVisible()` 在点击 Artifact 后
- `expect(composer.value).toContain('@{artifactId}')` 在拖拽后

### E5 — Canvas
- `expect(canvasRenderer).toBeDefined()`
- `expect(nodes.length).toBeGreaterThan(0)` 在 run.started 后
- `expect(node.position).not.toEqual({x: hardcoded, y: hardcoded})`
- 拖拽 `expect(node.dragging).toBe(true)` → `expect(node.position).toChange()`
- 展开 `expect(detailPanel).toBeVisible()` 在节点双击后

### E6 — Shell
- `expect(rightPanel.offsetWidth).toBe(320)`
- `expect(1200px → layout is 3-column)`
- `expect(1000px → layout is still usable)`
- `expect(Composer).toBeVisible()` 在所有断点下

---

## DoD (Definition of Done)

### 研发完成判断标准

1. **功能完成**: 所有功能点代码已实现，单元测试通过
2. **验收标准通过**: 所有 expect() 条目在 gstack QA 中通过
3. **Spec 完成**: `specs/` 下每个 Epic 对应 spec 文件四态定义完整
4. **无硬编码问题**: specs/ 中无硬编码间距/颜色（非 Token）
5. **空状态有引导**: Thread 列表空状态 / Canvas 空状态均有引导文案
6. **响应式验证**: 三栏布局在 1440p / 1024p / 768p 均正常
7. **文档更新**: README.md 更新环境变量说明

### 代码合并标准
- 所有 Story 验收标准通过
- 无 TypeScript 编译错误
- 无 Console Error（Error level）
- Canvas 渲染层集成测试通过

---

## 依赖关系图

```
E1 (SSE)  ─────┬──→ E3 (Run Engine) ──→ E5 (Canvas) ──→ E6 (Shell)
                │      ↑                        │
                │      └────────────────────────┘
E2 (Thread) ────┴──→ E4 (Artifact) ──→ E6 (Shell)
     ↑                ↑                      ↑
     └────────────────┴──────────────────────┘
              (所有 Epic 依赖 Shell 布局)
```

### 关键路径
`E1 (SSE)` → `E3 (Run)` → `E5 (Canvas)` 是核心价值链，必须优先完成。

---

## 技术风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| R1: Canvas 渲染层 `@xyflow/svelte` 适配问题 | 高 | Phase 1 先完成其他 Epic，E5 独立交付 |
| R2: Backend mock 限制 | 中 | Phase 1 接受 mock，Phase 2 接入真实 backend |
| R3: SSE URL 跨环境 | 低 | 环境变量化已规划 |
| R4: IndexedDB 容量限制 | 低 | Dexie.js 提供容量检测 API |
| R5: 节点布局随机 | 低 | dagre 自动布局已规划 |
