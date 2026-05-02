<!--
  SpecGraphViewer.svelte — central spec graph with parent/current/children cards
  Behavior B1 from FEAT-spec-graph-expansion: click spec → load parent/current/children graph
  Integrates with: spec-agent-context-store, spec-explorer-store, spec-display
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { specAgentContextStore } from '$lib/stores/spec-agent-context-store';
	import { extractSpecDisplay, type SpecSlotSummary } from '$lib/workbench/spec-display';
	import SpecAddChildCard from './SpecAddChildCard.svelte';
	import { get } from 'svelte/store';

	interface Props {
		specPath?: string | null;
		specContent?: string;
		onCreateChild?: (intent: CreateChildIntent) => void;
	}

	let { specPath = null, specContent = '', onCreateChild }: Props = $props();

	// ── derived state ─────────────────────────────────────────────
	let displayMeta = $derived.by(() => {
		if (!specPath || !specContent) return null;
		return extractSpecDisplay(specContent, specPath);
	});

	let currentLane = $derived.by(() => {
		if (!displayMeta) return 'L1';
		const raw = displayMeta.rawLevel;
		if (raw.includes('1')) return 'L1';
		if (raw.includes('2')) return 'L2';
		if (raw.includes('3')) return 'L3';
		if (raw.includes('4')) return 'L4';
		if (raw.includes('5')) return 'L5';
		return displayMeta.level === 'UNKNOWN' ? 'SPEC' : displayMeta.level;
	});

	// L1=8%, L2=29%, L3=50%, L4=71%, L5=92%
	function lanePercent(lane: string): number {
		const map: Record<string, number> = { L1: 8, L2: 29, L3: 50, L4: 71, L5: 92 };
		return map[lane] ?? 50;
	}

	let currentLeft = $derived.by(() => lanePercent(currentLane));
	let parentLeft = $derived.by(() => lanePercent(getParentLane(currentLane)));
	let childLeft = $derived.by(() => lanePercent(getChildLane(currentLane)));

	function getParentLane(current: string): string {
		const order = ['L1', 'L2', 'L3', 'L4', 'L5'];
		const idx = order.indexOf(current);
		return idx > 0 ? order[idx - 1] : 'L1';
	}

	function getChildLane(current: string): string {
		const order = ['L1', 'L2', 'L3', 'L4', 'L5'];
		const idx = order.indexOf(current);
		return idx < order.length - 1 ? order[idx + 1] : 'L5';
	}

	// ── actions ────────────────────────────────────────────────────
	function attachCurrentSpec() {
		if (!displayMeta) return;
		specAgentContextStore.addSpec(displayMeta, specContent);
	}

	function slotValue(slot: SpecSlotSummary): string {
		if (slot.status === 'present') return slot.count > 0 ? `${slot.count}` : '✓';
		if (slot.status === 'empty') return '无';
		if (slot.status === 'na') return '不适用';
		return '待补充';
	}

	function handleCreateChild(intent: CreateChildIntent) {
		onCreateChild?.(intent);
	}

	export type CreateChildIntent = {
		parentName: string;
		parentPath: string;
		targetLevel: string;
		targetDir: string;
		existingChildren: string[];
	};
</script>

