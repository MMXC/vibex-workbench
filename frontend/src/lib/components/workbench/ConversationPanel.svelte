<!-- ============================================================
ConversationPanel — 当前线程对话（SSE message.delta）
Spec: specs/feature/workbench-shell/workbench-conversation_feature.yaml
开发者维护，gen.py 永不覆盖
============================================================ -->

<script lang="ts">
  import { tick } from 'svelte';
  import {
    currentMessages,
    currentThread,
    stripReasoningTags,
    type Message,
  } from '$lib/stores/thread-store';
  import { runStore } from '$lib/stores/run-store';
  import { aiBlocksStore } from '$lib/stores/ai-blocks-store';
  import ToolChatBubble from '$lib/components/workbench/ToolChatBubble.svelte';

  let messages = $state<Message[]>([]);
  let activeRunStatus = $state<string | null>(null);
  let currentThreadId = $state<string | null>(null);
  let threadHeadline = $state('当前会话');

  $effect(() => {
    const unsub = currentMessages.subscribe(m => {
      messages = m;
    });
    return unsub;
  });

  $effect(() => {
    const unsub = currentThread.subscribe(t => {
      currentThreadId = t?.id ?? null;
      threadHeadline =
        (t?.title?.trim() || t?.goal?.slice(0, 52)?.trim() || '当前会话') ?? '当前会话';
    });
    return unsub;
  });

  $effect(() => {
    const unsub = runStore.subscribe(s => {
      activeRunStatus = s.runs.find(r => r.id === s.active_run_id)?.status ?? null;
    });
    return unsub;
  });

  let scrollEl = $state<HTMLDivElement | undefined>(undefined);

  $effect(() => {
    messages;
    aiBlocksStore.ingestThread(currentThreadId, messages);
    tick().then(() => {
      scrollEl?.scrollTo({ top: scrollEl.scrollHeight, behavior: 'smooth' });
    });
  });

  function displayContent(m: Message): string {
    if (m.role === 'assistant') return stripReasoningTags(m.content);
    return m.content;
  }

</script>

<section class="conversation-panel" aria-label="工作台对话">
  <header class="chat-head">
    <div>
      <span class="k">Workbench Chat</span>
      <h3>{threadHeadline}</h3>
    </div>
    <div class="chat-head-right">
      {#if activeRunStatus}
        <span class="status-pill run">{activeRunStatus}</span>
      {/if}
      {#if messages.length === 0}
        <span class="hdr-hint">发送后回复出现在下方</span>
      {/if}
    </div>
  </header>
  <div class="messages" bind:this={scrollEl}>
    {#each messages as m (m.id)}
      <div class="msg {m.role}">
        <span>{m.role}</span>
        {#if m.role === 'tool'}
          <ToolChatBubble content={m.content} openPath={m.toolOpenPath} />
        {:else}
          <pre>{displayContent(m)}</pre>
        {/if}
      </div>
    {/each}
  </div>
</section>

<style>
  /* 与 SpecSlotChatPane 槽位对话区对齐：背景、气泡、字色、圆角 */
  .conversation-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    flex: 1;
    min-height: 0;
    min-width: 0;
    background: #11141b;
    border-bottom: 1px solid #303746;
  }

  .chat-head {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 14px;
    border-bottom: 1px solid #303746;
    background: rgba(28, 32, 42, 0.84);
  }

  .chat-head-right {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
  }

  .k {
    display: block;
    margin-bottom: 4px;
    color: #72d6d0;
    font-family: 'Cascadia Code', ui-monospace, monospace;
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .chat-head h3 {
    margin: 0;
    color: #eef0f5;
    font-size: 14px;
    line-height: 1.3;
    font-weight: 700;
    max-width: min(100%, 280px);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .hdr-hint {
    font-size: 11px;
    color: #6f7888;
    white-space: nowrap;
  }

  .status-pill {
    border: 1px solid #465064;
    border-radius: 999px;
    padding: 4px 9px;
    color: #a3abb9;
    font-family: ui-monospace, monospace;
    font-size: 10px;
    font-weight: 800;
    text-transform: lowercase;
  }

  .status-pill.run {
    color: #efc66b;
    border-color: rgba(239, 198, 107, 0.5);
  }

  .messages {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .msg {
    display: flex;
    flex-direction: column;
    gap: 5px;
    max-width: 100%;
  }

  .msg.user {
    align-items: flex-end;
  }

  .msg span {
    color: #6f7888;
    font-family: ui-monospace, monospace;
    font-size: 10px;
    text-transform: uppercase;
  }

  .msg pre {
    max-width: min(100%, 760px);
    margin: 0;
    padding: 10px 12px;
    border: 1px solid #303746;
    border-radius: 12px;
    background: #171b24;
    color: #d4d8e3;
    font-family: ui-sans-serif, system-ui, sans-serif;
    font-size: 13px;
    line-height: 1.55;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .msg.user pre {
    background: rgba(122, 162, 255, 0.16);
    border-color: rgba(122, 162, 255, 0.42);
    color: #eef0f5;
  }

  .msg.system pre {
    background: rgba(239, 198, 107, 0.08);
    border-color: rgba(239, 198, 107, 0.34);
    color: #d4d8e3;
  }
</style>
