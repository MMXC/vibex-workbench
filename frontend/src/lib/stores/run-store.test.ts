/**
 * Unit tests for run-store.ts
 *
 * Strategy: direct import with vi.resetModules() in beforeEach to clear
 * the module cache between tests, giving each test a fresh singleton.
 */
import { describe, it, expect, beforeEach } from 'vitest';
import { runStore, activeRun, type Run, type ToolInvocation } from './run-store';

let _runStore: any;
let _activeRun: any;

async function getFreshStores() {
  // Reset module cache so we get fresh singletons
  vi.resetModules();
  const mod = await import('./run-store');
  return { store: mod.runStore, active: mod.activeRun };
}

// Helper: read current store state synchronously
function snap(store: any) {
  let s: any;
  const u = store.subscribe(x => { s = x; });
  u();
  return s;
}

// Helper: wait for derived store value
function waitForDerived<T>(store: { subscribe: (fn: (v: T) => void) => () => void }): T {
  let val!: T;
  const u = store.subscribe(v => { val = v; });
  u();
  return val;
}

beforeEach(async () => {
  // Re-import the module to get a fresh store instance
  const { store, active } = await getFreshStores();
  _runStore = store;
  _activeRun = active;
});

// ── createRun ────────────────────────────────────────────────
describe('runStore — createRun()', () => {
  it('creates a run with status="pending" and adds it to runs array', () => {
    const s0 = snap(_runStore);
    expect(s0.runs).toHaveLength(0);
    _runStore.createRun('thread-1');
    const s1 = snap(_runStore);
    expect(s1.runs).toHaveLength(1);
    expect(s1.runs[0].status).toBe('pending');
    expect(s1.runs[0].thread_id).toBe('thread-1');
  });

  it('returns the created Run object', () => {
    const run = _runStore.createRun('thread-2');
    expect(run).toBeDefined();
    expect(run.id).toBeTruthy();
    expect(run.thread_id).toBe('thread-2');
    expect(run.status).toBe('pending');
    expect(run.created_at).toBeTruthy();
  });

  it('generates unique ids for each run', () => {
    const r1 = _runStore.createRun('t1');
    const r2 = _runStore.createRun('t2');
    expect(r1.id).not.toBe(r2.id);
  });
});

// ── updateRunStatus ─────────────────────────────────────────
describe('runStore — updateRunStatus()', () => {
  it('updates the run status', () => {
    const run = _runStore.createRun('thread-x');
    _runStore.updateRunStatus(run.id, 'executing');
    const found = snap(_runStore).runs.find((r: Run) => r.id === run.id);
    expect(found?.status).toBe('executing');
  });

  it('sets finished_at timestamp when finishedAt is provided', () => {
    const run = _runStore.createRun('thread-y');
    const ts = '2025-01-01T12:00:00.000Z';
    _runStore.updateRunStatus(run.id, 'completed', ts);
    const found = snap(_runStore).runs.find((r: Run) => r.id === run.id);
    expect(found?.finished_at).toBe(ts);
  });

  it('does NOT auto-set finished_at when finishedAt is omitted', () => {
    const run = _runStore.createRun('thread-z');
    _runStore.updateRunStatus(run.id, 'completed');
    const found = snap(_runStore).runs.find((r: Run) => r.id === run.id);
    // finished_at stays undefined — caller must pass finishedAt explicitly
    expect(found?.finished_at).toBeUndefined();
  });

  it('clears active_run_id when status is "completed"', () => {
    const run = _runStore.createRun('thread-a');
    _runStore.setActiveRun(run.id);
    expect(snap(_runStore).active_run_id).toBe(run.id);
    _runStore.updateRunStatus(run.id, 'completed');
    expect(snap(_runStore).active_run_id).toBeNull();
  });

  it('clears active_run_id when status is "failed"', () => {
    const run = _runStore.createRun('thread-b');
    _runStore.setActiveRun(run.id);
    expect(snap(_runStore).active_run_id).toBe(run.id);
    _runStore.updateRunStatus(run.id, 'failed');
    expect(snap(_runStore).active_run_id).toBeNull();
  });

  it('clears active_run_id when status is "cancelled"', () => {
    const run = _runStore.createRun('thread-c');
    _runStore.setActiveRun(run.id);
    expect(snap(_runStore).active_run_id).toBe(run.id);
    _runStore.updateRunStatus(run.id, 'cancelled');
    expect(snap(_runStore).active_run_id).toBeNull();
  });

  it('keeps active_run_id set when status is "executing"', () => {
    const run = _runStore.createRun('thread-d');
    _runStore.setActiveRun(run.id);
    _runStore.updateRunStatus(run.id, 'executing');
    expect(snap(_runStore).active_run_id).toBe(run.id);
  });

  it('keeps active_run_id set when status is "planning"', () => {
    const run = _runStore.createRun('thread-e');
    _runStore.setActiveRun(run.id);
    _runStore.updateRunStatus(run.id, 'planning');
    expect(snap(_runStore).active_run_id).toBe(run.id);
  });
});

