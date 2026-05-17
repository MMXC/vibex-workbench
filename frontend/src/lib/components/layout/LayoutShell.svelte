<script lang="ts">
	import TopNavBar from './TopNavBar.svelte';
	import AgentHubPanel from '$lib/components/agents/AgentHubPanel.svelte';
	import SpecPilotPreview from '$lib/components/prototype/SpecPilotPreview.svelte';
	import ProtoDesignPanel from '$lib/components/proto/ProtoDesignPanel.svelte';
	import StatusBar from '$lib/components/workbench/StatusBar.svelte';

	// Props
	let { workspaceRoot = '—' }: { workspaceRoot?: string } = $props();

	// Panel widths
	let leftW = $state(340);
	let rightW = $state(380);
	let leftCollapsed = $state(false);
	let rightCollapsed = $state(false);

	// Resizing state
	let draggingLeft = $state(false);
	let draggingRight = $state(false);

	function startDragLeft(e: PointerEvent) {
		draggingLeft = true;
		(e.target as HTMLElement).setPointerCapture(e.pointerId);
		e.preventDefault();
	}
	function startDragRight(e: PointerEvent) {
		draggingRight = true;
		(e.target as HTMLElement).setPointerCapture(e.pointerId);
		e.preventDefault();
	}
	function onPointerMove(e: PointerEvent) {
		if (draggingLeft) {
			const nw = e.clientX;
			leftW = Math.min(500, Math.max(200, nw));
		}
		if (draggingRight) {
			const nw = window.innerWidth - e.clientX;
			rightW = Math.min(560, Math.max(240, nw));
		}
	}
	function onPointerUp() {
		draggingLeft = false;
		draggingRight = false;
	}

	function toggleLeft() { leftCollapsed = !leftCollapsed; }
	function toggleRight() { rightCollapsed = !rightCollapsed; }

	const actualLeftW = $derived(leftCollapsed ? 40 : leftW);
	const actualRightW = $derived(rightCollapsed ? 40 : rightW);
</script>

<svelte:window onpointermove={onPointerMove} onpointerup={onPointerUp} />

<TopNavBar />

<div class="shell" class:dragging={draggingLeft || draggingRight}>
	<!-- Left Panel -->
	<div class="left" style:width="{actualLeftW}px" style:min-width="{actualLeftW}px">
		{#if !leftCollapsed}
			<AgentHubPanel {workspaceRoot} />
		{/if}
		<button class="collapse-btn left-btn" onclick={toggleLeft} title={leftCollapsed ? '展开左侧' : '收起左侧'}>
			{leftCollapsed ? '▶' : '◀'}
		</button>
	</div>

	<!-- Left Drag Handle -->
	{#if !leftCollapsed}
		<div
			class="drag-handle"
			onpointerdown={startDragLeft}
			role="separator"
			aria-orientation="vertical"
			title="拖动调整宽度"
		></div>
	{/if}

	<!-- Center -->
	<div
		class="center"
		style:left="{actualLeftW + (leftCollapsed ? 0 : 6)}px"
		style:right="{actualRightW + (rightCollapsed ? 0 : 6)}px"
	>
		<SpecPilotPreview />
	</div>

	<!-- Right Drag Handle -->
	{#if !rightCollapsed}
		<div
			class="drag-handle right"
			onpointerdown={startDragRight}
			role="separator"
			aria-orientation="vertical"
			title="拖动调整宽度"
		></div>
	{/if}

	<!-- Right Panel -->
	<div class="right" style:width="{actualRightW}px" style:min-width="{actualRightW}px">
		<button class="collapse-btn right-btn" onclick={toggleRight} title={rightCollapsed ? '展开右侧' : '收起右侧'}>
			{rightCollapsed ? '◀' : '▶'}
		</button>
		{#if !rightCollapsed}
			<ProtoDesignPanel />
		{/if}
	</div>
</div>

<StatusBar />

<style>
	.shell {
		position: fixed;
		top: 42px;
		left: 0;
		right: 0;
		bottom: 24px;
		display: flex;
		overflow: hidden;
		user-select: none;
	}
	.shell.dragging { cursor: col-resize; }

	.left, .right {
		position: relative;
		flex-shrink: 0;
		height: 100%;
		overflow: hidden;
		transition: width 0.15s ease;
	}
	.left { background: #0b0d12; }
	.right { background: #0b0d12; }

	.center {
		position: absolute;
		top: 0;
		bottom: 0;
		overflow: hidden;
	}

	.drag-handle {
		width: 6px;
		flex-shrink: 0;
		background: transparent;
		cursor: col-resize;
		z-index: 10;
		transition: background 0.15s;
	}
	.drag-handle:hover, .dragging .drag-handle {
		background: rgba(122, 162, 255, 0.25);
	}
	.drag-handle.right { order: 3; }

	.collapse-btn {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		z-index: 20;
		width: 16px;
		height: 48px;
		border: none;
		border-radius: 4px;
		background: #1e2030;
		color: #556;
		font-size: 9px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: background 0.15s, color 0.15s;
	}
	.collapse-btn:hover { background: #2a2d42; color: #99a; }
	.left-btn { right: -10px; border-left: 1px solid #1e2030; }
	.right-btn { left: -10px; border-right: 1px solid #1e2030; }
</style>
