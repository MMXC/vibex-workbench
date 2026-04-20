// Artifact Store — 管理 Artifact 注册表
import { writable, derived } from 'svelte/store';
import { db } from '$lib/db';

export interface Artifact {
  id: string;
  thread_id?: string;
  run_id?: string;
  type: string;
  name: string;
  content: string;
  language?: string;
  version?: number;
  mime_type?: string;
  size_bytes?: number;
  tags: string[];
  status?: string;
  deleted_at?: string | null;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at?: string;
  last_accessed_at?: string;
}

export interface ArtifactState {
  artifacts: Artifact[];
  selected_artifact_id: string | null;
  search_query: string;
  filter_type: string | null;
  loading: boolean;
  error: string | null;
}

function createArtifactStore() {
  const { subscribe, update } = writable<ArtifactState>({
    artifacts: [],
    selected_artifact_id: null,
    search_query: '',
    filter_type: null,
    loading: false,
    error: null,
  });

  return {
    subscribe,

    // E4-U1: 从 IndexedDB 加载所有 Artifact
    async loadFromDB() {
      update(s => ({ ...s, loading: true, error: null }));
      try {
        const rows = await db.artifacts.toArray();
        const artifacts: Artifact[] = rows.map(r => ({
          id: r.id,
          name: r.name,
          type: r.type,
          content: r.content,
          mime_type: r.mime_type,
          tags: r.tags,
          thread_id: r.thread_id,
          run_id: r.run_id,
          created_at: r.created_at,
          updated_at: r.updated_at,
        }));
        update(s => ({ ...s, artifacts, loading: false }));
      } catch (e) {
        update(s => ({
          ...s,
          loading: false,
          error: `无法加载 Artifact: ${e instanceof Error ? e.message : String(e)}`,
        }));
      }
    },

    create(artifact: Omit<Artifact, 'id' | 'created_at'>): Artifact {
      const a: Artifact = {
        ...artifact,
        id: crypto.randomUUID(),
        created_at: new Date().toISOString(),
        tags: artifact.tags ?? [],
      };
      update(s => ({ ...s, artifacts: [...s.artifacts, a] }));
      // E4-U1: 持久化到 IndexedDB
      db.artifacts.put({
        id: a.id,
        name: a.name,
        type: a.type,
        content: a.content,
        mime_type: a.mime_type ?? 'text/plain',
        tags: a.tags,
        thread_id: a.thread_id,
        run_id: a.run_id,
        created_at: a.created_at,
        updated_at: a.updated_at,
        is_deleted: 0,
      }).catch(e => console.error('[artifactStore] Failed to persist:', e));
      return a;
    },

    update(id: string, patch: Partial<Artifact>) {
      update(s => ({
        ...s,
        artifacts: s.artifacts.map(a => a.id === id ? { ...a, ...patch } : a),
      }));
      db.artifacts.update(id, {
        ...patch,
        updated_at: new Date().toISOString(),
      }).catch(e => console.error('[artifactStore] Failed to update:', e));
    },

    remove(id: string) {
      update(s => ({ ...s, artifacts: s.artifacts.filter(a => a.id !== id) }));
      db.artifacts.delete(id).catch(e => console.error('[artifactStore] Failed to delete:', e));
    },

    select(id: string | null) {
      update(s => ({ ...s, selected_artifact_id: id }));
    },

    setSearch(q: string) {
      update(s => ({ ...s, search_query: q }));
    },

    setFilter(type: string | null) {
      update(s => ({ ...s, filter_type: type }));
    },
  };
}

export const artifactStore = createArtifactStore();

export const filteredArtifacts = derived(artifactStore, $s => {
  let items = $s.artifacts;
  if ($s.filter_type) items = items.filter(a => a.type === $s.filter_type);
  if ($s.search_query) items = items.filter(a =>
    a.name.toLowerCase().includes($s.search_query.toLowerCase()) ||
    a.content.toLowerCase().includes($s.search_query.toLowerCase())
  );
  return items;
});

export const selectedArtifact = derived(artifactStore, $s =>
  $s.artifacts.find(a => a.id === $s.selected_artifact_id) ?? null
);