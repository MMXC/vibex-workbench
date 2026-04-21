// SSE Consumer — 订阅后端事件并分发到 Stores
import { threadStore } from './thread-store';
import { runStore } from './run-store';
import { artifactStore } from './artifact-store';
import { canvasStore } from './canvasStore';

const HANDLERS: Record<string, (data: any) => void> = {
  // ── Thread events ──────────────────────────────────────
  'thread.created': (data: any) => threadStore.addThread(data.thread),
  'thread.updated': (data: any) => threadStore.updateThread(data.thread_id, data.patch),
  'message.created': (data: any) => threadStore.appendMessage(data.thread_id, data.message),

  // ── Run events ─────────────────────────────────────────
  'run.started': (data: any) => runStore.updateRunStatus(data.run_id ?? data.runId, 'executing'),
  'run.planning': (data: any) => runStore.updateRunStatus(data.run_id ?? data.runId, 'planning'),
  'run.completed': (data: any) => runStore.updateRunStatus(data.run_id ?? data.runId, 'completed', new Date().toISOString()),
  'run.failed': (data: any) => runStore.updateRunStatus(data.run_id ?? data.runId, 'failed'),
  'run.cancelled': (data: any) => runStore.updateRunStatus(data.run_id ?? data.runId, 'cancelled'),

  // ── Tool events ───────────────────────────────────────
  'tool.called': (data: any) => {
    const id = data.invocationId ?? data.call_id;
    runStore.addToolInvocation({
      id,
      run_id: data.runId ?? data.run_id,
      tool_name: data.toolName ?? data.tool,
      args: data.args,
      status: 'running',
    });
    // Canvas node creation via generic node.added
    canvasStore.addNode(data.thread_id ?? 'default', 'tool', data.toolName ?? data.tool, 'sequence', {
      status: 'running',
      args: JSON.stringify(data.args ?? {}),
    });
  },
  'tool.completed': (data: any) => {
    const id = data.invocationId ?? data.call_id;
    runStore.updateToolInvocation(id, {
      status: 'completed',
      result: data.result,
      finished_at: new Date().toISOString(),
    });
    canvasStore.updateNode?.(id, { status: 'done' });
  },

  // ── Artifact events ────────────────────────────────────
  'artifact.created': (data: any) => artifactStore.create({
    name: data.name, type: data.type, content: data.content ?? '',
    mime_type: data.mime_type, tags: data.tags ?? [], thread_id: data.thread_id, run_id: data.run_id,
  }),
  'artifact.updated': (data: any) => artifactStore.update(data.artifact_id, { content: data.content }),

  // ── Canvas events ─────────────────────────────────────
  'canvas.spec_created': (data: any) => {
    // Generic canvas node for spec creation events
    canvasStore.addNode(
      data.thread_id ?? data.spec_id ?? 'default',
      'spec_created',
      data.title ?? 'New Spec',
      'sequence',
      { ...data }
    );
    console.info('[SSE] canvas.spec_created:', data);
  },
  'canvas.tdd_nodes': (data: any) => {
    // TDD test case nodes from tdd_design
    canvasStore.addNode(
      data.spec_id ?? 'default',
      'tdd_nodes',
      `TDD: ${data.test_count ?? 0} tests`,
      'iteration',
      { test_file: data.test_file, phases: data.phases, nodes: data.nodes, ...data }
    );
    console.info('[SSE] canvas.tdd_nodes:', data);
  },
  'canvas.tdd_cycle': (data: any) => {
    // TDD cycle update (RED/GREEN/REFACTOR) from tdd_run/tdd_iterate
    canvasStore.addNode(
      data.spec_id ?? 'default',
      'tdd_cycle',
      `TDD ${data.phase ?? '?'}: ${data.passed ?? 0}p ${data.failed ?? 0}f`,
      data.phase === 'GREEN' ? 'sequence' : 'iteration',
      { phase: data.phase, color: data.color, passed: data.passed, failed: data.failed, ...data }
    );
    console.info('[SSE] canvas.tdd_cycle:', data);
  },
  // ── Generic canvas sync ────────────────────────────────
  'node.added': (data: any) => {
    canvasStore.addNode(
      data.threadId ?? data.thread_id ?? 'default',
      data.eventId ?? 'node',
      data.label ?? 'Node',
      data.nodeType ?? 'sequence',
      data.payload ?? {}
    );
  },
  'node.removed': (data: any) => {
    if (canvasStore.removeNode) canvasStore.removeNode(data.node_id);
  },
  'edge.added': (data: any) => {
    if (canvasStore.addEdge) canvasStore.addEdge(data.edge);
  },

  // ── Agent self-reflection ─────────────────────────────
  'agent.self_reflection': (data: any) => {
    // Display as a system message in thread
    const threadId = data.thread_id ?? 'default';
    threadStore.appendMessage(threadId, {
      id: `refl-${Date.now()}`,
      role: 'system',
      content: `🔄 Self-reflection: ${data.summary ?? data}`,
      timestamp: new Date().toISOString(),
    });
    console.info('[SSE] agent.self_reflection:', data);
  },
};

class SSEConsumer {
  private es: EventSource | null = null;

  connect(url: string) {
    if (this.es) this.es.close();
    this.es = new EventSource(url);
    for (const [event, handler] of Object.entries(HANDLERS)) {
      this.es.addEventListener(event, (e: MessageEvent) => {
        try { handler(JSON.parse(e.data)); }
        catch (err) { console.error('[SSE]', event, err); }
      });
    }
    // Register canvas.<*> dynamic events (e.g. canvas_update emits canvas.<eventType>)
    this.es.addEventListener('message', (e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data);
        if (parsed?.event?.startsWith('canvas.')) {
          const dynamicHandler = HANDLERS[parsed.event];
          if (dynamicHandler) dynamicHandler(parsed.data ?? parsed);
        }
      } catch {}
    });
    this.es.onerror = () => {
      console.warn('[SSE] reconnecting...');
      setTimeout(() => { if (this.es) this.connect(url); }, 3000);
    };
  }

  disconnect() {
    this.es?.close();
    this.es = null;
  }
}

export const sseConsumer = new SSEConsumer();
