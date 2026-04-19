<script lang="ts">
  interface Props {
    id: string; label: string; level?: string; type?: 'rect' | 'circle' | 'diamond';
    selected?: boolean; onclick?: (id: string) => void;
  }
  let { id, label, level = 'L4', type = 'rect', selected = false, onclick }: Props = $props();
  const levelColors: Record<string, string> = { 'L1': '#FF6B6B', 'L2': '#FFA94D', 'L3': '#FFD93D', 'L4': '#6BCB77', 'L5': '#4D96FF' };
  let color = $derived(levelColors[level] || '#888');
</script>

<div class="MermaidNode {type}" class:selected style="--node-color: {color}"
  onclick={() => onclick?.(id)} role="button" tabindex="0"
  onkeydown={(e) => e.key === 'Enter' && onclick?.(id)}>
  <span class="node-level">{level}</span>
  <span class="node-label">{label}</span>
</div>

<style>
  .MermaidNode { display: inline-flex; flex-direction: column; align-items: center; gap: 2px; padding: 6px 12px; border-radius: 6px; border: 2px solid var(--node-color); background: #1a1a2e; cursor: pointer; transition: all 0.15s; user-select: none; }
  .circle { border-radius: 50px; }
  .diamond { border-radius: 4px; }
  .MermaidNode:hover { background: #252540; transform: scale(1.02); }
  .MermaidNode.selected { background: color-mix(in srgb, var(--node-color) 20%, #1a1a2e); box-shadow: 0 0 8px var(--node-color); }
  .node-level { font-size: 9px; color: var(--node-color); font-weight: 600; }
  .node-label { font-size: 12px; color: #e0e0e0; white-space: nowrap; }
</style>
