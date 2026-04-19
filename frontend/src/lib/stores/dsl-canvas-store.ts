// DSL Canvas Store — 管理 Spec 从属关系图的可视化状态
import { writable, derived } from 'svelte/store';

// ── Types ────────────────────────────────────────────────────
export interface SpecNode {
  id: string;
  label: string;
  level: string; // L1–L5
  levelIndex: number;
  parent: string | null;
  filePath: string;
}

export interface DependencyEdge {
  from: string;
  to: string;
}

export interface DslCanvasState {
  // 渲染状态
  mermaidCode: string;
  renderStatus: 'idle' | 'loading' | 'rendered' | 'error';
  errorMessage: string;

  // 交互状态
  highlightedSpecId: string | null;
  selectedSpecId: string | null;

  // 视口状态
  zoomLevel: number; // 0.25–3.0
  panOffset: { x: number; y: number };

  // 视图模式
  viewMode: 'mermaid' | 'canvas';

  // 解析后的图数据
  nodes: SpecNode[];
  edges: DependencyEdge[];
}

// ── Store ───────────────────────────────────────────────────
const initialState: DslCanvasState = {
  mermaidCode: '',
  renderStatus: 'idle',
  errorMessage: '',
  highlightedSpecId: null,
  selectedSpecId: null,
  zoomLevel: 1.0,
  panOffset: { x: 0, y: 0 },
  viewMode: 'mermaid',
  nodes: [],
  edges: [],
};

function createDslCanvasStore() {
  const { subscribe, set, update } = writable<DslCanvasState>(initialState);

  return {
    subscribe,

    /** 设置 Mermaid 代码并解析为节点/边 */
    setMermaidCode(code: string) {
      update(s => ({
        ...s,
        mermaidCode: code,
        renderStatus: 'idle',
        errorMessage: '',
        nodes: [],
        edges: [],
      }));
      // Parse nodes and edges from mermaid code
      const nodes = parseNodesFromMermaid(code);
      const edges = parseEdgesFromMermaid(code);
      update(s => ({ ...s, nodes, edges }));
    },

    /** 设置渲染状态 */
    setRenderStatus(status: DslCanvasState['renderStatus'], errorMessage = '') {
      update(s => ({ ...s, renderStatus: status, errorMessage }));
    },

    /** 高亮指定 spec */
    highlightSpec(specId: string | null) {
      update(s => ({ ...s, highlightedSpecId: specId }));
    },

    /** 选中指定 spec */
    selectSpec(specId: string | null) {
      update(s => ({ ...s, selectedSpecId: specId }));
    },

    /** 缩放 */
    setZoom(level: number) {
      update(s => ({ ...s, zoomLevel: Math.max(0.25, Math.min(3.0, level)) }));
    },

    /** 缩放步进 */
    zoomIn() {
      update(s => ({ ...s, zoomLevel: Math.min(3.0, s.zoomLevel + 0.25) }));
    },

    zoomOut() {
      update(s => ({ ...s, zoomLevel: Math.max(0.25, s.zoomLevel - 0.25) }));
    },

    /** 平移 */
    setPan(offset: { x: number; y: number }) {
      update(s => ({ ...s, panOffset: offset }));
    },

    /** 切换视图模式 */
    setViewMode(mode: DslCanvasState['viewMode']) {
      update(s => ({ ...s, viewMode: mode }));
    },

    /** 重置视口 */
    resetViewport() {
      update(s => ({ ...s, zoomLevel: 1.0, panOffset: { x: 0, y: 0 } }));
    },

    /** 重新加载依赖图 */
    async refreshGraph() {
      update(s => ({ ...s, renderStatus: 'loading' }));
      try {
        const res = await fetch('/api/specs/mermaid');
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json() as { mermaid?: string; code?: string };
        const code = data.mermaid || data.code || '';
        update(s => ({ ...s, mermaidCode: code, renderStatus: 'idle' }));
        update(s => {
          const nodes = parseNodesFromMermaid(code);
          const edges = parseEdgesFromMermaid(code);
          return { ...s, nodes, edges };
        });
      } catch (e) {
        update(s => ({
          ...s,
          renderStatus: 'error',
          errorMessage: `无法加载依赖图: ${e instanceof Error ? e.message : String(e)}`,
        }));
      }
    },

    /** 清空状态 */
    reset() {
      set(initialState);
    },
  };
}

// ── Parsers ──────────────────────────────────────────────────

const LEVEL_RE = /:::(lv\d+)/;
const NODE_RE = /\s+(\w+)\["([^\"]+)"\]/g;
const EDGE_RE = /\s+(\w+)\s+-->\s+(\w+)/g;

/** 从 Mermaid 代码解析出节点列表 */
function parseNodesFromMermaid(code: string): SpecNode[] {
  const nodes: SpecNode[] = [];
  const seen = new Set<string>();

  let match: RegExpExecArray | null;
  const nodeRe = /\s+(\w+)["\[]/g;
  while ((match = nodeRe.exec(code)) !== null) {
    const id = match[1];
    if (seen.has(id) || id === 'spec_hierarchy') continue;
    seen.add(id);

    // Extract label from ["label"] or [label]
    const labelMatch = code.slice(match.index).match(/^\s*\w+\["([^\"]+)"\]/);
    const label = labelMatch ? labelMatch[1] : id;

    // Infer level from ID patterns
    const level = inferLevel(id);
    const levelIndex = parseInt(level.replace('L', ''), 10);

    nodes.push({ id, label, level, levelIndex, parent: null, filePath: '' });
  }

  return nodes.sort((a, b) => a.levelIndex - b.levelIndex);
}

/** 从 Mermaid 代码解析出边列表 */
function parseEdgesFromMermaid(code: string): DependencyEdge[] {
  const edges: DependencyEdge[] = [];
  let match: RegExpExecArray | null;
  const edgeRe = /\s+(\w+)\s+-->\s+(\w+)/g;
  while ((match = edgeRe.exec(code)) !== null) {
    edges.push({ from: match[1], to: match[2] });
  }
  return edges;
}

/** 从节点 ID 推断层级 */
function inferLevel(nodeId: string): string {
  const id = nodeId.toLowerCase();
  if (id.includes('goal') || id.includes('project')) return 'L1';
  if (id.includes('skeleton') || id.includes('arch')) return 'L2';
  if (id.includes('module') || id.includes('mod_')) return 'L3';
  if (id.includes('feature') || id.includes('canvas') || id.includes('editor') || id.includes('tool')) return 'L4';
  return 'L5';
}

export const dslCanvasStore = createDslCanvasStore();

// ── Derived stores ──────────────────────────────────────────

/** 按层级分组的节点 */
export const nodesByLevel = derived(dslCanvasStore, $s => {
  const grouped: Record<string, SpecNode[]> = {};
  for (const node of $s.nodes) {
    if (!grouped[node.level]) grouped[node.level] = [];
    grouped[node.level].push(node);
  }
  return grouped;
});

/** 当前选中的节点数据 */
export const selectedNode = derived(dslCanvasStore, $s =>
  $s.nodes.find(n => n.id === $s.selectedSpecId) ?? null
);
