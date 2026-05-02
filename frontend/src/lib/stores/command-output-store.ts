// command-output-store.ts — Stores command output and problems from run-make API.
// SLICE-command-output-problems-store
import { writable, derived } from 'svelte/store';

export type CommandStatus = 'idle' | 'running' | 'success' | 'failure' | 'timeout';

export interface CommandEntry {
  id: string;
  target: string;
  status: CommandStatus;
  exitCode: number;
  stdout: string;
  stderr: string;
  startedAt: string;
  finishedAt?: string;
  duration?: string;
  workspaceRoot: string;
  problems: ProblemEntry[];
}

export interface ProblemEntry {
  line?: number;
  message: string;
  severity: 'error' | 'warning' | 'info';
  file?: string;
}

interface CommandStoreState {
  commands: CommandEntry[];
  activeId: string | null;
  loading: boolean;
  error: string | null;
}

function createCommandOutputStore() {
  const { subscribe, update, set } = writable<CommandStoreState>({
    commands: [],
    activeId: null,
    loading: false,
    error: null,
  });

  return {
    subscribe,

    async runMake(workspaceRoot: string, target: string) {
      update(s => ({ ...s, loading: true, error: null }));

      const id = `cmd-${Date.now()}`;
      const startedAt = new Date().toISOString();

      // Add pending entry
      update(s => ({
        ...s,
        commands: [...s.commands, {
          id, target, status: 'running', exitCode: -1,
          stdout: '', stderr: '', startedAt,
          workspaceRoot, problems: [],
        }],
        activeId: id,
        loading: false,
      }));

      try {
        const res = await fetch('/api/workspace/run-make', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ workspace_root: workspaceRoot, target }),
        });

        const data = await res.json();
        const finishedAt = new Date().toISOString();

        // Parse problems from stderr
        const problems = parseProblems(data.output ?? '');

        update(s => ({
          ...s,
          commands: s.commands.map(cmd =>
            cmd.id === id
              ? {
                  ...cmd,
                  status: data.timeout ? 'timeout' as CommandStatus
                    : (data.exitCode === 0 ? 'success' as CommandStatus : 'failure' as CommandStatus),
                  exitCode: data.exitCode ?? -1,
                  stdout: truncate(data.output ?? '', 500),
                  stderr: truncate(data.stderr ?? '', 500),
                  finishedAt,
                  duration: data.duration ?? '',
                  problems,
                }
              : cmd
          ),
        }));
      } catch (e) {
        update(s => ({
          ...s,
          commands: s.commands.map(cmd =>
            cmd.id === id ? { ...cmd, status: 'failure' as CommandStatus, stderr: String(e) } : cmd
          ),
          error: String(e),
        }));
      }
    },

    setActive(id: string | null) {
      update(s => ({ ...s, activeId: id }));
    },

    clear() {
      update(s => ({ ...s, commands: [], activeId: null }));
    },

    setError(err: string | null) {
      update(s => ({ ...s, error: err }));
    },
  };
}

export const commandOutputStore = createCommandOutputStore();

// Derived: all problems across commands
export const allProblems = derived(commandOutputStore, $s =>
  $s.commands.flatMap(c => c.problems)
);

function parseProblems(output: string): ProblemEntry[] {
  const problems: ProblemEntry[] = [];
  const lines = output.split('\n');
  // Simple parser: look for common error patterns
  for (const line of lines) {
    const fileMatch = line.match(/^(\S+:\d+:\d+:?\s*)/);
    if (fileMatch && (line.toLowerCase().includes('error') || line.toLowerCase().includes('warning'))) {
      const [filePath, rest] = line.split(':');
      const parts = rest.split(':');
      problems.push({
        line: parts[0] ? parseInt(parts[0]) : undefined,
        message: line,
        severity: line.toLowerCase().includes('error') ? 'error' : 'warning',
        file: filePath,
      });
    }
  }
  return problems;
}

function truncate(s: string, max: number): string {
  if (!s) return '';
  if (s.length <= max) return s;
  return s.slice(0, max) + '...[truncated, see terminal]';
}