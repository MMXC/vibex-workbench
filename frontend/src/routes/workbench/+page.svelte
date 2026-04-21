<!-- ============================================================
VibeX Workbench — 主工作台页面
开发者维护，gen.py 永不覆盖
============================================================ -->

<script lang="ts">
  import { sseConsumer } from '$lib/sse';
  import {
    connectWorkbenchMessageBridge,
    disconnectWorkbenchMessageBridge,
  } from '$lib/workbench/workbench-message-sse-bridge';
  import { get } from 'svelte/store';
  import {
    workbenchLayoutStore,
    workbenchMainAreaHeight,
    type WorkbenchLayoutDims,
  } from '$lib/stores/workbench-layout-store';
  import { threadStore, currentThread } from '$lib/stores/thread-store';
  import WorkbenchLayoutResizable from '$lib/components/workbench/WorkbenchLayoutResizable.svelte';
  import ThreadList from '$lib/components/workbench/ThreadList.svelte';
  import ConversationPanel from '$lib/components/workbench/ConversationPanel.svelte';
  import CanvasRenderer from '$lib/components/workbench/CanvasRenderer.svelte';
  import ArtifactPanel from '$lib/components/workbench/ArtifactPanel.svelte';
  import Composer from '$lib/components/workbench/Composer.svelte';

  const SSE_URL = import.meta.env.VITE_SSE_URL || 'http://localhost:33335';
  /** 为 true 时走 `backend/` 的 mock（/api/sse/threads、/api/runs）；false 走 `agent`（/api/sse、/api/chat） */
  const useMockBackend =
    import.meta.env.VITE_MOCK_SSE === '1' || import.meta.env.VITE_MOCK_SSE === 'true';
  let prevThreadId: string | null = null;

  let layout = $state<WorkbenchLayoutDims>({
    sidebarLeftPx: 280,
    panelRightPx: 320,
    composerBarPx: 172,
    conversationPanePx: 260,
  });
  let mainAreaH = $state(400);

  $effect(() => {
    const unsub = workbenchLayoutStore.subscribe(v => {
      layout = v;
    });
    return unsub;
  });

  $effect(() => {
    const unsub = workbenchMainAreaHeight.subscribe(h => {
      mainAreaH = h > 0 ? h : 400;
    });
    return unsub;
  });

  function beginConversationCanvasSplit(e: PointerEvent) {
    if (typeof window !== 'undefined' && window.matchMedia('(max-width: 767px)').matches) return;
    e.preventDefault();
    const startY = e.clientY;
    const startH = get(workbenchLayoutStore).conversationPanePx;
    function move(ev: PointerEvent) {
      workbenchLayoutStore.previewConversationPanePx(
        startH + (ev.clientY - startY),
        mainAreaH
      );
    }
    function end() {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', end);
      workbenchLayoutStore.commit();
    }
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', end);
  }

  function sseConnectPath(tid: string) {
    return useMockBackend
      ? `${SSE_URL}/api/sse/threads/${tid}`
      : `${SSE_URL}/api/sse/${tid}`;
  }

  $effect(() => {
    const tid = $currentThread?.id ?? null;

    // Thread 切换时重连 SSE
    if (tid && tid !== prevThreadId) {
      sseConsumer.disconnect();
      disconnectWorkbenchMessageBridge();
      const url = sseConnectPath(tid);
      sseConsumer.connect(url);
      connectWorkbenchMessageBridge(url);
      prevThreadId = tid;
    }

    // cleanup: 组件销毁或 effect 重新运行时断开
    return () => {
      sseConsumer.disconnect();
      disconnectWorkbenchMessageBridge();
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
      const url = sseConnectPath(t.id);
      sseConsumer.disconnect();
      disconnectWorkbenchMessageBridge();
      sseConsumer.connect(url);
      connectWorkbenchMessageBridge(url);
      prevThreadId = t.id;
    }

    try {
      const threadKey = tid || prevThreadId;
      if (useMockBackend) {
        await fetch(`${SSE_URL}/api/runs`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ threadId: threadKey, goal: content }),
        });
      } else {
        await fetch(`${SSE_URL}/api/chat`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ threadId: threadKey, input: content }),
        });
      }
    } catch (e) {
      console.error('[Workbench] Failed to start run:', e);
    }
  }
</script>

<div class="workbench-root">
  <WorkbenchLayoutResizable>
    {#snippet sidebar()}
      <ThreadList />
    {/snippet}

    {#snippet main()}
      <div class="main-stack">
        <div class="conv-wrap" style:height="{layout.conversationPanePx}px">
          <ConversationPanel />
        </div>
        <button
          type="button"
          class="split-conv"
          aria-label="拖动调整对话区与画布高度"
          onpointerdown={beginConversationCanvasSplit}
        ></button>
        <div class="canvas-area">
          <CanvasRenderer />
        </div>
      </div>
    {/snippet}

    {#snippet panel()}
      <ArtifactPanel />
    {/snippet}

    {#snippet composer()}
      <Composer onsubmit={handleSubmit} />
    {/snippet}
  </WorkbenchLayoutResizable>
</div>

<style>
  .workbench-root { width: 100vw; height: 100vh; overflow: hidden; }

  .main-stack {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .conv-wrap {
    flex-shrink: 0;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .split-conv {
    flex-shrink: 0;
    height: 5px;
    margin: 0;
    padding: 0;
    border: none;
    cursor: row-resize;
    touch-action: none;
    background: var(--wb-splitter, #2a2a2a);
    z-index: 4;
  }

  .split-conv:hover,
  .split-conv:active {
    background: rgba(129, 140, 248, 0.45);
  }

  .canvas-area {
    flex: 1;
    min-height: 0;
    position: relative;
  }

  @media (max-width: 767px) {
    .conv-wrap {
      height: auto !important;
      flex: 0 1 auto;
      max-height: min(42vh, 280px);
    }
    .split-conv {
      display: none;
    }
    .canvas-area {
      flex: 1;
    }
  }
</style>