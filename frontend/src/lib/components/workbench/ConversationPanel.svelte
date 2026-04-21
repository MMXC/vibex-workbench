<!-- ============================================================
ConversationPanel — 当前线程对话（SSE message.delta）
Spec: specs/feature/workbench-shell/workbench-conversation_feature.yaml
开发者维护，gen.py 永不覆盖
============================================================ -->

<script lang="ts">
  import {
    currentMessages,
    stripReasoningTags,
    type Message,
  } from '$lib/stores/thread-store';

  let messages = $state<Message[]>([]);

  $effect(() => {
    const unsub = currentMessages.subscribe(m => {
      messages = m;
    });
    return unsub;
  });

  let scrollEl = $state<HTMLDivElement | undefined>(undefined);

  $effect(() => {
    messages;
    queueMicrotask(() => {
      scrollEl?.scrollTo({ top: scrollEl.scrollHeight, behavior: 'smooth' });
    });
  });

  function displayContent(m: Message): string {
    if (m.role === 'assistant') return stripReasoningTags(m.content);
    return m.content;
  }
</script>

<section class="conversation-panel" aria-label="对话">
  <header class="hdr">
    <span class="hdr-title">对话</span>
    {#if messages.length === 0}
      <span class="hdr-hint">发送消息后，回复会出现在此处</span>
    {/if}
  </header>
  <div class="scroll" bind:this={scrollEl}>
    {#each messages as m (m.id)}
      <div class="row" data-role={m.role}>
        <span class="role">{m.role}</span>
        <pre class="bubble">{displayContent(m)}</pre>
      </div>
    {/each}
  </div>
</section>

<style>
  .conversation-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    flex: 1;
    min-height: 0;
    border-bottom: 1px solid var(--wb-border, #262626);
    background: var(--wb-conv-bg, #0f0f0f);
    min-width: 0;
    box-shadow: inset 0 -1px 0 rgba(79, 70, 229, 0.12);
  }

  .hdr {
    flex-shrink: 0;
    display: flex;
    align-items: baseline;
    gap: 10px;
    padding: 8px 12px;
    border-bottom: 1px solid #1f1f1f;
  }

  .hdr-title {
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: #a3a3a3;
    text-transform: uppercase;
  }

  .hdr-hint {
    font-size: 11px;
    color: #525252;
  }

  .scroll {
    flex: 1;
    overflow: auto;
    padding: 10px 12px 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-height: 0;
  }

  .row {
    display: flex;
    flex-direction: column;
    gap: 4px;
    align-items: flex-start;
    max-width: 100%;
  }

  .row[data-role='user'] {
    align-items: flex-end;
  }

  .row[data-role='user'] .bubble {
    background: #1e3a5f;
    border-color: #2563eb44;
    color: #e2e8f0;
  }

  .row[data-role='assistant'] .bubble {
    background: #171717;
    border-color: #333;
    color: #d4d4d4;
  }

  .row[data-role='system'] .bubble {
    background: #3a1a1a;
    border-color: #7f1d1d88;
    color: #fecaca;
    font-size: 12px;
  }

  .role {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #737373;
  }

  .bubble {
    margin: 0;
    max-width: min(100%, 52rem);
    padding: 8px 11px;
    border-radius: 8px;
    border: 1px solid #333;
    font-family: ui-sans-serif, system-ui, sans-serif;
    font-size: 13px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
  }
</style>
