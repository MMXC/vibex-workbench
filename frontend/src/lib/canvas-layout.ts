// ============================================================
// Canvas Layout — dagre 自动布局封装
// E5-U2: 使用 dagre 对 Canvas 节点自动布局
// ============================================================

import dagre from 'dagre';

export interface LayoutNode {
  id: string;
  type: string;
  width?: number;
  height?: number;
}

export interface LayoutOptions {
  direction?: 'TB' | 'LR' | 'BT' | 'RL';
  nodeWidth?: number;
  nodeHeight?: number;
  rankSeparation?: number;
  nodeSeparation?: number;
}

/** 使用 dagre 自动布局节点，返回 {x, y} map */
export function layoutNodes(
  nodes: LayoutNode[],
  edges: { source: string; target: string }[],
  options: LayoutOptions = {}
): Record<string, { x: number; y: number }> {
  const {
    direction = 'TB',
    nodeWidth = 200,
    nodeHeight = 80,
    rankSeparation = 50,
    nodeSeparation = 30,
  } = options;

  const g = new dagre.graphlib.Graph();
  g.setGraph({ direction, ranksep: rankSeparation, nodesep: nodeSeparation });
  g.setDefaultEdgeLabel(() => ({}));

  for (const node of nodes) {
    g.setNode(node.id, {
      width: node.width ?? nodeWidth,
      height: node.height ?? nodeHeight,
    });
  }

  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  const pos: Record<string, { x: number; y: number }> = {};
  g.nodes().forEach(nodeId => {
    const { x, y } = g.node(nodeId);
    pos[nodeId] = { x, y };
  });

  return pos;
}

/** 提取 canvasStore nodes + edges 用于布局 */
export function layoutCanvasStoreNodes(
  nodes: Array<{ id: string; type: string; parent_id?: string; position?: { x: number; y: number } }>,
  edges: Array<{ source: string; target: string }>
): Record<string, { x: number; y: number }> {
  return layoutNodes(nodes, edges);
}