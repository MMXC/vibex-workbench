<!-- ============================================================
⚠️  此文件由 spec-to-code 自动生成
来自: specs
生成时间: 2026-04-23
⚠️  不要直接编辑此文件 — 改 *.svelte
============================================================ -->

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
  // CanvasRenderer 骨架 — Generated from canvas-renderer_uiux.yaml
  // Reactive state driven by canvasStore
  let storeNodes = $state<Node[]>([]);
  let storeEdges = $state<Edge[]>([]);
  let selectedNodeId = $state<string | null>(null);
  let detailNode = $state<Node | null>(null);

  $effect(() => {
    const unsub = canvasStore.subscribe(s => {
      storeNodes = (s.nodes as unknown as Node[]);
      storeEdges = (s.edges as unknown as Edge[]);
    });
    return unsub;
  });

  // onnodeclick → selectedNodeId
  function handleNodeClick(_: MouseEvent, node: Node) {
    selectedNodeId = node.id!;
  }

  // onnodedoubleclick → detailNode
  function handleNodeDoubleClick(_: MouseEvent, node: Node) {
    detailNode = node;
    selectedNodeId = node.id!;
  }

  // onnodedragstop → persist position
  function handleNodeDragStop(_: MouseEvent, node: Node) {
    canvasStore.updateNode(node.id!, { position: node.position } as any);
  }

  function closeDetail() {
    detailNode = null;
    selectedNodeId = null;
  }
</script>

<div class="canvas-renderer">
  <SvelteFlow
    nodes={{storeNodes}}
    edges={{storeEdges}}
    fitView
    onnodeclick={handleNodeClick}
    onnodedoubleclick={handleNodeDoubleClick}
    onnodedragstop={handleNodeDragStop}
  >
    <Controls />
    <Background />
  </SvelteFlow>

  <!-- 工具栏浮层 -->
  <div class="canvas-toolbar">
    <button title="Zoom In" class="tool-btn" onclick={() => { console.log("[Canvas] ZoomIn") }}>+</button>
    <button title="Zoom Out" class="tool-btn" onclick={() => { console.log("[Canvas] ZoomOut") }}>−</button>
    <button title="Fit View" class="tool-btn" onclick={() => { console.log("[Canvas] FitView") }}>⊡</button>
    <button title="Toggle" class="tool-btn" onclick={() => { console.log("[Canvas] ToggleInteractivity") }}>⊙</button>
  </div>

  <!-- 节点详情浮层 -->
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
          {#if detailNode.data?.status}
            <div class="detail-section">
              <span class="detail-key">status:</span>
              <span class="detail-status"
                class:running={detailNode.data?.status === 'running'}
                class:completed={detailNode.data?.status === 'completed'}
                class:failed={detailNode.data?.status === 'failed'}>
                {detailNode.data?.status}
              </span>
            </div>
          {/if}
          {#if detailNode.data?.args}
            <div class="detail-section">
              <span class="detail-key">args:</span>
              <pre class="detail-code">{JSON.stringify(detailNode.data?.args, null, 2)}</pre>
            </div>
          {/if}
          {#if detailNode.data?.result}
            <div class="detail-section">
              <span class="detail-key">result:</span>
              <pre class="detail-code">{JSON.stringify(detailNode.data?.result, null, 2)}</pre>
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

  /* 工具栏 */
  .canvas-toolbar {
    position: absolute;
    bottom: 16px;
    left: 16px;
    display: flex;
    gap: 4px;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 4px;
    z-index: 10;
  }
  .tool-btn {
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    width: 28px;
    height: 28px;
    border-radius: 4px;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .tool-btn:hover { background: #333; color: #fff; }

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
  .detail-type { color: #6366f1; font-size: 12px; }
  .detail-label { color: #e2e8f0; font-size: 13px; flex: 1; font-weight: 500; }
  .detail-header button {
    background: none;
    border: none;
    color: #666;
    font-size: 18px;
    cursor: pointer;
  }
  .detail-header button:hover { color: #fff; }
  .detail-body { padding: 14px; display: flex; flex-direction: column; gap: 10px; }
  .detail-section { display: flex; flex-direction: column; gap: 4px; }
  .detail-key { color: #888; font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; }
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
  .detail-status { font-size: 12px; padding: 2px 8px; border-radius: 10px; }
  .detail-status.running { background: #1e3a5f; color: #60a5fa; }
  .detail-status.completed { background: #1a3a2a; color: #4ade80; }
  .detail-status.failed { background: #3a1a1a; color: #f87171; }
</style>
