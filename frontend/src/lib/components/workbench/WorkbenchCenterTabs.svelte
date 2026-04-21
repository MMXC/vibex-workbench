<!-- R2 中央编辑器区：画布 | 构件 | Spec 文本 | Diff — region_map.center_editor.tab_kinds -->
<script lang="ts">
	import CanvasRenderer from '$lib/components/workbench/CanvasRenderer.svelte';
	import ArtifactPanel from '$lib/components/workbench/ArtifactPanel.svelte';

	type TabId = 'canvas' | 'artifacts' | 'spec_text' | 'diff';

	let active = $state<TabId>('canvas');

	const tabs: { id: TabId; label: string }[] = [
		{ id: 'canvas', label: '画布' },
		{ id: 'artifacts', label: '构件' },
		{ id: 'spec_text', label: 'Spec 文本' },
		{ id: 'diff', label: 'Diff' },
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
		{#if active === 'canvas'}
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
		background: var(--bg-base, #0d0d0e);
	}

	.tab-bar {
		flex-shrink: 0;
		height: var(--tab-h, 36px);
		display: flex;
		align-items: flex-end;
		background: var(--bg-panel, #131314);
		border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.07));
		overflow-x: auto;
	}

	.editor-tab {
		display: flex;
		align-items: center;
		padding: 0 14px;
		height: 35px;
		font-size: 13px;
		color: var(--text-muted, #555558);
		border: none;
		background: transparent;
		cursor: pointer;
		border-right: 1px solid var(--border, rgba(255, 255, 255, 0.07));
		transition:
			background 150ms ease,
			color 150ms ease;
		flex-shrink: 0;
		position: relative;
		font-family: var(--font-ui, 'Inter', sans-serif);
	}

	.editor-tab:hover {
		background: rgba(255, 255, 255, 0.05);
		color: var(--text-secondary, #8a8a8e);
	}

	.editor-tab.active {
		color: var(--text-primary, #e8e8ed);
		background: var(--bg-base, #0d0d0e);
		border-bottom: 1px solid var(--bg-base, #0d0d0e);
		margin-bottom: -1px;
	}

	.editor-tab.active::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 1px;
		background: var(--brand, #5856d6);
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
		background: var(--bg-base, #0d0d0e);
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
