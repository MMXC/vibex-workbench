<script lang="ts">
  interface Props {
    onsubmit?: (text: string) => void;
    placeholder?: string;
  }
  let { onsubmit, placeholder = '描述 bug 或 feature 需求...' }: Props = $props();
  let value = $state('');
  let submitting = $state(false);
  async function handleSubmit() {
    if (!value.trim()) return;
    submitting = true;
    await onsubmit?.(value.trim());
    value = ''; submitting = false;
  }
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSubmit();
  }
</script>

<div class="ChangeInput">
  <textarea bind:value {placeholder} onkeydown={handleKeydown} rows={3}></textarea>
  <div class="actions">
    <span class="hint">Ctrl+Enter 提交</span>
    <button onclick={handleSubmit} disabled={submitting || !value.trim()}>
      {submitting ? '⏳ 分析中...' : '🔍 分析从属'}
    </button>
  </div>
</div>

<style>
  .ChangeInput { display: flex; flex-direction: column; gap: 8px; }
  textarea { width: 100%; background: #1e1e2e; color: #e0e0e0; border: 1px solid #333; border-radius: 6px; padding: 10px; font-family: inherit; font-size: 13px; resize: vertical; box-sizing: border-box; }
  textarea:focus { outline: none; border-color: #4D96FF; }
  .actions { display: flex; justify-content: space-between; align-items: center; }
  .hint { color: #666; font-size: 11px; }
  button { background: #4D96FF; border: none; color: white; padding: 6px 16px; border-radius: 6px; cursor: pointer; font-size: 13px; }
  button:disabled { background: #333; color: #666; cursor: not-allowed; }
  button:not(:disabled):hover { background: #3a7be0; }
</style>
