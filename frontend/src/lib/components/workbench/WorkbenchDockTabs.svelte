<!-- 底部 Dock（对齐 R2：文件 / 编辑 / 终端 / 任务 / 设置） -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	type DockId = 'files' | 'edit' | 'terminal' | 'tasks' | 'settings';

	interface Props {
		filesBody?: Snippet;
	}

	let { filesBody }: Props = $props();

	let tab = $state<DockId>('files');

	const tabs: { id: DockId; label: string }[] = [
		{ id: 'files', label: '文件' },
		{ id: 'edit', label: '编辑' },
		{ id: 'terminal', label: '终端' },
		{ id: 'tasks', label: '任务' },
		{ id: 'settings', label: '设置' },
	];
</script>

<div class="dock">
	<div class="dock-tabs" role="tablist" aria-label="底部面板">
		{#each tabs as { id, label } (id)}
			<button
				type="button"
				role="tab"
				class:active={tab === id}
				onclick={() => (tab = id)}
			>
				{label}
			</button>
		{/each}
	</div>
	<div class="dock-body">
		{#if tab === 'files'}
			<div class="dock-panel files">
				{#if filesBody}
					{@render filesBody()}
				{:else}
					<p class="muted">Artifacts / 生成物将显示于此</p>
				{/if}
			</div>
		{:else if tab === 'edit'}
			<div class="dock-panel">
				<p class="muted">编辑 — 占位（多光标 / 格式化等）</p>
			</div>
		{:else if tab === 'terminal'}
			<div class="dock-panel">
				<p class="muted">终端 — 占位</p>
			</div>
		{:else if tab === 'tasks'}
			<div class="dock-panel">
				<p class="muted">任务 — 占位</p>
			</div>
		{:else}
			<div class="dock-panel">
				<p class="muted">设置 — 占位</p>
			</div>
		{/if}
	</div>
</div>

<style>
	.dock {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-height: 0;
		background: var(--wb-panel-bg, #131314);
	}

	.dock-tabs {
		display: flex;
		flex-shrink: 0;
		height: 28px;
		align-items: stretch;
		background: var(--wb-base, #0d0d0e);
		border-bottom: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
	}

	.dock-tabs button {
		padding: 0 12px;
		font-size: 11px;
		border: none;
		background: transparent;
		color: var(--wb-muted, #555558);
		cursor: pointer;
		border-right: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
		transition:
			background 150ms ease,
			color 150ms ease;
	}

	.dock-tabs button:hover {
		color: var(--wb-text-sec, #8a8a8e);
		background: rgba(255, 255, 255, 0.04);
	}

	.dock-tabs button.active {
		color: var(--wb-text, #e8e8ed);
		background: rgba(122, 162, 255, 0.12);
		box-shadow: inset 0 -2px 0 var(--wb-brand, #7aa2ff);
	}

	.dock-body {
		flex: 1;
		min-height: 0;
		overflow: auto;
	}

	.dock-panel {
		padding: 10px 12px;
		font-size: 12px;
	}

	.dock-panel.files {
		padding: 6px 8px;
		min-height: 100px;
	}

	.muted {
		margin: 0;
		color: var(--wb-muted, #555558);
	}
</style>
