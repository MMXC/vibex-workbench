// Thread Store — 管理 Thread 和 Message
import { writable, derived } from 'svelte/store';
import { db } from '$lib/db';
import type { DBThread } from '$lib/db';
import type { Thread } from '$lib/types';

export type { Thread };

// Message 存储在 threadStore 内部（不从 generated.ts 导出）
export interface Message {
  id: string;
  threadId: string;
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  createdAt: string;
}

export interface ThreadState {
  threads: Thread[];
  currentThreadId: string | null;
  loading: boolean;
  error: string | null;
  /** 每个 threadId 下的对话气泡（由 SSE message.delta 等填充） */
  messagesByThread: Record<string, Message[]>;
  /**
   * 每个 thread 最后一条 assistant 消息的 ID。
   * 用于 appendDelta 在非 is_final 时追加到同一气泡，
   * 避免多轮对话时 agent 回复串到同一个 STREAMING_ASSISTANT_ID 气泡里。
   */
  pendingAssistantIdByThread: Record<string, string>;
  /**
   * 流式期间发送的用户消息先在这里排队。
   * 等 agent 回复完成后（is_final=true）一次性 flush 进去，
   * 保证消息顺序：msg1 → agent1 → msg2 → agent2，不会穿插。
   */
  queuedUserMessages: Record<string, Message[]>;
}

/** 去掉推理模型包裹块，避免正文区刷屏（MiniMax / DeepSeek 等） */
export function stripReasoningTags(text: string): string {
  if (!text) return '';
  return text
    .replace(/<(?:think|redacted_reasoning)>[\s\S]*?<\/(?:think|redacted_reasoning)>/gi, '')
    .trim();
}

const STREAMING_ASSISTANT_ID = '__streaming-assistant__';

