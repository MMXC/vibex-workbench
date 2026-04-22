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
   * 当前 assistant 消息 ID（由 is_final=false 流式 chunk 设置）。
   * 用于 appendDelta 在非 is_final 时追加到同一气泡，
   * 避免多轮对话时 agent 回复串到同一个气泡里。
   */
  pendingAssistantIdByThread: Record<string, string>;
}

/** 去掉推理模型包裹块，避免正文区刷屏（MiniMax / DeepSeek 等） */
export function stripReasoningTags(text: string): string {
  if (!text) return '';
  return text
    .replace(/<(?:think|redacted_reasoning)>[\s\S]*?<\/(?:think|redacted_reasoning)>/gi, '')
    .trim();
}

function createThreadStore() {
  const { subscribe, set, update } = writable<ThreadState>({
    threads: [],
    currentThreadId: null,
    loading: false,
    error: null,
    messagesByThread: {},
    pendingAssistantIdByThread: {},
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

    /**
     * 来自 SSE message.created（由 sseConsumer 调用）的消息追加。
     * 当前主要用于 system/reflection 类型的消息追加。
     */
    appendMessage(threadId: string, message: Message) {
      update(s => {
        const existing = s.messagesByThread[threadId] ?? [];
        if (existing.some(m => m.id === message.id)) return s;
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
     * - user：追加到对话（用户消息通过 SSE 回显）。
     * - assistant + is_final:false：追加到当前 pending 气泡或新建。
     * - assistant + is_final:true：替换 pending 气泡内容。
     *
     * 不再使用排队机制——SSE 事件本身保证顺序。
     * pendingAssistantIdByThread 在 run.completed / run.failed 时由外部清除。
     */
    appendDelta(
      threadId: string,
      payload: { role: string; delta: string; is_final?: boolean }
    ) {
      const role = payload.role ?? 'assistant';
      const raw = payload.delta ?? '';
      const isFinal = payload.is_final === true;
      const iso = new Date().toISOString();

      // 用户消息：直接追加（server 通过 message.delta 回显用户输入）
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
          // 追加到 pending 气泡，或新建
          const targetId = pendingId ?? '__streaming__';
          const idx = prev.findIndex(m => m.id === targetId);
          if (idx >= 0) {
            const cur = prev[idx];
            prev[idx] = { ...cur, content: cur.content + raw, createdAt: iso };
          } else {
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

        // is_final: true → 替换 pending 气泡内容（去除推理标签）
        const withoutStreaming = prev.filter(
          m => m.id !== '__streaming__' && m.id !== pendingId
        );
        const content = stripReasoningTags(raw);
        if (!content) {
          // 空回复也保留结构，清除 pending
          return {
            ...s,
            messagesByThread: { ...s.messagesByThread, [threadId]: withoutStreaming },
            pendingAssistantIdByThread: { ...s.pendingAssistantIdByThread, [threadId]: '' },
          };
        }
        const finalId = crypto.randomUUID();
        return {
          ...s,
          messagesByThread: {
            ...s.messagesByThread,
            [threadId]: [...withoutStreaming, { id: finalId, threadId, role: 'assistant', content, createdAt: iso }],
          },
          pendingAssistantIdByThread: { ...s.pendingAssistantIdByThread, [threadId]: '' },
        };
      });
    },

    /** run.completed / run.failed 时清除 pending 状态，确保下一轮正确识别 */
    clearPendingAssistant(threadId: string) {
      update(s => ({
        ...s,
        pendingAssistantIdByThread: { ...s.pendingAssistantIdByThread, [threadId]: '' },
      }));
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