// ── setActiveRun ─────────────────────────────────────────────
describe('runStore — setActiveRun()', () => {
  it('sets active_run_id', () => {
    const run = _runStore.createRun('thread-f');
    _runStore.setActiveRun(run.id);
    expect(snap(_runStore).active_run_id).toBe(run.id);
  });

  it('can set active_run_id to null', () => {
    const run = _runStore.createRun('thread-g');
    _runStore.setActiveRun(run.id);
    _runStore.setActiveRun(null);
    expect(snap(_runStore).active_run_id).toBeNull();
  });
});

// ── activeRun derived store ─────────────────────────────────
describe('runStore — activeRun derived store', () => {
  it('returns the current active run', () => {
    const run = _runStore.createRun('thread-h');
    _runStore.setActiveRun(run.id);
    const val = waitForDerived(_activeRun);
    expect(val?.id).toBe(run.id);
  });

  it('returns null when no run is active', () => {
    _runStore.setActiveRun(null);
    const val = waitForDerived(_activeRun);
    expect(val).toBeNull();
  });
});

// ── addToolInvocation ────────────────────────────────────────
describe('runStore — addToolInvocation()', () => {
  it('adds a tool invocation to the toolInvocations array', () => {
    const run = _runStore.createRun('thread-i');
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'bash', status: 'pending' });
    const s = snap(_runStore);
    expect(s.toolInvocations).toHaveLength(1);
    expect(s.toolInvocations[0].tool_name).toBe('bash');
    expect(s.toolInvocations[0].run_id).toBe(run.id);
  });

  it('auto-generates id if not provided', () => {
    const run = _runStore.createRun('thread-j');
    const inv = _runStore.addToolInvocation({ run_id: run.id, tool_name: 'read_file', status: 'pending' });
    expect(inv.id).toBeTruthy();
    expect(typeof inv.id).toBe('string');
  });

  it('uses provided id when given', () => {
    const run = _runStore.createRun('thread-k');
    const inv = _runStore.addToolInvocation({ id: 'my-custom-id', run_id: run.id, tool_name: 'write', status: 'pending' });
    expect(inv.id).toBe('my-custom-id');
  });

  it('sets started_at to current time if not provided', () => {
    const run = _runStore.createRun('thread-l');
    const inv = _runStore.addToolInvocation({ run_id: run.id, tool_name: 'bash', status: 'pending' });
    expect(inv.started_at).toBeTruthy();
  });

  it('accumulates multiple tool invocations', () => {
    const run = _runStore.createRun('thread-m');
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'tool-1', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'tool-2', status: 'pending' });
    expect(snap(_runStore).toolInvocations).toHaveLength(2);
  });
});

