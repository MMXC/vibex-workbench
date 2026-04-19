// ============================================================
// VibeX Workbench — 主工作台页面
// 开发者维护，gen.py 永不覆盖
// ============================================================

<script lang="ts">
  import { onMount } from 'svelte';
  import { sseConsumer } from '$lib/sse';
  import { threadStore, currentThread } from '$lib/stores/thread-store';
  import { canvasStore } from '$lib/stores/canvas-store';
  import WorkbenchShell from '$lib/components/workbench/WorkbenchShell.svelte';
  import ThreadList from '$lib/components/workbench/ThreadList.svelte';
  import ArtifactPanel from '$lib/components/workbench/ArtifactPanel.svelte';
  import Composer from '$lib/components/workbench/Composer.svelte';

  const SSE_URL = import.meta.env.VITE_SSE_URL || 'http://localhost:33335';

  let prevThreadId: string | null = null;
  let canvasNodes = $state(0);

  $effect(() => {
    const unsub = canvasStore.subscribe(s => { canvasNodes = s.nodes.length; });
    return unsub;
  });

  $effect(() => {
    const tid = $currentThread?.id ?? null;

    // Thread 切换时重连 SSE
    if (tid && tid !== prevThreadId) {
      sseConsumer.disconnect(); // 切换前先断开旧连接，防止内存泄漏
      sseConsumer.connect(`${SSE_URL}/api/sse/threads/${tid}`);
      prevThreadId = tid;
    }

    // cleanup: 组件销毁或 effect 重新运行时断开
    return () => {
      sseConsumer.disconnect();
      prevThreadId = null;
    };
  });

  // ── Composer 提交触发 Run ─────────────────────────────────
  async function handleSubmit(content: string, mode: string) {
    const tid = $currentThread?.id;
    if (!tid) {
      // 无 thread 时自动创建一个
      const t = {
        id: crypto.randomUUID(),
        goal: content.slice(0, 50),
        title: content.slice(0, 20),
        createdAt: new Date().toISOString(),
      };
      threadStore.addThread(t);
      threadStore.setCurrentThread(t.id);
      // 连接 SSE
      sseConsumer.connect(`${SSE_URL}/api/sse/threads/${t.id}`);
      prevThreadId = t.id;
    }

    // POST /api/runs 触发 backend mock run
    try {
      await fetch(`${SSE_URL}/api/runs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ threadId: tid || prevThreadId, goal: content }),
      });
    } catch (e) {
      console.error('[Workbench] Failed to start run:', e);
    }
  }
</script>

<div class="workbench-root">
  <WorkbenchShell>
    {#snippet sidebar()}
      <ThreadList />
    {/snippet}

    {#snippet main()}
      <div class="canvas-area">
        <p class="placeholder">Canvas Orchestration ({canvasNodes} nodes)</p>
      </div>
    {/snippet}

    {#snippet panel()}
      <ArtifactPanel />
    {/snippet}

    {#snippet composer()}
      <Composer onsubmit={handleSubmit} />
    {/snippet}
  </WorkbenchShell>
</div>

<style>
  .workbench-root { width: 100vw; height: 100vh; overflow: hidden; }
  .canvas-area { flex: 1; overflow: hidden; }
  .placeholder { color: #555; font-size: 13px; padding: 16px; }
</style>
