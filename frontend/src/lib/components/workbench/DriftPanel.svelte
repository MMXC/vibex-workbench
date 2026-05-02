<!-- DriftPanel.svelte — Spec drift detection display panel
     Shows drifted files from drift-state.json with Accept/Ignore actions
     SLICE-drift-panel
-->
<script lang="ts">
  export interface DriftEntry {
    spec_path: string;
    missing?: string[];
    extra?: string[];
    modified?: string[];
    detected_at: string;
    status: string;
  }

  interface Props {
    workspaceRoot?: string;
    onAccept?: (specPath: string) => void;
    onIgnore?: (specPath: string) => void;
  }

  let { workspaceRoot = '', onAccept, onIgnore }: Props = $props();

  let entries = $state<DriftEntry[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function loadDrift() {
    loading = true;
    error = null;
    try {
      const res = await fetch(`/api/spec/drift/list?workspaceRoot=${encodeURIComponent(workspaceRoot)}`);
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      entries = data.entries ?? [];
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function acceptDrift(specPath: string) {
    try {
      await fetch(`/api/spec/drift/accept`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ spec_path: specPath }),
      });
      onAccept?.(specPath);
      await loadDrift();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  }

  async function ignoreDrift(specPath: string) {
    try {
      await fetch(`/api/spec/drift/reject`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ spec_path: specPath }),
      });
      onIgnore?.(specPath);
      await loadDrift();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  }

  $effect(() => {
    loadDrift();
  });

  const pendingCount = $derived(entries.filter(e => e.status === 'pending').length);
</script>

<div class="drift-panel">
  <div class="drift-header">
    <span class="drift-title">Spec Drift</span>
    <div class="drift-actions">
      <button type="button" class="btn-refresh" onclick={loadDrift} disabled={loading}>
        {loading ? '加载中…' : '↻'}
      </button>
    </div>
  </div>

  {#if error}
    <div class="drift-error">{error}</div>
  {/if}

  {#if loading}
    <div class="drift-loading">加载中…</div>
  {:else if entries.length === 0}
    <div class="drift-empty">
      <p>无 drift 条目</p>
    </div>
  {:else}
    <div class="drift-list">
      {#each entries as entry}
        <div class="drift-item" class:accepted={entry.status === 'accepted'} class:rejected={entry.status === 'rejected'}>
          <div class="drift-item-header">
            <span class="drift-spec-path">{entry.spec_path}</span>
            <span class="drift-status" data-status={entry.status}>{entry.status}</span>
          </div>

          {#if entry.missing?.length}
            <div class="drift-section">
              <span class="drift-section-label">缺失文件</span>
              {#each entry.missing as f}
                <span class="drift-file missing">{f}</span>
              {/each}
            </div>
          {/if}

          {#if entry.extra?.length}
            <div class="drift-section">
              <span class="drift-section-label">多余文件</span>
              {#each entry.extra as f}
                <span class="drift-file extra">{f}</span>
              {/each}
            </div>
          {/if}

          {#if entry.modified?.length}
            <div class="drift-section">
              <span class="drift-section-label">已修改</span>
              {#each entry.modified as f}
                <span class="drift-file modified">{f}</span>
              {/each}
            </div>
          {/if}

          {#if entry.status === 'pending'}
            <div class="drift-item-actions">
              <button type="button" class="btn-accept" onclick={() => acceptDrift(entry.spec_path)}>
                Accept
              </button>
              <button type="button" class="btn-ignore" onclick={() => ignoreDrift(entry.spec_path)}>
                Ignore
              </button>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .drift-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    font-family: ui-monospace, monospace;
    background: #0d0d0e;
  }

  .drift-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #27272a;
    flex-shrink: 0;
  }

  .drift-title {
    font-size: 11px;
    font-weight: 600;
    color: #a1a1aa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .btn-refresh {
    font-size: 11px;
    padding: 0.2rem 0.4rem;
    background: none;
    border: none;
    color: #71717a;
    cursor: pointer;
  }
  .btn-refresh:hover { color: #e4e4e7; }
  .btn-refresh:disabled { opacity: 0.4; cursor: not-allowed; }

  .drift-error {
    padding: 0.5rem 0.75rem;
    font-size: 11px;
    color: #fca5a5;
    background: rgba(239, 68, 68, 0.1);
  }

  .drift-loading, .drift-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    color: #52525b;
  }

  .drift-list {
    flex: 1;
    overflow: auto;
    padding: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .drift-item {
    border: 1px solid #27272a;
    border-radius: 6px;
    padding: 0.6rem;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .drift-item.accepted {
    border-color: rgba(34, 197, 94, 0.3);
    background: rgba(34, 197, 94, 0.05);
  }

  .drift-item.rejected {
    border-color: rgba(239, 68, 68, 0.3);
    background: rgba(239, 68, 68, 0.05);
  }

  .drift-item-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .drift-spec-path {
    font-size: 11px;
    color: #e4e4e7;
    font-weight: 600;
  }

  .drift-status {
    font-size: 9px;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    text-transform: uppercase;
    font-weight: 600;
  }

  .drift-status[data-status="pending"] { background: rgba(245,158,11,0.2); color: #f59e0b; }
  .drift-status[data-status="accepted"] { background: rgba(34,197,94,0.2); color: #22c55e; }
  .drift-status[data-status="rejected"] { background: rgba(239,68,68,0.2); color: #ef4444; }

  .drift-section {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    align-items: center;
  }

  .drift-section-label {
    font-size: 9px;
    color: #71717a;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin-right: 0.2rem;
  }

  .drift-file {
    font-size: 10px;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    font-family: ui-monospace, monospace;
  }

  .drift-file.missing { background: rgba(239,68,68,0.15); color: #fca5a5; }
  .drift-file.extra { background: rgba(245,158,11,0.15); color: #fcd34d; }
  .drift-file.modified { background: rgba(122,162,255,0.15); color: #9fc0ff; }

  .drift-item-actions {
    display: flex;
    gap: 0.4rem;
    margin-top: 0.2rem;
  }

  .btn-accept, .btn-ignore {
    font-size: 10px;
    padding: 0.2rem 0.6rem;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 600;
  }

  .btn-accept {
    background: #166534;
    border: 1px solid #22c55e44;
    color: #86efac;
  }

  .btn-ignore {
    background: #27272a;
    border: 1px solid #3f3f46;
    color: #a1a1aa;
  }
</style>