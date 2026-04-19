// Run Store — 管理 Run 状态机
import { writable, derived } from 'svelte/store';

export interface Run {
  id: string;
  thread_id: string;
  goal: string;
  status: string;
  stage?: string;
  visibility_layer?: number;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  result_summary?: string;
  error_message?: string;
}

export interface ToolInvocation {
  id: string;
  run_id: string;
  tool_name: string;
  tool_display_name?: string;
  args?: Record<string, unknown>;
  result?: Record<string, unknown>;
  error?: string;
  status: string;
  order?: number;
  started_at?: string;
  finished_at?: string;
  duration_ms?: number;
}

export interface RunState {
  runs: Run[];
  active_run_id: string | null;
}

function createRunStore() {
  const { subscribe, update } = writable<RunState>({
    runs: [],
    active_run_id: null,
  });

  return {
    subscribe,
    createRun(threadId: string): Run {
      const run: Run = {
        id: crypto.randomUUID(),
        thread_id: threadId,
        goal: '',
        status: 'pending',
        created_at: new Date().toISOString(),
      };
      update(s => ({ ...s, runs: [...s.runs, run] }));
      return run;
    },
    updateRunStatus(runId: string, status: string, finishedAt?: string) {
      update(s => ({
        ...s,
        runs: s.runs.map(r =>
          r.id === runId
            ? { ...r, status, finished_at: finishedAt ?? r.finished_at }
            : r
        ),
        active_run_id: ['completed','failed','cancelled'].includes(status) ? null : (s.active_run_id ?? runId),
      }));
    },
    setActiveRun(id: string | null) {
      update(s => ({ ...s, active_run_id: id }));
    },
  };
}

export const runStore = createRunStore();
export const activeRun = derived(runStore, $s => $s.runs.find(r => r.id === $s.active_run_id) ?? null);
