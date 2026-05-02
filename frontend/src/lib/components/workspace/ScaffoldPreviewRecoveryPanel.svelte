<!-- ScaffoldPreviewRecoveryPanel.svelte — Scaffold preview and recovery panel
     Shows scaffold preview status and allows recovery from failed scaffolding
     SLICE-scaffold-preview-recovery-panel
-->
<script lang="ts">
  interface Props {
    workspaceRoot?: string;
    onRecoveryComplete?: () => void;
  }

  let { workspaceRoot = '', onRecoveryComplete }: Props = $props();

  let previewState = $state<{
    state: 'empty' | 'partial' | 'ready' | 'error';
    signals: string[];
    suggestions: string[];
    error?: string;
  } | null>(null);

  let loading = $state(false);
  let error = $state<string | null>(null);
  let recovering = $state(false);

  async function detectState() {
    loading = true;
    error = null;
    try {
      const res = await fetch(
        `/api/workspace/detect-state?workspaceRoot=${encodeURIComponent(workspaceRoot)}`,
        { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' }
      );
      const data = await res.json();
      previewState = data;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function recover() {
    recovering = true;
    error = null;
    try {
      const res = await fetch('/api/workspace/scaffold', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          workspace_root: workspaceRoot,
          confirm: true,
          mode: 'partial',
        }),
      });
      const data = await res.json();
      if (data.ok) {
        await detectState();
        onRecoveryComplete?.();
      } else {
        error = data.error ?? '恢复失败';
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      recovering = false;
    }
  }

  const stateColor: Record<string, string> = {
    empty: '#f09a6a',
    partial: '#efc66b',
    ready: '#87cf8a',
    error: '#ef4444',
  };

  const stateLabel: Record<string, string> = {
    empty: '空仓库',
    partial: '半成品',
    ready: '就绪',
    error: '错误',
  };

  $effect(() => {
    if (workspaceRoot) detectState();
  });
</script>

<div class="recovery-panel">
  <div class="recovery-header">
    <span class="recovery-title">Scaffold Preview</span>
    <button type="button" class="btn-refresh" onclick={detectState} disabled={loading}>
      {loading ? '…' : '↻'}
    </button>
  </div>

  {#if error}
    <div class="recovery-error">{error}</div>
  {/if}

  {#if previewState}
    <div class="state-display">
      <span
        class="state-badge"
        style="background: {stateColor[previewState.state]}22; color: {stateColor[previewState.state]}; border-color: {stateColor[previewState.state]}55"
      >
        {stateLabel[previewState.state] ?? previewState.state}
      </span>
    </div>

    {#if previewState.signals?.length}
      <div class="signals-section">
        <span class="section-label">检测信号</span>
        {#each previewState.signals as signal}
          <span class="signal-tag">{signal}</span>
        {/each}
      </div>
    {/if}

    {#if previewState.suggestions?.length}
      <div class="suggestions-section">
        <span class="section-label">建议</span>
        {#each previewState.suggestions as s}
          <div class="suggestion-item">{s}</div>
        {/each}
      </div>
    {/if}

    {#if previewState.state === 'partial' || previewState.state === 'empty'}
      <div class="recovery-actions">
        <button
          type="button"
          class="btn-recover"
          onclick={recover}
          disabled={recovering}
        >
          {recovering ? '恢复中…' : '恢复 Scaffold'}
        </button>
      </div>
    {/if}
  {:else if loading}
    <div class="recovery-loading">检测中…</div>
  {:else}
    <div class="recovery-empty">
      <p>选择一个 workspace 开始预览</p>
    </div>
  {/if}
</div>

<style>
  .recovery-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #0d0d0e;
  }

  .recovery-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #27272a;
    flex-shrink: 0;
  }

  .recovery-title {
    font-size: 11px;
    font-weight: 600;
    color: #a1a1aa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
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

  .recovery-error {
    margin: 0.5rem;
    padding: 0.5rem;
    font-size: 11px;
    color: #fca5a5;
    background: rgba(239, 68, 68, 0.1);
    border-radius: 6px;
  }

  .state-display {
    padding: 0.75rem;
    display: flex;
    align-items: center;
  }

  .state-badge {
    font-size: 12px;
    padding: 0.3rem 0.8rem;
    border-radius: 6px;
    border: 1px solid;
    font-weight: 600;
  }

  .signals-section, .suggestions-section {
    padding: 0.5rem 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .section-label {
    font-size: 9px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #52525b;
  }

  .signal-tag {
    font-size: 10px;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    background: #1a1a1a;
    color: #a1a1aa;
    display: inline-block;
    width: fit-content;
  }

  .suggestion-item {
    font-size: 11px;
    color: #e4e4e7;
    line-height: 1.4;
  }

  .recovery-actions {
    padding: 0.5rem 0.75rem;
  }

  .btn-recover {
    width: 100%;
    padding: 0.5rem;
    background: #166534;
    border: 1px solid #22c55e44;
    color: #86efac;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }
  .btn-recover:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .recovery-loading, .recovery-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    color: #52525b;
  }
</style>