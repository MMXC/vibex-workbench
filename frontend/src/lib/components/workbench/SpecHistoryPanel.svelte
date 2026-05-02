<!-- SpecHistoryPanel.svelte — Spec history and rollback panel
     Shows git history for spec files with rollback options
     SLICE-spec-history-rollback
-->
<script lang="ts">
  export interface HistoryEntry {
    commit: string;
    author: string;
    date: string;
    message: string;
    spec_path: string;
  }

  interface Props {
    specPath?: string;
    workspaceRoot?: string;
    onRollback?: (commit: string) => void;
  }

  let { specPath = '', workspaceRoot = '', onRollback }: Props = $props();

  let history = $state<HistoryEntry[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function loadHistory() {
    if (!specPath) return;
    loading = true;
    error = null;
    try {
      const res = await fetch(
        `/api/workspace/specs/history?specPath=${encodeURIComponent(specPath)}`
      );
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      history = data.history ?? [];
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (specPath) loadHistory();
  });

  function formatDate(d: string) {
    try {
      return new Date(d).toLocaleString('zh-CN', {
        month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit',
      });
    } catch {
      return d;
    }
  }

  function shortHash(commit: string) {
    return commit.slice(0, 7);
  }
</script>

<div class="history-panel">
  <div class="history-header">
    <span class="history-title">Spec History</span>
    {#if specPath}
      <span class="history-spec">{specPath}</span>
    {/if}
    <button type="button" class="btn-refresh" onclick={loadHistory} disabled={loading}>
      {loading ? '…' : '↻'}
    </button>
  </div>

  <div class="history-body">
    {#if loading}
      <div class="history-loading">加载中…</div>
    {:else if error}
      <div class="history-error">{error}</div>
    {:else if history.length === 0}
      <div class="history-empty">
        <p>尚无 git 历史</p>
      </div>
    {:else}
      <div class="history-list">
        {#each history as entry}
          <div class="history-entry">
            <div class="entry-head">
              <span class="entry-hash">{shortHash(entry.commit)}</span>
              <span class="entry-date">{formatDate(entry.date)}</span>
              <span class="entry-author">{entry.author}</span>
            </div>
            <div class="entry-msg">{entry.message}</div>
            <div class="entry-actions">
              <button type="button" class="btn-rollback" onclick={() => onRollback?.(entry.commit)}>
                回滚至此
              </button>
              <button type="button" class="btn-view" onclick={() => {/* TODO: view at commit */}}>
                查看
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .history-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #0d0d0e;
  }

  .history-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #27272a;
    flex-shrink: 0;
  }

  .history-title {
    font-size: 11px;
    font-weight: 600;
    color: #a1a1aa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .history-spec {
    font-size: 10px;
    color: #9fc0ff;
    font-family: ui-monospace, monospace;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .btn-refresh {
    font-size: 11px;
    background: none;
    border: none;
    color: #71717a;
    cursor: pointer;
    padding: 0 0.2rem;
  }
  .btn-refresh:hover { color: #e4e4e7; }

  .history-body {
    flex: 1;
    overflow: auto;
    padding: 0.5rem;
  }

  .history-loading, .history-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    font-size: 11px;
    color: #52525b;
  }

  .history-error {
    font-size: 11px;
    color: #fca5a5;
    padding: 0.5rem;
    background: rgba(239, 68, 68, 0.1);
    border-radius: 6px;
  }

  .history-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .history-entry {
    border: 1px solid #1f1f23;
    border-radius: 6px;
    padding: 0.6rem;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .entry-head {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .entry-hash {
    font-size: 10px;
    font-weight: 700;
    color: #9fc0ff;
    font-family: ui-monospace, monospace;
    background: rgba(122, 162, 255, 0.1);
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
  }

  .entry-date, .entry-author {
    font-size: 10px;
    color: #71717a;
  }

  .entry-msg {
    font-size: 11px;
    color: #e4e4e7;
  }

  .entry-actions {
    display: flex;
    gap: 0.4rem;
    margin-top: 0.2rem;
  }

  .btn-rollback, .btn-view {
    font-size: 10px;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 600;
  }

  .btn-rollback {
    background: #27272a;
    border: 1px solid #3f3f46;
    color: #a1a1aa;
  }

  .btn-view {
    background: none;
    border: 1px solid #27272a;
    color: #71717a;
  }
</style>