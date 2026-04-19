// ============================================================
// Composer — 多模态输入框
// E3-U2: 底部 RunStatusBar 显示运行状态
// E4-U3: 拖拽 Artifact 到 Composer 注入 @artifactId
// 开发者维护，gen.py 永不覆盖
// ============================================================

<script lang="ts">
  import { runStore, activeRun } from '$lib/stores/run-store';

  interface Props {
    onsubmit?: (content: string, mode: string) => Promise<void> | void;
  }

  let { onsubmit }: Props = $props();

  let content = $state('');
  let mode = $state<'text'|'image'|'file'|'url'>('text');
  let submitting = $state(false);

  // E3-U2: RunStatusBar reactive state
  let activeRunData = $state<{ id: string; status: string; goal: string; result_summary?: string; error_message?: string } | null>(null);
  let toolCount = $state(0);
  let hideStatusBarTimer: ReturnType<typeof setTimeout> | null = null;
  let showStatus = $state(false);
  let statusMessage = $state('');
  let statusType = $state<'running'|'completed'|'failed'>('running');

  // E4-U3: Composer drag-over + drop 支持
  let dragOver = $state(false);

  function handleDragOver(e: DragEvent) {
    if (e.dataTransfer?.types.includes('text/vibex-artifact')) {
      e.preventDefault();
      e.dataTransfer!.dropEffect = 'copy';
      dragOver = true;
    }
  }

  function handleDragLeave() {
    dragOver = false;
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragOver = false;
    const artifactId = e.dataTransfer?.getData('text/vibex-artifact');
    if (artifactId) {
      // E4-U3: 注入 @artifactId 引用
      content = content ? `${content.trimEnd()} @{${artifactId}}` : `@{${artifactId}}`;
    }
  }

  $effect(() => {
    const unsub = runStore.subscribe(s => {
      const ar = s.runs.find(r => r.id === s.active_run_id);
      activeRunData = ar ?? null;
      toolCount = s.toolInvocations.length;
    });
    return unsub;
  });

  // 监听 activeRun 状态变化，显示状态条
  $effect(() => {
    const run = activeRunData;
    if (!run) {
      showStatus = false;
      return;
    }
    if (run.status === 'executing' || run.status === 'planning') {
      statusMessage = `运行中… (${toolCount} tools)`;
      statusType = 'running';
      showStatus = true;
    } else if (run.status === 'completed') {
      statusMessage = run.result_summary ?? '✅ 运行完成';
      statusType = 'completed';
      showStatus = true;
      if (hideStatusBarTimer) clearTimeout(hideStatusBarTimer);
      hideStatusBarTimer = setTimeout(() => { showStatus = false; }, 5000);
    } else if (run.status === 'failed') {
      statusMessage = `❌ 失败: ${run.error_message ?? '未知错误'}`;
      statusType = 'failed';
      showStatus = true;
    }
  });

  async function submit() {
    if (!content.trim() || submitting) return;
    submitting = true;
    try {
      await onsubmit?.(content, mode);
      content = '';
    } finally {
      submitting = false;
    }
  }
</script>

<div
  class="composer"
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
  class:drag-over={dragOver}
>
  <!-- E3-U2: RunStatusBar -->
  {#if showStatus}
    <div class="run-status-bar" class:running={statusType==='running'} class:completed={statusType==='completed'} class:failed={statusType==='failed'}>
      <span class="status-icon">
        {#if statusType === 'running'}⟳{/if}
        {#if statusType === 'completed'}✓{/if}
        {#if statusType === 'failed'}✗{/if}
      </span>
      <span class="status-msg">{statusMessage}</span>
    </div>
  {/if}

  <div class="mode-tabs">
    <button class:active={mode==='text'} onclick={() => mode='text'}>文本</button>
    <button class:active={mode==='image'} onclick={() => mode='image'}>图片</button>
    <button class:active={mode==='file'} onclick={() => mode='file'}>文件</button>
    <button class:active={mode==='url'} onclick={() => mode='url'}>URL</button>
  </div>
  <textarea
    bind:value={content}
    placeholder="输入消息，或 @ 引用 Artifact..."
    rows={3}
    onkeydown={(e) => { if (e.key === 'Enter' && e.ctrlKey) submit(); }}
  ></textarea>
  <div class="actions">
    <span class="hint">Ctrl+Enter 发送 · {toolCount} tools · 拖拽 Artifact 到此</span>
    <button class="submit-btn" onclick={submit} disabled={submitting}>
      {submitting ? '发送中…' : '发送 ⌘↵'}
    </button>
  </div>
</div>

<style>
  .composer {
    padding: 8px 16px;
    background: #1a1a1a;
    border-top: 1px solid #333;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  textarea { background: #222; border: 1px solid #444; border-radius: 8px; color: #eee; padding: 8px; resize: none; box-sizing: border-box; }
  .mode-tabs { display: flex; gap: 4px; }
  .mode-tabs button { background: transparent; border: none; color: #888; cursor: pointer; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
  .mode-tabs button.active { background: #333; color: #fff; }
  .actions { display: flex; justify-content: space-between; align-items: center; }
  .hint { color: #555; font-size: 11px; }
  .submit-btn { background: #4f46e5; color: white; border: none; padding: 6px 16px; border-radius: 6px; cursor: pointer; font-size: 13px; }
  .submit-btn:disabled { opacity: 0.6; cursor: not-allowed; }

  /* E4-U3: drag-over highlight */
  .composer.drag-over { border-color: #4f46e5; background: #1e1a2e; }

  /* E3-U2: RunStatusBar styles */
  .run-status-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 12px;
    animation: slideIn 0.2s ease-out;
  }
  @keyframes slideIn {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .run-status-bar.running { background: #1e3a5f; color: #60a5fa; border: 1px solid #2563eb; }
  .run-status-bar.completed { background: #1a3a2a; color: #4ade80; border: 1px solid #22c55e; }
  .run-status-bar.failed { background: #3a1a1a; color: #f87171; border: 1px solid #ef4444; }
  .status-icon { font-size: 14px; }
  .status-msg { flex: 1; }
  .run-status-bar.running .status-icon { animation: spin 1s linear infinite; }
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>