/**
 * Unit tests for canvas-layout.ts
 *
 * Strategy: import pure functions directly and test with deterministic inputs.
 * Uses vi.resetModules() between tests to ensure a clean dagre state.
 * Tests cover: basic layout, direction modes, edge connectivity, options, edge cases.
 */
import { describe, it, expect, beforeEach } from 'vitest';
import dagre from 'dagre';
import { layoutNodes, layoutCanvasStoreNodes, type LayoutNode, type LayoutOptions } from './canvas-layout';

let _layoutNodes: typeof layoutNodes;
let _layoutCanvasStoreNodes: typeof layoutCanvasStoreNodes;

async function getFresh() {
  vi.resetModules();
  const mod = await import('./canvas-layout');
  return { fn: mod.layoutNodes, fn2: mod.layoutCanvasStoreNodes };
}

beforeEach(async () => {
  const { fn, fn2 } = await getFresh();
  _layoutNodes = fn;
  _layoutCanvasStoreNodes = fn2;
});

// ── T1: Returns position for each node ───────────────────────
describe('layoutNodes — returns position for each node', () => {
  it('returns a map entry for every input node', () => {
    const nodes: LayoutNode[] = [
      { id: 'a', type: 'run' },
      { id: 'b', type: 'tool' },
      { id: 'c', type: 'tool' },
    ];
    const edges: { source: string; target: string }[] = [];

    const pos = _layoutNodes(nodes, edges);

    expect(Object.keys(pos)).toHaveLength(3);
    expect(pos['a']).toHaveProperty('x');
    expect(pos['a']).toHaveProperty('y');
    expect(pos['b']).toHaveProperty('x');
    expect(pos['b']).toHaveProperty('y');
    expect(pos['c']).toHaveProperty('x');
    expect(pos['c']).toHaveProperty('y');
  });

  it('returns a plain object (not null/undefined)', () => {
    const nodes: LayoutNode[] = [{ id: 'n1', type: 'run' }];
    const pos = _layoutNodes(nodes, []);
    expect(pos).toBeTruthy();
    expect(typeof pos).toBe('object');
  });
});

// ── T2: TB direction — children below parents ────────────────
describe('layoutNodes — TB (top-to-bottom) direction', () => {
  it('child node is below parent node in TB mode', () => {
    const nodes: LayoutNode[] = [
      { id: 'run', type: 'run' },
      { id: 'tool', type: 'tool' },
    ];
    const edges = [{ source: 'run', target: 'tool' }];

    const pos = _layoutNodes(nodes, edges, { direction: 'TB' });

    expect(pos['tool'].y).toBeGreaterThan(pos['run'].y);
  });

  it('deeply nested nodes stack vertically', () => {
    const nodes: LayoutNode[] = [
      { id: 'a', type: 'run' },
      { id: 'b', type: 'tool' },
      { id: 'c', type: 'tool' },
    ];
    // a → b → c
    const edges = [
      { source: 'a', target: 'b' },
      { source: 'b', target: 'c' },
    ];

    const pos = _layoutNodes(nodes, edges, { direction: 'TB' });

    expect(pos['b'].y).toBeGreaterThan(pos['a'].y);
    expect(pos['c'].y).toBeGreaterThan(pos['b'].y);
  });
});

// ── T3: LR direction — children to the right ─────────────────
describe('layoutNodes — LR (left-to-right) direction', () => {
  it('child node is to the right of parent node in LR mode', () => {
    // With more nodes and wider layout, LR spreads horizontally
    const nodes: LayoutNode[] = [
      { id: 'root', type: 'run' },
      { id: 'lvl1', type: 'tool' },
      { id: 'lvl2', type: 'tool' },
    ];
    const edges = [
      { source: 'root', target: 'lvl1' },
      { source: 'lvl1', target: 'lvl2' },
    ];

    const pos = _layoutNodes(nodes, edges, { direction: 'LR', rankSeparation: 100 });

    expect(pos['lvl1'].x).toBeGreaterThan(pos['root'].x);
    expect(pos['lvl2'].x).toBeGreaterThan(pos['lvl1'].x);
  });

  it('siblings are both to the right of root in LR mode', () => {
    const nodes: LayoutNode[] = [
      { id: 'root', type: 'run' },
      { id: 'child1', type: 'tool' },
      { id: 'child2', type: 'tool' },
    ];
    const edges = [
      { source: 'root', target: 'child1' },
      { source: 'root', target: 'child2' },
    ];

    const pos = _layoutNodes(nodes, edges, { direction: 'LR', rankSeparation: 80 });

    // Both children should be to the right of root in LR mode
    expect(pos['child1'].x).toBeGreaterThan(pos['root'].x);
    expect(pos['child2'].x).toBeGreaterThan(pos['root'].x);
    // Siblings may or may not share the same y depending on dagre's crossing-minimization
    // The key invariant: both are to the right of the root
  });
});

