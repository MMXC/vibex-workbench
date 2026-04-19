// SSE Consumer — 订阅后端事件并分发到 Stores
import { threadStore } from './thread-store';
import { runStore } from './run-store';
import { artifactStore } from './artifact-store';
import { canvasStore } from './canvas-store';

const HANDLERS: Record<string, (data: any) => void> = {
  'thread.created': (data: any) => threadStore.addThread(data.thread),
  'thread.updated': (data: any) => threadStore.updateThread(data.thread_id, data.patch),
  'message.created': (data: any) => threadStore.appendMessage(data.thread_id, data.message),
  'run.started': (data: any) => runStore.updateRunStatus(data.run_id, 'executing'),
  'run.planning': (data: any) => runStore.updateRunStatus(data.run_id, 'planning'),
  'run.completed': (data: any) => runStore.updateRunStatus(data.run_id, 'completed', new Date().toISOString()),
  'run.failed': (data: any) => runStore.updateRunStatus(data.run_id, 'failed'),
  'run.cancelled': (data: any) => runStore.updateRunStatus(data.run_id, 'cancelled'),
  'artifact.created': (data: any) => artifactStore.create({
    name: data.name, type: data.type, content: data.content ?? '',
    mime_type: data.mime_type, tags: data.tags ?? [], thread_id: data.thread_id, run_id: data.run_id,
  }),
  'artifact.updated': (data: any) => artifactStore.update(data.artifact_id, { content: data.content }),
  'node.added': (data: any) => canvasStore.addNode(data.node),
  'node.removed': (data: any) => canvasStore.removeNode(data.node_id),
  'edge.added': (data: any) => canvasStore.addEdge(data.edge),
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
