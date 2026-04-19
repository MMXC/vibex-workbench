<script lang="ts">
  interface Props {
    zoom?: number;
    onzoomchange?: (z: number) => void;
    ontoolselect?: (tool: string) => void;
  }

  let { zoom = $bindable(100), onzoomchange, ontoolselect }: Props = $props();

  let activeTool = $state('select');
  const tools = [
    { id: 'select', icon: '⬚', label: '选择' },
    { id: 'pan', icon: '✋', label: '平移' },
    { id: 'add', icon: '➕', label: '添加节点' },
    { id: 'connect', icon: '🔗', label: '连线' },
    { id: 'delete', icon: '🗑️', label: '删除' },
  ];

  function selectTool(id: string) {
    activeTool = id;
    ontoolselect?.(id);
  }
  function zoomIn() { zoom = Math.min(200, zoom + 10); onzoomchange?.(zoom); }
  function zoomOut() { zoom = Math.max(10, zoom - 10); onzoomchange?.(zoom); }
  function zoomReset() { zoom = 100; onzoomchange?.(zoom); }
</script>

<div class="CanvasToolbar">
  <div class="tool-group">
    {#each tools as tool}
      <button class="tool-btn" class:active={activeTool === tool.id}
        onclick={() => selectTool(tool.id)} title={tool.label}>{tool.icon}</button>
    {/each}
  </div>
  <div class="divider"></div>
  <div class="zoom-group">
    <button onclick={zoomOut} title="缩小">➖</button>
    <button onclick={zoomReset} class="zoom-label">{zoom}%</button>
    <button onclick={zoomIn} title="放大">➕</button>
  </div>
</div>

<style>
  .CanvasToolbar { display: flex; align-items: center; gap: 8px; padding: 6px 12px; background: #1a1a2e; border-bottom: 1px solid #333; }
  .tool-group { display: flex; gap: 2px; }
  .tool-btn { background: transparent; border: 1px solid transparent; color: #888; padding: 4px 8px; border-radius: 4px; cursor: pointer; font-size: 14px; }
  .tool-btn:hover { background: #252540; color: #e0e0e0; }
  .tool-btn.active { background: #4D96FF; color: #fff; border-color: #4D96FF; }
  .divider { width: 1px; height: 20px; background: #333; }
  .zoom-group { display: flex; align-items: center; gap: 4px; }
  .zoom-group button { background: transparent; border: 1px solid #333; color: #888; padding: 2px 6px; border-radius: 4px; cursor: pointer; font-size: 12px; }
  .zoom-group button:hover { background: #252540; color: #e0e0e0; }
  .zoom-label { min-width: 48px; text-align: center; font-size: 11px; cursor: pointer; }
</style>
