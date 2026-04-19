# Spec — E5: Canvas Orchestration

> PRD: `prd.md` E5 章节
> 规范层级: L4 Feature Spec

## Canvas 渲染层集成

### 技术选型
- 方案A（推荐）：`@xyflow/svelte`（SvelteKit 原生适配）
- 方案B：`svelte-flow`
- 方案C：简化降级，DOM 渲染

> **决策**: 方案A。成本最低，与项目架构一致。

### 依赖安装
```bash
npm install @xyflow/svelte
npm install dagre @types/dagre  # 自动布局
```

---

## CanvasRenderer 组件

```svelte
<!-- CanvasRenderer.svelte -->
<script>
  import { Canvas } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';
  import { canvasStore } from '$lib/stores/canvas-store';
  
  let nodes = $derived($canvasStore.nodes);
  let edges = $derived($canvasStore.edges);
</script>

<Canvas {nodes} {edges} fitView onnodeclick={(e) => {...}}>
  <!-- 自定义节点 -->
</Canvas>
```

---

## 节点类型定义

### RunNode
```typescript
interface RunNodeData {
  label: string;      // "Run: xxx"
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  goal?: string;
  summary?: string;
  error?: string;
}
```

### ToolNode
```typescript
interface ToolNodeData {
  label: string;       // toolName
  status: 'running' | 'completed' | 'failed';
  args?: Record<string, unknown>;
  result?: Record<string, unknown>;
  error?: string;
  duration_ms?: number;
}
```

### ArtifactNode
```typescript
interface ArtifactNodeData {
  label: string;       // artifact name
  type: 'code' | 'image' | 'text';
  status: 'pending' | 'created';
}
```

---

## 节点四态规范

### RunNode

| 状态 | 边框颜色 | 背景色 | 图标 |
|------|----------|--------|------|
| pending | `#666` | `#1a1a1a` | ○ |
| running | `#4f46e5` | `#1a1a2e` | ◐ (pulse) |
| completed | `#22c55e` | `#1a2e1a` | ✓ |
| failed | `#ef4444` | `#2e1a1a` | ✗ |
| cancelled | `#f59e0b` | `#2e2a1a` | — |

### ToolNode

| 状态 | 边框颜色 | 背景色 |
|------|----------|--------|
| running | `#4f46e5` | `#1a1a2e` |
| completed | `#22c55e` | `#1a2e1a` |
| failed | `#ef4444` | `#2e1a1a` |

---

## 自动布局规范

### 布局算法
- 初始布局：dagre（从左到右 DAG）
- 用户拖拽：覆盖自动布局结果（不还原）

### dagre 配置
```typescript
const layoutConfig = {
  rankdir: 'TB',        // top to bottom
  nodesep: 60,          // 节点间距
  ranksep: 100,         // 层间距
  marginx: 20,
  marginy: 20,
};
```

### 硬编码禁止
```
禁止: { x: 300, y: 100 }  ← Math.random() 属于降级，Phase 1 可接受
必须: { x: layout(node).x, y: layout(node).y }
```

---

## Canvas 空状态

```
┌──────────────────────────────────────────────────┐
│                                                  │
│               [空画布 SVG 插图]                   │
│                                                  │
│          还没有 Run 记录                          │
│      发送消息，Canvas 将展示执行过程              │
│                                                  │
└──────────────────────────────────────────────────┘
```
- 插图：简单 SVG 节点+连线图（示意）
- 引导文案：「还没有 Run 记录」「发送消息，Canvas 将展示执行过程」
- 禁止只写"无内容"或留白

---

## 节点交互规范

### 拖拽
- 鼠标拖拽节点 → `canvasStore.updateNode(id, { position: { x, y } })`
- 拖拽时 `node.dragging = true`

### 双击展开
- 双击 ToolNode → 展开显示 `args` 和 `result`
- 使用 side panel 或 tooltip
- 内容：`JSON.stringify(args, null, 2)` + `JSON.stringify(result, null, 2)`

### 连线高亮
- Run 节点 → Tool 节点连线：`animated: true`（流动箭头）
- 完成态：`animated: false`，颜色 `#22c55e`

---

## 间距/颜色 Token

所有 CSS 必须使用 Token：
```css
--color-bg: #111;
--color-surface: #1a1a1a;
--color-border: #222;
--color-primary: #4f46e5;
--color-success: #22c55e;
--color-error: #ef4444;
--color-warning: #f59e0b;
--space-1: 8px;  /* 基准间距 */
--space-2: 16px;
--space-3: 24px;
--space-4: 32px;
```

---

## 验收标准

```typescript
// F5.1 — 渲染层
expect(canvasRenderer).toBeDefined();
expect(document.querySelector('.svelte-flow')).not.toBeNull();

// F5.2 — 自动布局
// run.started 后检查节点位置
const runNode = nodes.find(n => n.type === 'run');
expect(runNode.position.x).not.toBeNaN();
expect(runNode.position.y).not.toBeNaN();
// 无 Math.random() 硬编码
expect(source).not.toMatch(/Math\.random\(\).*position/);

// F5.3 — 节点交互
await drag(node('run-1')).to({ x: 500, y: 200 });
expect(canvasStore.nodes[0].position.x).toBe(500);

await dblclick(node('tool-1'));
expect(detailPanel).toBeVisible();
expect(detailContent).toContain('args');
```

### DoD
- [ ] `@xyflow/svelte` 安装并可 import
- [ ] `CanvasRenderer.svelte` 渲染 `$canvasStore.nodes`
- [ ] run.started 事件触发 → Canvas 出现 Run 节点
- [ ] tool.called 事件触发 → Canvas 出现 Tool 节点（自动连线到父 Run）
- [ ] 节点可拖拽，位置更新到 canvasStore
- [ ] 双击 ToolNode 展开显示 args/result
- [ ] dagre 自动布局已集成（Phase 1 可选，但必须移除 Math.random 硬编码）
- [ ] Canvas 空状态有引导文案
- [ ] 所有间距使用 8 倍数，颜色使用 CSS Token
