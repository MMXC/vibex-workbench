// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: specs
//     生成时间: 2026-04-19
//     ⚠️  不要直接编辑此文件
// ============================================================

// SSE Consumer — 订阅后端事件并分发到 Stores
import { get } from 'svelte/store';
import { threadStore } from '$lib/stores/thread-store';
import { runStore } from '$lib/stores/run-store';
import { artifactStore } from '$lib/stores/artifact-store';
import { canvasStore } from '$lib/stores/canvas-store';
import { getAgentApiBase } from '$lib/runtime/agent-transport';
import {
  extractToolWorkspacePath,
  formatToolCalledBody,
  formatToolCompletedFooter,
} from '$lib/workbench/tool-call-chat';

function toolCallThreadId(data: Record<string, unknown>): string {
  return String(data.threadId ?? data.runId ?? data.run_id ?? '');
}

type SSEEventHandler = (data: unknown) => void;

const HANDLERS: Record<string, SSEEventHandler> = {
  // ── Thread events ────────────────────────────────────────
  'message.created': (data: any) => {
    threadStore.appendMessage(data.threadId, data.message);
  },
  'thread.created': (data: any) => {
    threadStore.addThread(data.thread);
  },
  'thread.updated': (data: any) => {
    threadStore.updateThread(data.threadId, data.patch);
  },

  // ── Run events ───────────────────────────────────────────
  'run.started': (data: any) => {
    runStore.updateRunStatus(data.runId, 'executing');
    // Canvas sync: create RunNode
    canvasStore.addNode({
      id: data.runId,
      type: 'run',
      position: { x: 300, y: 100 },
      data: {
        label: `Run: ${(data.goal ?? '').slice(0, 20)}`,
        status: 'running',
        stage: data.stage,
      },
    });
  },
  'run.stage_changed': (data: any) => {
    runStore.updateRunStatus(data.runId, data.stage ?? 'executing');
    canvasStore.updateNode(data.runId, { data: { status: data.stage } });
  },
  'run.completed': (data: any) => {
    runStore.updateRunStatus(data.runId, 'completed');
    canvasStore.updateNode(data.runId, { data: { status: 'completed', summary: data.summary } });
  },
  'run.failed': (data: any) => {
    runStore.updateRunStatus(data.runId, 'failed');
    canvasStore.updateNode(data.runId, { data: { status: 'failed', error: data.error } });
  },
  'run.cancelled': (data: any) => {
    runStore.updateRunStatus(data.runId, 'cancelled');
    canvasStore.updateNode(data.runId, { data: { status: 'cancelled' } });
  },

  // ── Tool events (map to canvas nodes) ──────────────────
  'tool.called': (data: any) => {
    const callId = String(data.invocationId ?? data.call_id ?? crypto.randomUUID());
    const runId = data.runId ?? data.run_id;
    const toolName = String(data.toolName ?? data.tool ?? 'tool');
    const args = data.args;
    // E3-U1: 追踪 tool invocation
    runStore.addToolInvocation({
      id: callId,
      run_id: runId,
      tool_name: toolName,
      tool_display_name: toolName,
      args,
      status: 'running',
      order: data.order ?? 0,
    });
    // Canvas node creation
    canvasStore.addNode({
      id: callId,
      type: 'tool',
      position: { x: 150 + Math.random() * 300, y: 200 + Math.random() * 200 },
      parent_id: runId,
      data: {
        label: toolName,
        status: 'running',
        args,
      },
    });
    // E5-U4: 自动创建 edge (run → tool)
    if (runId) {
      canvasStore.addEdge({
        id: `${runId}-${callId}`,
        source: runId,
        target: callId,
        label: '',
      });
    }
    const threadId = toolCallThreadId(data);
    if (threadId) {
      const openPath = extractToolWorkspacePath(toolName, args);
      threadStore.appendToolCallMessage(threadId, {
        id: `tool-inline:${callId}`,
        threadId,
        role: 'tool',
        content: formatToolCalledBody(toolName, args),
        createdAt: new Date().toISOString(),
        toolOpenPath: openPath,
      });
    }
  },
  'tool.completed': (data: any) => {
    const callId = String(data.invocationId ?? data.call_id ?? '');
    const rawResult = data.result;
    const resultStr =
      typeof rawResult === 'string' ? rawResult : JSON.stringify(rawResult ?? '');
    const failed = /^error:|^blocked:|^filter_rejected:/i.test(resultStr.trim());
    // E3-U1: 更新 tool invocation 状态
    if (callId) {
      runStore.updateToolInvocation(callId, {
        status: 'completed',
        result: rawResult ?? resultStr,
        finished_at: new Date().toISOString(),
      });
      canvasStore.updateNode(callId, {
        data: { status: failed ? 'failed' : 'completed', result: rawResult ?? resultStr },
      });
    }
    const threadId = toolCallThreadId(data);
    if (threadId && callId) {
      const msgId = `tool-inline:${callId}`;
      const prev =
        get(threadStore).messagesByThread[threadId]?.find(m => m.id === msgId)?.content ?? '';
      threadStore.patchMessage(threadId, msgId, {
        content: prev + formatToolCompletedFooter(resultStr, failed),
      });
    }
  },
  'tool.failed': (data: any) => {
    const callId = String(data.invocationId ?? data.call_id ?? '');
    const err = String(data.error ?? '');
    // E3-U1: 更新 tool invocation 错误状态
    if (callId) {
      runStore.updateToolInvocation(callId, {
        status: 'failed',
        error: err,
        finished_at: new Date().toISOString(),
      });
      canvasStore.updateNode(callId, { data: { status: 'failed', error: err } });
    }
    const threadId = toolCallThreadId(data);
    if (threadId && callId) {
      const msgId = `tool-inline:${callId}`;
      const prev =
        get(threadStore).messagesByThread[threadId]?.find(m => m.id === msgId)?.content ?? '';
      threadStore.patchMessage(threadId, msgId, {
        content: prev + formatToolCompletedFooter(err, true),
      });
    }
  },

  // ── Artifact events ──────────────────────────────────────
  'artifact.created': (data: any) => {
    const a = data.artifact ?? data; // 支持 { artifact: {...} } 或直接对象
    artifactStore.create({
      name: a.name,
      type: a.type,
      content: a.content,
      mime_type: a.mime_type ?? a.mimeType ?? 'text/plain',
      tags: a.tags ?? [],
    });
  },
  'artifact.updated': (data: any) => {
    artifactStore.update(data.artifactId, { content: data.content });
  },

  // ── Canvas sync ──────────────────────────────────────────
  'node.added': (data: any) => {
    canvasStore.addNode(data.node);
  },
  'node.removed': (data: any) => {
    canvasStore.removeNode(data.nodeId);
  },
  'edge.added': (data: any) => {
    canvasStore.addEdge(data.edge);
  },

  // ── TDD events (backend: canvas.tdd_nodes / canvas.tdd_cycle) ──
  'canvas.tdd_nodes': (data: any) => {
    // TDD test case design: RED phase
    canvasStore.addNode({
      id: `tdd-nodes-${data.spec_id ?? 'default'}`,
      type: 'iteration',
      position: { x: 200, y: 300 },
      data: {
        label: `TDD: ${data.test_count ?? 0} test cases`,
        status: 'running',
        test_file: data.test_file,
        phases: data.phases ?? ['RED', 'GREEN', 'REFACTOR'],
        ...data,
      },
    });
  },
  'canvas.tdd_cycle': (data: any) => {
    // TDD cycle: RED/GREEN/REFACTOR update
    const statusMap: Record<string, string> = { RED: 'error', GREEN: 'done', REFACTOR: 'running' };
    canvasStore.addNode({
      id: `tdd-cycle-${data.spec_id ?? 'default'}`,
      type: data.phase === 'GREEN' ? 'sequence' : 'iteration',
      position: { x: 350, y: 350 },
      data: {
        label: `TDD ${data.phase ?? '?'}: ${data.passed ?? 0}p / ${data.failed ?? 0}f`,
        status: statusMap[data.phase ?? ''] ?? 'running',
        color: data.color,
        phase: data.phase,
        ...data,
      },
    });
  },

  // ── Spec creation (backend: canvas.spec_created) ────────────
  'canvas.spec_created': (data: any) => {
    canvasStore.addNode({
      id: `spec-${Date.now()}`,
      type: 'input',
      position: { x: 100, y: 100 },
      data: {
        label: data.title ?? 'New Spec',
        status: 'done',
        ...data,
      },
    });
  },

  // ── Agent self-reflection ────────────────────────────────
  'agent.self_reflection': (data: any) => {
    console.info('[SSE] agent.self_reflection:', data.summary ?? data);
    // Could emit a thread message here if threadStore supports it
  },
};

