<!-- 非 L1 总目标时的轻量「图谱」：中心为 spec 标识，周围为顶层段落卡片 -->
<script lang="ts">
	import { parse as parseYaml } from 'yaml';
	import { specAgentContextStore } from '$lib/stores/spec-agent-context-store';
	import { extractSpecDisplay } from '$lib/workbench/spec-display';

	let { specPath, content }: { specPath: string; content: string } = $props();

	const parsed = $derived.by(() => {
		try {
			return parseYaml(content) as Record<string, unknown>;
		} catch {
			return null;
		}
	});

	const specMeta = $derived.by(() => extractSpecDisplay(content, specPath));

	const title = $derived.by(() => {
		return specMeta.display.title;
	});

	function attachCurrentSpec() {
		specAgentContextStore.addSpec(specMeta, content);
	}

	const level = $derived.by(() => {
		const p = parsed;
		const spec = p?.spec as Record<string, unknown> | undefined;
		const l = spec?.level;
		return typeof l === 'string' ? l : '';
	});

	const sectionKeys = $derived.by(() => {
		const p = parsed;
		if (!p) return [] as string[];
		return Object.keys(p).filter(k => k !== 'spec');
	});

	function levelShort(raw: string): string {
		if (raw.includes('1')) return 'L1';
		if (raw.includes('2')) return 'L2';
		if (raw.includes('3')) return 'L3';
		if (raw.includes('4')) return 'L4';
		if (raw.includes('5')) return 'L5';
		return specMeta.level === 'UNKNOWN' ? 'SPEC' : specMeta.level;
	}

	function laneLeft(lane: string): number {
		const order = ['L1', 'L2', 'L3', 'L4', 'L5'];
		const idx = Math.max(order.indexOf(lane), 2);
		return 8 + idx * 21;
	}

	const currentLane = $derived.by(() => levelShort(level));
	const currentLeft = $derived.by(() => laneLeft(currentLane));
	const parentLeft = $derived.by(() => Math.max(8, currentLeft - 21));
	const childLeft = $derived.by(() => Math.min(92, currentLeft + 21));
</script>

