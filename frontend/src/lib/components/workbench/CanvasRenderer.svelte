// ============================================================
// CanvasRenderer — @xyflow/svelte 包装层
// E5-U1: Canvas 渲染层集成
// E5-U2: dagre 自动布局
// E5-U3: 节点交互（拖拽、展开）
// E5-U4: SSE → canvasStore → 渲染层同步
// ============================================================

<script lang="ts">
  import {
    SvelteFlow,
    Controls,
    Background,
    type Node,
    type Edge,
  } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';
  import { canvasStore } from '$lib/stores/canvas-store';
  import { layoutNodes } from '$lib/canvas-layout';

  // Reactive state driven by canvasStore
  let storeNodes = $state<Node[]>([]);
  let storeEdges = $state<Edge[]>([]);
  let selectedNodeId = $state<string | null>(null);
  let detailNode = $state<Node | null>(null);

  $effect(() => {
    const unsub = canvasStore.subscribe(s => {
      storeNodes = s.nodes as unknown as Node[];
      storeEdges = s.edges as unknown as Edge[];
    });
    return unsub;
  });

  // E5-U2: 自动布局触发 — 首次加载时对无坐标节点布局
  $effect(() => {
    if (storeNodes.length === 0) return;
    const needsLayout = storeNodes.filter(n => !n.position || (n.position.x === 0 && n.position.y === 0));
    if (needsLayout.length > 0) {
      // 提取 edges（从 canvasStore 或 SSE events 建立的 edge）
      const edges = storeEdges.map(e => ({ source: e.source, target: e.target }));
      const posMap = layoutNodes(
        storeNodes.map(n => ({ id: n.id!, type: n.type ?? 'default' })),
        edges
      );
      for (const node of storeNodes) {
        const pos = posMap[node.id!];
        if (pos) {
          canvasStore.updateNode(node.id!, { position: { x: pos.x, y: pos.y } } as any);
        }
      }
    }
  });

  // E5-U3: 双击节点 → 展开详情
  function handleNodeDoubleClick(_: MouseEvent, node: Node) {
    selectedNodeId = node.id!;
    detailNode = node;
  }

  function closeDetail() {
    detailNode = null;
    selectedNodeId = null;
  }

  // E5-U3: 节点拖拽后保存位置（不再自动重排）
  function handleNodeDragStop(_: MouseEvent, node: Node) {
    canvasStore.updateNode(node.id!, { position: node.position } as any);
  }
</script>

<div class="canvas-renderer">
  <SvelteFlow
    nodes={storeNodes}
    edges={storeEdges}
    fitView
    onnodeclick={(_, node) => { selectedNodeId = node.id!; }}
    onnodedoubleclick={handleNodeDoubleClick}
    onnodedragstop={handleNodeDragStop}
  >
    <Controls />
    <Background />
  </SvelteFlow>

  <!-- E5-U3: 节点详情面板 -->
  {#if detailNode}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="detail-overlay" onclick={closeDetail}>
      <div class="detail-panel" onclick={(e) => e.stopPropagation()}>
        <div class="detail-header">
          <span class="detail-type">[{detailNode.type ?? 'node'}]</span>
          <span class="detail-label">{detailNode.data?.label ?? detailNode.id}</span>
          <button onclick={closeDetail}>×</button>
        </div>
        <div class="detail-body">
          {#if detailNode.data?.args}
            <div class="detail-section">
              <span class="detail-key">args:</span>
              <pre class="detail-code">{JSON.stringify(detailNode.data.args, null, 2)}</pre>
            </div>
          {/if}
          {#if detailNode.data?.result}
            <div class="detail-section">
              <span class="detail-key">result:</span>
              <pre class="detail-code">{JSON.stringify(detailNode.data.result, null, 2)}</pre>
            </div>
          {/if}
          {#if detailNode.data?.error}
            <div class="detail-section">
              <span class="detail-key error">error:</span>
              <pre class="detail-code error">{detailNode.data.error}</pre>
            </div>
          {/if}
          {#if detailNode.data?.status}
            <div class="detail-section">
              <span class="detail-key">status:</span>
              <span class="detail-status" class:running={detailNode.data.status === 'running'} class:completed={detailNode.data.status === 'completed'} class:failed={detailNode.data.status === 'failed'}>{detailNode.data.status}</span>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .canvas-renderer { width: 100%; height: 100%; position: relative; }
  :global(.svelte-flow) { background: #0a0a0a; }
  :global(.svelte-flow .node) { border-radius: 6px; }

  /* 详情面板 */
  .detail-overlay {
    position: absolute;
    inset: 0;
    background: rgba(0,0,0,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }
  .detail-panel {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 10px;
    width: 480px;
    max-height: 70vh;
    overflow: auto;
  }
  .detail-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 14px;
    border-bottom: 1px solid #333;
  }
  .detail-type { color: #4f46e5; font-size: 12px; }
  .detail-label { color: #e2e8f0; font-size: 13px; flex: 1; font-weight: 500; }
  .detail-header button { background: none; border: none; color: #666; font-size: 18px; cursor: pointer; }
  .detail-header button:hover { color: #fff; }
  .detail-body { padding: 14px; display: flex; flex-direction: column; gap: 10px; }
  .detail-section { display: flex; flex-direction: column; gap: 4px; }
  .detail-key { color: #888; font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; }
  .detail-key.error { color: #f87171; }
  .detail-code {
    background: #111;
    border: 1px solid #222;
    border-radius: 6px;
    padding: 8px;
    font-size: 11px;
    color: #ccc;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 200px;
    overflow: auto;
    margin: 0;
  }
  .detail-code.error { border-color: #ef4444; color: #f87171; }
  .detail-status { font-size: 12px; padding: 2px 8px; border-radius: 10px; }
  .detail-status.running { background: #1e3a5f; color: #60a5fa; }
  .detail-status.completed { background: #1a3a2a; color: #4ade80; }
  .detail-status.failed { background: #3a1a1a; color: #f87171; }
</style>