<!-- ImplementationJournalPanel.svelte — Implementation journal viewer panel
     Shows checkpoints and decisions from spec implementation journals
     SLICE-implementation-journal-viewer
-->
<script lang="ts">
  interface CheckpointEntry {
    id: string;
    timestamp: string;
    agent: string;
    action: string;
    category?: string;
    files?: string[];
    result?: string;
    next_action?: string;
  }

  interface Props {
    specName?: string;
    workspaceRoot?: string;
  }

  let { specName = '', workspaceRoot = '' }: Props = $props();

  let entries = $state<CheckpointEntry[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function loadJournal() {
    if (!specName) return;
    loading = true;
    error = null;
    try {
      const safeName = specName.replace(/[^a-zA-Z0-9_-]/g, '_');
      const journalPath = `specs/journal/${safeName}.implementation.yaml`;
      const res = await fetch(`/api/workspace/specs/read?path=${encodeURIComponent(journalPath)}`);
      if (!res.ok) {
        entries = [];
        return;
      }
      const data = await res.json();
      entries = parseJournalEntries(data.content ?? '');
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  function parseJournalEntries(content: string): CheckpointEntry[] {
    const entries: CheckpointEntry[] = [];
    const lines = content.split('\n');
    let current: Partial<CheckpointEntry> = {};

    for (const line of lines) {
      if (line.startsWith('# id:')) {
        if (current.id) entries.push(current as CheckpointEntry);
        current = { id: line.slice(5).trim() };
      }
      if (line.startsWith('# timestamp:')) current.timestamp = line.slice(12).trim();
      if (line.startsWith('# agent:')) current.agent = line.slice(8).trim();
      if (line.startsWith('# action:')) current.action = line.slice(10).trim();
      if (line.startsWith('# category:')) current.category = line.slice(11).trim();
      if (line.startsWith('# files:')) current.files = line.slice(8).trim().split(',').map(s => s.trim()).filter(Boolean);
      if (line.startsWith('# result:')) current.result = line.slice(9).trim();
      if (line.startsWith('# next_action:')) current.next_action = line.slice(14).trim();
    }
    if (current.id) entries.push(current as CheckpointEntry);
    return entries;
  }

  $effect(() => {
    if (specName) loadJournal();
  });

  function formatTime(ts: string) {
    try {
      return new Date(ts).toLocaleString('zh-CN', {
        month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit',
      });
    } catch {
      return ts;
    }
  }
</script>

<div class="journal-panel">
  <div class="journal-header">
    <span class="journal-title">Implementation Journal</span>
    {#if specName}
      <span class="journal-spec">{specName}</span>
    {/if}
  </div>

  <div class="journal-body">
    {#if loading}
      <div class="journal-loading">加载中…</div>
    {:else if error}
      <div class="journal-error">{error}</div>
    {:else if entries.length === 0}
      <div class="journal-empty">
        <p>尚无 checkpoint 记录</p>
        <p class="journal-empty-hint">实现此 spec 时，checkpoint writer 会自动追加记录</p>
      </div>
    {:else}
      <div class="journal-entries">
        {#each entries as entry, i}
          <div class="journal-entry">
            <div class="entry-header">
              <span class="entry-num">#{i + 1}</span>
              <span class="entry-time">{formatTime(entry.timestamp)}</span>
              <span class="entry-agent">{entry.agent}</span>
              {#if entry.category}
                <span class="entry-category">{entry.category}</span>
              {/if}
            </div>
            <div class="entry-action">{entry.action}</div>
            {#if entry.result}
              <div class="entry-result">{entry.result}</div>
            {/if}
            {#if entry.files?.length}
              <div class="entry-files">
                {#each entry.files as f}
                  <span class="entry-file">{f}</span>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .journal-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #0d0d0e;
  }

  .journal-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #27272a;
    flex-shrink: 0;
  }

  .journal-title {
    font-size: 11px;
    font-weight: 600;
    color: #a1a1aa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .journal-spec {
    font-size: 10px;
    color: #9fc0ff;
    font-family: ui-monospace, monospace;
  }

  .journal-body {
    flex: 1;
    overflow: auto;
    padding: 0.5rem;
  }

  .journal-loading, .journal-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    font-size: 11px;
    color: #52525b;
    text-align: center;
    gap: 0.3rem;
  }

  .journal-empty-hint {
    font-size: 10px;
    color: #3f3f46;
  }

  .journal-error {
    font-size: 11px;
    color: #fca5a5;
    padding: 0.5rem;
    background: rgba(239, 68, 68, 0.1);
    border-radius: 6px;
  }

  .journal-entries {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .journal-entry {
    border: 1px solid #1f1f23;
    border-radius: 6px;
    padding: 0.6rem;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    background: #0d0d0e;
  }

  .entry-header {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    flex-wrap: wrap;
  }

  .entry-num {
    font-size: 9px;
    font-weight: 700;
    color: #52525b;
  }

  .entry-time {
    font-size: 10px;
    color: #71717a;
  }

  .entry-agent {
    font-size: 10px;
    color: #9fc0ff;
  }

  .entry-category {
    font-size: 9px;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    background: rgba(245, 158, 11, 0.15);
    color: #fcd34d;
    text-transform: uppercase;
  }

  .entry-action {
    font-size: 11px;
    color: #e4e4e7;
    font-weight: 600;
  }

  .entry-result {
    font-size: 10px;
    color: #a1a1aa;
  }

  .entry-files {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
  }

  .entry-file {
    font-size: 10px;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    background: #1a1a1a;
    color: #71717a;
    font-family: ui-monospace, monospace;
  }
</style>