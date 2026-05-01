<!-- ============================================================
⚠️  此文件由 spec-to-code 自动生成
来自: specs
生成时间: 2026-04-23
⚠️  不要直接编辑此文件 — 改 *.svelte
============================================================ -->

<script lang="ts">
  import { threadStore, currentThread, threadCount } from '$lib/stores/thread-store';
  // ThreadList 骨架 — Generated from thread-manager_uiux.yaml
  interface Props {
    onNewThread?: () => void;
  }
  let { onNewThread }: Props = $props();
  let threads = $state($threadStore.threads);
  let current = $state($currentThread);
  let count   = $state($threadCount);

  $effect(() => {
    const unsub = threadStore.subscribe(s => { threads = s.threads; });
    return unsub;
  });
</script>

<div class="thread-list">
  <div class="header">
    <span>线程 ({count})</span>
    <button onclick={() => onNewThread?.()}>+ 新建</button>
  </div>
  <div class="items">
    {#each threads as thread (thread.id)}
      <div
        class="thread-item"
        class:active={current?.id === thread.id}
        onclick={() => threadStore.setCurrentThread(thread.id)}
        onkeydown={(e) => e.key === 'Enter' && threadStore.setCurrentThread(thread.id)}
        role="button"
        tabindex="0"
      >
        <span class="name">{thread.title ?? thread.goal?.slice(0, 20) ?? '新线程'}</span>
        <span class="meta">{thread.status ?? 'draft'}</span>
      </div>
    {/each}
    {#if threads.length === 0}
      <p class="empty">暂无线程，点击「+ 新建」创建</p>
    {/if}
  </div>
</div>

<style>
  .thread-list  { height: 100%; display: flex; flex-direction: column; background: var(--wb-bg-base, #0b0c10); }
  .header       { display: flex; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid var(--wb-border, #303746); color: var(--wb-text, #eef0f5); font-size: 13px; }
  .header button { background: var(--wb-accent, #72d6d0); border: none; color: #071513; padding: 4px 10px; border-radius: 999px; cursor: pointer; font-size: 12px; font-weight: 800; }
  .items        { flex: 1; overflow-y: auto; }
  .thread-item  { padding: 10px 16px; cursor: pointer; border-bottom: 1px solid var(--wb-border, #303746); display: flex; justify-content: space-between; font-size: 13px; }
  .thread-item:hover   { background: var(--wb-bg-panel-2, #1c202a); }
  .thread-item.active  { background: rgba(122, 162, 255, .13); border-left: 3px solid var(--wb-brand, #7aa2ff); }
  .name         { color: var(--wb-text, #eef0f5); }
  .meta         { color: var(--wb-muted, #6f7888); font-size: 11px; }
  .empty        { color: var(--wb-muted, #6f7888); font-size: 12px; padding: 16px; text-align: center; }
</style>
