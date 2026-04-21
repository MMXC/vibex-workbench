<!-- ============================================================
VibeX Workbench — ThreadList 组件
开发者维护，gen.py 永不覆盖

四态 UI: 骨架屏 / 空态 / 正常 / 错误重试
============================================================ -->

<script lang="ts">
  import { threadStore, currentThread, threadCount, type Thread } from '$lib/stores/thread-store';

  let threads = $state<Thread[]>([]);
  let current = $state<Thread | null>(null);
  let count = $state(0);
  let loading = $state(false);
  let error = $state<string | null>(null);

  $effect(() => {
    const unsub = threadStore.subscribe(s => {
      threads = s.threads;
      current = s.currentThreadId ? threads.find(t => t.id === s.currentThreadId) ?? null : null;
      count = s.threads.length;
      loading = s.loading;
      error = s.error;
    });
    return unsub;
  });

  // 页面初始化时从 IndexedDB 加载
  threadStore.loadFromDB();

  function newThread() {
    const t: Thread = {
      id: crypto.randomUUID(),
      goal: '新线程 ' + Date.now(),
      title: '新线程',
      status: 'draft',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    threadStore.addThread(t);
    threadStore.setCurrentThread(t.id);
  }

  function handleRetry() {
    threadStore.setError(null);
    threadStore.loadFromDB();
  }
</script>

<div class="thread-list">
  <div class="header">
    <span>线程 ({count})</span>
    <button onclick={newThread}>+ 新建</button>
  </div>

  <div class="items">
    {#if loading}
      <!-- 状态1: 骨架屏 -->
      {#each { length: 4 } as _, i}
        <div class="thread-item skeleton">
          <div class="skel-name" style="width: {60 + (i * 17) % 40}%"></div>
          <div class="skel-meta"></div>
        </div>
      {/each}

    {:else if error}
      <!-- 状态2: 错误重试 -->
      <div class="error-state">
        <span class="error-icon">⚠</span>
        <p class="error-msg">{error}</p>
        <button class="retry-btn" onclick={handleRetry}>重试</button>
      </div>

    {:else if threads.length === 0}
      <!-- 状态3: 空态引导 -->
      <div class="empty-state">
        <p class="empty-msg">暂无线程</p>
        <p class="empty-hint">点击「+ 新建」创建第一个线程</p>
        <button class="new-btn" onclick={newThread}>+ 新建线程</button>
      </div>

    {:else}
      <!-- 状态4: 正常列表 -->
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
    {/if}
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

  /* 骨架屏 */
  .thread-item.skeleton { pointer-events: none; }
  .skel-name, .skel-meta {
    height: 8px;
    border-radius: 4px;
    background: linear-gradient(90deg, #2a2a2a 25%, #3a3a3a 50%, #2a2a2a 75%);
    background-size: 200% 100%;
    animation: shimmer 1.4s ease-in-out infinite;
  }
  .skel-name { margin-bottom: 6px; }
  .skel-meta { width: 30%; }
  @keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

  /* 错误态 */
  .error-state { display: flex; flex-direction: column; align-items: center; padding: 24px 16px; gap: 8px; }
  .error-icon { font-size: 24px; }
  .error-msg { color: #f87171; font-size: 12px; text-align: center; margin: 0; }
  .retry-btn { background: #374151; border: none; color: #ccc; padding: 6px 14px; border-radius: 4px; cursor: pointer; font-size: 12px; }

  /* 空态 */
  .empty-state { display: flex; flex-direction: column; align-items: center; padding: 24px 16px; gap: 6px; }
  .empty-msg { color: #555; font-size: 14px; margin: 0; }
  .empty-hint { color: #444; font-size: 11px; margin: 0; }
  .new-btn { background: #4f46e5; border: none; color: white; padding: 8px 16px; border-radius: 4px; cursor: pointer; font-size: 12px; margin-top: 8px; }
</style>