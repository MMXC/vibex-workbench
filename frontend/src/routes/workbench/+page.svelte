<!-- ============================================================
VibeX Workbench — Cursor 式：左侧活动栏+文件树 / 中央画布或 Spec / 右侧会话+Composer / 底部 Dock
对齐 prototypes/vibex-ide-chrome-r2.html
开发者维护，gen.py 永不覆盖
============================================================ -->

<script lang="ts">
	import { sseConsumer } from '$lib/sse';
	import {
		connectWorkbenchMessageBridge,
		disconnectWorkbenchMessageBridge,
	} from '$lib/workbench/workbench-message-sse-bridge';
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
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';

	const SSE_URL = import.meta.env.VITE_SSE_URL || 'http://localhost:33335';
	const useMockBackend =
		import.meta.env.VITE_MOCK_SSE === '1' || import.meta.env.VITE_MOCK_SSE === 'true';
	let prevThreadId: string | null = null;

	function sseConnectPath(tid: string) {
		return useMockBackend
			? `${SSE_URL}/api/sse/threads/${tid}`
			: `${SSE_URL}/api/sse/${tid}`;
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

	async function handleSubmit(content: string, mode: string) {
		const tid = $currentThread?.id;
		let effectiveTid = tid;
		if (!effectiveTid) {
			const t = {
				id: crypto.randomUUID(),
				goal: content.slice(0, 50),
				title: content.slice(0, 20),
				createdAt: new Date().toISOString(),
			};
			threadStore.addThread(t);
			threadStore.setCurrentThread(t.id);
			effectiveTid = t.id;
			const url = sseConnectPath(t.id);
			sseConsumer.disconnect();
			disconnectWorkbenchMessageBridge();
			sseConsumer.connect(url);
			connectWorkbenchMessageBridge(url);
			prevThreadId = t.id;
		}

		// 先本地存储用户消息，显示在对话区
		threadStore.appendMessage(effectiveTid!, {
			id: crypto.randomUUID(),
			threadId: effectiveTid!,
			role: 'user',
			content,
			createdAt: new Date().toISOString(),
		});

		try {
			const threadKey = effectiveTid || prevThreadId;
			if (useMockBackend) {
				await fetch(`${SSE_URL}/api/runs`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ threadId: threadKey, goal: content }),
				});
			} else {
				await fetch(`${SSE_URL}/api/chat`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ threadId: threadKey, input: content }),
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
	</WorkbenchLayoutResizable>
</div>

<style>
	.workbench-root {
		width: 100vw;
		height: 100vh;
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
