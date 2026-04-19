// Thread Store — 管理 Thread 和 Message
import { writable, derived } from 'svelte/store';
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
    addThread(thread: Thread) {
      update(s => ({ ...s, threads: [...s.threads, thread] }));
    },
    setCurrentThread(id: string | null) {
      update(s => ({ ...s, currentThreadId: id }));
    },
    appendMessage(threadId: string, message: Message) {
      // Thread 不直接存储 messages；由 sse.ts 中的 SSE 事件驱动
      // 此方法存在仅为 API 兼容性
    },
    updateThread(id: string, patch: Partial<Thread>) {
      update(s => ({
        ...s,
        threads: s.threads.map(t => t.id === id ? { ...t, ...patch } : t),
      }));
    },
    removeThread(id: string) {
      update(s => ({
        ...s,
        threads: s.threads.filter(t => t.id !== id),
        currentThreadId: s.currentThreadId === id ? null : s.currentThreadId,
      }));
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
