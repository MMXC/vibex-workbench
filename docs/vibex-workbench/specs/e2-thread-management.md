# Spec — E2: Thread Management

> PRD: `prd.md` E2 章节
> 规范层级: L4 Feature Spec

## Thread 列表组件状态规范

组件: `ThreadList.svelte`

### 四态定义

#### 1. 加载态（Loading）
```
┌─────────────────────┐
│  线程 (--)          │
│ ┌─────────────────┐ │
│ │ ████████  ████  │ │  ← 骨架屏，3条假数据
│ │ ████████  ████  │ │
│ │ ████████         │ │
│ └─────────────────┘ │
└─────────────────────┘
```
- 骨架屏：3 条灰色占位条，宽度 60%/40% 交替
- 间距：`padding: 12px 16px`，骨架条 `height: 40px`，`border-radius: 4px`
- 背景色：`#1a1a1a` 骨架条：`#222`
- 加载态判断：`threadStore.loading === true`

#### 2. 空状态（Empty）
```
┌─────────────────────┐
│  线程 (0)           │
│                     │
│      [插图:对话气泡]   │
│    还没有对话记录      │
│  开始一个新对话吧～      │
│                     │
│  [+ 新建对话]        │  ← 居中按钮
└─────────────────────┘
```
- 插图：简单 SVG 对话气泡或留白图形
- 引导文案：「还没有对话记录」「开始一个新对话吧～」
- 新建按钮：`background: #4f46e5`, `color: white`, `border-radius: 8px`, `padding: 8px 20px`
- 字体：`font-size: 13px`，颜色 `#888`
- 空状态判断：`threadStore.threads.length === 0 && !threadStore.loading`

#### 3. 正常态（Populated）
```
┌─────────────────────┐
│  线程 (3)    [+新建] │
├─────────────────────┤
│ ● Thread A    draft │  ← active: 左侧 #4f46e5 边线
│ ● Thread B   active │
│ ● Thread C   done   │
└─────────────────────┘
```
- 间距：`gap: 0`，`border-bottom: 1px solid #1a1a1a`
- hover：`background: #1a1a1a`
- active：`background: #1e293b`, `border-left: 3px solid #4f46e5`
- 字体：`font-size: 13px`，标题 `#e2e8f0`，meta `#666`

#### 4. 错误态（Error）
```
┌─────────────────────┐
│  线程 (--)     [重试] │
│                     │
│      ⚠️              │
│   加载失败，请重试     │
│                     │
└─────────────────────┘
```
- 图标：`⚠️` 或 SVG 警告图标
- 引导文案：「加载失败，请重试」
- 重试按钮：`background: transparent`, `border: 1px solid #4f46e5`, `color: #4f46e5`
- 错误态判断：`threadStore.error !== null`
- 错误信息存储：`threadStore.error` 显示

---

## Thread 持久化规范

### IndexedDB Schema（via Dexie.js）

```typescript
// 表: threads
interface ThreadRecord {
  id: string;          // 主键
  goal: string;
  title: string;
  status: string;
  createdAt: string;
  updatedAt: string;
  deletedAt: string | null;  // 软删除
}

// 表: messages（可选，Phase 2）
interface MessageRecord {
  id: string;
  threadId: string;
  role: string;
  content: string;
  createdAt: string;
}
```

### 同步策略
- **写时同步**: 每次 `threadStore.addThread/updateThread/removeThread` 后同步到 IndexedDB
- **读时恢复**: `workbench/+page.svelte` onMount 时从 IndexedDB 恢复

### 间距/颜色 Token
- 所有间距使用 8 的倍数: `8px / 16px / 24px`
- 颜色使用 CSS 变量: `--color-bg: #111`, `--color-surface: #1a1a1a`, `--color-border: #222`, `--color-primary: #4f46e5`

---

## 验收标准

```typescript
// F2.1 — 持久化
expect(threadCount).toBeGreaterThan(0);           // 新建后计数 > 0
await refreshPage();
expect(threadCount).toBeGreaterThan(0);           // 刷新后恢复

// F2.2 — 四态
expect(ThreadListSkeleton).toBeVisible();         // 加载态
expect(emptyStateText).toContainText('还没有对话记录');  // 空态
expect(retryButton).toBeVisible();                // 错误态
expect(threadItems.length).toBeGreaterThan(0);    // 正常态
```

### DoD
- [ ] Dexie.js 初始化，`threads` 表已创建
- [ ] Thread 新建 → IndexedDB 写入
- [ ] 页面刷新 → IndexedDB 读取 → `threadStore.threads` 恢复
- [ ] Thread 删除 → 软删除（`deletedAt` 设置）
- [ ] Thread 列表四态定义完整（骨架屏/空态引导/正常/错误重试）
- [ ] 空状态有引导文案，禁止只留白
- [ ] specs/ 中无硬编码颜色/间距
