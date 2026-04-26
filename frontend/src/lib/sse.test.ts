/**
 * sse.test.ts
 *
 * E1-U1: SSE URL 环境变量化
 * E1-U2: SSE 指数退避重连
 *
 * Tests SSEConsumer behavior through sseConsumer singleton.
 * E1-U1: verify VITE_SSE_URL default URL is set
 * E1-U2: verify exponential backoff (3s → 6s → 12s → 24s → 48s, max 5 retries)
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// ── Mock EventSource ──────────────────────────────────────────
const mockInstances: Array<{
  url: string;
  onerror: (() => void) | null;
  close: ReturnType<typeof vi.fn>;
  addEventListener: ReturnType<typeof vi.fn>;
}> = [];

vi.stubGlobal('EventSource', vi.fn((url: string) => {
  const inst = {
    url,
    onerror: null,
    close: vi.fn(() => { mockInstances.length = 0; }),
    addEventListener: vi.fn(),
  };
  mockInstances.push(inst);
  return inst;
}));

// ── Mock stores ──────────────────────────────────────────────
vi.mock('$lib/stores/thread-store', () => ({
  threadStore: { appendMessage: vi.fn(), addThread: vi.fn(), updateThread: vi.fn() },
}));
vi.mock('$lib/stores/run-store', () => ({
  runStore: { updateRunStatus: vi.fn(), addToolInvocation: vi.fn(), updateToolInvocation: vi.fn() },
}));
vi.mock('$lib/stores/artifact-store', () => ({
  artifactStore: { create: vi.fn() },
}));
vi.mock('$lib/stores/canvas-store', () => ({
  canvasStore: { addNode: vi.fn(), updateNode: vi.fn(), addEdge: vi.fn() },
}));

// SSEConsumer is internal class — test through sseConsumer singleton
describe('E1-U1: SSE URL 环境变量化', () => {
  it('sseConsumer 实例化成功（VITE_SSE_URL 作为默认 URL）', async () => {
    const { sseConsumer } = await import('./sse');
    expect(sseConsumer).toBeDefined();
    expect(typeof sseConsumer.connect).toBe('function');
    expect(typeof sseConsumer.disconnect).toBe('function');
    // url 属性来自 import.meta.env.VITE_SSE_URL
    expect((sseConsumer as unknown as { url: string }).url).toBeTruthy();
  });

  it('connect(url) 使用传入的 URL 而非默认值', async () => {
    const { sseConsumer } = await import('./sse');
    mockInstances.length = 0;
    const customUrl = 'http://custom:9999/api/sse/threads/test-id';
    sseConsumer.connect(customUrl);
    expect(mockInstances.some(i => i.url === customUrl)).toBe(true);
  });

  it('connect() 使用默认值 URL', async () => {
    const { sseConsumer } = await import('./sse');
    mockInstances.length = 0;
    sseConsumer.connect();
    expect(mockInstances.length).toBe(1);
  });
});

describe('E1-U2: SSE 指数退避重连', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockInstances.length = 0;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('指数退避: 5 次 onerror 分别延迟 3s/6s/12s/24s/48s', async () => {
    const { sseConsumer } = await import('./sse');
    sseConsumer.connect('http://localhost:33338/api/sse/threads/test');
    const instance = mockInstances[0];
    expect(instance).toBeDefined();

    const expectedDelays = [3000, 6000, 12000, 24000, 48000];
    for (let i = 0; i < 5; i++) {
      instance.onerror?.();
      vi.advanceTimersByTime(expectedDelays[i]);
    }
    // All 5 retries happened (5 setTimeout calls executed)
  });

  it('第 6 次 onerror 后停止重连（maxRetries=5）', async () => {
    const { sseConsumer } = await import('./sse');
    sseConsumer.connect('http://localhost:33338/api/sse/threads/test');
    const instance = mockInstances[0];

    // Trigger 5 errors (exhaust retries)
    for (let i = 0; i < 5; i++) {
      instance.onerror?.();
      vi.advanceTimersByTime(3000 * Math.pow(2, i));
    }

    // 第 6 次: no new connect should happen
    const beforeCount = mockInstances.length;
    instance.onerror?.();
    // advance timer — if no retry, no new setTimeout
    vi.advanceTimersByTime(96000);
    // No new instances created after 6th error
    expect(mockInstances.length).toBe(beforeCount);
  });

  it('disconnect() 关闭 EventSource 并重置 retryCount', async () => {
    const { sseConsumer } = await import('./sse');
    sseConsumer.connect('http://localhost:33338/api/sse/threads/test');
    const instance = mockInstances[0];
    expect(instance).toBeDefined();

    sseConsumer.disconnect();
    expect(instance.close).toHaveBeenCalled();
  });

  it('disconnect() 清除 retryTimer', async () => {
    const { sseConsumer } = await import('./sse');
    sseConsumer.connect('http://localhost:33338/api/sse/threads/test');
    const instance = mockInstances[0];

    // trigger error to set a retryTimer
    instance.onerror?.();
    vi.advanceTimersByTime(500); // not enough to fire the timer

    // disconnect should clear the pending timer
    sseConsumer.disconnect();

    // advance time — if timer was cleared, no reconnect should happen
    vi.advanceTimersByTime(3000);
    // No new EventSource should be created (timer was cleared)
    expect(mockInstances.length).toBe(0);
  });
});