// ── updateToolInvocation ────────────────────────────────────
describe('runStore — updateToolInvocation()', () => {
  it('updates an existing tool invocation', () => {
    const run = _runStore.createRun('thread-n');
    _runStore.addToolInvocation({ id: 'inv-1', run_id: run.id, tool_name: 'bash', status: 'pending' });
    _runStore.updateToolInvocation('inv-1', { status: 'completed', result: { output: 'ok' } });
    const found = snap(_runStore).toolInvocations.find((t: ToolInvocation) => t.id === 'inv-1');
    expect(found?.status).toBe('completed');
    expect(found?.result).toEqual({ output: 'ok' });
  });

  it('preserves unchanged fields when updating', () => {
    const run = _runStore.createRun('thread-o');
    _runStore.addToolInvocation({
      id: 'inv-2', run_id: run.id, tool_name: 'read_file',
      tool_display_name: 'Read File', status: 'pending', order: 1,
    });
    _runStore.updateToolInvocation('inv-2', { status: 'completed' });
    const found = snap(_runStore).toolInvocations.find((t: ToolInvocation) => t.id === 'inv-2');
    expect(found?.tool_name).toBe('read_file');
    expect(found?.tool_display_name).toBe('Read File');
    expect(found?.order).toBe(1);
    expect(found?.status).toBe('completed');
  });

  it('does nothing if tool invocation id does not exist', () => {
    const run = _runStore.createRun('thread-p');
    _runStore.addToolInvocation({ id: 'inv-3', run_id: run.id, tool_name: 'bash', status: 'pending' });
    expect(snap(_runStore).toolInvocations).toHaveLength(1);
    _runStore.updateToolInvocation('nonexistent', { status: 'completed' });
    expect(snap(_runStore).toolInvocations).toHaveLength(1);
    expect(snap(_runStore).toolInvocations[0].status).toBe('pending');
  });
});

// ── clearToolInvocationsForRun ──────────────────────────────
describe('runStore — clearToolInvocationsForRun()', () => {
  it('removes all tool invocations for a given run', () => {
    const run1 = _runStore.createRun('thread-q');
    const run2 = _runStore.createRun('thread-r');
    _runStore.addToolInvocation({ run_id: run1.id, tool_name: 'tool-A', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run1.id, tool_name: 'tool-B', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run2.id, tool_name: 'tool-C', status: 'pending' });
    expect(snap(_runStore).toolInvocations).toHaveLength(3);

    _runStore.clearToolInvocationsForRun(run1.id);
    expect(snap(_runStore).toolInvocations).toHaveLength(1);
    expect(snap(_runStore).toolInvocations[0].tool_name).toBe('tool-C');
  });

  it('does nothing if no matching runId', () => {
    const run = _runStore.createRun('thread-s');
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'bash', status: 'pending' });
    expect(snap(_runStore).toolInvocations).toHaveLength(1);
    _runStore.clearToolInvocationsForRun('nonexistent-id');
    expect(snap(_runStore).toolInvocations).toHaveLength(1);
  });

  it('clears all tools for a run that has multiple', () => {
    const run = _runStore.createRun('thread-t');
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 't1', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 't2', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 't3', status: 'pending' });
    expect(snap(_runStore).toolInvocations).toHaveLength(3);
    _runStore.clearToolInvocationsForRun(run.id);
    expect(snap(_runStore).toolInvocations).toHaveLength(0);
  });
});

// ── integration: toolInvocations count ─────────────────────
describe('runStore — toolInvocations count', () => {
  it('reflects add operations', () => {
    const run = _runStore.createRun('thread-u');
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'a', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'b', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'c', status: 'pending' });
    expect(snap(_runStore).toolInvocations).toHaveLength(3);
  });

  it('reflects clear operations', () => {
    const run = _runStore.createRun('thread-v');
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'x', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run.id, tool_name: 'y', status: 'pending' });
    _runStore.clearToolInvocationsForRun(run.id);
    expect(snap(_runStore).toolInvocations).toHaveLength(0);
  });

  it('reflects mixed add/clear operations', () => {
    const run1 = _runStore.createRun('thread-w1');
    const run2 = _runStore.createRun('thread-w2');
    _runStore.addToolInvocation({ run_id: run1.id, tool_name: 'a', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run2.id, tool_name: 'b', status: 'pending' });
    _runStore.addToolInvocation({ run_id: run1.id, tool_name: 'c', status: 'pending' });
    expect(snap(_runStore).toolInvocations).toHaveLength(3);
    _runStore.clearToolInvocationsForRun(run1.id);
    expect(snap(_runStore).toolInvocations).toHaveLength(1);
    _runStore.clearToolInvocationsForRun(run2.id);
    expect(snap(_runStore).toolInvocations).toHaveLength(0);
  });
});
