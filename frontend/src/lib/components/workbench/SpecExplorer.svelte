<!-- SpecExplorer — Wails filesystem binding 驱动
     列表数据来自 specExplorerStore.specs（store.loadList 内部调用 wailsListSpecs）
     生产用 Wails binding，开发用 HTTP fallback（见 wails-filesystem.ts）
-->
<script lang="ts">
	import { specExplorerStore, workspaceDisplayName } from '$lib/stores/spec-explorer-store';

	// 订阅 store 中的 specs 列表
	let specs = $state<{ path: string; level: number; name: string }[]>([]);
	let specsLoading = $state(false);
	let specsError = $state<string | null>(null);
	let selectedPath = $state<string | null>(null);
	let currentWorkspaceRoot = $state('');

	// 订阅 store（列表 + 选中状态）
	$effect(() => {
		const unsub = specExplorerStore.subscribe(s => {
			specs = s.specs;
			specsLoading = s.specsLoading;
			specsError = s.specsError;
			selectedPath = s.selectedSpecPath;
			currentWorkspaceRoot = s.workspaceRoot;
		});
		return unsub;
	});

	/** 手动刷新（点击 ↻ 按钮） */
	function reload() {
		if (!currentWorkspaceRoot) return;
		specExplorerStore.loadList(currentWorkspaceRoot);
	}

	function depthIndent(path: string): number {
		return path.split('/').length - 2;
	}
</script>

<div class="spec-explorer">
	<div class="hdr">
		<span class="hdr-title">资源管理器</span>
		<button type="button" class="reload" title="刷新列表" onclick={reload}>↻</button>
	</div>
	<div class="workspace-head">
		<span class="chevron">▾</span>
		<span class="workspace-name">{$workspaceDisplayName}</span>
		<span class="workspace-actions">···</span>
	</div>

	{#if !currentWorkspaceRoot}
		<p class="muted pad">未设置工作区</p>
	{:else if specsLoading}
		<p class="muted pad">加载中…</p>
	{:else if specsError}
		<p class="err pad">{specsError}</p>
	{:else if specs.length === 0}
		<p class="muted pad">无 spec 文件（点击工具栏「初始化」开始）</p>
	{:else}
		<div class="tree" role="tree">
			<div class="tree-section">
				<span class="chevron">▾</span>
				<span>specs</span>
			</div>
			{#each specs as item (item.path)}
				<button
					type="button"
					class="ws-item"
					class:active={selectedPath === item.path}
					style:padding-left="{10 + depthIndent(item.path) * 12}px"
					onclick={() => specExplorerStore.selectSpec(item.path)}
				>
					<span class="ws-icon">◇</span>
					{#if item.level > 0}
						<span class="level-badge">L{item.level}</span>
					{/if}
					<span class="ws-name">{item.path.replace(/^specs\//, '')}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.spec-explorer {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background: #252526;
		border-right: 1px solid #2d2d2d;
		font-family:
			'Segoe UI',
			'Microsoft YaHei',
			-apple-system,
			sans-serif;
		font-size: 13px;
	}

	.hdr {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 12px 8px;
		border-bottom: 1px solid #2d2d2d;
	}

	.hdr-title {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.07em;
		text-transform: uppercase;
		color: #bbbbbb;
	}

	.workspace-head,
	.tree-section {
		display: flex;
		align-items: center;
		gap: 5px;
		height: 24px;
		padding: 0 8px;
		color: #cccccc;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.workspace-head {
		border-bottom: 1px solid #2d2d2d;
	}

	.workspace-name {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.workspace-actions {
		color: #858585;
		font-weight: 400;
		letter-spacing: 0.08em;
	}

	.chevron {
		color: #858585;
		font-size: 10px;
	}

	.reload {
		background: none;
		border: none;
		color: #858585;
		cursor: pointer;
		padding: 2px 6px;
		border-radius: 4px;
		font-size: 14px;
		line-height: 1;
	}

	.reload:hover {
		color: #cccccc;
		background: #2a2d2e;
	}

	.pad {
		padding: 10px 12px;
	}

	.muted {
		color: #858585;
		font-size: 12px;
	}

	.err {
		color: #f87171;
		font-size: 12px;
	}

	.tree {
		flex: 1;
		overflow-y: auto;
		padding: 4px 0;
	}

	.ws-item {
		display: flex;
		align-items: center;
		gap: 5px;
		width: 100%;
		padding: 3px 10px 3px 16px;
		border: none;
		background: none;
		cursor: pointer;
		color: #cccccc;
		text-align: left;
		font: inherit;
		transition:
			background 150ms ease,
			color 150ms ease;
	}

	.ws-item:hover {
		background: #2a2d2e;
		color: #ffffff;
	}

	.ws-item.active {
		background: #04395e;
		color: #ffffff;
	}

	.ws-icon {
		flex-shrink: 0;
		width: 12px;
		text-align: center;
		font-size: 12px;
		opacity: 0.85;
		color: #519aba;
	}

	.level-badge {
		flex-shrink: 0;
		font-size: 9px;
		font-weight: 700;
		letter-spacing: 0.02em;
		line-height: 1;
		padding: 2px 5px;
		border-radius: 4px;
		background: rgba(0, 122, 204, 0.22);
		color: #9cdcfe;
	}

	.ws-name {
		flex: 1;
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		font-size: 12.5px;
	}
</style>
