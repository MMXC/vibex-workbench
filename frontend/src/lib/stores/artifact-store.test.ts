/**
 * Unit tests for artifact-store.ts
 * Strategy: replace the $lib/db module exports via vi.mock, expose via getter.
 * Uses mockImplementation per test to avoid queue-stomping between tests.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// ── Mock data ──────────────────────────────────────────────────
const THREE_ARTIFACTS = [
  {
    id: 'a1', name: 'Hello.py', type: 'code', content: 'print("hello")',
    mime_type: 'text/plain', tags: ['python'], thread_id: 't1', run_id: 'r1',
    created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-02T00:00:00Z',
  },
  {
    id: 'a2', name: 'diagram.png', type: 'image', content: 'base64img',
    mime_type: 'image/png', tags: ['diagram'], thread_id: 't1', run_id: 'r2',
    created_at: '2024-01-03T00:00:00Z',
  },
  {
    id: 'a3', name: 'util.js', type: 'code', content: 'const x = 1',
    mime_type: 'text/plain', tags: [], thread_id: 't1', run_id: 'r3',
    created_at: '2024-01-04T00:00:00Z',
  },
];

const TWO_ARTIFACTS = [
  {
    id: 'a1', name: 'One.py', type: 'code', content: 'one',
    mime_type: 'text/plain', tags: [], created_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'a2', name: 'Two.png', type: 'image', content: 'two',
    mime_type: 'image/png', tags: [], created_at: '2024-01-02T00:00:00Z',
  },
];

// ── Mock Dexie via vi.hoisted ──────────────────────────────────
const { toArray, put, update, delete: dbDelete } = vi.hoisted(() => {
  // Default impl returns [] so unmocked calls don't throw
  const impl = vi.fn<() => Promise<unknown[]>>(() => Promise.resolve([]));
  return {
    toArray: impl,
    put: vi.fn(() => Promise.resolve()),
    update: vi.fn(() => Promise.resolve()),
    delete: vi.fn(() => Promise.resolve()),
  };
});

vi.mock('$lib/db', () => ({
  db: { artifacts: { toArray, put, update, delete: dbDelete } },
}));

import { artifactStore, filteredArtifacts, selectedArtifact } from './artifact-store';

// ── Helpers ──────────────────────────────────────────────────
function snap() {
  let s: import('./artifact-store').ArtifactState = {
    artifacts: [],
    selected_artifact_id: null,
    search_query: '',
    filter_type: null,
    loading: false,
    error: null,
  };
  const u = artifactStore.subscribe(x => { s = x; });
  u();
  return s;
}

function readDerived<T>(derived: import('svelte/store').Readable<T>): T {
  return get(derived);
}

async function loadStore(data: unknown[]) {
  toArray.mockImplementation(() => Promise.resolve(data));
  await artifactStore.loadFromDB();
}

// ── Tests ────────────────────────────────────────────────────
describe('artifactStore', () => {

  // ── loadFromDB ────────────────────────────────────────────────
  describe('loadFromDB()', () => {
    it('loads artifacts from IndexedDB', async () => {
      await loadStore([THREE_ARTIFACTS[0]]);
      expect(snap().artifacts).toHaveLength(1);
      expect(snap().artifacts[0].id).toBe('a1');
      expect(snap().artifacts[0].name).toBe('Hello.py');
    });

    it('sets loading=true during fetch, false after', async () => {
      toArray.mockImplementationOnce(() => new Promise(r => setTimeout(r, 10)).then(() => []));
      const p = artifactStore.loadFromDB();
      expect(snap().loading).toBe(true);
      await p;
      expect(snap().loading).toBe(false);
    });

    it('sets error string on DB failure', async () => {
      toArray.mockImplementationOnce(() => Promise.reject(new Error('boom')));
      await artifactStore.loadFromDB();
      expect(snap().error).toBe('无法加载 Artifact: boom');
      expect(snap().loading).toBe(false);
    });

    it('clears error on successful retry', async () => {
      toArray.mockImplementationOnce(() => Promise.reject(new Error('boom')));
      await artifactStore.loadFromDB();
      expect(snap().error).toBeTruthy();

      toArray.mockImplementationOnce(() => Promise.resolve([]));
      await artifactStore.loadFromDB();
      expect(snap().error).toBeNull();
    });
  });

  // ── create ────────────────────────────────────────────────────
  describe('create()', () => {
    beforeEach(async () => { await loadStore([]); });

    it('adds artifact to store array', async () => {
      put.mockClear();
      const a = artifactStore.create({
        name: 'New.py', type: 'code', content: 'x',
        mime_type: 'text/plain', tags: [],
      });
      expect(snap().artifacts).toHaveLength(1);
      expect(snap().artifacts[0].name).toBe('New.py');
      expect(a.id).toBeTruthy();
      expect(a.created_at).toBeTruthy();
    });

    it('returns artifact with generated id and created_at', () => {
      const a = artifactStore.create({
        name: 'X.ts', type: 'code', content: 'const x = 1',
        mime_type: 'text/plain', tags: ['ts'],
      });
      expect(typeof a.id).toBe('string');
      expect(a.id.length).toBeGreaterThan(0);
      expect(a.created_at).toMatch(/^\d{4}-\d{2}-\d{2}T/);
      expect(a.tags).toEqual(['ts']);
    });

    it('calls db.artifacts.put for persistence', () => {
      put.mockClear();
      artifactStore.create({
        name: 'Persist.kt', type: 'code', content: 'val x',
        mime_type: 'text/plain', tags: [],
      });
      expect(put).toHaveBeenCalledTimes(1);
      expect(put).toHaveBeenCalledWith(expect.objectContaining({
        name: 'Persist.kt', type: 'code', mime_type: 'text/plain',
      }));
    });

    it('defaults tags to [] when undefined', () => {
      const a = artifactStore.create({
        name: 'NoTags.txt', type: 'code', content: 'no tags',
        mime_type: 'text/plain', tags: undefined as unknown as string[],
      } as Parameters<typeof artifactStore.create>[0]);
      expect(a.tags).toEqual([]);
    });
  });

  // ── update ────────────────────────────────────────────────────
  describe('update()', () => {
    beforeEach(async () => { await loadStore([...TWO_ARTIFACTS]); });

    it('updates artifact in store', () => {
      update.mockClear();
      artifactStore.update('a1', { name: 'Renamed.py', content: 'updated' });
      const found = snap().artifacts.find(a => a.id === 'a1');
      expect(found?.name).toBe('Renamed.py');
      expect(found?.content).toBe('updated');
    });

    it('calls db.artifacts.update for persistence', () => {
      update.mockClear();
      artifactStore.update('a1', { name: 'Patch.kt' });
      expect(update).toHaveBeenCalledTimes(1);
      expect(update).toHaveBeenCalledWith('a1', expect.objectContaining({
        name: 'Patch.kt', updated_at: expect.any(String),
      }));
    });

    it('leaves other artifacts unchanged', () => {
      artifactStore.update('a1', { name: 'Patched' });
      const unchanged = snap().artifacts.find(a => a.id === 'a2');
      expect(unchanged?.name).toBe('Two.png');
    });
  });

  // ── remove ────────────────────────────────────────────────────
  describe('remove()', () => {
    beforeEach(async () => { await loadStore([...TWO_ARTIFACTS]); });

    it('removes artifact from store', () => {
      dbDelete.mockClear();
      artifactStore.remove('a1');
      expect(snap().artifacts.find(a => a.id === 'a1')).toBeUndefined();
      expect(snap().artifacts).toHaveLength(1);
      expect(snap().artifacts[0].id).toBe('a2');
    });

    it('calls db.artifacts.delete for persistence', () => {
      dbDelete.mockClear();
      artifactStore.remove('a1');
      expect(dbDelete).toHaveBeenCalledTimes(1);
      expect(dbDelete).toHaveBeenCalledWith('a1');
    });

    it('handles remove of non-existent id gracefully', () => {
      expect(() => artifactStore.remove('nonexistent')).not.toThrow();
    });
  });

  // ── select ────────────────────────────────────────────────────
  describe('select()', () => {
    it('sets selected_artifact_id', () => {
      artifactStore.select('a1');
      expect(snap().selected_artifact_id).toBe('a1');
    });

    it('can deselect by passing null', () => {
      artifactStore.select('a1');
      artifactStore.select(null);
      expect(snap().selected_artifact_id).toBeNull();
    });
  });

  // ── setSearch ────────────────────────────────────────────────
  describe('setSearch()', () => {
    it('sets search_query', () => {
      artifactStore.setSearch('hello');
      expect(snap().search_query).toBe('hello');
    });

    it('supports clearing search', () => {
      artifactStore.setSearch('hello');
      artifactStore.setSearch('');
      expect(snap().search_query).toBe('');
    });
  });

  // ── setFilter ────────────────────────────────────────────────
  describe('setFilter()', () => {
    it('sets filter_type to code', () => {
      artifactStore.setFilter('code');
      expect(snap().filter_type).toBe('code');
    });

    it('sets filter_type to image', () => {
      artifactStore.setFilter('image');
      expect(snap().filter_type).toBe('image');
    });

    it('can clear filter by passing null', () => {
      artifactStore.setFilter('code');
      artifactStore.setFilter(null);
      expect(snap().filter_type).toBeNull();
    });
  });

  // ── filteredArtifacts derived ─────────────────────────────────
  describe('filteredArtifacts derived', () => {
    beforeEach(async () => {
      // Load 3 artifacts fresh for each test
      await loadStore([...THREE_ARTIFACTS]);
      // Reset search/filter from previous test
      artifactStore.setSearch('');
      artifactStore.setFilter(null);
    });

    it('returns all artifacts when no filter set', () => {
      const items = readDerived(filteredArtifacts);
      expect(items).toHaveLength(3);
    });

    it('filters by type=code', () => {
      artifactStore.setFilter('code');
      const items = readDerived(filteredArtifacts);
      expect(items).toHaveLength(2);
      expect(items.every(a => a.type === 'code')).toBe(true);
    });

    it('filters by type=image', () => {
      artifactStore.setFilter('image');
      const items = readDerived(filteredArtifacts);
      expect(items).toHaveLength(1);
      expect(items[0].type).toBe('image');
    });

    it('filters by search_query on name', () => {
      artifactStore.setSearch('Hello');
      const items = readDerived(filteredArtifacts);
      expect(items).toHaveLength(1);
      expect(items[0].name).toBe('Hello.py');
    });

    it('filters by search_query on content', () => {
      artifactStore.setSearch('print');
      const items = readDerived(filteredArtifacts);
      expect(items).toHaveLength(1);
      expect(items[0].name).toBe('Hello.py');
    });

    it('search is case-insensitive', () => {
      artifactStore.setSearch('HELLO');
      const items = readDerived(filteredArtifacts);
      expect(items).toHaveLength(1);
      expect(items[0].name).toBe('Hello.py');
    });

    it('combines filter_type and search_query', () => {
      artifactStore.setFilter('code');
      artifactStore.setSearch('util');
      const items = readDerived(filteredArtifacts);
      expect(items).toHaveLength(1);
      expect(items[0].name).toBe('util.js');
      expect(items[0].type).toBe('code');
    });

    it('returns empty array when nothing matches', () => {
      artifactStore.setSearch('nonexistent12345');
      const items = readDerived(filteredArtifacts);
      expect(items).toHaveLength(0);
    });
  });

  // ── selectedArtifact derived ───────────────────────────────────
  describe('selectedArtifact derived', () => {
    beforeEach(async () => {
      await loadStore([{ id: 'a1', name: 'Selected', type: 'code', content: 'x',
        mime_type: 'text/plain', tags: [], created_at: '2024-01-01T00:00:00Z' }]);
      artifactStore.select(null);
    });

    it('returns selected artifact', () => {
      artifactStore.select('a1');
      const found = readDerived(selectedArtifact);
      expect(found?.id).toBe('a1');
      expect(found?.name).toBe('Selected');
    });

    it('returns null when nothing selected', () => {
      const found = readDerived(selectedArtifact);
      expect(found).toBeNull();
    });

    it('returns null when selected id not found', () => {
      artifactStore.select('nonexistent');
      const found = readDerived(selectedArtifact);
      expect(found).toBeNull();
    });
  });
});
