// Artifact Store — 管理 Artifact 注册表
import { writable, derived } from 'svelte/store';

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
}

function createArtifactStore() {
  const { subscribe, update } = writable<ArtifactState>({
    artifacts: [],
    selected_artifact_id: null,
    search_query: '',
    filter_type: null,
    loading: false,
  });

  return {
    subscribe,
    create(artifact: Omit<Artifact, 'id' | 'created_at'>): Artifact {
      const a: Artifact = {
        ...artifact,
        id: crypto.randomUUID(),
        created_at: new Date().toISOString(),
        tags: artifact.tags ?? [],
      };
      update(s => ({ ...s, artifacts: [...s.artifacts, a] }));
      return a;
    },
    update(id: string, patch: Partial<Artifact>) {
      update(s => ({
        ...s,
        artifacts: s.artifacts.map(a => a.id === id ? { ...a, ...patch } : a),
      }));
    },
    remove(id: string) {
      update(s => ({ ...s, artifacts: s.artifacts.filter(a => a.id !== id) }));
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
