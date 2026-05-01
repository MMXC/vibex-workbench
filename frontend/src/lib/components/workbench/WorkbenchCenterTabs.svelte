<!-- R2 中央编辑器区：画布 | 构件 | Spec 文本 | Diff — region_map.center_editor.tab_kinds -->
<script lang="ts">
	import WorkbenchBootstrapPrototype from '$lib/components/workbench/WorkbenchBootstrapPrototype.svelte';
	import CanvasRenderer from '$lib/components/workbench/CanvasRenderer.svelte';
	import ArtifactPanel from '$lib/components/workbench/ArtifactPanel.svelte';

	type TabId = 'bootstrap' | 'canvas' | 'artifacts' | 'spec_text' | 'diff';

	let active = $state<TabId>('canvas');

	const tabs: { id: TabId; label: string }[] = [
		{ id: 'canvas', label: '画布' },
		{ id: 'artifacts', label: '构件' },
		{ id: 'spec_text', label: 'Spec 文本' },
		{ id: 'diff', label: 'Diff' },
		{ id: 'bootstrap', label: '自举原型' },
	];
</script>

<div class="center-r2">
	<div class="tab-bar" role="tablist" aria-label="编辑器视图">
		{#each tabs as { id, label } (id)}
			<button
				type="button"
				class="editor-tab"
				class:active={active === id}
				role="tab"
				aria-selected={active === id}
				onclick={() => (active = id)}
			>
				{label}
			</button>
		{/each}
	</div>

	<div class="pane-stack">
		{#if active === 'bootstrap'}
			<div class="pane prototype-pane">
				<WorkbenchBootstrapPrototype />
			</div>
		{:else if active === 'canvas'}
			<div class="pane canvas-pane">
				<CanvasRenderer />
			</div>
		{:else if active === 'artifacts'}
			<div class="pane artifacts-pane">
				<ArtifactPanel />
			</div>
		{:else if active === 'spec_text'}
			<div class="pane placeholder-pane">
				<p class="hint">
					Spec 源码请在左侧<strong>资源管理器</strong>选择 YAML；选中后主区切换为规格视图。
				</p>
				<p class="sub">与 R2「Spec 文本」Tab 对齐；图谱见 SpecViewer。</p>
			</div>
		{:else}
			<div class="pane placeholder-pane">
				<p class="hint">Diff — 占位（变更对比）</p>
			</div>
		{/if}
	</div>
</div>

<style>
	.center-r2 {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		background: var(--wb-bg-base, #0b0c10);
	}

	.tab-bar {
		flex-shrink: 0;
		height: var(--tab-h, 35px);
		display: flex;
		align-items: flex-end;
		background: var(--wb-bg-panel, #151820);
		border-bottom: 1px solid var(--wb-border, #303746);
		overflow-x: auto;
	}

	.editor-tab {
		display: flex;
		align-items: center;
		padding: 0 13px;
		height: 34px;
		font-size: 12.5px;
		color: var(--wb-text-sec, #a3abb9);
		border: none;
		background: transparent;
		cursor: pointer;
		border-right: 1px solid var(--wb-border, #303746);
		transition:
			background 150ms ease,
			color 150ms ease;
		flex-shrink: 0;
		position: relative;
		font-family: var(--font-ui, 'Segoe UI', 'Microsoft YaHei', system-ui, sans-serif);
	}

	.editor-tab:hover {
		background: var(--wb-bg-panel-2, #1c202a);
		color: var(--wb-text, #eef0f5);
	}

	.editor-tab.active {
		color: var(--wb-text, #eef0f5);
		background: var(--wb-bg-base, #0b0c10);
		border-bottom: 1px solid var(--wb-bg-base, #0b0c10);
		margin-bottom: -1px;
	}

	.editor-tab.active::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 1px;
		background: var(--wb-accent, #72d6d0);
	}

	.pane-stack {
		flex: 1;
		min-height: 0;
		position: relative;
		overflow: hidden;
	}

	.pane {
		position: absolute;
		inset: 0;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.canvas-pane {
		background: var(--wb-bg-base, #0b0c10);
	}

	.prototype-pane {
		background: #0a0f17;
	}

	.artifacts-pane :global(.artifact-panel) {
		height: 100%;
		background: transparent;
	}

	.placeholder-pane {
		padding: 1.25rem;
		overflow: auto;
		color: var(--text-secondary, #8a8a8e);
		font-size: 13px;
		line-height: 1.55;
	}

	.hint {
		margin: 0 0 0.5rem;
		color: var(--text-primary, #e8e8ed);
	}

	.sub {
		margin: 0;
		font-size: 12px;
		color: var(--text-muted, #555558);
	}
</style>