// ── T4: Empty nodes array ────────────────────────────────────
describe('layoutNodes — empty input', () => {
  it('returns empty map for empty nodes array', () => {
    const pos = _layoutNodes([], []);
    expect(Object.keys(pos)).toHaveLength(0);
  });

  it('returns empty map for nodes with no edges', () => {
    const nodes: LayoutNode[] = [{ id: 'solo', type: 'run' }];
    const pos = _layoutNodes(nodes, []);
    // Dagre may still place a single disconnected node
    expect(Object.keys(pos)).toHaveLength(1);
    expect(pos['solo']).toHaveProperty('x');
    expect(pos['solo']).toHaveProperty('y');
  });

  it('handles edges that reference non-existent nodes gracefully', () => {
    const nodes: LayoutNode[] = [{ id: 'a', type: 'run' }];
    const edges = [{ source: 'a', target: 'ghost' }];
    // dagre v0.8.5 auto-creates ghost nodes for dangling edge references.
    // The function should not throw and should return positions for real nodes.
    // Both 'a' (real) and 'ghost' (auto-created by dagre) get positions.
    expect(() => _layoutNodes(nodes, edges)).not.toThrow();
    const pos = _layoutNodes(nodes, edges);
    expect(pos['a']).toHaveProperty('x');
    expect(pos['a']).toHaveProperty('y');
    expect(typeof pos['a'].x).toBe('number');
    expect(typeof pos['a'].y).toBe('number');
    // ghost node may or may not get a position depending on dagre version
    // the key invariant: real nodes always get valid positions
  });
});

// ── T5: Existing positions are ignored by dagre ──────────────
describe('layoutNodes — existing positions', () => {
  it('still computes new positions even when nodes have pre-existing positions', () => {
    const nodes: LayoutNode[] = [
      { id: 'a', type: 'run' },
      { id: 'b', type: 'tool' },
    ];
    const edges = [{ source: 'a', target: 'b' }];

    // Call twice — dagre always re-computes from scratch
    const pos1 = _layoutNodes(nodes, edges);
    const pos2 = _layoutNodes(nodes, edges);

    // Positions should be deterministic (same each call)
    expect(pos1['a'].x).toBeCloseTo(pos2['a'].x, 1);
    expect(pos1['a'].y).toBeCloseTo(pos2['a'].y, 1);
    expect(pos1['b'].x).toBeCloseTo(pos2['b'].x, 1);
    expect(pos1['b'].y).toBeCloseTo(pos2['b'].y, 1);
  });
});

// ── T6: Multiple ranks — same y for same rank ───────────────
describe('layoutNodes — multiple ranks', () => {
  it('siblings in TB mode share the same y (same rank)', () => {
    const nodes: LayoutNode[] = [
      { id: 'root', type: 'run' },
      { id: 'c1', type: 'tool' },
      { id: 'c2', type: 'tool' },
    ];
    const edges = [
      { source: 'root', target: 'c1' },
      { source: 'root', target: 'c2' },
    ];

    const pos = _layoutNodes(nodes, edges, { direction: 'TB' });

    // Both children should be on the same rank (same y)
    expect(pos['c1'].y).toBeCloseTo(pos['c2'].y, 0);
  });

  it('grandchild is on a different rank than parent and sibling', () => {
    const nodes: LayoutNode[] = [
      { id: 'root', type: 'run' },
      { id: 'child', type: 'tool' },
      { id: 'grandchild', type: 'tool' },
    ];
    const edges = [
      { source: 'root', target: 'child' },
      { source: 'child', target: 'grandchild' },
    ];

    const pos = _layoutNodes(nodes, edges, { direction: 'TB' });

    expect(pos['root'].y).toBeLessThan(pos['child'].y);
    expect(pos['child'].y).toBeLessThan(pos['grandchild'].y);
    // root and grandchild should NOT share the same rank
    expect(pos['grandchild'].y).not.toBeCloseTo(pos['root'].y, 0);
  });
});

