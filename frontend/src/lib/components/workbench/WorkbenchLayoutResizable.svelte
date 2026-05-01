<!-- ============================================================
WorkbenchLayoutResizable — Cursor 式：活动栏+侧栏 | 主区 | 右侧 AI 栏；底部 Dock
Spec: workbench-layout_resize_feature + workbench-ide-chrome
开发者维护，gen.py 永不覆盖
============================================================ -->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { browser } from '$app/environment';
	import {
		workbenchLayoutStore,
		workbenchMainAreaHeight,
		type WorkbenchLayoutDims,
	} from '$lib/stores/workbench-layout-store';

	interface Props {
		titlebar?: Snippet;
		activityBar?: Snippet;
		sidebar?: Snippet;
		main?: Snippet;
		rightPanel?: Snippet;
		dock?: Snippet;
		statusbar?: Snippet;
	}

	let { titlebar, activityBar, sidebar, main, rightPanel, dock, statusbar }: Props = $props();

	let dims = $state<WorkbenchLayoutDims>({
		sidebarLeftPx: 260,
		panelRightPx: 380,
		aiComposerBarPx: 156,
		bottomDockPx: 200,
	});

	let mainEl = $state<HTMLElement | undefined>(undefined);

	$effect(() => {
		const unsub = workbenchLayoutStore.subscribe(v => {
			dims = v;
		});
		return unsub;
	});

	$effect(() => {
		if (!browser || !mainEl) return;
		const ro = new ResizeObserver(entries => {
			const h = entries[0]?.contentRect.height ?? 0;
			workbenchMainAreaHeight.set(Math.round(h));
		});
		ro.observe(mainEl);
		workbenchMainAreaHeight.set(mainEl.clientHeight);
		return () => ro.disconnect();
	});

	function beginLeftResize(e: PointerEvent) {
		if (window.matchMedia('(max-width: 767px)').matches) return;
		e.preventDefault();
		const startX = e.clientX;
		const startW = dims.sidebarLeftPx;
		function move(ev: PointerEvent) {
			workbenchLayoutStore.previewSidebarLeftPx(startW + (ev.clientX - startX));
		}
		function end() {
			window.removeEventListener('pointermove', move);
			window.removeEventListener('pointerup', end);
			workbenchLayoutStore.commit();
		}
		window.addEventListener('pointermove', move);
		window.addEventListener('pointerup', end);
	}

	function beginRightResize(e: PointerEvent) {
		if (window.matchMedia('(max-width: 767px)').matches) return;
		e.preventDefault();
		const startX = e.clientX;
		const startW = dims.panelRightPx;
		function move(ev: PointerEvent) {
			workbenchLayoutStore.previewPanelRightPx(startW - (ev.clientX - startX));
		}
		function end() {
			window.removeEventListener('pointermove', move);
			window.removeEventListener('pointerup', end);
			workbenchLayoutStore.commit();
		}
		window.addEventListener('pointermove', move);
		window.addEventListener('pointerup', end);
	}

	/** LR2-003：Dock 顶边横向分隔条 → bottomDockPx（与 layout_resize v2 一致） */
	function beginDockResize(e: PointerEvent) {
		if (window.matchMedia('(max-width: 767px)').matches) return;
		e.preventDefault();
		const startY = e.clientY;
		const startDock = dims.bottomDockPx;
		function move(ev: PointerEvent) {
			workbenchLayoutStore.previewBottomDockPx(startDock - (ev.clientY - startY));
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

<div class="wb-root">
	{#if titlebar}
		<div class="wb-titlebar-slot">
			{@render titlebar()}
		</div>
	{/if}
	<div class="wb-row">
		<div class="wb-left-composite" style:width="{dims.sidebarLeftPx}px">
			<div class="wb-activity-slot">
				{#if activityBar}
					{@render activityBar()}
				{/if}
			</div>
			<div class="wb-sidebar-slot">
				{#if sidebar}
					{@render sidebar()}
				{/if}
			</div>
		</div>

		<button
			type="button"
			class="wb-gutter wb-gutter-v"
			aria-label="拖动调整左侧栏总宽度"
			onpointerdown={beginLeftResize}
		></button>

		<main class="wb-main" bind:this={mainEl}>
			{#if main}
				{@render main()}
			{/if}
		</main>

		<button
			type="button"
			class="wb-gutter wb-gutter-v wb-gutter-main-right"
			aria-label="拖动调整右侧 AI 栏宽度"
			onpointerdown={beginRightResize}
		></button>

		<aside class="wb-right" style:width="{dims.panelRightPx}px">
			{#if rightPanel}
				{@render rightPanel()}
			{/if}
		</aside>
	</div>

	<button
		type="button"
		class="wb-gutter wb-gutter-h wb-gutter-dock"
		aria-label="拖动调整底部 Dock 高度"
		onpointerdown={beginDockResize}
	></button>

	<div class="wb-dock-wrap" style:height="{dims.bottomDockPx}px">
		{#if dock}
			{@render dock()}
		{/if}
	</div>

	{#if statusbar}
		{@render statusbar()}
	{/if}
</div>

<style>
	.wb-titlebar-slot {
		flex-shrink: 0;
		z-index: 100;
	}

	.wb-root {
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100vh;
		min-height: 0;
		overflow: hidden;
		background: var(--wb-bg-base, #0b0c10);
		--bg-base: var(--wb-bg-base, #0b0c10);
		--bg-panel: var(--wb-bg-panel, #151820);
		--bg-surface: var(--wb-bg-panel-2, #1c202a);
		--border: var(--wb-border, #303746);
		--brand: var(--wb-brand, #7aa2ff);
		--text-primary: var(--wb-text, #eef0f5);
		--text-secondary: var(--wb-text-sec, #a3abb9);
		--text-muted: var(--wb-muted, #6f7888);
		--font-ui: 'Segoe UI', 'Microsoft YaHei', system-ui, sans-serif;
		--font-mono: 'Cascadia Mono', 'JetBrains Mono', Consolas, monospace;
	}

	.wb-row {
		display: flex;
		flex: 1;
		flex-direction: row;
		min-height: 0;
		min-width: 0;
	}

	.wb-left-composite {
		flex-shrink: 0;
		display: flex;
		flex-direction: row;
		min-width: 0;
		overflow: hidden;
		background: var(--wb-bg-panel, #151820);
		border-right: 1px solid var(--wb-border, #303746);
	}

	.wb-activity-slot {
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
	}

	.wb-sidebar-slot {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.wb-right {
		flex-shrink: 0;
		overflow: hidden;
		display: flex;
		flex-direction: column;
		min-width: 0;
		background: var(--wb-bg-panel, #151820);
		border-left: 1px solid var(--wb-border, #303746);
	}

	.wb-main {
		flex: 1;
		min-width: 0;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		position: relative;
	}

	.wb-dock-wrap {
		flex-shrink: 0;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		border-top: 1px solid var(--wb-border, #303746);
	}

	.wb-dock-wrap :global(.dock) {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.wb-gutter {
		flex-shrink: 0;
		margin: 0;
		padding: 0;
		border: none;
		background: var(--wb-splitter);
		cursor: col-resize;
		touch-action: none;
		z-index: 5;
	}

	.wb-gutter:focus-visible {
		outline: 2px solid var(--wb-splitter-hover);
		outline-offset: -1px;
	}

	.wb-gutter-v {
		width: 1px;
		cursor: col-resize;
	}

	.wb-gutter-v:hover,
	.wb-gutter-v:active {
		background: var(--wb-splitter-hover);
	}

	/* 横向分隔：须覆盖 .wb-gutter 的纵向 cursor */
	.wb-gutter-h {
		cursor: row-resize;
		width: 100%;
		height: 1px;
		touch-action: none;
	}

	.wb-gutter-h:hover,
	.wb-gutter-h:active {
		background: var(--wb-splitter-hover);
	}

	@media (max-width: 1023px) and (min-width: 768px) {
		.wb-right {
			display: none;
		}
		.wb-gutter-main-right {
			display: none;
		}
	}

	@media (max-width: 767px) {
		.wb-left-composite,
		.wb-gutter-v {
			display: none !important;
		}
		.wb-main {
			width: 100%;
			flex: 1;
			min-height: 0;
		}
		.wb-gutter-dock {
			display: none;
		}
		.wb-dock-wrap {
			display: none;
		}
	}
</style>
