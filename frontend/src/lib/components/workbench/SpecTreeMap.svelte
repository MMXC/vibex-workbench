<!-- SpecTreeMap.svelte — Tree map visualization of spec hierarchy
     Visualizes spec tree as a zoomable treemap
     SLICE-spec-tree-map
-->
<script lang="ts">
  import type { SpecTreeNode } from '$lib/stores/spec-explorer-store';

  interface Props {
    tree?: SpecTreeNode[];
    onSelectSpec?: (path: string) => void;
    workspaceRoot?: string;
  }

  let { tree = [], onSelectSpec, workspaceRoot = '' }: Props = $props();

  let selectedPath = $state<string | null>(null);

  function levelColor(level: string) {
    switch (level) {
      case 'L1': return '#7aa2ff';
      case 'L2': return '#22c55e';
      case 'L3': return '#f59e0b';
      case 'L4': return '#a78bfa';
      case 'L5': return '#f472b6';
      default: return '#71717a';
    }
  }

  function handleSelect(path: string) {
    selectedPath = path;
    onSelectSpec?.(path);
  }

  // Simple flat list layout
  const flatItems = $derived(
    tree.map((node, i) => ({
      ...node,
      color: levelColor(node.level ?? 'unknown'),
      size: node.children?.length ? Math.min(node.children.length * 30, 120) : 60,
    }))
  );
</script>

<div class="treemap">
  <div class="treemap-header">
    <span class="treemap-title">Spec Tree Map</span>
    <span class="treemap-count">{tree.length} specs</span>
  </div>

  {#if tree.length === 0}
    <div class="treemap-empty">
      <p>加载中或暂无 spec 数据</p>
    </div>
  {:else}
    <div class="treemap-grid">
      {#each flatItems as item}
        <button
          type="button"
          class="treemap-node"
          style="border-color: {item.color}55; background: {item.color}11"
          onclick={() => handleSelect(item.path)}
          title={item.path}
        >
          <span class="node-level" style="color: {item.color}">{item.level}</span>
          <span class="node-name">{item.name}</span>
        </button>
      {/each}
    </div>
  {/if}

  {#if selectedPath}
    <div class="treemap-selected">
      <span>选中：</span>
      <code>{selectedPath}</code>
    </div>
  {/if}
</div>

<style>
  .treemap {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #0d0d0e;
  }

  .treemap-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #27272a;
    flex-shrink: 0;
  }

  .treemap-title {
    font-size: 11px;
    font-weight: 600;
    color: #a1a1aa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .treemap-count {
    font-size: 10px;
    color: #71717a;
  }

  .treemap-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    color: #52525b;
  }

  .treemap-grid {
    flex: 1;
    overflow: auto;
    padding: 0.5rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    align-content: flex-start;
  }

  .treemap-node {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    padding: 0.4rem 0.6rem;
    border-radius: 6px;
    border: 1px solid;
    cursor: pointer;
    min-width: 80px;
    max-width: 140px;
    text-align: left;
    transition: background 0.15s;
  }

  .treemap-node:hover {
    filter: brightness(1.2);
  }

  .node-level {
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .node-name {
    font-size: 10px;
    color: #e4e4e7;
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 100%;
  }

  .treemap-selected {
    flex-shrink: 0;
    padding: 0.4rem 0.75rem;
    border-top: 1px solid #1f1f23;
    font-size: 10px;
    color: #71717a;
    display: flex;
    gap: 0.3rem;
    align-items: center;
  }

  .treemap-selected code {
    color: #9fc0ff;
    font-family: ui-monospace, monospace;
    font-size: 10px;
  }
</style>