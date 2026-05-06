<!-- ============================================================
VibeX Workbench — Cursor 式：左侧活动栏+文件树 / 中央画布或 Spec / 右侧会话+Composer / 底部 Dock
对齐 prototypes/vibex-ide-chrome-r2.html
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
	import { threadStore, currentThread } from '$lib/stores/thread-store';
	import WorkbenchLayoutResizable from '$lib/components/workbench/WorkbenchLayoutResizable.svelte';
	import ActivityBar from '$lib/components/workbench/ActivityBar.svelte';
	import SpecExplorer from '$lib/components/workbench/SpecExplorer.svelte';
	import LeftPlaceholderView from '$lib/components/workbench/LeftPlaceholderView.svelte';
	import AiChatColumn from '$lib/components/workbench/AiChatColumn.svelte';
	import WorkbenchTitlebar from '$lib/components/workbench/WorkbenchTitlebar.svelte';
	import WorkbenchCenterTabs from '$lib/components/workbench/WorkbenchCenterTabs.svelte';
	import R2Dock from '$lib/components/workbench/R2Dock.svelte';
	import SpecViewer from '$lib/components/workbench/SpecViewer.svelte';
	import StatusBar from '$lib/components/workbench/StatusBar.svelte';
	import SpecSlotDrawer from '$lib/components/workbench/SpecSlotDrawer.svelte';
	import WindowResizeFrame from '$lib/components/workbench/WindowResizeFrame.svelte';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import {
		currentFocusedSpecContext,
		formatSpecContextForPrompt,
		specAgentContextStore,
	} from '$lib/stores/spec-agent-context-store';
	import { specSlotSessionStore } from '$lib/stores/spec-slot-session-store';
	import { runStore } from '$lib/stores/run-store';
	import { eventsOn } from '$lib/wails-runtime';
	import { wailsReadSpecFile } from '$lib/wails-filesystem';
	import { appendOutput, clearOutput, outputText } from '$lib/stores/workspace-output-store';
	import { extractSpecDisplay, type SpecSlotSummary } from '$lib/workbench/spec-display';

	const SSE_URL = import.meta.env.VITE_SSE_URL || 'http://localhost:33338';
	const useMockBackend =
		import.meta.env.VITE_MOCK_SSE === '1' || import.meta.env.VITE_MOCK_SSE === 'true';

	// Status bar state
	let workspaceRoot = $state('—');
	let backendStatus = $state<'connecting' | 'ready' | 'error'>('connecting');
	let workspaceState = $state<'empty' | 'partial' | 'ready'>('empty');

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

	// 监听 Wails backend 事件
	// 启动时从 localStorage 恢复 workspaceRoot
	onMount(() => {
		const saved = localStorage.getItem('vibex-workspace-root');
		if (saved && isLikelyFullPath(saved)) {
			workspaceRoot = saved;
			specExplorerStore.setWorkspaceRoot(saved);
			detectWorkspaceState(saved);
		} else if (saved) {
			console.warn('[Workbench] Ignore invalid workspace root in localStorage:', saved);
			localStorage.removeItem('vibex-workspace-root');
			void restoreBackendWorkspaceRoot();
		} else {
			void restoreBackendWorkspaceRoot();
		}

		const rt = (window as any).runtime;
		if (!rt) return;

		rt.EventsOn('backend:ready', (data: any) => {
			backendStatus = 'ready';
			console.log('[Workbench] Backend ready:', data);
		});
		rt.EventsOn('backend:error', (_msg: any) => {
			backendStatus = 'error';
		});
		rt.EventsOn('workspace:selected', (path: string) => {
			if (!isLikelyFullPath(path)) {
				console.warn('[Workbench] Ignore invalid workspace:selected path:', path);
				return;
			}
			workspaceRoot = path;
			specExplorerStore.setWorkspaceRoot(path);
			detectWorkspaceState(path);
		});

		// menu:run-generate → POST /api/workspace/run-make { target: "generate", workspace_root } → append to output
		eventsOn('menu:run-generate', async () => {
			clearOutput();
			appendOutput(`[run-make] Starting generate...\n`);
			try {
				const res = await fetch('/api/workspace/run-make', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ target: 'generate', workspace_root: workspaceRoot }),
				});
				const data = await res.json();
				if (data.output) appendOutput(data.output);
				if (!data.ok) appendOutput(`[run-make] FAILED: ${data.error ?? 'unknown error'}\n`);
				else appendOutput(`[run-make] Done — exit code: ${data.exitCode}\n`);
			} catch (e) {
				appendOutput(`[run-make] Network error: ${e}\n`);
			}
		});

		// menu:run-lint → POST /api/workspace/run-make { target: "lint-specs", workspace_root } → append to output
		eventsOn('menu:run-lint', async () => {
			clearOutput();
			appendOutput(`[run-make] Starting lint-specs...\n`);
			try {
				const res = await fetch('/api/workspace/run-make', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ target: 'lint-specs', workspace_root: workspaceRoot }),
				});
				const data = await res.json();
				if (data.output) appendOutput(data.output);
				if (!data.ok) appendOutput(`[run-make] FAILED: ${data.error ?? 'unknown error'}\n`);
				else appendOutput(`[run-make] Done — exit code: ${data.exitCode}\n`);
			} catch (e) {
				appendOutput(`[run-make] Network error: ${e}\n`);
			}
		});
	});

	/** 检测 workspace 状态：空 / 半成品 / 就绪 */
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
		} catch {
			// ignore — state stays 'empty'
		}
	}

	function sseConnectPath(tid: string) {
		return useMockBackend
			? `${SSE_URL}/api/sse/threads/${tid}`
			: `${SSE_URL}/api/sse/${tid}`;
	}

	function ensureLocalThread(goal: string): string {
		if ($currentThread?.id) return $currentThread.id;
		const thread = {
			id: crypto.randomUUID(),
			goal: goal.slice(0, 50),
			title: goal.slice(0, 20),
			createdAt: new Date().toISOString(),
		};
		threadStore.addThread(thread as any);
		threadStore.setCurrentThread(thread.id);
		return thread.id;
	}

	function appendLocalWorkbenchMessage(threadId: string, content: string) {
		threadStore.appendMessage(threadId, {
			id: crypto.randomUUID(),
			threadId,
			role: 'assistant',
			content,
			createdAt: new Date().toISOString(),
		});
	}

	function extractCommandQuery(content: string, command: string): string {
		return content
			.trim()
			.slice(command.length)
			.trim()
			.replace(/^["'“”]+|["'“”]+$/g, '')
			.trim();
	}

	function specSearchText(spec: (typeof $specExplorerStore.specs)[number]): string {
		return [
			spec.path,
			spec.name,
			spec.status,
			spec.display?.title,
			spec.display?.summary,
			spec.display?.description,
		]
			.filter(Boolean)
			.join(' ')
			.toLowerCase();
	}

	function scoreSpecMatch(spec: (typeof $specExplorerStore.specs)[number], query: string): number {
		const q = query.toLowerCase();
		const path = spec.path.toLowerCase();
		const name = spec.name.toLowerCase();
		const title = spec.display?.title?.toLowerCase() ?? '';
		if (path === q || name === q || title === q) return 100;
		if (path.endsWith(q) || name.includes(q)) return 80;
		if (title.includes(q)) return 70;
		if (specSearchText(spec).includes(q)) return 40;
		return 0;
	}

	function searchSpecs(query: string) {
		const q = query.trim();
		if (!q) return [];
		return $specExplorerStore.specs
			.map(spec => ({ spec, score: scoreSpecMatch(spec, q) }))
			.filter(item => item.score > 0)
			.sort((a, b) => b.score - a.score || a.spec.path.localeCompare(b.spec.path))
			.slice(0, 8)
			.map(item => item.spec);
	}

	function inferSlotFromText(text: string, slots: SpecSlotSummary[]): SpecSlotSummary | null {
		const lower = text.toLowerCase();
		const candidates: [string, string[]][] = [
			['structure', ['structure', '结构']],
			['input', ['input', '输入', '入参']],
			['output', ['output', '输出', '出参']],
			['constraints', ['constraint', 'constraints', '约束']],
			['prototype', ['prototype', '原型']],
			['implementation', ['implementation', 'implement', '实现']],
		];
		for (const [id, aliases] of candidates) {
			if (aliases.some(alias => lower.includes(alias))) {
				return slots.find(slot => slot.id === id) ?? null;
			}
		}
		return slots.find(slot => slot.status === 'missing' || slot.status === 'empty') ?? slots[0] ?? null;
	}

	async function tryOpenSlotCommand(content: string, workspaceForAgent: string): Promise<boolean> {
		if (!content.trimStart().toLowerCase().startsWith('/open-slot')) return false;
		const focused = currentFocusedSpecContext();
		const selectedPath = $specExplorerStore.selectedSpecPath ?? focused?.path ?? null;
		if (!selectedPath) {
			specAgentContextStore.prefillCommand('/open-slot "请先在左侧或中央选择一个 spec，再指定结构/输入/输出/原型/实现槽位"');
			return true;
		}
		try {
			const raw = focused?.path === selectedPath && focused.content
				? focused.content
				: await wailsReadSpecFile(workspaceForAgent, selectedPath);
			const meta = extractSpecDisplay(raw, selectedPath);
			const slot = inferSlotFromText(content, meta.slots.all);
			if (!slot) return true;
			specAgentContextStore.addSpec(meta, raw);
			specSlotSessionStore.open({ spec: meta, slot, content: raw });
		} catch (e) {
			specAgentContextStore.prefillCommand(`/open-slot "打开槽位失败：${e instanceof Error ? e.message : String(e)}"`);
		}
		return true;
	}

	async function tryOpenSpecCommand(content: string, workspaceForAgent: string): Promise<boolean> {
		const lower = content.trimStart().toLowerCase();
		const command = lower.startsWith('/open-spec')
			? '/open-spec'
			: lower.startsWith('/search-spec')
				? '/search-spec'
				: null;
		if (!command) return false;

		const threadId = ensureLocalThread(content);
		const query = extractCommandQuery(content, command);
		if (!query) {
			appendLocalWorkbenchMessage(threadId, `${command} 需要一个关键词，例如：/open-spec "FEAT-ide-agent-panel"`);
			return true;
		}

		let matches = searchSpecs(query);
		if (matches.length === 0 && $specExplorerStore.specs.length === 0 && workspaceForAgent) {
			await specExplorerStore.loadList(workspaceForAgent);
			matches = searchSpecs(query);
		}
		if (matches.length === 0) {
			appendLocalWorkbenchMessage(threadId, `没有找到匹配 "${query}" 的 spec。可以用 /specs 先查看当前列表。`);
			return true;
		}

		if (command === '/search-spec' || matches.length > 1) {
			const candidates = matches
				.map((spec, index) => {
					const title = spec.display?.title ?? spec.name;
					const summary = spec.display?.summary ? ` — ${spec.display.summary}` : '';
					return `${index + 1}. ${title}\n   path: ${spec.path}${summary}`;
				})
				.join('\n');
			appendLocalWorkbenchMessage(
				threadId,
				`找到 ${matches.length} 个候选，请复制更精确的 path/name 使用 /open-spec 打开：\n${candidates}`
			);
			return true;
		}

		const spec = matches[0];
		try {
			const raw = await wailsReadSpecFile(workspaceForAgent, spec.path);
			const meta = extractSpecDisplay(raw, spec.path);
			specExplorerStore.selectSpec(spec.path);
			specAgentContextStore.addSpec(meta, raw);
			appendLocalWorkbenchMessage(threadId, `已打开 spec：${meta.display.title}\npath: ${spec.path}`);
		} catch (e) {
			appendLocalWorkbenchMessage(threadId, `打开 spec 失败：${e instanceof Error ? e.message : String(e)}`);
		}
		return true;
	}

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

	/** Workbench → Agent Inspector：WebSocket 推送 UI 状态快照（mock 后端时关闭）。 */
	$effect(() => {
		const handle = startWorkbenchInspectorStream(SSE_URL, { disabled: useMockBackend });
		return () => handle.close();
	});

	async function handleSubmit(content: string, mode: string) {
		const workspaceForAgent = workspaceRoot !== '—' ? workspaceRoot : $specExplorerStore.workspaceRoot;
		if (await tryOpenSpecCommand(content, workspaceForAgent)) return;
		if (await tryOpenSlotCommand(content, workspaceForAgent)) return;

		const tid = $currentThread?.id;
		let effectiveTid = tid;
		if (!effectiveTid) {
			const t = {
				id: crypto.randomUUID(),
				goal: content.slice(0, 50),
				title: content.slice(0, 20),
				createdAt: new Date().toISOString(),
			};
			threadStore.addThread(t as any);
			threadStore.setCurrentThread(t.id);
			effectiveTid = t.id;
			const url = sseConnectPath(t.id);
			sseConsumer.disconnect();
			disconnectWorkbenchMessageBridge();
			sseConsumer.connect(url);
			connectWorkbenchMessageBridge(url);
			prevThreadId = t.id;
		}

		const activeRun = $runStore.runs.find(run => run.id === $runStore.active_run_id) ?? null;
		const recentOutput = $outputText.trim().slice(-1200);
		const inputWithContext = `${content}${formatSpecContextForPrompt({
			workspaceRoot: workspaceForAgent,
			specs: $specExplorerStore.specs,
			workbench: {
				workspaceRoot: workspaceForAgent,
				workspaceState,
				backendStatus,
				leftActivity: $specExplorerStore.leftActivity,
				centerView: $specExplorerStore.selectedSpecPath
					? $specExplorerStore.centerView
					: 'dashboard',
				selectedSpecPath: $specExplorerStore.selectedSpecPath,
				dashboardLevel: $specExplorerStore.dashboardLevel,
				specCount: $specExplorerStore.specs.length,
				mode,
				activeRun: activeRun
					? {
							id: activeRun.id,
							status: activeRun.status,
							toolCount: $runStore.toolInvocations.length,
						}
					: undefined,
				recentOutput,
			},
		})}`;

		// 用户消息通过 SSE message.delta(role='user') 由后端回显，作为唯一来源。
		// 不再本地提前创建，避免 SSE bridge echo 时 ID 不同导致重复/排队混乱。
		try {
			const threadKey = effectiveTid || prevThreadId;
			if (useMockBackend) {
				await fetch(`${SSE_URL}/api/runs`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ threadId: threadKey, goal: inputWithContext }),
				});
			} else {
				await fetch(`${SSE_URL}/api/chat`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						threadId: threadKey,
						input: inputWithContext,
						workspaceRoot: workspaceForAgent,
					}),
				});
			}
		} catch (e) {
			console.error('[Workbench] Failed to start run:', e);
		}
	}