class SSEConsumer {
  private es: EventSource | null = null;
  private url: string = getAgentApiBase();
  private retryCount = 0;
  private maxRetries = 5;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;

  connect(url?: string) {
    if (url) this.url = url;
    if (this.es) this.es.close();
    this.es = new EventSource(url ?? this.url);
    this.retryCount = 0; // 重置计数器

    for (const [event, handler] of Object.entries(HANDLERS)) {
      this.es.addEventListener(event, (e: MessageEvent) => {
        try {
          handler(JSON.parse(e.data));
        } catch (err) {
          console.error(`[SSE] Failed to parse ${event}:`, err);
        }
      });
    }

    this.es.onerror = () => {
      if (this.retryCount >= this.maxRetries) {
        console.error('[SSE] Max retries reached, giving up.');
        this.disconnect();
        return;
      }
      const delay = 3000 * Math.pow(2, this.retryCount);
      console.warn(`[SSE] Retry ${this.retryCount + 1}/${this.maxRetries} in ${delay}ms`);
      this.retryTimer = setTimeout(() => {
        this.retryCount++;
        this.connect();
      }, delay);
    };
  }

  disconnect() {
    if (this.retryTimer) clearTimeout(this.retryTimer);
    this.retryTimer = null;
    this.es?.close();
    this.es = null;
    this.retryCount = 0; // disconnect 时重置
  }
}

export const sseConsumer = new SSEConsumer();
