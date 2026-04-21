/**
 * Agent 对话文本 SSE — 单独 bridge，避免修改 spec-to-code 生成的 `$lib/sse.ts`。
 * Spec: specs/feature/workbench-shell/workbench-conversation_feature.yaml
 *
 * 使用第二条 EventSource 订阅 `message.delta`（及应用层 `error`），与 sseConsumer 并行；
 * 代价为多一条连接，待 gen 支持后可合并回单一 Consumer。
 */
import { get } from 'svelte/store';
import { threadStore } from '$lib/stores/thread-store';

let bridge: EventSource | null = null;

function parseThreadId(data: Record<string, unknown>): string | null {
  const tid =
    (typeof data.threadId === 'string' && data.threadId) ||
    (typeof data.thread_id === 'string' && data.thread_id) ||
    get(threadStore).currentThreadId;
  return tid ?? null;
}

function onMessageDelta(e: MessageEvent) {
  try {
    const data = JSON.parse(String(e.data)) as Record<string, unknown>;
    const tid = parseThreadId(data);
    if (!tid) return;
    threadStore.appendDelta(tid, {
      role: String(data.role ?? 'assistant'),
      delta: typeof data.delta === 'string' ? data.delta : '',
      is_final: data.is_final === true,
    });
  } catch {
    console.error('[workbench-message-sse-bridge] message.delta parse failed', e.data);
  }
}

function onAppError(e: Event) {
  const me = e as MessageEvent;
  if (typeof me.data !== 'string' || !me.data.trim().startsWith('{')) return;
  try {
    const data = JSON.parse(me.data) as Record<string, unknown>;
    const tid = parseThreadId(data);
    if (!tid) return;
    const err =
      typeof data.error === 'string'
        ? data.error
        : typeof data.message === 'string'
          ? data.message
          : JSON.stringify(data);
    threadStore.appendMessage(tid, {
      id: crypto.randomUUID(),
      threadId: tid,
      role: 'system',
      content: err,
      createdAt: new Date().toISOString(),
    });
  } catch {
    /* 连接层 error 等非 JSON，忽略 */
  }
}

/** 与 `sseConsumer.connect(url)` 使用同一 `url`。 */
export function connectWorkbenchMessageBridge(url: string) {
  disconnectWorkbenchMessageBridge();
  bridge = new EventSource(url);
  bridge.addEventListener('message.delta', onMessageDelta as EventListener);
  bridge.addEventListener('error', onAppError as EventListener);
}

export function disconnectWorkbenchMessageBridge() {
  bridge?.close();
  bridge = null;
}
