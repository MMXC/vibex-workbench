// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: /root/vibex-workbench/specs
//     生成时间: 2026-04-21
//     ⚠️  不要直接编辑此文件 — 改 *.svelte
// ============================================================

<script lang="ts">
  // Composer 骨架 — Generated from workbench-shell_uiux.yaml
  // mode: text | image | file | url
  interface Props {
    onsubmit?: (content: string, mode: string) => void;
  }
  let { onsubmit }: Props = $props();
  let content = $state('');
  let mode    = $state<'text'|'image'|'file'|'url'>('text');

  function submit() {
    if (!content.trim()) return;
    onsubmit?.(content, mode);
    content = '';
  }
</script>

<div class="composer">
  <div class="mode-tabs">
    <button class:active={mode==='text'}   onclick={() => mode='text'}>文本</button>
    <button class:active={mode==='image'}  onclick={() => mode='image'}>图片</button>
    <button class:active={mode==='file'}   onclick={() => mode='file'}>文件</button>
    <button class:active={mode==='url'}     onclick={() => mode='url'}>URL</button>
  </div>
  <textarea
    bind:value={content}
    placeholder="输入消息，或 @ 引用 Artifact..."
    rows={3}
    onkeydown={(e) => { if (e.key === 'Enter' && e.ctrlKey) submit(); }}
  ></textarea>
  <div class="actions">
    <button class="submit-btn" onclick={submit}>发送 ⌘↵</button>
  </div>
</div>

<style>
  .composer { padding: 8px 16px; background: #1a1a1a; border-top: 1px solid #333; }
  textarea  { width: 100%; background: #222; border: 1px solid #444; border-radius: 8px; color: #eee; padding: 8px; resize: none; }
  .mode-tabs { display: flex; gap: 4px; margin-bottom: 6px; }
  .mode-tabs button { background: transparent; border: none; color: #888; cursor: pointer; padding: 4px 8px; border-radius: 4px; }
  .mode-tabs button.active { background: #333; color: #fff; }
  .actions  { display: flex; justify-content: flex-end; margin-top: 6px; }
  .submit-btn { background: #4f46e5; color: white; border: none; padding: 6px 16px; border-radius: 6px; cursor: pointer; }
</style>
