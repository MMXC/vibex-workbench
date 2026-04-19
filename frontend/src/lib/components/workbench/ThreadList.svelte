// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: /root/vibex-workbench/specs
//     生成时间: 2026-04-19
//     ⚠️  不要直接编辑此文件
// ============================================================

<script lang="ts">
  import { threadStore, currentThread, threadCount, type Thread } from '$lib/stores/thread-store';
  let threads = $state($threadStore.threads);
  let current = $state($currentThread);
  let count = $state($threadCount);

  $effect(() => {
    const unsub = threadStore.subscribe(s => { threads = s.threads; });
    return unsub;
  });

  function newThread() {
    const t: Thread = {
      id: crypto.randomUUID(),
      goal: '新线程 ' + Date.now(),
      title: '新线程',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    threadStore.addThread(t);
    threadStore.setCurrentThread(t.id);
  }
</script>

<div class="thread-list">
  <div class="header">
    <span>线程 ({count})</span>
    <button onclick={newThread}>+ 新建</button>
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
        <span class="name">{thread.title ?? thread.goal.slice(0, 20)}</span>
        <span class="meta">{thread.status ?? 'draft'}</span>
      </div>
    {/each}
  </div>
</div>

<style>
  .thread-list { height: 100%; display: flex; flex-direction: column; background: #111; }
  .header { display: flex; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid #222; color: #ccc; font-size: 13px; }
  .header button { background: #4f46e5; border: none; color: white; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; }
  .items { flex: 1; overflow-y: auto; }
  .thread-item { padding: 10px 16px; cursor: pointer; border-bottom: 1px solid #1a1a1a; display: flex; justify-content: space-between; font-size: 13px; }
  .thread-item:hover { background: #1a1a1a; }
  .thread-item.active { background: #1e293b; border-left: 3px solid #4f46e5; }
  .name { color: #e2e8f0; }
  .meta { color: #666; font-size: 11px; }
</style>
