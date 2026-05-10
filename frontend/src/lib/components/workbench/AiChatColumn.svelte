<!-- 右侧 AI 栏：会话历史（线程） + 对话内容 + Composer（对齐 Cursor） -->
<script lang="ts">
	import ThreadList from '$lib/components/workbench/ThreadList.svelte';
	import ConversationPanel from '$lib/components/workbench/ConversationPanel.svelte';
	import Composer from '$lib/components/workbench/Composer.svelte';
	import { browser } from '$app/environment';
	import {
		workbenchLayoutStore,
		workbenchRightPanelHeight,
		type WorkbenchLayoutDims,
	} from '$lib/stores/workbench-layout-store';

	interface Props {
		onsubmit?: (content: string, mode: string) => Promise<void> | void;
	onquickaction?: (action: 'detect' | 'init' | 'align' | 'validate') => void;
	}

let { onsubmit, onquickaction }: Props = $props();

	let dims = $state<WorkbenchLayoutDims>({
		sidebarLeftPx: 260,
		panelRightPx: 380,
		aiComposerBarPx: 156,
		bottomDockPx: 200,
	});

	let rootEl = $state<HTMLElement | undefined>(undefined);
let showHistory = $state(false);

	$effect(() => {
		const unsub = workbenchLayoutStore.subscribe(v => {
			dims = v;
		});
		return unsub;
	});

	$effect(() => {
		if (!browser || !rootEl) return;
		const ro = new ResizeObserver(entries => {
			const h = entries[0]?.contentRect.height ?? 0;
			workbenchRightPanelHeight.set(Math.round(h));
		});
		ro.observe(rootEl);
		workbenchRightPanelHeight.set(rootEl.clientHeight);
		return () => ro.disconnect();
	});

	function beginComposerResize(e: PointerEvent) {
		if (!browser || window.matchMedia('(max-width: 767px)').matches) return;
		e.preventDefault();
		const startY = e.clientY;
		const startH = dims.aiComposerBarPx;
		const inner = rootEl?.clientHeight ?? 400;
		function move(ev: PointerEvent) {
			const innerH = rootEl?.clientHeight ?? inner;
			workbenchLayoutStore.previewAiComposerBarPx(startH + (startY - ev.clientY), innerH);
		}
		function end() {
			window.removeEventListener('pointermove', move);
			window.removeEventListener('pointerup', end);
			workbenchLayoutStore.commit();
		}
		window.addEventListener('pointermove', move);
		window.addEventListener('pointerup', end);
	}

	function triggerQuickAction(action: 'detect' | 'init' | 'align' | 'validate') {
		onquickaction?.(action);
	}
</script>

