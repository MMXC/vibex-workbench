<!-- SpecExplorer — Wails filesystem binding 驱动
     列表数据来自 specExplorerStore.specs（store.loadList 内部调用 wailsListSpecs）
     生产用 Wails binding，开发用 HTTP fallback（见 wails-filesystem.ts）
-->
<script lang="ts">
	import { specExplorerStore, workspaceDisplayName } from '$lib/stores/spec-explorer-store';
	import type { SpecDisplay } from '$lib/workbench/spec-display';
	import { fallbackDisplayTitle } from '$lib/workbench/spec-display';

	// 订阅 store 中的 specs 列表
	let specs = $state<
		{ path: string; level: number; name: string; status: string; display?: SpecDisplay }[]
	>([]);
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

	function levelClass(level: number): string {
		if (level <= 1) return 'goal';
		if (level === 2) return 'skeleton';
		if (level === 3) return 'module';
		if (level === 4) return 'feature';
		return 'slice';
	}

	function compactPath(path: string): string {
		return path.replace(/^specs\//, '').replace(/\.ya?ml$/, '');
	}
</script>

<div class="spec-explorer">
	<div class="hdr">
		<div>
			<span class="eyebrow">Spec Index</span>
			<span class="hdr-title">资源管理器</span>
		</div>
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
				{@const title = item.display?.title || fallbackDisplayTitle(item.name)}
				{@const summary = item.display?.summary || compactPath(item.path)}
				{@const level = item.level > 0 ? `L${item.level}` : 'SPEC'}
				<button
					type="button"
					class="ws-item"
					class:goal={item.level <= 1}
					class:skeleton={item.level === 2}
					class:module={item.level === 3}
					class:feature={item.level === 4}
					class:slice={item.level >= 5}
					class:active={selectedPath === item.path}
					style:--depth="{depthIndent(item.path)}"
					onclick={() => specExplorerStore.selectSpec(item.path)}
				>
					<span class="ws-accent"></span>
					<span class="ws-main">
						<span class="ws-top">
							<span class="ws-title">{title}</span>
							<span class="level-badge {levelClass(item.level)}">{level}</span>
						</span>
						<span class="ws-summary">{summary}</span>
						<span class="ws-machine">{item.status} · {compactPath(item.path)}</span>
					</span>
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
		background:
			radial-gradient(circle at 18% 0%, rgba(122, 162, 255, 0.1), transparent 34%),
			#151820;
		border-right: 1px solid #303746;
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
		padding: 14px 14px 12px;
		border-bottom: 1px solid #303746;
		background: rgba(28, 32, 42, 0.78);
	}

	.eyebrow {
		display: block;
		margin-bottom: 4px;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.13em;
		text-transform: uppercase;
	}

	.hdr-title {
		display: block;
		font-size: 11px;
		font-weight: 800;
		letter-spacing: 0.07em;
		text-transform: uppercase;
		color: #eef0f5;
	}

	.workspace-head,
	.tree-section {
		display: flex;
		align-items: center;
		gap: 5px;
		min-height: 30px;
		padding: 0 12px;
		color: #eef0f5;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.workspace-head {
		border-bottom: 1px solid #303746;
		background: rgba(12, 14, 19, 0.36);
	}

	.workspace-name {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.workspace-actions {
		color: var(--wb-muted, #6f7888);
		font-weight: 400;
		letter-spacing: 0.08em;
	}

	.chevron {
		color: var(--wb-muted, #6f7888);
		font-size: 10px;
	}

	.reload {
		width: 30px;
		height: 30px;
		background: rgba(36, 41, 54, 0.86);
		border: 1px solid #465064;
		color: #a3abb9;
		cursor: pointer;
		padding: 0;
		border-radius: 999px;
		font-size: 14px;
		line-height: 1;
	}

	.reload:hover {
		color: #eef0f5;
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.14);
	}

	.pad {
		padding: 10px 12px;
	}

	.muted {
		color: var(--wb-muted, #6f7888);
		font-size: 12px;
	}

	.err {
		color: #f87171;
		font-size: 12px;
	}

	.tree {
		flex: 1;
		overflow-y: auto;
		padding: 10px;
		display: grid;
		align-content: start;
		gap: 7px;
	}

	.ws-item {
		position: relative;
		display: grid;
		grid-template-columns: 5px 1fr;
		align-items: stretch;
		gap: 0;
		width: 100%;
		padding: 0;
		border: 1px solid transparent;
		border-radius: 13px;
		background: rgba(28, 32, 42, 0.62);
		cursor: pointer;
		color: #eef0f5;
		text-align: left;
		font: inherit;
		overflow: hidden;
		transition:
			background 150ms ease,
			border-color 150ms ease,
			transform 150ms ease;
	}

	.ws-item:hover {
		background: rgba(36, 41, 54, 0.86);
		border-color: #465064;
		transform: translateY(-1px);
	}

	.ws-item.active {
		background: rgba(122, 162, 255, 0.13);
		border-color: #7aa2ff;
		color: #ffffff;
	}

	.ws-accent {
		display: block;
		background: #7aa2ff;
	}

	.ws-item.goal .ws-accent {
		background: #72d6d0;
	}

	.ws-item.skeleton .ws-accent {
		background: #87cf8a;
	}

	.ws-item.module .ws-accent {
		background: #7aa2ff;
	}

	.ws-item.feature .ws-accent {
		background: #efc66b;
	}

	.ws-item.slice .ws-accent {
		background: #f09a6a;
	}

	.ws-main {
		min-width: 0;
		padding: 10px 10px 9px;
		padding-left: calc(10px + var(--depth, 0) * 4px);
		display: grid;
		gap: 5px;
	}

	.ws-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}

	.ws-title {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: 12px;
		font-weight: 800;
	}

	.level-badge {
		flex-shrink: 0;
		height: 21px;
		display: inline-flex;
		align-items: center;
		border: 1px solid #465064;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.02em;
		line-height: 1;
		padding: 0 8px;
		border-radius: 999px;
		background: rgba(12, 14, 19, 0.5);
		color: #a3abb9;
	}

	.level-badge.goal {
		color: #72d6d0;
		border-color: rgba(114, 214, 208, 0.5);
	}

	.level-badge.skeleton {
		color: #87cf8a;
		border-color: rgba(135, 207, 138, 0.5);
	}

	.level-badge.module {
		color: #7aa2ff;
		border-color: rgba(122, 162, 255, 0.5);
	}

	.level-badge.feature {
		color: #efc66b;
		border-color: rgba(239, 198, 107, 0.5);
	}

	.level-badge.slice {
		color: #f09a6a;
		border-color: rgba(240, 154, 106, 0.5);
	}

	.ws-summary,
	.ws-machine {
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.ws-summary {
		color: #a3abb9;
		font-size: 11px;
	}

	.ws-machine {
		color: #6f7888;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
	}
</style>
