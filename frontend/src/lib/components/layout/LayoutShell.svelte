<!-- LayoutShell.svelte
三栏布局外壳：左 AgentHub(340px) | 中 SpecPilot 原型(flex) | 右 ProtoDesign(360px)
顶部导航栏 42px，底部状态栏 24px。
不删除现有组件文件，逐步迁移。
-->
<script lang="ts">
	import TopNavBar from './TopNavBar.svelte';
	import AgentHubPanel from '$lib/components/agents/AgentHubPanel.svelte';
	import SpecPilotPreview from '$lib/components/prototype/SpecPilotPreview.svelte';
	import ProtoDesignPanel from '$lib/components/proto/ProtoDesignPanel.svelte';
	import StatusBar from '$lib/components/workbench/StatusBar.svelte';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { specSlotSessionStore } from '$lib/stores/spec-slot-session-store';
	import { onDestroy } from 'svelte';

	interface Props {
		backendStatus?: 'connecting' | 'ready' | 'error';
		workspaceState?: 'empty' | 'partial' | 'ready';
		canvasAutoUiEnabled?: boolean;
		ontoggleCanvasAutoUi?: () => void;
	}
	let { backendStatus = 'connecting', workspaceState = 'empty', canvasAutoUiEnabled = true, ontoggleCanvasAutoUi }: Props = $props();

	const TOP_H = 42;
	const BOT_H = 24;
	const LEFT_MIN = 200;
	const LEFT_MAX = 500;
	const RIGHT_MIN = 240;
	const RIGHT_MAX = 560;

	let leftOpen = $state(true);
	let rightOpen = $state(true);
	let leftWidth = $state(340);
	let rightWidth = $state(360);

	let workspaceRoot = $derived($specExplorerStore.workspaceRoot ?? '—');

	// specId = basename of selected spec path (without .yaml)
	let specId = $derived.by(() => {
		const path = $specExplorerStore.selectedSpecPath;
		if (!path) return '';
		// e.g. /path/to/workspace/specs/L5-slice-name.yaml → L5-slice-name
		const base = path.split('/').pop() ?? '';
		return base.replace(/\.ya?ml$/, '');
	});

	// protoSpecId = derived from session's bound prototype path
	// pptDemoPath = ".specpilot/prototypes/{protoSpecId}.html" → protoSpecId
	let sessionState = $state({ activeKey: null as string | null, sessions: {} as Record<string, { pptDemoPath?: string | null }> });
	$effect(() => {
		const unsub = specSlotSessionStore.subscribe(v => {
			sessionState = { activeKey: v.activeKey, sessions: v.sessions };
		});
		return unsub;
	});
	let protoSpecId = $derived.by(() => {
		const key = sessionState.activeKey;
		if (!key) return '';
		const ppt = sessionState.sessions[key]?.pptDemoPath ?? '';
		if (!ppt) return '';
		// ".specpilot/prototypes/foo.html" → "foo"
		const m = ppt.match(/([^/]+)\.html$/);
		return m ? m[1] : '';
	});

	// Drag left divider
	let draggingLeft = $state(false);
	let dragLeftX = 0;
	let dragLeftW = 0;
	function startLeftDrag(e: MouseEvent) {
		draggingLeft = true;
		dragLeftX = e.clientX;
		dragLeftW = leftWidth;
		window.addEventListener('mousemove', moveLeftDrag);
		window.addEventListener('mouseup', stopLeftDrag);
	}
	function moveLeftDrag(e: MouseEvent) {
		if (!draggingLeft) return;
		leftWidth = Math.max(LEFT_MIN, Math.min(LEFT_MAX, dragLeftW + (e.clientX - dragLeftX)));
	}
	function stopLeftDrag() {
		draggingLeft = false;
		window.removeEventListener('mousemove', moveLeftDrag);
		window.removeEventListener('mouseup', stopLeftDrag);
	}

	// Drag right divider
	let draggingRight = $state(false);
	let dragRightX = 0;
	let dragRightW = 0;
	function startRightDrag(e: MouseEvent) {
		draggingRight = true;
		dragRightX = e.clientX;
		dragRightW = rightWidth;
		window.addEventListener('mousemove', moveRightDrag);
		window.addEventListener('mouseup', stopRightDrag);
	}
	function moveRightDrag(e: MouseEvent) {
		if (!draggingRight) return;
		rightWidth = Math.max(RIGHT_MIN, Math.min(RIGHT_MAX, dragRightW + (dragRightX - e.clientX)));
	}
	function stopRightDrag() {
		draggingRight = false;
		window.removeEventListener('mousemove', moveRightDrag);
		window.removeEventListener('mouseup', stopRightDrag);
	}

	onDestroy(() => {
		window.removeEventListener('mousemove', moveLeftDrag);
		window.removeEventListener('mouseup', stopLeftDrag);
		window.removeEventListener('mousemove', moveRightDrag);
		window.removeEventListener('mouseup', stopRightDrag);
	});
</script>

<!-- Top NavBar: fixed top -->
<div class="ls-top">
	<TopNavBar />
</div>

	<!-- Bottom StatusBar: fixed bottom -->
	<div class="ls-bot">
		<StatusBar {workspaceRoot} {backendStatus} {workspaceState} {canvasAutoUiEnabled} {ontoggleCanvasAutoUi} />
	</div>

<!-- Middle row: flex -->
<div class="ls-middle">

	{#if leftOpen}
	<!-- Left Agent Hub -->
	<aside class="ls-left" style="width:{leftWidth}px">
		<AgentHubPanel />
	</aside>
	<!-- Left resize handle -->
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="ls-divider"
		class:active={draggingLeft}
		onmousedown={startLeftDrag}
		role="separator"
		aria-orientation="vertical"
	></div>
	{/if}

	<!-- Center Prototype Preview -->
	<main class="ls-center">
		<SpecPilotPreview specId={protoSpecId || specId} />
	</main>

	{#if rightOpen}
	<!-- Right resize handle -->
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="ls-divider"
		class:active={draggingRight}
		onmousedown={startRightDrag}
		role="separator"
		aria-orientation="vertical"
	></div>
	<!-- Right ProtoDesign Panel -->
	<aside class="ls-right" style="width:{rightWidth}px">
		<ProtoDesignPanel />
	</aside>
	{/if}

</div>

<style>
.ls-top {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	height: 42px;
	z-index: 100;
}
.ls-bot {
	position: fixed;
	bottom: 0;
	left: 0;
	right: 0;
	height: 24px;
	z-index: 100;
}
.ls-middle {
	display: flex;
	flex-direction: row;
	position: fixed;
	top: 42px;
	bottom: 24px;
	left: 0;
	right: 0;
	overflow: hidden;
}
.ls-left {
	flex-shrink: 0;
	height: 100%;
	overflow: hidden;
	border-right: 1px solid var(--wb-border, rgba(255,255,255,0.07));
	background: var(--wb-bg, #0b0d12);
}
.ls-center {
	flex: 1;
	height: 100%;
	overflow: hidden;
}
.ls-right {
	flex-shrink: 0;
	height: 100%;
	overflow: hidden;
	border-left: 1px solid var(--wb-border, rgba(255,255,255,0.07));
	background: var(--wb-bg-panel, #10131a);
}
.ls-divider {
	width: 4px;
	height: 100%;
	cursor: col-resize;
	background: transparent;
	transition: background 0.15s;
	margin: 0 -2px;
	z-index: 10;
	flex-shrink: 0;
}
.ls-divider:hover,
.ls-divider.active {
	background: var(--wb-accent, #72d6d0);
}
</style>
