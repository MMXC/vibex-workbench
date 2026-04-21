<!-- Spec 文件树（对齐 R2 sidebar ws-tree）；点击 → specExplorerStore.selectSpec -->
<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
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

	$effect(() => {
		const unsub = specExplorerStore.subscribe(s => {
			selectedPath = s.selectedSpecPath;
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

	async function loadList() {
		loading = true;
		loadErr = null;
		try {
			const r = await fetch('/api/workspace/specs/list');
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

	function depthIndent(path: string): number {
		return path.split('/').length - 2;
	}

	function badgeShort(specTypeId: string): string {
		const m = specTypeId.match(/^(L\d+[a-z]*)/i);
		if (m) return m[1];
		return specTypeId === 'meta_binding' ? 'meta' : specTypeId.slice(0, 6);
	}

	$effect(() => {
		loadConvention();
		loadList();
	});
</script>

<div class="spec-explorer">
	<div class="hdr">
		<span class="hdr-title">资源管理器</span>
		<button type="button" class="reload" title="刷新列表" onclick={() => loadList()}>↻</button>
	</div>

	{#if loading}
		<p class="muted pad">加载中…</p>
	{:else if loadErr}
		<p class="err pad">{loadErr}</p>
	{:else}
		<div class="tree" role="tree">
			{#each paths as p (p)}
				<button
					type="button"
					class="ws-item"
					class:active={selectedPath === p}
					style:padding-left="{10 + depthIndent(p) * 12}px"
					onclick={() => specExplorerStore.selectSpec(p)}
				>
					<span class="ws-icon">📄</span>
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
		background: var(--wb-panel-bg, #131314);
		border-right: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
		font-family:
			'Inter',
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
		border-bottom: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
	}

	.hdr-title {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.07em;
		text-transform: uppercase;
		color: var(--wb-muted, #555558);
	}

	.reload {
		background: none;
		border: none;
		color: var(--wb-muted, #555558);
		cursor: pointer;
		padding: 2px 6px;
		border-radius: 4px;
		font-size: 14px;
		line-height: 1;
	}

	.reload:hover {
		color: var(--wb-text-sec, #8a8a8e);
		background: rgba(255, 255, 255, 0.05);
	}

	.pad {
		padding: 10px 12px;
	}

	.muted {
		color: var(--wb-muted, #555558);
		font-size: 12px;
	}

	.err {
		color: #f87171;
		font-size: 12px;
	}

	.tree {
		flex: 1;
		overflow-y: auto;
		padding: 6px 0;
	}

	.ws-item {
		display: flex;
		align-items: center;
		gap: 6px;
		width: 100%;
		padding: 5px 12px 5px 16px;
		border: none;
		background: none;
		cursor: pointer;
		color: var(--wb-text-sec, #8a8a8e);
		text-align: left;
		font: inherit;
		transition:
			background 150ms ease,
			color 150ms ease;
	}

	.ws-item:hover {
		background: rgba(255, 255, 255, 0.05);
		color: var(--wb-text, #e8e8ed);
	}

	.ws-item.active {
		background: rgba(88, 86, 214, 0.15);
		color: var(--wb-text, #e8e8ed);
	}

	.ws-icon {
		flex-shrink: 0;
		font-size: 12px;
		opacity: 0.85;
	}

	.type-pill {
		flex-shrink: 0;
		font-size: 9px;
		font-weight: 700;
		letter-spacing: 0.02em;
		line-height: 1;
		padding: 2px 5px;
		border-radius: 4px;
		background: rgba(88, 86, 214, 0.22);
		color: #c4b5fd;
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
