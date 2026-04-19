<script lang="ts">
  import { onMount } from 'svelte';

  interface SpecNode {
    name: string;
    path: string;
    level: number;
    children: SpecNode[];
    expanded?: boolean;
  }

  interface Props {
    onselect?: (path: string, name: string) => void;
    selectedPath?: string;
  }

  let { onselect, selectedPath = $bindable('') }: Props = $props();

  let tree: SpecNode[] = $state([]);

  async function loadTree() {
    try {
      const fs = await import('fs');
      const path = await import('path');

      function buildNode(basePath: string, baseName: string, level: number): SpecNode {
        const entries = fs.readdirSync(basePath).sort();
        const children: SpecNode[] = [];

        for (const entry of entries) {
          const fullPath = path.join(basePath, entry);
          const stat = fs.statSync(fullPath);

          if (stat.isDirectory()) {
            children.push(buildNode(fullPath, entry, level + 1));
          } else if (entry.endsWith('.yaml')) {
            children.push({
              name: entry.replace('.yaml', ''),
              path: fullPath,
              level,
              children: [],
            });
          }
        }

        return { name: baseName, path: basePath, level, children };
      }

      const root = buildNode('/root/vibex-spec/specs', 'specs', 0);
      tree = [root];
      root.expanded = true;
    } catch (e) {
      console.warn('Failed to load spec tree:', e);
    }
  }

  onMount(() => {
    loadTree();
  });

  function toggle(node: SpecNode) {
    node.expanded = !node.expanded;
  }

  function select(node: SpecNode) {
    selectedPath = node.path;
    onselect?.(node.path, node.name);
  }

  function getFileIcon(name: string): string {
    if (name.includes('goal')) return '🎯';
    if (name.includes('architecture')) return '🏗️';
    if (name.includes('service')) return '⚙️';
    if (name.includes('data')) return '🗄️';
    if (name.includes('feature')) return '✨';
    if (name.includes('uiux')) return '🎨';
    if (name.includes('tauri')) return '🪟';
    if (name.includes('test')) return '🧪';
    return '📄';
  }

  function getLevel(name: string): string {
    if (name.includes('goal')) return '1';
    if (name.includes('architecture')) return '2';
    if (name.includes('service') || name.includes('data')) return '3';
    if (name.includes('feature')) return '4';
    if (name.includes('uiux') || name.includes('tauri') || name.includes('test')) return '5';
    return '?';
  }

  // Flatten tree for rendering
  function flatten(nodes: SpecNode[]): SpecNode[] {
    const result: SpecNode[] = [];
    for (const node of nodes) {
      result.push(node);
      if (node.expanded && node.children) {
        result.push(...flatten(node.children));
      }
    }
    return result;
  }

  let flatTree = $derived(flatten(tree));
</script>

<div class="file-tree">
  <div class="tree-header">
    <span>📂 规格文件</span>
    <button onclick={() => { tree = []; loadTree(); }} title="刷新">🔄</button>
  </div>

  <div class="tree-content">
    {#each flatTree as item}
      {@const isDir = item.children.length > 0}
      <div
        class="tree-item"
        class:dir={isDir}
        class:file={!isDir}
        class:selected={selectedPath === item.path}
        style="padding-left: {item.level * 16 + 8}px"
        onclick={() => isDir ? toggle(item) : select(item)}
        role="button"
        tabindex="0"
        onkeydown={(e) => e.key === 'Enter' && (isDir ? toggle(item) : select(item))}
      >
        <span class="icon">{isDir ? (item.expanded ? '📂' : '📁') : getFileIcon(item.name)}</span>
        <span class="name">{item.name}</span>
        {#if !isDir}
          <span class="level-badge">L{getLevel(item.name)}</span>
        {/if}
      </div>
    {/each}
  </div>
</div>

<style>
  .file-tree { display: flex; flex-direction: column; height: 100%; background: #1a1a2e; color: #e0e0e0; font-size: 12px; }
  .tree-header { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 1px solid #333; font-weight: 600; font-size: 11px; color: #888; text-transform: uppercase; }
  .tree-header button { background: transparent; border: none; cursor: pointer; font-size: 12px; padding: 2px; }
  .tree-content { flex: 1; overflow: auto; padding: 4px 0; }
  .tree-item { display: flex; align-items: center; gap: 6px; padding: 4px 8px; cursor: pointer; user-select: none; transition: background 0.1s; }
  .tree-item:hover { background: #252540; }
  .tree-item.selected { background: #2d4a7a; color: #fff; }
  .icon { font-size: 12px; flex-shrink: 0; }
  .name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .level-badge { font-size: 9px; background: #333; color: #888; padding: 1px 4px; border-radius: 3px; flex-shrink: 0; }
</style>
