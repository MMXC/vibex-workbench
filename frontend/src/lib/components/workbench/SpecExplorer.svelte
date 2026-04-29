<!-- Spec 文件树（对齐 R2 sidebar ws-tree）；点击 → specExplorerStore.selectSpec -->
<script lang="ts">
	import { specExplorerStore, workspaceDisplayName } from '$lib/stores/spec-explorer-store';
	import {
		type ConventionPayload,
		inferSpecTypeId,
		specTypeLabel,
	} from '$lib/workbench/spec-convention';

	let paths = $state<string[]>([]);
	let loadErr = $state<string | null>(null);
	let loading = $state(true);
	let convention = $state<ConventionPayload['convention'] | null>(null);

	let selectedPath = $state<string | null>(null);
	let currentWorkspaceRoot = $state('');

	// 订阅 selectedPath（来自 store，选中状态）
	$effect(() => {
		const unsub = specExplorerStore.subscribe(s => {
			selectedPath = s.selectedSpecPath;
		});
		return unsub;
	});

	// 订阅 workspaceRoot（来自 store，切换时触发重新加载）
	$effect(() => {
		const unsub = specExplorerStore.subscribe(s => {
			const root = s.workspaceRoot;
			if (root && root !== currentWorkspaceRoot) {
				currentWorkspaceRoot = root;
				loadWithRoot(root);
			}
		});
		return unsub;
	});

	async function loadConvention() {
		try {
			const r = await fetch('/api/workspace/specs/convention');
			if (!r.ok) return;
			const j = (await r.json()) as ConventionPayload;
			convention = j.convention ?? null;
		} catch {
			convention = null;
		}
	}

	/**
	 * 带 workspaceRoot 的 specs 列表加载
	 * @param root workspace 根路径，为空时跳过加载
	 */
	async function loadWithRoot(root: string) {
		loading = true;
		loadErr = null;
		loadConvention();
		try {
			const url = `/api/workspace/specs/list?workspaceRoot=${encodeURIComponent(root)}`;
			const r = await fetch(url);
			if (!r.ok) throw new Error(await r.text());
			const data = (await r.json()) as { paths: string[] };
			paths = data.paths ?? [];
		} catch (e) {
			loadErr = e instanceof Error ? e.message : String(e);
			paths = [];
		} finally {
			loading = false;
		}
	}

	/** 手动刷新（点击 ↻ 按钮） */
	async function reload() {
		if (!currentWorkspaceRoot) return;
		await loadWithRoot(currentWorkspaceRoot);
	}

	function depthIndent(path: string): number {
		return path.split('/').length - 2;
	}

	function badgeShort(specTypeId: string): string {
		const m = specTypeId.match(/^(L\d+[a-z]*)/i);
		if (m) return m[1];
		return specTypeId === 'meta_binding' ? 'meta' : specTypeId.slice(0, 6);
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
	{:else if loading}
		<p class="muted pad">加载中…</p>
	{:else if loadErr}
		<p class="err pad">{loadErr}</p>
	{:else}
		<div class="tree" role="tree">
			<div class="tree-section">
				<span class="chevron">▾</span>
				<span>specs</span>
			</div>
			{#each paths as p (p)}
				<button
					type="button"
					class="ws-item"
					class:active={selectedPath === p}
					style:padding-left="{10 + depthIndent(p) * 12}px"
					onclick={() => specExplorerStore.selectSpec(p)}
				>
					<span class="ws-icon">{p.endsWith('.yaml') ? '◇' : '·'}</span>
					{#if convention}
						{@const tid = inferSpecTypeId(p, convention)}
						{#if tid}
							<span class="type-pill" title={specTypeLabel(convention, tid) ?? tid}>{badgeShort(tid)}</span>
						{/if}
					{/if}
					<span class="ws-name">{p.replace(/^specs\//, '')}</span>
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

	.type-pill {
		flex-shrink: 0;
		font-size: 9px;
		font-weight: 700;
		letter-spacing: 0.02em;
		line-height: 1;
		padding: 2px 5px;
		border-radius: 4px;
		background: rgba(0, 122, 204, 0.22);
		color: #9cdcfe;
		max-width: 44px;
		overflow: hidden;
		text-overflow: ellipsis;
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