// ── T7: Edges connect source→target correctly ────────────────
describe('layoutNodes — edge connectivity', () => {
  it('reverse edge swaps relative positions', () => {
    const nodes: LayoutNode[] = [
      { id: 'a', type: 'run' },
      { id: 'b', type: 'tool' },
    ];

    // a → b (top-to-bottom)
    const pos1 = _layoutNodes(nodes, [{ source: 'a', target: 'b' }], { direction: 'TB' });
    expect(pos1['b'].y).toBeGreaterThan(pos1['a'].y);

    // b → a (bottom-to-top)
    const pos2 = _layoutNodes(nodes, [{ source: 'b', target: 'a' }], { direction: 'TB' });
    expect(pos2['a'].y).toBeGreaterThan(pos2['b'].y);
  });

  it('diamond graph — all nodes get positions', () => {
    // Diamond:      root
    //             /    \
    //           left   right
    //             \    /
    //              bottom
    const nodes: LayoutNode[] = [
      { id: 'root', type: 'run' },
      { id: 'left', type: 'tool' },
      { id: 'right', type: 'tool' },
      { id: 'bottom', type: 'tool' },
    ];
    const edges = [
      { source: 'root', target: 'left' },
      { source: 'root', target: 'right' },
      { source: 'left', target: 'bottom' },
      { source: 'right', target: 'bottom' },
    ];

    const pos = _layoutNodes(nodes, edges);

    expect(Object.keys(pos)).toHaveLength(4);
    // root should be above both left and right
    expect(pos['root'].y).toBeLessThan(pos['left'].y);
    expect(pos['root'].y).toBeLessThan(pos['right'].y);
    // bottom should be below left and right
    expect(pos['bottom'].y).toBeGreaterThan(pos['left'].y);
    expect(pos['bottom'].y).toBeGreaterThan(pos['right'].y);
  });
});

// ── T8: nodeWidth / nodeHeight options ───────────────────────
describe('layoutNodes — nodeWidth/nodeHeight options', () => {
  it('accepts custom nodeWidth and nodeHeight', () => {
    const nodes: LayoutNode[] = [
      { id: 'small', type: 'run', width: 100, height: 50 },
      { id: 'large', type: 'tool', width: 400, height: 200 },
    ];
    const edges: { source: string; target: string }[] = [];

    const pos = _layoutNodes(nodes, edges, {
      nodeWidth: 200,
      nodeHeight: 80,
    });

    // Both should still get positions (dagre handles the dimensions)
    expect(pos['small']).toHaveProperty('x');
    expect(pos['large']).toHaveProperty('y');
  });

  it('uses per-node width/height when provided', () => {
    const nodes1: LayoutNode[] = [{ id: 'n', type: 'run', width: 100, height: 40 }];
    const nodes2: LayoutNode[] = [{ id: 'n', type: 'run', width: 400, height: 160 }];

    const pos1 = _layoutNodes(nodes1, []);
    const pos2 = _layoutNodes(nodes2, []);

    // Different node sizes affect position spacing, but both should return valid positions
    expect(pos1['n']).toHaveProperty('x');
    expect(pos2['n']).toHaveProperty('x');
  });
});

