// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: specs
//     生成时间: 2026-04-19
//     ⚠️  不要直接编辑此文件
// ============================================================

// SSE Consumer — 订阅后端事件并分发到 Stores
import { threadStore } from '$lib/stores/thread-store';
import { runStore } from '$lib/stores/run-store';
import { artifactStore } from '$lib/stores/artifact-store';
import { canvasStore } from '$lib/stores/canvas-store';

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
    // E3-U1: 追踪 tool invocation
    runStore.addToolInvocation({
      id: data.invocationId,
      run_id: data.runId,
      tool_name: data.toolName,
      tool_display_name: data.toolName,
      args: data.args,
      status: 'running',
      order: data.order ?? 0,
    });
    // Canvas node creation
    canvasStore.addNode({
      id: data.invocationId,
      type: 'tool',
      position: { x: 150 + Math.random() * 300, y: 200 + Math.random() * 200 },
      parent_id: data.runId,
      data: {
        label: data.toolName,
        status: 'running',
        args: data.args,
      },
    });
  },
  'tool.completed': (data: any) => {
    // E3-U1: 更新 tool invocation 状态
    runStore.updateToolInvocation(data.invocationId, {
      status: 'completed',
      result: data.result,
      finished_at: new Date().toISOString(),
    });
    canvasStore.updateNode(data.invocationId, { data: { status: 'completed', result: data.result } });
  },
  'tool.failed': (data: any) => {
    // E3-U1: 更新 tool invocation 错误状态
    runStore.updateToolInvocation(data.invocationId, {
      status: 'failed',
      error: data.error,
      finished_at: new Date().toISOString(),
    });
    canvasStore.updateNode(data.invocationId, { data: { status: 'failed', error: data.error } });
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
};

class SSEConsumer {
  private es: EventSource | null = null;
  private url: string = import.meta.env.VITE_SSE_URL || 'http://localhost:33335';
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
