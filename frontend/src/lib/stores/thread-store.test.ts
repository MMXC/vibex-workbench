/**
 * Unit tests for thread-store.ts
 * Strategy: directly replace the db module exports via vi.mock, expose via getter
 */
import { describe, it, expect, vi } from 'vitest';

// ── Mock Dexie via vi.mock ─────────────────────────────────────
// We define the mocks in vi.hoisted so they can be shared with the test
const { toArray, put, update } = vi.hoisted(() => {
  let storage: Record<string, unknown>[] = [
    { id: 't1', title: 'Thread One', goal: 'Goal one', status: 'draft', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' },
    { id: 't2', title: 'Thread Two', goal: 'Goal two', status: 'active', createdAt: '2024-01-02T00:00:00Z', updatedAt: '2024-01-02T00:00:00Z' },
  ];
  return {
    toArray: vi.fn(() => Promise.resolve([...storage])),
    put: vi.fn((item: Record<string, unknown>) => { storage.push(item); return Promise.resolve(); }),
    update: vi.fn((id: string, patch: Record<string, unknown>) => {
      const idx = storage.findIndex(d => d['id'] === id);
      if (idx !== -1) storage[idx] = { ...storage[idx], ...patch };
      return Promise.resolve();
    }),
  };
});

vi.mock('$lib/db', () => ({
  db: {
    threads: {
      toArray, // same reference thread-store.ts will import
      put,
      update,
    },
  },
}));

// Must import AFTER vi.mock
import { threadStore, threadCount } from './thread-store';

// ── Helpers ──────────────────────────────────────────────────
function snap() {
  let s = {
    threads: [] as unknown[],
    loading: false,
    error: null as string | null,
    messagesByThread: {} as Record<string, unknown[]>,
  };
  const u = threadStore.subscribe(x => { s = x; });
  u();
  return s;
}

// ── Tests ────────────────────────────────────────────────────
describe('threadStore', () => {

  describe('loadFromDB()', () => {
    it('loads threads, filters soft-deleted rows', async () => {
      toArray.mockResolvedValueOnce([
        { id: 't1', title: 'A', goal: '', status: 'draft', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' },
        { id: 't2', title: 'B', goal: '', status: 'active', createdAt: '2024-01-02T00:00:00Z', updatedAt: '2024-01-02T00:00:00Z' },
        { id: 't-deleted', title: 'Deleted', goal: '', status: 'draft', createdAt: '2024-01-03T00:00:00Z', updatedAt: '2024-01-03T00:00:00Z', deleted_at: '2024-01-03T00:00:00Z' },
      ]);
      await threadStore.loadFromDB();
      const s = snap();
      expect(s.threads).toHaveLength(2);
      expect(s.threads.map((t: any) => t.id)).toEqual(['t1', 't2']);
    });

    it('sets loading=true during fetch, false after', async () => {
      toArray.mockResolvedValueOnce([]);
      const p = threadStore.loadFromDB();
      expect(snap().loading).toBe(true);
      await p;
      expect(snap().loading).toBe(false);
    });

    it('sets error string on DB failure', async () => {
      toArray.mockRejectedValueOnce(new Error('boom'));
      await threadStore.loadFromDB();
      const s = snap();
      expect(s.error).toBe('无法加载 Thread: boom');
    });
  });

  describe('addThread()', () => {
    it('adds thread to store array', async () => {
      toArray.mockResolvedValueOnce([]);
      await threadStore.loadFromDB();
      put.mockClear();

      threadStore.addThread({ id: 'new', title: 'X', goal: 'Y', status: 'draft', createdAt: '2024-02-01T00:00:00Z', updatedAt: '2024-02-01T00:00:00Z' });
      expect((snap().threads as any[]).find((t: any) => t.id === 'new')).toBeTruthy();
    });

    it('calls db.threads.put', async () => {
      put.mockClear();
      threadStore.addThread({ id: 'p1', title: 'P', goal: 'Q', status: 'draft', createdAt: '2024-02-02T00:00:00Z', updatedAt: '2024-02-02T00:00:00Z' });
      expect(put).toHaveBeenCalledWith(expect.objectContaining({ id: 'p1', title: 'P' }));
    });
  });

  describe('updateThread()', () => {
    it('updates thread in store', async () => {
      toArray.mockResolvedValueOnce([{ id: 't1', title: 'Old', goal: '', status: 'draft', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' }]);
      await threadStore.loadFromDB();
      update.mockClear();

      threadStore.updateThread('t1', { title: 'New Title' });
      expect(((snap().threads as any[]).find((t: any) => t.id === 't1') as any).title).toBe('New Title');
    });

    it('calls db.threads.update', async () => {
      update.mockClear();
      threadStore.updateThread('t1', { title: 'Patch', status: 'active' });
      expect(update).toHaveBeenCalledWith('t1', expect.objectContaining({ title: 'Patch', status: 'active' }));
    });
  });

  describe('removeThread()', () => {
    it('removes thread from active list', async () => {
      toArray.mockResolvedValueOnce([
        { id: 't1', title: 'A', goal: '', status: 'draft', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' },
        { id: 't2', title: 'B', goal: '', status: 'active', createdAt: '2024-01-02T00:00:00Z', updatedAt: '2024-01-02T00:00:00Z' },
      ]);
      await threadStore.loadFromDB();
      update.mockClear();

      threadStore.removeThread('t1');
      expect((snap().threads as any[]).find((t: any) => t.id === 't1')).toBeUndefined();
      expect(snap().threads).toHaveLength(1);
    });

    it('persists deletedAt via db.threads.update', async () => {
      update.mockClear();
      threadStore.removeThread('t1');
      expect(update).toHaveBeenCalledWith('t1', expect.objectContaining({ deleted_at: expect.any(String) }));
    });
  });

  describe('threadCount', () => {
    it('is 2 after loading 2 threads', async () => {
      toArray.mockResolvedValueOnce([
        { id: 't1', title: 'A', goal: '', status: 'draft', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' },
        { id: 't2', title: 'B', goal: '', status: 'active', createdAt: '2024-01-02T00:00:00Z', updatedAt: '2024-01-02T00:00:00Z' },
      ]);
      await threadStore.loadFromDB();
      const c = await new Promise<number>(res => { const u = threadCount.subscribe(v => { res(v); u(); }); });
      expect(c).toBe(2);
    });

    it('is 0 when empty', async () => {
      toArray.mockResolvedValueOnce([]);
      await threadStore.loadFromDB();
      const c = await new Promise<number>(res => { const u = threadCount.subscribe(v => { res(v); u(); }); });
      expect(c).toBe(0);
    });

    it('increments after addThread', async () => {
      toArray.mockResolvedValueOnce([]);
      await threadStore.loadFromDB();

      threadStore.addThread({ id: 'x', title: 'X', goal: '', status: 'draft', createdAt: '2024-02-01T00:00:00Z', updatedAt: '2024-02-01T00:00:00Z' });

      // Read current store state synchronously
      let s: any;
      const u = threadStore.subscribe(x => { s = x; });
      u();

      expect(s.threads.length).toBe(1);
    });
  });

  describe('error / retry', () => {
    it('sets error message on load failure', async () => {
      toArray.mockRejectedValueOnce(new Error('DB fail'));
      await threadStore.loadFromDB();
      expect(snap().error).toBe('无法加载 Thread: DB fail');
    });

    it('clears error on retry (setError null + loadFromDB)', async () => {
      threadStore.setError('old');
      toArray.mockResolvedValueOnce([{ id: 't1', title: 'A', goal: '', status: 'draft', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' }]);
      threadStore.setError(null);
      await threadStore.loadFromDB();
      expect(snap().error).toBeNull();
      expect(snap().threads).toHaveLength(1);
    });
  });
});
