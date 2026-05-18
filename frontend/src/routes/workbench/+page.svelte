<!-- ============================================================
VibeX Workbench — 三栏布局：左 AgentHub / 中 SpecPilot / 右 ProtoDesign
对齐 SPEC.md L1 架构
开发者维护，gen.py 永不覆盖
============================================================ -->

<script lang="ts">
	import { onMount } from 'svelte';
	import { sseConsumer } from '$lib/sse';
	import {
		connectWorkbenchMessageBridge,
		disconnectWorkbenchMessageBridge,
	} from '$lib/workbench/workbench-message-sse-bridge';
	import { startWorkbenchInspectorStream } from '$lib/workbench/workbench-inspector-ws';
	import { threadStore, currentThread, currentMessages } from '$lib/stores/thread-store';
	import LayoutShell from '$lib/components/layout/LayoutShell.svelte';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { currentFocusedSpecContext, formatSpecContextForPrompt, specAgentContextStore } from '$lib/stores/spec-agent-context-store';
	import { parseLeadingAgentSlashCommand, specSlotSessionStore } from '$lib/stores/spec-slot-session-store';
	import { runStore } from '$lib/stores/run-store';
	import { eventsOn } from '$lib/wails-runtime';
	import { wailsReadSpecFile } from '$lib/wails-filesystem';
	import { appendOutput, clearOutput, outputText } from '$lib/stores/workspace-output-store';
	import { extractSpecDisplay, type SpecSlotSummary } from '$lib/workbench/spec-display';
	import { agentApiUrl, getAgentApiBase, postAgentChat } from '$lib/runtime/agent-transport';
	import { uiPreferencesStore } from '$lib/stores/ui-preferences-store';

	const useMockBackend =
		import.meta.env.VITE_MOCK_SSE === '1' || import.meta.env.VITE_MOCK_SSE === 'true';

	let workspaceRoot = $state('—');
	let backendStatus = $state<'connecting' | 'ready' | 'error'>('connecting');
	let workspaceState = $state<'empty' | 'partial' | 'ready'>('empty');
	let canvasAutoUiEnabled = $state(true);
	let threadMessages = $state<{ role: string; content: string }[]>([]);

	function toggleCanvasAutoUi() {
		uiPreferencesStore.toggleCanvasAutoUi();
	}
	const CANVAS_POST_TAG = '[canvas-postprocess]';
	const processedCompletedRuns = new Set<string>();
	let prevThreadId: string | null = null;

	function isLikelyFullPath(p: string): boolean {
		return !!p && (p.includes('/') || p.includes('\\'));
	}

	async function restoreBackendWorkspaceRoot() {
		try {
			const res = await fetch('/api/workspace/detect-state');
			if (!res.ok) return;
			const data = await res.json();
			const root = data.workspaceRoot ?? data.workspace_root;
			if (!isLikelyFullPath(root)) return;
			localStorage.setItem('vibex-workspace-root', root);
			workspaceRoot = root;
			specExplorerStore.setWorkspaceRoot(root);
			workspaceState = data.state === 'ready' ? 'ready' : data.state === 'half' ? 'partial' : 'empty';
		} catch (e) {
			console.warn('[Workbench] failed to restore backend workspace root:', e);
		}
	}

	onMount(() => {
		const saved = localStorage.getItem('vibex-workspace-root');
		if (saved && isLikelyFullPath(saved)) {
			workspaceRoot = saved;
			specExplorerStore.setWorkspaceRoot(saved);
			detectWorkspaceState(saved);
		} else if (saved) {
			localStorage.removeItem('vibex-workspace-root');
			void restoreBackendWorkspaceRoot();
		} else {
			void restoreBackendWorkspaceRoot();
		}

		const rt = (window as any).runtime;
		if (!rt) return;

		rt.EventsOn('backend:ready', () => { backendStatus = 'ready'; });
		rt.EventsOn('backend:error', () => { backendStatus = 'error'; });
		rt.EventsOn('workspace:selected', (path: string) => {
			if (!isLikelyFullPath(path)) return;
			workspaceRoot = path;
			specExplorerStore.setWorkspaceRoot(path);
			detectWorkspaceState(path);
		});

		eventsOn('menu:run-generate', async () => {
			clearOutput();
			appendOutput(`[run-make] Starting generate...\n`);
			try {
				const res = await fetch('/api/workspace/run-make', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ target: 'generate', workspace_root: workspaceRoot }),
				});
				const d = await res.json();
				if (d.output) appendOutput(d.output);
				if (!d.ok) appendOutput(`[run-make] FAILED: ${d.error ?? 'unknown error'}\n`);
				else appendOutput(`[run-make] Done — exit code: ${d.exitCode}\n`);
			} catch (e) { appendOutput(`[run-make] Network error: ${e}\n`); }
		});

		eventsOn('menu:run-lint', async () => {
			clearOutput();
			appendOutput(`[run-make] Starting lint-specs...\n`);
			try {
				const res = await fetch('/api/workspace/run-make', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ target: 'lint-specs', workspace_root: workspaceRoot }),
				});
				const d = await res.json();
				if (d.output) appendOutput(d.output);
				if (!d.ok) appendOutput(`[run-make] FAILED: ${d.error ?? 'unknown error'}\n`);
				else appendOutput(`[run-make] Done — exit code: ${d.exitCode}\n`);
			} catch (e) { appendOutput(`[run-make] Network error: ${e}\n`); }
		});
	});

	async function detectWorkspaceState(root: string) {
		try {
			const res = await fetch(`/api/workspace/detect-state`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ workspace_root: root }),
			});
			if (res.ok) {
				const data = await res.json();
				workspaceState = data.state ?? 'empty';
			}
		} catch { /* ignore */ }
	}

	function sseConnectPath(tid: string) {
		return useMockBackend ? agentApiUrl(`/api/sse/threads/${tid}`) : agentApiUrl(`/api/sse/${tid}`);
	}

	function ensureLocalThread(goal: string): string {
		if ($currentThread?.id) return $currentThread.id;
		const thread = { id: crypto.randomUUID(), goal: goal.slice(0, 50), title: goal.slice(0, 20), createdAt: new Date().toISOString() };
		threadStore.addThread(thread as any);
		threadStore.setCurrentThread(thread.id);
		return thread.id;
	}

	$effect(() => {
		const unsub = uiPreferencesStore.subscribe(v => { canvasAutoUiEnabled = v.canvasAutoUiEnabled; });
		return unsub;
	});

	async function triggerCanvasPostProcess(threadId: string, workspaceForAgent: string) {
		const latestExcerpt = [...threadMessages].reverse().find(m => m.role === 'assistant')?.content ?? '';
		if (!latestExcerpt.trim()) return;
		if (!/(\/[a-z-]+|```|^\s*[-*]\s+|^\s*\d+\.\s+)/im.test(latestExcerpt)) return;
		const postInput = `${CANVAS_POST_TAG}\n将以下 assistant 结果转换为 CanvasSkillPayload(JSON)…\n\nworkspace_root: ${workspaceForAgent || 'unknown'}\n\nsource_assistant_result:\n${latestExcerpt.slice(-2200)}`;
		try {
			await postAgentChat({ threadId, input: postInput, workspaceRoot: workspaceForAgent });
		} catch { /* ignore */ }
	}

	$effect(() => {
		const unsub = currentMessages.subscribe(messages => {
			threadMessages = messages.map(m => ({ role: m.role, content: m.content ?? '' }));
		});
		return unsub;
	});

	$effect(() => {
		const tid = $currentThread?.id ?? null;
		if (tid && tid !== prevThreadId) {
			sseConsumer.disconnect();
			disconnectWorkbenchMessageBridge();
			const url = sseConnectPath(tid);
			sseConsumer.connect(url);
			connectWorkbenchMessageBridge(url);
			prevThreadId = tid;
		}
		return () => {
			sseConsumer.disconnect();
			disconnectWorkbenchMessageBridge();
			prevThreadId = null;
		};
	});

	$effect(() => {
		const tid = $currentThread?.id;
		if (!tid || !canvasAutoUiEnabled) return;
		const workspaceForAgent = workspaceRoot !== '—' ? workspaceRoot : $specExplorerStore.workspaceRoot;
		for (const run of $runStore.runs) {
			if (run.thread_id !== tid || run.status !== 'completed') continue;
			if (processedCompletedRuns.has(run.id) || (run.goal ?? '').includes(CANVAS_POST_TAG)) continue;
			processedCompletedRuns.add(run.id);
			void triggerCanvasPostProcess(tid, workspaceForAgent);
			break;
		}
	});

	$effect(() => {
		const handle = startWorkbenchInspectorStream(getAgentApiBase(), { disabled: useMockBackend });
		return () => { if (handle && 'close' in handle) handle.close(); };
	});
</script>

<div class="wb-root">
	<LayoutShell {backendStatus} {workspaceState} {canvasAutoUiEnabled} ontoggleCanvasAutoUi={toggleCanvasAutoUi} />
</div>

<style>
	.wb-root {
		width: 100%;
		height: 100%;
		overflow: hidden;
		--wb-base: #0d0d0e;
		--wb-panel-bg: #131314;
		--wb-border: rgba(255, 255, 255, 0.07);
		--wb-brand: #5856d6;
		--wb-text: #e8e8ed;
		--wb-text-sec: #8a8a8e;
		--wb-muted: #555558;
		--wb-main-bg: #0d0d0e;
		--wb-bg-base: #0d0d0e;
		--wb-bg-secondary: #131314;
		--wb-bg: #0b0d12;
		--wb-accent: #72d6d0;
		--bg-base: #0d0d0e;
		--bg-panel: #131314;
		--bg-surface: #1a1a1c;
		--bg-hover: rgba(255, 255, 255, 0.05);
		--border: rgba(255, 255, 255, 0.07);
		--text-primary: #e8e8ed;
		--text-secondary: #8a8a8e;
		--text-muted: #555558;
		--brand: #5856d6;
		--accent-green: #87cf8a;
		--accent-yellow: #efc66b;
		--accent-red: #e16d75;
		--accent-orange: #f09a6a;
	}
</style>
