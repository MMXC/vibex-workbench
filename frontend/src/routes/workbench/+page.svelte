// ============================================================
// VibeX Workbench — 主工作台页面
// 开发者维护，gen.py 永不覆盖
// ============================================================

<script lang="ts">
  import { sseConsumer } from '$lib/sse';
  import { threadStore, currentThread } from '$lib/stores/thread-store';
  import WorkbenchShell from '$lib/components/workbench/WorkbenchShell.svelte';
  import ThreadList from '$lib/components/workbench/ThreadList.svelte';
  import CanvasRenderer from '$lib/components/workbench/CanvasRenderer.svelte';
  import ArtifactPanel from '$lib/components/workbench/ArtifactPanel.svelte';
  import Composer from '$lib/components/workbench/Composer.svelte';

  const SSE_URL = import.meta.env.VITE_SSE_URL || 'http://localhost:33335';
  let prevThreadId: string | null = null;

  $effect(() => {
    const tid = $currentThread?.id ?? null;

    // Thread 切换时重连 SSE
    if (tid && tid !== prevThreadId) {
      sseConsumer.disconnect();
      sseConsumer.connect(`${SSE_URL}/api/sse/threads/${tid}`);
      prevThreadId = tid;
    }

    // cleanup: 组件销毁或 effect 重新运行时断开
    return () => {
      sseConsumer.disconnect();
      prevThreadId = null;
    };
  });

  // Composer 提交触发 Run
  async function handleSubmit(content: string, mode: string) {
    const tid = $currentThread?.id;
    if (!tid) {
      const t = {
        id: crypto.randomUUID(),
        goal: content.slice(0, 50),
        title: content.slice(0, 20),
        createdAt: new Date().toISOString(),
      };
      threadStore.addThread(t);
      threadStore.setCurrentThread(t.id);
      sseConsumer.connect(`${SSE_URL}/api/sse/threads/${t.id}`);
      prevThreadId = t.id;
    }

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
      <CanvasRenderer />
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
</style>