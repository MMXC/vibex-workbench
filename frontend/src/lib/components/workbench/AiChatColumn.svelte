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
	}

	let { onsubmit }: Props = $props();

	let dims = $state<WorkbenchLayoutDims>({
		sidebarLeftPx: 260,
		panelRightPx: 380,
		aiComposerBarPx: 156,
		bottomDockPx: 200,
	});

	let rootEl = $state<HTMLElement | undefined>(undefined);

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
</script>

<div class="ai-column" bind:this={rootEl}>
	<div class="hdr">
		<div>
			<span class="hdr-title">Agent Workspace</span>
			<span class="hdr-sub">Context · Commands · Draft output</span>
		</div>
		<span class="hdr-dot" title="SSE / backend"></span>
	</div>
	<div class="thread-region">
		<ThreadList />
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

	.thread-region {
		flex-shrink: 0;
		max-height: min(28vh, 200px);
		min-height: 88px;
		overflow: hidden;
		display: flex;
		flex-direction: column;
		border-bottom: 1px solid #303746;
		background: rgba(12, 14, 19, 0.32);
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
</style>
