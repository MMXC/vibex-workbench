<!-- SpecExplorer — Wails filesystem binding 驱动
     列表数据来自 specExplorerStore.specs（store.loadList 内部调用 wailsListSpecs）
     生产用 Wails binding，开发用 HTTP fallback（见 wails-filesystem.ts）
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { specExplorerStore, workspaceDisplayName } from '$lib/stores/spec-explorer-store';

	import { agentApiUrl } from '$lib/runtime/agent-transport';
	type TreeNode = { name: string; path: string; type: 'file' | 'dir' };
	const childrenMap = $state<Record<string, TreeNode[]>>({});
	const expanded = $state<Record<string, boolean>>({});
	const loadingMap = $state<Record<string, boolean>>({});
	let treeError = $state('');

	function isLikelyFullPath(path: string | null): path is string {
		return !!path && (path.includes('/') || path.includes('\\'));
	}

	function isSpecFile(path: string): boolean {
		return path.startsWith('specs/') && /\.ya?ml$/i.test(path);
	}

	function depthOf(path: string): number {
		return path ? path.split('/').length - 1 : 0;
	}

	async function fetchChildren(path = '') {
		const ws = $specExplorerStore.workspaceRoot;
		if (!ws) return;
		loadingMap[path] = true;
		treeError = '';
		try {
			const url = `${agentApiUrl('/api/workspace/tree')}?workspaceRoot=${encodeURIComponent(ws)}&path=${encodeURIComponent(path || '.')}`;
			const res = await fetch(url);
			const data = await res.json();
			if (!res.ok || !data.ok) throw new Error(data.error || `HTTP ${res.status}`);
			childrenMap[path] = data.nodes || [];
		} catch (e) {
			treeError = e instanceof Error ? e.message : String(e);
		} finally {
			loadingMap[path] = false;
		}
	}

	async function toggleDir(path: string) {
		expanded[path] = !expanded[path];
		if (expanded[path] && !childrenMap[path]) {
			await fetchChildren(path);
		}
	}

	function clickFile(path: string) {
		if (isSpecFile(path)) {
			specExplorerStore.selectSpec(path);
		} else {
			specExplorerStore.selectFile(path);
		}
	}

	async function reload() {
		await fetchChildren('');
	}

	async function restoreWorkspaceRoot() {
		const saved = localStorage.getItem('vibex-workspace-root');
		if (isLikelyFullPath(saved)) {
			specExplorerStore.setWorkspaceRoot(saved);
			return;
		}
		try {
			const res = await fetch('/api/workspace/detect-state');
			if (!res.ok) return;
			const data = await res.json();
			const root = data.workspaceRoot ?? data.workspace_root;
			if (isLikelyFullPath(root)) {
				localStorage.setItem('vibex-workspace-root', root);
				specExplorerStore.setWorkspaceRoot(root);
			}
		} catch (e) {
			console.warn('[SpecExplorer] failed to restore backend workspace root:', e);
		}
	}

	onMount(() => {
		if (!get(specExplorerStore).workspaceRoot) void restoreWorkspaceRoot();
	});

	$effect(() => {
		const ws = $specExplorerStore.workspaceRoot;
		if (!ws) return;
		void fetchChildren('');
	});
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

	{#if !$specExplorerStore.workspaceRoot}
		<p class="muted pad">未设置工作区</p>
	{:else if loadingMap['']}
		<p class="muted pad">加载中…</p>
	{:else if treeError}
		<p class="err pad">{treeError}</p>
	{:else if !childrenMap[''] || childrenMap[''].length === 0}
		<p class="muted pad">当前目录为空</p>
	{:else}
		<div class="tree" role="tree">
			<div class="tree-section">
				<span class="chevron">▾</span>
				<span>workspace</span>
				<span class="tree-count">{childrenMap[''].length}</span>
			</div>
			{#each childrenMap[''] as node (node.path)}
				<div class="line" style:--depth-indent="{depthOf(node.path) * 10}px">
					{#if node.type === 'dir'}
						<button type="button" class="tree-btn dir" onclick={() => toggleDir(node.path)}>
							<span>{expanded[node.path] ? '▾' : '▸'}</span>
							<span>{node.name}</span>
						</button>
						{#if expanded[node.path]}
							{#if loadingMap[node.path]}
								<div class="nested muted">加载中…</div>
							{:else}
								{#each childrenMap[node.path] || [] as sub (sub.path)}
									<div class="nested" style:--depth-indent="{depthOf(sub.path) * 10}px">
										{#if sub.type === 'dir'}
											<button type="button" class="tree-btn dir" onclick={() => toggleDir(sub.path)}>
												<span>{expanded[sub.path] ? '▾' : '▸'}</span>
												<span>{sub.name}</span>
											</button>
										{:else}
											<button type="button" class="tree-btn file"
												class:active={$specExplorerStore.selectedSpecPath === sub.path || $specExplorerStore.selectedFilePath === sub.path}
												onclick={() => clickFile(sub.path)}>
												<span>{isSpecFile(sub.path) ? '◆' : '•'}</span>
												<span>{sub.name}</span>
											</button>
										{/if}
									</div>
								{/each}
							{/if}
						{/if}
					{:else}
						<button type="button" class="tree-btn file"
							class:active={$specExplorerStore.selectedSpecPath === node.path || $specExplorerStore.selectedFilePath === node.path}
							onclick={() => clickFile(node.path)}>
							<span>{isSpecFile(node.path) ? '◆' : '•'}</span>
							<span>{node.name}</span>
						</button>
					{/if}
				</div>
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

	.tree-count {
		margin-left: auto;
		min-width: 18px;
		padding: 1px 6px;
		border-radius: 999px;
		background: rgba(122, 162, 255, 0.14);
		color: var(--wb-text-sec, #a3abb9);
		text-align: center;
		font-size: 10px;
		font-weight: 700;
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
		min-height: 0;
		overflow-y: auto;
		padding: 10px;
		display: flex;
		flex-direction: column;
		gap: 7px;
	}

	.line, .nested { padding-left: var(--depth-indent, 0px); }
	.tree-btn {
		width: 100%;
		display: flex;
		align-items: center;
		gap: 6px;
		border: 1px solid transparent;
		border-radius: 8px;
		background: transparent;
		color: #dbe4f3;
		cursor: pointer;
		padding: 6px 8px;
		text-align: left;
	}
	.tree-btn:hover { background: rgba(122,162,255,.09); border-color: #465064; }
	.tree-btn.active { background: rgba(122,162,255,.16); border-color: #7aa2ff; }
	.tree-btn.dir { color: #9dc2ff; font-weight: 600; }
	.tree-btn.file { color: #dbe4f3; }
</style>