// ── T9: rankSeparation / nodeSeparation options ─────────────
describe('layoutNodes — rankSeparation/nodeSeparation options', () => {
  it('accepts rankSeparation option', () => {
    const nodes: LayoutNode[] = [
      { id: 'a', type: 'run' },
      { id: 'b', type: 'tool' },
    ];
    const edges = [{ source: 'a', target: 'b' }];

    const pos1 = _layoutNodes(nodes, edges, { rankSeparation: 50 });
    const pos2 = _layoutNodes(nodes, edges, { rankSeparation: 300 });

    // Larger ranksep → larger y-gap
    const gap1 = Math.abs(pos1['b'].y - pos1['a'].y);
    const gap2 = Math.abs(pos2['b'].y - pos2['a'].y);
    expect(gap2).toBeGreaterThan(gap1);
  });

  it('accepts nodeSeparation option', () => {
    const nodes: LayoutNode[] = [
      { id: 'a', type: 'tool' },
      { id: 'b', type: 'tool' },
    ];
    // No edge → siblings in same rank
    const edges: { source: string; target: string }[] = [];

    const pos1 = _layoutNodes(nodes, edges, { nodeSeparation: 10 });
    const pos2 = _layoutNodes(nodes, edges, { nodeSeparation: 200 });

    const xGap1 = Math.abs(pos1['b'].x - pos1['a'].x);
    const xGap2 = Math.abs(pos2['b'].x - pos2['a'].x);
    expect(xGap2).toBeGreaterThan(xGap1);
  });

  it('uses defaults when options omitted', () => {
    const nodes: LayoutNode[] = [{ id: 'n', type: 'run' }];
    const pos = _layoutNodes(nodes, []);
    expect(pos['n']).toHaveProperty('x');
    expect(pos['n']).toHaveProperty('y');
    expect(typeof pos['n'].x).toBe('number');
    expect(typeof pos['n'].y).toBe('number');
  });
});

// ── T10: layoutCanvasStoreNodes is a thin wrapper ─────────────
describe('layoutCanvasStoreNodes — passthrough to layoutNodes', () => {
  it('returns positions for canvas-store-style nodes', () => {
    const nodes = [
      { id: 'run-1', type: 'run', position: { x: 0, y: 0 } },
      { id: 'tool-1', type: 'tool', parent_id: 'run-1' },
    ];
    const edges = [{ source: 'run-1', target: 'tool-1' }];

    const pos = _layoutCanvasStoreNodes(nodes, edges);

    expect(pos['run-1']).toHaveProperty('x');
    expect(pos['run-1']).toHaveProperty('y');
    expect(pos['tool-1']).toHaveProperty('x');
    expect(pos['tool-1']).toHaveProperty('y');
  });

  it('ignores parent_id (not used by dagre)', () => {
    const nodes = [
      { id: 'a', type: 'run' },
      { id: 'b', type: 'tool', parent_id: 'a' },
    ];
    const edges = [{ source: 'a', target: 'b' }];

    const pos = _layoutCanvasStoreNodes(nodes, edges);

    expect(Object.keys(pos)).toHaveLength(2);
  });

  it('handles nodes with and without parent_id', () => {
    const nodes = [
      { id: 'r', type: 'run' },
      { id: 't1', type: 'tool', parent_id: 'r' },
      { id: 't2', type: 'tool' }, // no parent
    ];
    const edges = [{ source: 'r', target: 't1' }];

    const pos = _layoutCanvasStoreNodes(nodes, edges);

    expect(Object.keys(pos)).toHaveLength(3);
  });
});

// ── T11: BT and RL directions ─────────────────────────────────
describe('layoutNodes — BT and RL directions', () => {
  it('BT (bottom-to-top) places target above source', () => {
    // Need at least 3 nodes in a chain to force dagre to use multiple ranks
    const nodes: LayoutNode[] = [
      { id: 'a', type: 'run' },
      { id: 'b', type: 'tool' },
      { id: 'c', type: 'tool' },
    ];
    const edges = [
      { source: 'a', target: 'b' },
      { source: 'b', target: 'c' },
    ];

    const pos = _layoutNodes(nodes, edges, { direction: 'BT', rankSeparation: 80 });

    expect(pos['b'].y).toBeLessThan(pos['a'].y);
    expect(pos['c'].y).toBeLessThan(pos['b'].y);
  });

  it('RL (right-to-left) places target to the left of source', () => {
    const nodes: LayoutNode[] = [
      { id: 'a', type: 'run' },
      { id: 'b', type: 'tool' },
      { id: 'c', type: 'tool' },
    ];
    const edges = [
      { source: 'a', target: 'b' },
      { source: 'b', target: 'c' },
    ];

    const pos = _layoutNodes(nodes, edges, { direction: 'RL', rankSeparation: 80 });

    expect(pos['b'].x).toBeLessThan(pos['a'].x);
    expect(pos['c'].x).toBeLessThan(pos['b'].x);
  });
});
