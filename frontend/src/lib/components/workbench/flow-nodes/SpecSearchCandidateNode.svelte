<script lang="ts">
	import type { NodeProps } from '@xyflow/svelte';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';

	type CandData = {
		path: string;
		title?: string;
		snippet?: string;
		score?: number;
		query?: string;
	};

	let { data }: NodeProps = $props();
	const d = $derived(data as CandData);

	function openSpecDetail() {
		const p = d.path?.trim();
		if (p) specExplorerStore.selectSpec(p);
	}
</script>

<div class="spec-cand-root">
	<div class="row-top">
		<span class="badge">spec 候选</span>
		{#if d.score != null && !Number.isNaN(Number(d.score))}
			<span class="score">{typeof d.score === 'number' && d.score <= 1 ? `${Math.round(d.score * 100)}%` : String(d.score)}</span>
		{/if}
	</div>
	<strong class="title">{d.title ?? d.path}</strong>
	<code class="path">{d.path}</code>
	{#if d.snippet}
		<p class="snippet">{d.snippet}</p>
	{/if}
	{#if d.query}
		<p class="q"><span class="qk">query</span>{d.query}</p>
	{/if}
	<button type="button" class="open nodrag" onclick={openSpecDetail}>打开 spec 详情</button>
</div>

<style>
	.spec-cand-root {
		min-width: 220px;
		max-width: 280px;
		padding: 10px 12px;
		border-radius: 12px;
		border: 1px solid rgba(122, 162, 255, 0.45);
		background: linear-gradient(165deg, rgba(18, 26, 42, 0.98), rgba(12, 16, 24, 0.98));
		box-shadow: 0 8px 28px rgba(0, 0, 0, 0.35);
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.row-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}

	.badge {
		font-size: 9px;
		font-weight: 800;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: #9ec1ff;
		border: 1px solid rgba(122, 162, 255, 0.35);
		border-radius: 999px;
		padding: 2px 7px;
	}

	.score {
		font-size: 11px;
		font-weight: 700;
		color: #a7f3d0;
		font-family: ui-monospace, monospace;
	}

	.title {
		font-size: 13px;
		font-weight: 700;
		color: #eef0f5;
		line-height: 1.3;
		margin: 0;
	}

	.path {
		font-size: 10px;
		color: #7aa2ff;
		word-break: break-all;
		line-height: 1.35;
		margin: 0;
	}

	.snippet {
		margin: 0;
		font-size: 11px;
		color: #a3abb9;
		line-height: 1.45;
		max-height: 4.35em;
		overflow: hidden;
	}

	.q {
		margin: 0;
		font-size: 10px;
		color: #6f7888;
	}

	.qk {
		display: inline-block;
		margin-right: 6px;
		padding: 1px 5px;
		border-radius: 4px;
		background: rgba(255, 255, 255, 0.06);
		color: #9ec1ff;
		font-weight: 700;
		text-transform: uppercase;
		font-size: 9px;
	}

	.open {
		margin-top: 4px;
		align-self: flex-start;
		padding: 6px 12px;
		border-radius: 8px;
		border: 1px solid rgba(122, 162, 255, 0.55);
		background: rgba(122, 162, 255, 0.14);
		color: #e8eeff;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
	}

	.open:hover {
		background: rgba(122, 162, 255, 0.24);
		border-color: #7aa2ff;
		color: #fff;
	}
</style>
