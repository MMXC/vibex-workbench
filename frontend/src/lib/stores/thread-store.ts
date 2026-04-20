// Thread Store — 管理 Thread 和 Message
import { writable, derived } from 'svelte/store';
import { db } from '$lib/db';
import type { DBThread } from '$lib/db';
import type { Thread } from '$lib/types/generated';

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
}

function createThreadStore() {
  const { subscribe, set, update } = writable<ThreadState>({
    threads: [],
    currentThreadId: null,
    loading: false,
    error: null,
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
      // Thread 不直接存储 messages；由 sse.ts 中的 SSE 事件驱动
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