{#if !specPath || !displayMeta}
	<div class="sgv-empty">
		<p>在左侧选择一个 spec 以查看图谱</p>
	</div>
{:else}
	<div class="spec-graph-viewer">
		<!-- Header -->
		<div class="sgv-head">
			<div>
				<span class="eyebrow">Spec Graph</span>
				<h2>{displayMeta.display.title}</h2>
				<p>{displayMeta.display.summary}</p>
			</div>
			<span class="mode-badge">{currentLane}</span>
		</div>

		<!-- Graph canvas -->
		<div class="sgv-canvas">
			<!-- L1-L5 lane markers -->
			{#each ['L1', 'L2', 'L3', 'L4', 'L5'] as lane, i}
				<span class="lane" style:left="{8 + i * 21}%"><span>{lane}</span></span>
			{/each}

			<!-- Parent card -->
			{#if displayMeta.parent}
				<div class="relation-card parent-card" style:left="{parentLeft}%">
					<span class="k">parent</span>
					<strong>{displayMeta.parent}</strong>
				</div>
				<span
					class="edge parent-edge"
					style:left="{parentLeft + 9}%"
					style:width="{currentLeft - parentLeft - 13}%"
				></span>
			{/if}

			<!-- Center spec card (current) -->
			<div
				class="center-card"
				style:left="{currentLeft}%"
				role="button"
				tabindex="0"
				title="点击添加到右侧 Agent Context"
				onclick={attachCurrentSpec}
				onkeydown={(e) => {
					if (e.key === 'Enter' || e.key === ' ') attachCurrentSpec();
				}}
			>
				<span class="kicker">Spec</span>
				<h2 class="title">{displayMeta.display.title}</h2>
				<p class="summary">{displayMeta.display.summary}</p>
				<p class="meta">{currentLane} · {displayMeta.name}</p>
				<p class="path">{specPath}</p>
				<div class="slot-strip-mini">
					{#each displayMeta.slots.all as slot (slot.id)}
						<span class="slot-dot {slot.status}" title="{slot.label}: {slot.summary}">
							{slot.label[0]}
						</span>
					{/each}
				</div>
				<button
					type="button"
					class="attach-btn"
					onclick={(e) => {
						e.stopPropagation();
						attachCurrentSpec();
					}}
				>
					+ Add to Context
				</button>
			</div>

			<!-- Child slots from spec content -->
			{#each displayMeta.slots.all as slot, i}
				{#if slot.status === 'present'}
					<div
						class="orbit-card slot-card present"
						style:left="{childLeft}%"
						style:top="{15 + i * 12}%"
						title={slot.summary}
					>
						<span class="k">{slot.label}</span>
						<strong>{slotValue(slot)}</strong>
						<small>{slot.summary}</small>
					</div>
				{/if}
			{/each}

			<!-- + Add child card -->
			<SpecAddChildCard
				currentLevel={currentLane}
				parentName={displayMeta.name ?? specPath}
				parentPath={specPath}
				onCreateChild={handleCreateChild}
			/>
		</div>

		<!-- Footer stats -->
		<div class="sgv-foot">
			<div>
				<strong>{displayMeta.slots.all.filter(s => s.status === 'present').length}</strong>
				<span>ready slots</span>
			</div>
			<div>
				<strong>{displayMeta.status}</strong>
				<span>status</span>
			</div>
			<div>
				<strong>{displayMeta.parent ?? 'root'}</strong>
				<span>parent</span>
			</div>
		</div>
	</div>
{/if}

<style>
	.spec-graph-viewer {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background:
			linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			#0e1016;
		background-size: 26px 26px;
		color: #eef0f5;
	}

	.sgv-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		flex: 1;
		color: #6f7888;
		font-size: 13px;
	}

	.sgv-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		padding: 13px 15px;
		border-bottom: 1px solid #303746;
		background: rgba(28, 32, 42, 0.78);
	}

	.sgv-head h2 {
		margin: 0 0 5px;
		font-size: 15px;
		font-weight: 840;
		letter-spacing: -0.02em;
	}

	.sgv-head p {
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
	}

	.sgv-canvas {
		position: relative;
		flex: 1;
		min-height: 420px;
		border: 1px solid #303746;
		border-radius: 16px;
		margin: 12px;
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

	.center-card {
		position: absolute;
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
		transition: transform 160ms ease, border-color 160ms ease;
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

	.slot-strip-mini {
		display: flex;
		gap: 4px;
		margin: 0.5rem 0;
	}

	.slot-dot {
		width: 18px;
		height: 18px;
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 9px;
		font-weight: 700;
		font-family: ui-monospace, monospace;
	}

	.slot-dot.present {
		background: rgba(114, 214, 208, 0.15);
		color: #72d6d0;
		border: 1px solid rgba(114, 214, 208, 0.4);
	}

	.slot-dot.empty {
		background: rgba(111, 120, 136, 0.1);
		color: #6f7888;
		border: 1px solid rgba(111, 120, 136, 0.3);
	}

	.slot-dot.na {
		background: rgba(111, 120, 136, 0.1);
		color: #6f7888;
		border: 1px solid rgba(111, 120, 136, 0.3);
	}

	.slot-dot.missing {
		background: rgba(239, 198, 107, 0.12);
		color: #efc66b;
		border: 1px solid rgba(239, 198, 107, 0.4);
	}

	.attach-btn {
		margin-top: 0.75rem;
		width: 100%;
		border: 1px solid #72d6d0;
		border-radius: 10px;
		background: rgba(114, 214, 208, 0.12);
		color: #e6fffb;
		font-size: 12px;
		padding: 0.45rem 0.55rem;
		cursor: pointer;
		font-family: inherit;
	}

	.attach-btn:hover {
		background: rgba(114, 214, 208, 0.2);
		border-color: #72d6d0;
	}

	.relation-card {
		position: absolute;
		top: 50%;
		transform: translate(-50%, -50%);
		width: 142px;
		padding: 0.48rem 0.62rem;
		font-size: 10px;
		line-height: 1.2;
		border-radius: 12px;
		border: 1px dashed #465064;
		background: rgba(23, 27, 36, 0.72);
		color: #d4d4d8;
		text-align: left;
		pointer-events: none;
	}

	.parent-card {
		top: 50%;
	}

	.relation-card .k {
		display: block;
		margin-bottom: 4px;
		color: #6f7888;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 9px;
		text-transform: uppercase;
	}

	.relation-card strong {
		display: block;
		color: #eef0f5;
		font-size: 11px;
		font-weight: 800;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.orbit-card {
		position: absolute;
		transform: translate(-50%, -50%);
		width: 142px;
		padding: 0.48rem 0.62rem;
		font-size: 10px;
		line-height: 1.2;
		border-radius: 12px;
		border: 1px solid rgba(114, 214, 208, 0.58);
		background: rgba(114, 214, 208, 0.08);
		color: #d4d4d8;
		text-align: left;
		pointer-events: none;
	}

	.slot-card small {
		display: block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: #858fa1;
		font-size: 9px;
	}

	.orbit-card .k {
		display: block;
		margin-bottom: 4px;
		color: #6f7888;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 9px;
		text-transform: uppercase;
	}

	.orbit-card strong {
		display: block;
		color: #eef0f5;
		font-size: 11px;
		font-weight: 800;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.sgv-foot {
		flex-shrink: 0;
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 10px;
		margin: 0 12px 12px;
	}

	.sgv-foot div {
		min-width: 0;
		border: 1px solid #303746;
		border-radius: 12px;
		padding: 10px;
		background: rgba(28, 32, 42, 0.75);
	}

	.sgv-foot strong,
	.sgv-foot span {
		display: block;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.sgv-foot strong {
		color: #eef0f5;
		font-size: 12px;
		margin-bottom: 4px;
	}

	.sgv-foot span {
		color: #a3abb9;
		font-size: 11px;
	}
</style>