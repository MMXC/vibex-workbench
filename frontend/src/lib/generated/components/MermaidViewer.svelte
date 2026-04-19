<script lang="ts">
  interface Props {
    content?: string;
    onnodeclick?: (nodeId: string) => void;
  }

  let { content = $bindable(''), onnodeclick }: Props = $props();

  // Parse simple Mermaid syntax into nodes and edges
  interface MermaidNode { id: string; label: string; type: string; level: string; }
  interface MermaidEdge { from: string; to: string; label?: string; }

  function parseMermaid(mermaid: string): { nodes: MermaidNode[]; edges: MermaidEdge[] } {
    const nodes: MermaidNode[] = [];
    const edges: MermaidEdge[] = [];
    const seen = new Set<string>();

    // Match: A[label] or A((label)) or A{label}
    const nodeRe = /([A-Z0-9_-]+)\[([^\]]+)\]|([A-Z0-9_-]+)\(\(([^)]+)\)\)|([A-Z0-9_-]+)\{([^}]+)\}/gi;
    const edgeRe = /([A-Z0-9_-]+)---([A-Z0-9_-]+)(?:\|([^{}]+))?/gi;
    const arrowRe = /([A-Z0-9_-]+)-->([A-Z0-9_-]+)(?:\|([^{}]+))?/gi;

    let match;
    while ((match = nodeRe.exec(mermaid)) !== null) {
      const id = (match[1] || match[3] || match[5]);
      const label = (match[2] || match[4] || match[6]);
      if (!seen.has(id)) {
        seen.add(id);
        const level = id.startsWith('L') ? id.split('-')[0] : '?';
        nodes.push({ id, label, type: match[2] ? 'circle' : match[4] ? 'circle' : 'rect', level });
      }
    }

    while ((match = edgeRe.exec(mermaid)) !== null) {
      edges.push({ from: match[1], to: match[2], label: match[3] });
    }
    while ((match = arrowRe.exec(mermaid)) !== null) {
      edges.push({ from: match[1], to: match[2], label: match[3] });
    }

    return { nodes, edges };
  }

  const levelColors: Record<string, string> = {
    'L1': '#FF6B6B',
    'L2': '#FFA94D',
    'L3': '#FFD93D',
    'L4': '#6BCB77',
    'L5': '#4D96FF',
    '?': '#888',
  };

  const nodeColors: Record<string, string> = {
    rect: '#2d4a7a',
    circle: '#3d2d5a',
    diamond: '#4a2d3d',
  };

  let parsed = $derived(parseMermaid(content));
</script>

<div class="mermaid-viewer">
  <div class="viewer-header">
    <span>🕸️ 从属关系图</span>
    <span class="legend">
      {#each Object.entries(levelColors) as [lvl, color]}
        <span class="legend-item">
          <span class="dot" style="background: {color}"></span>
          {lvl}
        </span>
      {/each}
    </span>
  </div>

  {#if content}
    <div class="canvas">
      <div class="node-list">
        {#each parsed.nodes as node}
          <div
            class="node"
            style="border-color: {levelColors[node.level] || '#888'}"
            onclick={() => onnodeclick?.(node.id)}
            role="button"
            tabindex="0"
            onkeydown={(e) => e.key === 'Enter' && onnodeclick?.(node.id)}
          >
            <span class="node-id" style="color: {levelColors[node.level] || '#888'}">{node.id}</span>
            <span class="node-label">{node.label}</span>
          </div>
        {/each}
      </div>

      {#if parsed.edges.length > 0}
        <div class="edge-list">
          <div class="edge-header">边关系 ({parsed.edges.length})</div>
          {#each parsed.edges as edge}
            <div class="edge" onclick={() => { onnodeclick?.(edge.from); onnodeclick?.(edge.to); }} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && onnodeclick?.(edge.from)}>
              <span class="edge-from">{edge.from}</span>
              <span class="edge-arrow">→</span>
              <span class="edge-to">{edge.to}</span>
              {#if edge.label}<span class="edge-label">|{edge.label}|</span>{/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <details class="raw-block">
      <summary>Mermaid 源码</summary>
      <pre>{content}</pre>
    </details>
  {:else}
    <div class="empty">
      <p>暂无从属关系图</p>
      <p class="hint">在 canvas_feature.yaml 中定义 spec 间引用关系后自动生成</p>
    </div>
  {/if}
</div>

<style>
  .mermaid-viewer { display: flex; flex-direction: column; height: 100%; background: #1a1a2e; color: #e0e0e0; font-size: 12px; }
  .viewer-header { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 1px solid #333; font-weight: 600; }
  .legend { display: flex; gap: 8px; }
  .legend-item { display: flex; align-items: center; gap: 3px; font-size: 10px; color: #888; }
  .dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
  .canvas { flex: 1; overflow: auto; padding: 12px; display: flex; gap: 16px; }
  .node-list { display: flex; flex-direction: column; gap: 8px; flex: 1; }
  .node { padding: 8px 12px; background: #252540; border-left: 3px solid; border-radius: 4px; cursor: pointer; transition: background 0.1s; }
  .node:hover { background: #2d2d5a; }
  .node-id { font-weight: 700; font-size: 11px; display: block; }
  .node-label { color: #ccc; font-size: 11px; }
  .edge-list { flex-shrink: 0; min-width: 200px; }
  .edge-header { font-size: 10px; color: #666; margin-bottom: 6px; text-transform: uppercase; }
  .edge { display: flex; align-items: center; gap: 4px; padding: 3px 6px; border-radius: 3px; cursor: pointer; font-size: 11px; }
  .edge:hover { background: #252540; }
  .edge-from { color: #4D96FF; font-weight: 500; }
  .edge-arrow { color: #666; }
  .edge-to { color: #6BCB77; font-weight: 500; }
  .edge-label { color: #888; font-size: 10px; margin-left: 4px; }
  .raw-block { border-top: 1px solid #333; }
  .raw-block summary { padding: 6px 12px; cursor: pointer; font-size: 11px; color: #666; }
  .raw-block pre { padding: 8px 12px; margin: 0; font-size: 11px; color: #888; overflow: auto; max-height: 120px; }
  .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; color: #666; }
  .empty p { margin: 4px 0; }
  .hint { font-size: 11px; color: #444; }
</style>