</script>

<div class="workbench-root">
	<WorkbenchLayoutResizable>
		{#snippet titlebar()}
			<WorkbenchTitlebar />
		{/snippet}

		{#snippet activityBar()}
			<ActivityBar />
		{/snippet}

		{#snippet sidebar()}
			{#if $specExplorerStore.leftActivity === 'explorer'}
				<SpecExplorer />
			{:else if $specExplorerStore.leftActivity === 'git'}
				<LeftPlaceholderView
					title="源代码管理"
					hint="Git 状态与提交将在此展示；当前为占位。"
				/>
			{:else if $specExplorerStore.leftActivity === 'search'}
				<LeftPlaceholderView title="搜索" hint="全局搜索 specs 与代码；当前为占位。" />
			{:else}
				<LeftPlaceholderView title="扩展" hint="扩展市场与管理；当前为占位。" />
			{/if}
		{/snippet}

		{#snippet main()}
			{#if $specExplorerStore.selectedSpecPath}
				<SpecViewer />
			{:else}
				<WorkbenchCenterTabs />
			{/if}
		{/snippet}

		{#snippet rightPanel()}
			<AiChatColumn onsubmit={handleSubmit} />
		{/snippet}

		{#snippet dock()}
			<R2Dock />
		{/snippet}

		{#snippet statusbar()}
			<StatusBar
				{workspaceRoot}
				{backendStatus}
				{workspaceState}
			/>
		{/snippet}
	</WorkbenchLayoutResizable>
	<SpecSlotDrawer />
	<WindowResizeFrame />
</div>

<style>
	.workbench-root {
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
		--wb-main-bg: #0d0d0d;
		--wb-bg-base: #0d0d0e;
		--wb-bg-secondary: #131314;
		/* R2 :root 对齐（供 Titlebar / Center / Dock 子组件使用） */
		--bg-base: #0d0d0e;
		--bg-panel: #131314;
		--bg-surface: #1a1a1c;
		--bg-hover: rgba(255, 255, 255, 0.05);
		--border: rgba(255, 255, 255, 0.07);
		--text-primary: #e8e8ed;
		--text-secondary: #8a8a8e;
		--text-muted: #555558;
		--brand: #5856d6;
		--font-ui: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
		--font-mono: 'JetBrains Mono', 'Fira Code', monospace;
		--radius-sm: 3px;
		--titlebar-h: 38px;
		--tab-h: 36px;
		--dock-tab-h: 28px;
		--error: #ef4444;
	}
</style>
