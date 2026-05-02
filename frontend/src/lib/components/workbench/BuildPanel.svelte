<!-- BuildPanel.svelte — Build output display panel
     Shows run-make results with exitCode badge and truncated stdout/stderr
     SLICE-build-panel-output-display
-->
<script lang="ts">
  import { commandOutputStore, type CommandEntry } from '$lib/stores/command-output-store';

  interface Props {
    workspaceRoot?: string;
    onRunMake?: (target: string) => void;
  }

  let { workspaceRoot = '', onRunMake }: Props = $props();

  let storeValue = $state<ReturnType<typeof commandOutputStore['subscribe']>>(null as any);
  $effect(() => {
    const unsub = commandOutputStore.subscribe(v => { storeValue = v; });
    return unsub;
  });

  let selectedTarget = $state('validate');
  const targets = ['validate', 'lint-specs', 'generate', 'build'];

  function handleRun() {
    commandOutputStore.runMake(workspaceRoot, selectedTarget);
    onRunMake?.(selectedTarget);
  }

  function truncate(s: string, max = 500) {
    if (!s) return '';
    return s.length > max ? s.slice(0, max) + '\n... [truncated]' : s;
  }

  function statusColor(status: string) {
    switch (status) {
      case 'success': return '#22c55e';
      case 'failure': return '#ef4444';
      case 'timeout': return '#f59e0b';
      case 'running': return '#a78bfa';
      default: return '#71717a';
    }
  }

  function statusLabel(status: string) {
    switch (status) {
      case 'success': return '✅ 通过';
      case 'failure': return '❌ 失败';
      case 'timeout': return '⏰ 超时';
      case 'running': return '⏳ 运行中';
      default: return '—';
    }
  }

  const activeEntry = $derived(
    storeValue?.activeId
      ? storeValue?.commands.find(c => c.id === storeValue?.activeId)
      : storeValue?.commands[storeValue.commands.length - 1]
  );
</script>

<div class="build-panel">
  <div class="build-header">
    <span class="build-title">Build</span>
    <div class="build-controls">
      <select class="build-select" bind:value={selectedTarget}>
        {#each targets as t}
          <option value={t}>{t}</option>
        {/each}
      </select>
      <button
        type="button"
        class="build-run-btn"
        onclick={handleRun}
        disabled={storeValue?.loading}
      >
        {storeValue?.loading ? '运行中…' : '▶ Run'}
      </button>
    </div>
  </div>

  {#if activeEntry}
    <div class="build-result" style="border-color: {statusColor(activeEntry.status)}44">
      <div class="result-header">
        <span
          class="status-badge"
          style="background: {statusColor(activeEntry.status)}22; color: {statusColor(activeEntry.status)}; border-color: {statusColor(activeEntry.status)}55"
        >
          {statusLabel(activeEntry.status)}
        </span>
        {#if activeEntry.duration}
          <span class="result-meta">{activeEntry.duration}</span>
        {/if}
        <span class="result-meta">{activeEntry.target}</span>
      </div>

      {#if activeEntry.stdout}
        <div class="output-section">
          <div class="output-label">stdout</div>
          <pre class="output-text">{truncate(activeEntry.stdout)}</pre>
        </div>
      {/if}

      {#if activeEntry.stderr}
        <div class="output-section">
          <div class="output-label stderr-label">stderr</div>
          <pre class="output-text stderr-text">{truncate(activeEntry.stderr)}</pre>
        </div>
      {/if}

      {#if activeEntry.status === 'timeout'}
        <div class="timeout-msg">执行超时，请检查目录</div>
      {/if}
    </div>
  {:else}
    <div class="build-empty">
      <p>选择一个 target 并点击 Run 来执行构建</p>
    </div>
  {/if}
</div>

<style>
  .build-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    font-family: ui-monospace, monospace;
    background: #0d0d0e;
  }

  .build-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #27272a;
    flex-shrink: 0;
  }

  .build-title {
    font-size: 11px;
    font-weight: 600;
    color: #a1a1aa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .build-controls {
    display: flex;
    gap: 0.4rem;
    align-items: center;
  }

  .build-select {
    font-size: 10px;
    padding: 0.2rem 0.4rem;
    background: #1a1a1a;
    border: 1px solid #3f3f46;
    color: #e4e4e7;
    border-radius: 4px;
    cursor: pointer;
  }

  .build-run-btn {
    font-size: 10px;
    padding: 0.2rem 0.6rem;
    background: #166534;
    border: 1px solid #22c55e44;
    color: #86efac;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 600;
  }
  .build-run-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .build-result {
    flex: 1;
    overflow: auto;
    padding: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .result-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .status-badge {
    font-size: 10px;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    border: 1px solid;
    font-weight: 600;
  }

  .result-meta {
    font-size: 10px;
    color: #71717a;
  }

  .output-section {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .output-label {
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #22c55e;
  }

  .stderr-label {
    color: #ef4444;
  }

  .output-text {
    font-size: 10px;
    color: #d4d4d8;
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    padding: 0.4rem;
    background: #141416;
    border-radius: 4px;
    max-height: 200px;
    overflow: auto;
  }

  .stderr-text {
    color: #fca5a5;
  }

  .timeout-msg {
    font-size: 11px;
    color: #f59e0b;
    padding: 0.4rem 0.6rem;
    background: rgba(245, 158, 11, 0.1);
    border-radius: 4px;
  }

  .build-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .build-empty p {
    font-size: 11px;
    color: #52525b;
    margin: 0;
  }
</style>