<div class="ai-column" bind:this={rootEl}>
	<div class="hdr">
		<div>
			<span class="hdr-title">Agent Workspace</span>
			<span class="hdr-sub">Focused Chat · Actions · Composer</span>
		</div>
		<div class="hdr-tools">
			<span class="hdr-dot" title="SSE / backend"></span>
			<button
				type="button"
				class="history-btn"
				class:active={showHistory}
				title="会话历史"
				aria-label="打开会话历史"
				onclick={() => (showHistory = !showHistory)}
			>
				🕘
			</button>
		</div>
	</div>
	<div class="quick-actions" aria-label="快捷操作">
		<button type="button" onclick={() => triggerQuickAction('detect')}>检测状态</button>
		<button type="button" onclick={() => triggerQuickAction('init')}>新项目初始化</button>
		<button type="button" onclick={() => triggerQuickAction('align')}>旧项目对齐</button>
		<button type="button" onclick={() => triggerQuickAction('validate')}>治理校验</button>
	</div>
	<div class="chat-region">
		<ConversationPanel />
	</div>
	<button
		type="button"
		class="split-ai"
		aria-label="拖动调整对话区与 Composer 高度"
		onpointerdown={beginComposerResize}
	></button>
	<div class="composer-region" style:height="{dims.aiComposerBarPx}px">
		<Composer {onsubmit} />
	</div>

	{#if showHistory}
		<div
			class="history-overlay"
			role="button"
			tabindex="0"
			aria-label="关闭会话历史抽屉"
			onclick={() => (showHistory = false)}
			onkeydown={(e) => (e.key === 'Escape' || e.key === 'Enter') && (showHistory = false)}
		>
			<div
				class="history-drawer"
				role="dialog"
				tabindex="-1"
				aria-label="会话历史"
				onclick={(e) => e.stopPropagation()}
			>
				<div class="history-head">
					<strong>会话历史</strong>
					<button type="button" onclick={() => (showHistory = false)}>关闭</button>
				</div>
				<div class="history-list-wrap">
					<ThreadList />
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.ai-column {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background:
			radial-gradient(circle at 85% 4%, rgba(114, 214, 208, 0.08), transparent 34%),
			#151820;
		border-left: 1px solid #303746;
		overflow: hidden;
	}

	.hdr {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		padding: 14px 14px 12px;
		border-bottom: 1px solid #303746;
		background: rgba(28, 32, 42, 0.78);
	}

	.hdr-title {
		display: block;
		font-size: 13px;
		font-weight: 800;
		letter-spacing: 0.02em;
		text-transform: uppercase;
		color: #eef0f5;
	}

	.hdr-sub {
		display: block;
		margin-top: 2px;
		font-size: 10.5px;
		color: #a3abb9;
		white-space: nowrap;
	}

	.hdr-dot {
		width: 8px;
		height: 8px;
		border-radius: 999px;
		background: #22c55e;
		box-shadow: 0 0 12px rgba(34, 197, 94, 0.65);
	}

	.hdr-tools {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.history-btn {
		width: 28px;
		height: 28px;
		border-radius: 8px;
		border: 1px solid #303746;
		background: #12151c;
		color: #a3abb9;
		cursor: pointer;
	}

	.history-btn:hover,
	.history-btn.active {
		color: #eef0f5;
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.18);
	}

	.quick-actions {
		flex-shrink: 0;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
		padding: 10px 12px;
		border-bottom: 1px solid #303746;
		background: rgba(12, 14, 19, 0.26);
	}

	.quick-actions button {
		border: 1px solid #3a4356;
		background: #11141b;
		color: #d7deea;
		border-radius: 10px;
		padding: 6px 8px;
		font-size: 11.5px;
		cursor: pointer;
	}

	.quick-actions button:hover {
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.14);
	}

	.chat-region {
		flex: 1;
		min-height: 120px;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.split-ai {
		flex-shrink: 0;
		height: 5px;
		margin: 0;
		padding: 0;
		border: none;
		cursor: row-resize;
		touch-action: none;
		background: #0f1117;
		z-index: 4;
	}

	.split-ai:hover,
	.split-ai:active {
		background: #7aa2ff;
	}

	.composer-region {
		flex-shrink: 0;
		min-height: 0;
		overflow: hidden;
		display: flex;
		flex-direction: column;
		border-top: 1px solid #303746;
	}

	.composer-region :global(.composer) {
		flex: 1;
		min-height: 0;
		overflow: auto;
	}

	.history-overlay {
		position: absolute;
		inset: 0;
		background: rgba(7, 10, 16, 0.52);
		z-index: 30;
		display: flex;
		justify-content: flex-end;
	}

	.history-drawer {
		width: min(340px, 92%);
		height: 100%;
		background: #0f1218;
		border-left: 1px solid #303746;
		display: flex;
		flex-direction: column;
	}

	.history-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 10px 12px;
		border-bottom: 1px solid #303746;
	}

	.history-head strong {
		font-size: 12px;
		color: #eef0f5;
	}

	.history-head button {
		border: 1px solid #303746;
		background: #11141b;
		color: #a3abb9;
		border-radius: 8px;
		padding: 4px 8px;
		cursor: pointer;
	}

	.history-list-wrap {
		flex: 1;
		min-height: 0;
		overflow: hidden;
	}
</style>