function createThreadStore() {
  const { subscribe, set, update } = writable<ThreadState>({
    threads: [],
    currentThreadId: null,
    loading: false,
    error: null,
    messagesByThread: {},
    pendingAssistantIdByThread: {},
    queuedUserMessages: {},
  });

  return {
    subscribe,

    /** 从 IndexedDB 加载所有活跃 Thread（页面初始化时调用） */
    async loadFromDB() {
      update(s => ({ ...s, loading: true, error: null }));
      try {
        const rows = await db.threads.toArray();
        const threads = rows
          .filter(r => !r.deleted_at)
          .map(r => ({
          id: r.id,
          title: r.title,
          goal: r.goal,
          status: r.status as Thread['status'],
          createdAt: r.created_at,
          updatedAt: r.updated_at,
        })) as Thread[];
        update(s => ({ ...s, threads, loading: false }));
      } catch (e) {
        update(s => ({
          ...s,
          loading: false,
          error: `无法加载 Thread: ${e instanceof Error ? e.message : String(e)}`,
        }));
      }
    },

    addThread(thread: Thread) {
      update(s => ({ ...s, threads: [...s.threads, thread] }));
      db.threads.put({
        id: thread.id,
        title: thread.title ?? '',
        goal: thread.goal ?? '',
        status: (thread.status ?? 'idle') as string,
        created_at: thread.createdAt,
        updated_at: thread.updatedAt ?? thread.createdAt,
        is_deleted: 0,
      }).catch(e => console.error('[threadStore] Failed to persist:', e));
    },

    setCurrentThread(id: string | null) {
      update(s => ({ ...s, currentThreadId: id }));
    },

    appendMessage(threadId: string, message: Message) {
      update(s => {
        // 去重：同一 id 的消息不重复追加
        const existing = s.messagesByThread[threadId] ?? [];
        if (existing.some(m => m.id === message.id)) return s;

        // 如果当前有 pending assistant 消息，用户消息先排队（去重）
        if (s.pendingAssistantIdByThread[threadId]) {
          const queued = s.queuedUserMessages[threadId] ?? [];
          if (queued.some(m => m.id === message.id)) return s;
          return {
            ...s,
            queuedUserMessages: {
              ...s.queuedUserMessages,
              [threadId]: [...queued, message],
            },
          };
        }
        return {
          ...s,
          messagesByThread: {
            ...s.messagesByThread,
            [threadId]: [...existing, message],
          },
        };
      });
    },

    /**
     * Agent message.delta：
     * - user：每条追加；
     * - assistant + is_final:false：累积到当前 pending 气泡；
     * - assistant + is_final:true：去掉所有占位，追加最终答复，记住该 ID。
     *
     * 关键修复：使用 pendingAssistantIdByThread[tid] 跟踪当前 assistant 消息 ID，
     * 避免多轮对话时所有 agent 回复都追加到同一个 STREAMING_ASSISTANT_ID 气泡里。
     */
    appendDelta(
      threadId: string,
      payload: { role: string; delta: string; is_final?: boolean }
    ) {
      const role = payload.role ?? 'assistant';
      const raw = payload.delta ?? '';
      const isFinal = payload.is_final === true;
      const iso = new Date().toISOString();

      if (role === 'user' && raw) {
        const message: Message = {
          id: crypto.randomUUID(),
          threadId,
          role: 'user',
          content: raw,
          createdAt: iso,
        };
        update(s => ({
          ...s,
          messagesByThread: {
            ...s.messagesByThread,
            [threadId]: [...(s.messagesByThread[threadId] ?? []), message],
          },
        }));
        return;
      }

      if (role !== 'assistant') return;

      update(s => {
        const prev = [...(s.messagesByThread[threadId] ?? [])];
        const pendingId = s.pendingAssistantIdByThread[threadId];

        if (!isFinal) {
          // 优先用 pending ID，其次用 STREAMING_ASSISTANT_ID
          const targetId = pendingId ?? STREAMING_ASSISTANT_ID;
          const idx = prev.findIndex(m => m.id === targetId);
          if (idx >= 0) {
            const cur = prev[idx];
            prev[idx] = { ...cur, content: cur.content + raw, createdAt: iso };
          } else {
            // 没有占位消息 → 第一次流式，开始新的
            const newId = crypto.randomUUID();
            prev.push({ id: newId, threadId, role: 'assistant', content: raw, createdAt: iso });
            return {
              ...s,
              messagesByThread: { ...s.messagesByThread, [threadId]: prev },
              pendingAssistantIdByThread: { ...s.pendingAssistantIdByThread, [threadId]: newId },
            };
          }
          return {
            ...s,
            messagesByThread: { ...s.messagesByThread, [threadId]: prev },
          };
        }

        // is_final: true → 追加最终答复，再 flush 排队的用户消息
        const withoutPlaceholders = prev.filter(
          m => m.id !== STREAMING_ASSISTANT_ID && m.id !== pendingId
        );
        const content = stripReasoningTags(raw);
        const queued = s.queuedUserMessages[threadId] ?? [];
        if (!content) {
          // assistant 内容为空时也要 flush 排队的用户消息
          return {
            ...s,
            messagesByThread: {
              ...s.messagesByThread,
              [threadId]: [...withoutPlaceholders, ...queued],
            },
            pendingAssistantIdByThread: { ...s.pendingAssistantIdByThread, [threadId]: '' },
            queuedUserMessages: { ...s.queuedUserMessages, [threadId]: [] },
          };
        }
        const finalId = crypto.randomUUID();
        return {
          ...s,
          messagesByThread: {
            ...s.messagesByThread,
            [threadId]: [...withoutPlaceholders, { id: finalId, threadId, role: 'assistant', content, createdAt: iso }, ...queued],
          },
          pendingAssistantIdByThread: { ...s.pendingAssistantIdByThread, [threadId]: '' },
          queuedUserMessages: { ...s.queuedUserMessages, [threadId]: [] },
        };
      });
    },

    updateThread(id: string, patch: Partial<Thread>) {
      update(s => ({
        ...s,
        threads: s.threads.map(t => t.id === id ? { ...t, ...patch } : t),
      }));
      const now = new Date().toISOString();
      const updateData: Partial<DBThread> = { updated_at: now };
      if (patch.title !== undefined) updateData.title = patch.title;
      if (patch.goal !== undefined) updateData.goal = patch.goal;
      if (patch.status !== undefined) updateData.status = patch.status as string;
      db.threads.update(id, updateData).catch(e => console.error('[threadStore] Failed to update:', e));
    },

    /** 软删除 Thread（设置 deletedAt 标记） */
    removeThread(id: string) {
      update(s => ({
        ...s,
        threads: s.threads.filter(t => t.id !== id),
        currentThreadId: s.currentThreadId === id ? null : s.currentThreadId,
      }));
      db.threads.update(id, {
        deleted_at: new Date().toISOString(),
      } as Partial<DBThread>).catch(e => console.error('[threadStore] Failed to soft-delete:', e));
    },

    setLoading(v: boolean) {
      update(s => ({ ...s, loading: v }));
    },

    setError(e: string | null) {
      update(s => ({ ...s, error: e }));
    },
  };
}

export const threadStore = createThreadStore();

export const currentThread = derived(threadStore, $s =>
  $s.threads.find(t => t.id === $s.currentThreadId) ?? null
);

export const threadCount = derived(threadStore, $s => $s.threads.length);

/** 当前线程的对话列表（供 ConversationPanel） */
export const currentMessages = derived(threadStore, $s => {
  const tid = $s.currentThreadId;
  if (!tid) return [] as Message[];
  return $s.messagesByThread[tid] ?? [];
});