<div class="generic-graph">
	<div class="graph-head">
		<div>
			<span class="eyebrow">Layered DAG Canvas</span>
			<h2>{specMeta.display.title}</h2>
			<p>{specMeta.display.summary}</p>
		</div>
		<span class="mode-badge">{levelShort(level)}</span>
	</div>
	<div class="radial">
		{#each ['L1', 'L2', 'L3', 'L4', 'L5'] as lane, i (lane)}
			<span class="lane" style:left="{8 + i * 21}%"><span>{lane}</span></span>
		{/each}

		{#if specMeta.parent}
			<div class="relation-card parent-card" style:left="{parentLeft}%" style:top="50%">
				<span class="k">parent</span>
				<strong>{specMeta.parent}</strong>
			</div>
			<span class="edge parent-edge" style:left="{parentLeft + 9}%" style:width="{currentLeft - parentLeft - 13}%"></span>
		{/if}

		{#each sectionKeys.slice(0, 5) as key, i (key)}
			<div class="orbit-card" style:left="{childLeft}%" style:top="{18 + i * 14}%">
				<span class="k">section</span>
				<strong>{key}</strong>
			</div>
		{/each}

		{#if currentLane !== 'L5'}
			<div class="add-card" style:left="{childLeft}%" style:top="84%">
				<span class="k">Add child</span>
				<strong>next spec</strong>
			</div>
			<span class="edge child-edge" style:left="{currentLeft + 9}%" style:width="{childLeft - currentLeft - 13}%"></span>
		{/if}

		<div
			class="center-card"
			style:left="{currentLeft}%"
			style:top="50%"
			role="button"
			tabindex="0"
			title="点击添加到右侧 Agent Context"
			onclick={attachCurrentSpec}
			onkeydown={(e) => {
				if (e.key === 'Enter' || e.key === ' ') attachCurrentSpec();
			}}
		>
			<span class="kicker">Spec 文件</span>
			<h2 class="title">{title}</h2>
			<p class="summary">{specMeta.display.summary}</p>
			{#if level}
				<p class="meta">{level}</p>
			{/if}
			<p class="path">{specPath}</p>
			<button type="button" class="attach" onclick={(e) => { e.stopPropagation(); attachCurrentSpec(); }}>
				Add to Context
			</button>
		</div>
	</div>
	<div class="graph-foot">
		<div><strong>{sectionKeys.length}</strong><span>yaml sections</span></div>
		<div><strong>{specMeta.status}</strong><span>status</span></div>
		<div><strong>{specMeta.parent ?? 'root'}</strong><span>parent</span></div>
	</div>
</div>

<style>
	.generic-graph {
		height: 100%;
		min-height: 280px;
		display: flex;
		flex-direction: column;
		padding: 12px;
		background:
			linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			#0e1016;
		background-size: 26px 26px;
		color: #eef0f5;
	}

	.graph-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		margin-bottom: 12px;
		padding: 13px 15px;
		border: 1px solid #303746;
		border-radius: 16px;
		background: rgba(28, 32, 42, 0.78);
	}

	.graph-head h2 {
		margin: 0 0 5px;
		font-size: 15px;
		font-weight: 840;
		letter-spacing: -0.02em;
	}

	.graph-head p {
		margin: 0;
		color: #a3abb9;
		font-size: 12px;
		line-height: 1.45;
	}

	.eyebrow {
		display: block;
		margin-bottom: 6px;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	.mode-badge {
		flex-shrink: 0;
		border: 1px solid #465064;
		border-radius: 999px;
		padding: 5px 10px;
		color: #7aa2ff;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 11px;
		font-weight: 800;
		text-transform: uppercase;
	}

	.radial {
		position: relative;
		flex: 1;
		min-height: 420px;
		border: 1px solid #303746;
		border-radius: 16px;
		background: rgba(12, 14, 19, 0.74);
		overflow: hidden;
	}

	.lane {
		position: absolute;
		top: 18px;
		bottom: 18px;
		width: 1px;
		border-left: 1px solid #242b38;
		display: block;
	}

	.lane span {
		position: absolute;
		top: -2px;
		left: 6px;
		color: #6f7888;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		pointer-events: none;
	}

	.edge {
		position: absolute;
		height: 1.5px;
		background: #465064;
		transform: translateY(-50%);
		pointer-events: none;
	}

	.edge::after {
		content: '';
		position: absolute;
		right: -1px;
		top: 50%;
		width: 0;
		height: 0;
		border-top: 4px solid transparent;
		border-bottom: 4px solid transparent;
		border-left: 7px solid #465064;
		transform: translateY(-50%);
	}

	.parent-edge {
		top: 50%;
	}

	.child-edge {
		top: 84%;
		border-top: 1px dashed #465064;
		background: transparent;
	}

	.center-card {
		position: absolute;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		width: min(88%, 360px);
		padding: 1rem;
		border-radius: 16px;
		border: 2px solid #7aa2ff;
		background: linear-gradient(145deg, rgba(31, 36, 49, 0.96), rgba(15, 17, 23, 0.98));
		box-shadow: 0 18px 60px rgba(0, 0, 0, 0.42);
		cursor: pointer;
		outline: none;
		transition:
			transform 160ms ease,
			border-color 160ms ease;
	}

	.center-card:hover,
	.center-card:focus {
		transform: translate(-50%, -50%) translateY(-2px);
		border-color: #72d6d0;
	}

	.kicker {
		display: block;
		font-size: 10px;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #72d6d0;
		margin-bottom: 0.35rem;
	}

	.title {
		margin: 0 0 0.35rem;
		font-size: 1.1rem;
		font-weight: 840;
		color: #eef0f5;
	}

	.meta {
		margin: 0;
		font-size: 11px;
		color: #7aa2ff;
		font-family: 'Cascadia Code', ui-monospace, monospace;
	}

	.summary {
		margin: 0 0 0.45rem;
		font-size: 12px;
		line-height: 1.45;
		color: #c4c4cc;
	}

	.path {
		margin: 0.5rem 0 0;
		font-size: 10px;
		font-family: ui-monospace, monospace;
		color: #71717a;
		word-break: break-all;
		line-height: 1.35;
	}

	.attach {
		margin-top: 0.75rem;
		width: 100%;
		border: 1px solid #72d6d0;
		border-radius: 10px;
		background: rgba(114, 214, 208, 0.12);
		color: #e6fffb;
		font-size: 12px;
		padding: 0.45rem 0.55rem;
		cursor: pointer;
	}

	.attach:hover {
		background: rgba(114, 214, 208, 0.2);
		border-color: #72d6d0;
	}

	.orbit-card,
	.relation-card,
	.add-card {
		position: absolute;
		transform: translate(-50%, -50%);
		width: 142px;
		padding: 0.48rem 0.62rem;
		font-size: 10px;
		line-height: 1.2;
		border-radius: 12px;
		border: 1px solid #465064;
		background: #171b24;
		color: #d4d4d8;
		text-align: left;
		pointer-events: none;
	}

	.relation-card {
		border-style: dashed;
		background: rgba(23, 27, 36, 0.72);
	}

	.add-card {
		border-style: dashed;
		background: #10131a;
		opacity: 0.82;
	}

	.orbit-card .k,
	.relation-card .k,
	.add-card .k {
		display: block;
		margin-bottom: 4px;
		color: #6f7888;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 9px;
		text-transform: uppercase;
	}

	.orbit-card strong,
	.relation-card strong,
	.add-card strong {
		display: block;
		color: #eef0f5;
		font-size: 11px;
		font-weight: 800;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.graph-foot {
		flex-shrink: 0;
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 10px;
		margin-top: 12px;
	}

	.graph-foot div {
		min-width: 0;
		border: 1px solid #303746;
		border-radius: 12px;
		padding: 10px;
		background: rgba(28, 32, 42, 0.75);
	}

	.graph-foot strong,
	.graph-foot span {
		display: block;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.graph-foot strong {
		color: #eef0f5;
		font-size: 12px;
		margin-bottom: 4px;
	}

	.graph-foot span {
		color: #a3abb9;
		font-size: 11px;
	}
</style>
