/**
 * Unit tests for canvas-store.ts
 *
 * Strategy: direct import with vi.resetModules() in beforeEach to clear
 * the module cache between tests, giving each test a fresh singleton.
 * Covers: addNode, removeNode, updateNode, addEdge, removeEdge,
 *         selectNodes, setViewport, setTool, clear, nodeCount.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { canvasStore, nodeCount, type CanvasNode, type CanvasEdge } from './canvas-store';

let _store: typeof canvasStore;
let _nodeCount: typeof nodeCount;

// Helper: read current store state synchronously
function snap(store: { subscribe: (fn: (v: any) => void) => () => void }) {
  let s: any;
  const u = store.subscribe(x => { s = x; });
  u();
  return s;
}

// Helper: read derived store value
function snapDerived(store: { subscribe: (fn: (v: any) => void) => () => void }) {
  let s: any;
  const u = store.subscribe(x => { s = x; });
  u();
  return s;
}

// Helper: fresh store re-import
async function getFreshStores() {
  vi.resetModules();
  const mod = await import('./canvas-store');
  return { store: mod.canvasStore, nodeCount: mod.nodeCount };
}

beforeEach(async () => {
  const { store, nodeCount: nc } = await getFreshStores();
  _store = store;
  _nodeCount = nc;
});

// ── addNode ──────────────────────────────────────────────────
describe('canvasStore — addNode()', () => {
  it('adds a node to the nodes array', () => {
    const node: CanvasNode = {
      id: 'n1',
      type: 'run',
      position: { x: 100, y: 200 },
      data: { label: 'Test Run' },
    };
    _store.addNode(node);
    expect(snap(_store).nodes).toHaveLength(1);
    expect(snap(_store).nodes[0].id).toBe('n1');
  });

  it('accumulates multiple nodes', () => {
    _store.addNode({ id: 'a', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.addNode({ id: 'b', type: 'tool', position: { x: 10, y: 10 }, data: {} });
    _store.addNode({ id: 'c', type: 'tool', position: { x: 20, y: 20 }, data: {} });
    expect(snap(_store).nodes).toHaveLength(3);
  });

  it('does NOT reject duplicate ids (store allows it — caller responsibility)', () => {
    _store.addNode({ id: 'dup', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.addNode({ id: 'dup', type: 'tool', position: { x: 10, y: 10 }, data: {} });
    // Store blindly appends — no dedup
    expect(snap(_store).nodes).toHaveLength(2);
  });

  it('stores all node fields', () => {
    const node: CanvasNode = {
      id: 'full',
      type: 'artifact',
      position: { x: 50, y: 60 },
      width: 300,
      height: 150,
      data: { label: 'Art 1', args: { key: 'val' }, result: 'ok' },
      selected: false,
      expanded: true,
      z_index: 10,
      parent_id: 'parent-1',
    };
    _store.addNode(node);
    const saved = snap(_store).nodes[0];
    expect(saved.id).toBe('full');
    expect(saved.type).toBe('artifact');
    expect(saved.position).toEqual({ x: 50, y: 60 });
    expect(saved.width).toBe(300);
    expect(saved.height).toBe(150);
    expect(saved.data.label).toBe('Art 1');
    expect(saved.expanded).toBe(true);
    expect(saved.z_index).toBe(10);
    expect(saved.parent_id).toBe('parent-1');
  });
});

// ── removeNode ───────────────────────────────────────────────
describe('canvasStore — removeNode()', () => {
  it('removes a node by id', () => {
    _store.addNode({ id: 'to-remove', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.addNode({ id: 'to-keep', type: 'run', position: { x: 0, y: 0 }, data: {} });
    expect(snap(_store).nodes).toHaveLength(2);

    _store.removeNode('to-remove');
    expect(snap(_store).nodes).toHaveLength(1);
    expect(snap(_store).nodes[0].id).toBe('to-keep');
  });

  it('removes all edges connected to the removed node', () => {
    _store.addNode({ id: 'run', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.addNode({ id: 'tool', type: 'tool', position: { x: 10, y: 10 }, data: {} });
    _store.addEdge({ id: 'e1', source: 'run', target: 'tool' });
    _store.addEdge({ id: 'e2', source: 'tool', target: 'run' });
    expect(snap(_store).edges).toHaveLength(2);

    _store.removeNode('run');

    const remainingEdges = snap(_store).edges;
    expect(remainingEdges).toHaveLength(0);
  });

  it('also removes the node from selected_node_ids', () => {
    _store.addNode({ id: 'sel', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.selectNodes(['sel']);
    expect(snap(_store).selected_node_ids).toContain('sel');

    _store.removeNode('sel');
    expect(snap(_store).selected_node_ids).not.toContain('sel');
  });

  it('idempotent: removing a non-existent node does not throw', () => {
    expect(() => _store.removeNode('ghost')).not.toThrow();
  });
});

// ── updateNode ───────────────────────────────────────────────
describe('canvasStore — updateNode()', () => {
  it('updates node position', () => {
    _store.addNode({ id: 'n', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.updateNode('n', { position: { x: 999, y: 888 } });
    const updated = snap(_store).nodes.find((n: CanvasNode) => n.id === 'n');
    expect(updated?.position).toEqual({ x: 999, y: 888 });
  });

  it('updates node data', () => {
    _store.addNode({ id: 'n', type: 'run', position: { x: 0, y: 0 }, data: { label: 'old' } });
    _store.updateNode('n', { data: { label: 'new', status: 'completed' } } as any);
    const updated = snap(_store).nodes.find((n: CanvasNode) => n.id === 'n');
    expect(updated?.data.label).toBe('new');
    expect(updated?.data.status).toBe('completed');
  });

  it('merges partial data (shallow merge)', () => {
    _store.addNode({ id: 'n', type: 'run', position: { x: 0, y: 0 }, data: { label: 'old', extra: 'keep' } });
    _store.updateNode('n', { data: { label: 'new' } } as any);
    const updated = snap(_store).nodes.find((n: CanvasNode) => n.id === 'n');
    expect(updated?.data.label).toBe('new');
    expect(updated?.data.extra).toBeUndefined(); // data is replaced, not deep-merged
  });

  it('updates selected flag', () => {
    _store.addNode({ id: 'n', type: 'run', position: { x: 0, y: 0 }, data: {}, selected: false });
    _store.updateNode('n', { selected: true });
    expect(snap(_store).nodes[0].selected).toBe(true);
  });

  it('idempotent: updating non-existent node does not throw', () => {
    expect(() => _store.updateNode('ghost', { position: { x: 0, y: 0 } })).not.toThrow();
  });
});

// ── addEdge ──────────────────────────────────────────────────
describe('canvasStore — addEdge()', () => {
  it('adds an edge to the edges array', () => {
    const edge: CanvasEdge = { id: 'e1', source: 'a', target: 'b', label: 'calls' };
    _store.addEdge(edge);
    expect(snap(_store).edges).toHaveLength(1);
    expect(snap(_store).edges[0].id).toBe('e1');
    expect(snap(_store).edges[0].source).toBe('a');
    expect(snap(_store).edges[0].target).toBe('b');
  });

  it('accumulates multiple edges', () => {
    _store.addEdge({ id: 'e1', source: 'a', target: 'b' });
    _store.addEdge({ id: 'e2', source: 'b', target: 'c' });
    _store.addEdge({ id: 'e3', source: 'a', target: 'c' });
    expect(snap(_store).edges).toHaveLength(3);
  });

  it('stores animated and selected flags', () => {
    _store.addEdge({ id: 'e1', source: 'a', target: 'b', animated: true, selected: true });
    expect(snap(_store).edges[0].animated).toBe(true);
    expect(snap(_store).edges[0].selected).toBe(true);
  });

  it('allows duplicate edges (caller responsibility)', () => {
    _store.addEdge({ id: 'dup', source: 'a', target: 'b' });
    _store.addEdge({ id: 'dup', source: 'a', target: 'b' });
    expect(snap(_store).edges).toHaveLength(2);
  });
});

// ── removeEdge ───────────────────────────────────────────────
describe('canvasStore — removeEdge()', () => {
  it('removes an edge by id', () => {
    _store.addEdge({ id: 'kill', source: 'a', target: 'b' });
    _store.addEdge({ id: 'keep', source: 'c', target: 'd' });
    expect(snap(_store).edges).toHaveLength(2);

    _store.removeEdge('kill');
    expect(snap(_store).edges).toHaveLength(1);
    expect(snap(_store).edges[0].id).toBe('keep');
  });

  it('idempotent: removing non-existent edge does not throw', () => {
    expect(() => _store.removeEdge('ghost-edge')).not.toThrow();
    expect(snap(_store).edges).toHaveLength(0);
  });
});

// ── selectNodes ──────────────────────────────────────────────
describe('canvasStore — selectNodes()', () => {
  it('sets selected_node_ids to provided array', () => {
    _store.addNode({ id: 'a', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.addNode({ id: 'b', type: 'tool', position: { x: 10, y: 10 }, data: {} });

    _store.selectNodes(['a']);
    expect(snap(_store).selected_node_ids).toEqual(['a']);

    _store.selectNodes(['a', 'b']);
    expect(snap(_store).selected_node_ids).toEqual(['a', 'b']);
  });

  it('can clear selection by passing empty array', () => {
    _store.selectNodes(['a', 'b']);
    _store.selectNodes([]);
    expect(snap(_store).selected_node_ids).toEqual([]);
  });

  it('can re-select after clearing', () => {
    _store.selectNodes(['x']);
    _store.selectNodes([]);
    _store.selectNodes(['y']);
    expect(snap(_store).selected_node_ids).toEqual(['y']);
  });
});

// ── setViewport ──────────────────────────────────────────────
describe('canvasStore — setViewport()', () => {
  it('updates viewport fields', () => {
    _store.setViewport({ x: 100, y: 200, zoom: 1.5 });
    const vp = snap(_store).viewport;
    expect(vp.x).toBe(100);
    expect(vp.y).toBe(200);
    expect(vp.zoom).toBe(1.5);
  });

  it('partially updates viewport (merges)', () => {
    _store.setViewport({ x: 100, y: 200, zoom: 1 });
    _store.setViewport({ zoom: 2 });
    const vp = snap(_store).viewport;
    expect(vp.x).toBe(100); // preserved
    expect(vp.y).toBe(200); // preserved
    expect(vp.zoom).toBe(2);
  });

  it('can reset zoom to 1', () => {
    _store.setViewport({ zoom: 0.5 });
    _store.setViewport({ zoom: 1 });
    expect(snap(_store).viewport.zoom).toBe(1);
  });
});

// ── setTool ──────────────────────────────────────────────────
describe('canvasStore — setTool()', () => {
  it('changes tool to pan', () => {
    _store.setTool('pan');
    expect(snap(_store).tool).toBe('pan');
  });

  it('changes tool to add-node', () => {
    _store.setTool('add-node');
    expect(snap(_store).tool).toBe('add-node');
  });

  it('defaults to select', () => {
    // Initial state should be 'select'
    expect(snap(_store).tool).toBe('select');
  });

  it('can switch between all tool modes', () => {
    _store.setTool('pan');
    expect(snap(_store).tool).toBe('pan');
    _store.setTool('select');
    expect(snap(_store).tool).toBe('select');
    _store.setTool('add-node');
    expect(snap(_store).tool).toBe('add-node');
  });
});

// ── clear ───────────────────────────────────────────────────
describe('canvasStore — clear()', () => {
  it('empties nodes array', () => {
    _store.addNode({ id: 'n', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.clear();
    expect(snap(_store).nodes).toHaveLength(0);
  });

  it('empties edges array', () => {
    _store.addEdge({ id: 'e', source: 'a', target: 'b' });
    _store.clear();
    expect(snap(_store).edges).toHaveLength(0);
  });

  it('resets selected_node_ids', () => {
    _store.selectNodes(['n']);
    _store.clear();
    expect(snap(_store).selected_node_ids).toEqual([]);
  });

  it('preserves viewport after clear', () => {
    _store.setViewport({ x: 500, y: 300, zoom: 2 });
    _store.clear();
    expect(snap(_store).viewport).toEqual({ x: 500, y: 300, zoom: 2 });
  });

  it('preserves tool after clear', () => {
    _store.setTool('pan');
    _store.clear();
    expect(snap(_store).tool).toBe('pan');
  });
});

// ── nodeCount derived ───────────────────────────────────────
describe('canvasStore — nodeCount derived store', () => {
  it('is 0 initially', () => {
    expect(snapDerived(_nodeCount)).toBe(0);
  });

  it('increments when nodes are added', () => {
    _store.addNode({ id: 'a', type: 'run', position: { x: 0, y: 0 }, data: {} });
    expect(snapDerived(_nodeCount)).toBe(1);
    _store.addNode({ id: 'b', type: 'tool', position: { x: 10, y: 10 }, data: {} });
    expect(snapDerived(_nodeCount)).toBe(2);
  });

  it('decrements when nodes are removed', () => {
    _store.addNode({ id: 'a', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.addNode({ id: 'b', type: 'tool', position: { x: 10, y: 10 }, data: {} });
    _store.removeNode('a');
    expect(snapDerived(_nodeCount)).toBe(1);
  });

  it('resets to 0 after clear', () => {
    _store.addNode({ id: 'n', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.clear();
    expect(snapDerived(_nodeCount)).toBe(0);
  });

  it('correct count after mixed add/remove operations', () => {
    _store.addNode({ id: 'a', type: 'run', position: { x: 0, y: 0 }, data: {} });
    _store.addNode({ id: 'b', type: 'tool', position: { x: 10, y: 10 }, data: {} });
    _store.addNode({ id: 'c', type: 'tool', position: { x: 20, y: 20 }, data: {} });
    expect(snapDerived(_nodeCount)).toBe(3);
    _store.removeNode('b');
    expect(snapDerived(_nodeCount)).toBe(2);
    _store.addNode({ id: 'd', type: 'tool', position: { x: 30, y: 30 }, data: {} });
    expect(snapDerived(_nodeCount)).toBe(3);
  });
});

// ── Integration: full workflow ──────────────────────────────
describe('canvasStore — full workflow', () => {
  it('run → tool invocation → edge workflow mirrors SSE pipeline', () => {
    // Simulates the SSE pipeline in sse.ts for 'run.started' + 'tool.called'
    const runId = 'run-1';
    const toolId = 'tool-1';

    // E1: run.started creates a run node
    _store.addNode({
      id: runId,
      type: 'run',
      position: { x: 300, y: 100 },
      data: { label: 'Run: test', status: 'running' },
    });
    expect(snapDerived(_nodeCount)).toBe(1);

    // E2: tool.called creates a tool node + edge
    _store.addNode({
      id: toolId,
      type: 'tool',
      position: { x: 150, y: 200 },
      parent_id: runId,
      data: { label: 'bash', status: 'running', args: { cmd: 'ls' } },
    });
    _store.addEdge({ id: "e1", source: runId, target: toolId });
    expect(snapDerived(_nodeCount)).toBe(2);
    expect(snap(_store).edges).toHaveLength(1);
    expect(snap(_store).edges[0].source).toBe(runId);
    expect(snap(_store).edges[0].target).toBe(toolId);

    // E3: tool.completed updates the tool node
    _store.updateNode(toolId, { data: { label: 'bash', status: 'completed', result: { output: 'files' } } } as any);
    const toolNode = snap(_store).nodes.find((n: CanvasNode) => n.id === toolId);
    expect(toolNode?.data.status).toBe('completed');

    // E4: removing the run also removes the edge (tool is orphaned but still present)
    _store.removeNode(runId);
    expect(snap(_store).edges).toHaveLength(0);
    // Tool node still exists — only the edge was removed, not the orphaned tool
    expect(snap(_store).nodes.find((n: CanvasNode) => n.id === toolId)).toBeDefined();

    // E5: removing the orphaned tool cleans up completely
    _store.removeNode(toolId);
    expect(snap(_store).nodes).toHaveLength(0);
  });
});
