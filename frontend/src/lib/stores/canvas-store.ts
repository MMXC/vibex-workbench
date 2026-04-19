// Canvas Store — 管理 Canvas 节点图
import { writable, derived } from 'svelte/store';

export interface CanvasNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  width?: number;
  height?: number;
  data: Record<string, unknown>;
  selected?: boolean;
  dragging?: boolean;
  expanded?: boolean;
  z_index?: number;
  parent_id?: string;
  created_at?: string;
}

export interface CanvasEdge {
  id: string;
  source: string;
  target: string;
  type?: string;
  label?: string;
  animated?: boolean;
  selected?: boolean;
}

export interface Viewport {
  x: number;
  y: number;
  zoom: number;
}

export interface CanvasState {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  selected_node_ids: string[];
  viewport: Viewport;
  tool: 'select' | 'pan' | 'add-node';
}

function createCanvasStore() {
  const { subscribe, update } = writable<CanvasState>({
    nodes: [],
    edges: [],
    selected_node_ids: [],
    viewport: { x: 0, y: 0, zoom: 1 },
    tool: 'select',
  });

  return {
    subscribe,
    addNode(node: CanvasNode) {
      update(s => ({ ...s, nodes: [...s.nodes, node] }));
    },
    removeNode(id: string) {
      update(s => ({
        ...s,
        nodes: s.nodes.filter(n => n.id !== id),
        edges: s.edges.filter(e => e.source !== id && e.target !== id),
        selected_node_ids: s.selected_node_ids.filter(nid => nid !== id),
      }));
    },
    updateNode(id: string, patch: Partial<CanvasNode>) {
      update(s => ({
        ...s,
        nodes: s.nodes.map(n => n.id === id ? { ...n, ...patch } : n),
      }));
    },
    addEdge(edge: CanvasEdge) {
      update(s => ({ ...s, edges: [...s.edges, edge] }));
    },
    removeEdge(id: string) {
      update(s => ({ ...s, edges: s.edges.filter(e => e.id !== id) }));
    },
    selectNodes(ids: string[]) {
      update(s => ({ ...s, selected_node_ids: ids }));
    },
    setViewport(vp: Partial<Viewport>) {
      update(s => ({ ...s, viewport: { ...s.viewport, ...vp } }));
    },
    setTool(tool: CanvasState['tool']) {
      update(s => ({ ...s, tool }));
    },
    clear() {
      update(s => ({ ...s, nodes: [], edges: [], selected_node_ids: [] }));
    },
  };
}

export const canvasStore = createCanvasStore();
export const nodeCount = derived(canvasStore, $s => $s.nodes.length);
