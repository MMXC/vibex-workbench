// ============================================================
// Composer — 多模态输入框
// 开发者维护，gen.py 永不覆盖
// ============================================================

<script lang="ts">
  interface Props {
    onsubmit?: (content: string, mode: string) => Promise<void> | void;
  }

  let { onsubmit }: Props = $props();

  let content = $state('');
  let mode = $state<'text'|'image'|'file'|'url'>('text');
  let submitting = $state(false);

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

<div class="composer">
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
    <span class="hint">Ctrl+Enter 发送</span>
    <button class="submit-btn" onclick={submit} disabled={submitting}>
      {submitting ? '发送中…' : '发送 ⌘↵'}
    </button>
  </div>
</div>

<style>
  .composer { padding: 8px 16px; background: #1a1a1a; border-top: 1px solid #333; }
  textarea { width: 100%; background: #222; border: 1px solid #444; border-radius: 8px; color: #eee; padding: 8px; resize: none; box-sizing: border-box; }
  .mode-tabs { display: flex; gap: 4px; margin-bottom: 6px; }
  .mode-tabs button { background: transparent; border: none; color: #888; cursor: pointer; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
  .mode-tabs button.active { background: #333; color: #fff; }
  .actions { display: flex; justify-content: space-between; align-items: center; margin-top: 6px; }
  .hint { color: #555; font-size: 11px; }
  .submit-btn { background: #4f46e5; color: white; border: none; padding: 6px 16px; border-radius: 6px; cursor: pointer; font-size: 13px; }
  .submit-btn:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
