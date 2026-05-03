<script lang="ts">
	import type { SpecSlotSession } from '$lib/stores/spec-slot-session-store';

	let { session }: { session: SpecSlotSession } = $props();

	const slots = $derived.by(() => session.spec.slots.all);

	function statusText(status: string): string {
		if (status === 'present') return '已定义';
		if (status === 'empty') return '空';
		if (status === 'na') return '不适用';
		return '待补';
	}

	function questionForSlot(id: string): string {
		if (id === 'structure') return '它的 parent、依赖和影响文件是否已经完整？';
		if (id === 'input') return '上游输入、触发条件和数据格式是否明确？';
		if (id === 'output') return '用户可见结果、文件产物和校验信号是什么？';
		if (id === 'prototype') return '需要什么可视化原型来验证这个 spec？';
		if (id === 'implementation') return '应该路由到哪些工具或代码区域实现？';
		return '这个槽位还缺少什么确认信息？';
	}

	const routeDecisions = $derived.by(() => session.routePreview?.decisions ?? []);
	const fireworks = $derived.by(() => session.fireworksGraph);
</script>

<section class="visual-pane" aria-label="槽位可视化图谱">
	<div class="map-card focus">
		<span class="k">Current Slot</span>
		<h3>{session.slot.label}</h3>
		<p>{session.slot.summary}</p>
		<strong>{statusText(session.slot.status)}</strong>
	</div>

	<div class="flow">
		<div class="spec-node">
			<span class="k">{session.spec.level}</span>
			<h3>{session.spec.display.title}</h3>
			<p>{session.spec.display.summary}</p>
			<code>{session.spec.path}</code>
		</div>

		<div class="slot-grid">
			{#each slots as slot (slot.id)}
				<div class="slot-node {slot.status}" class:active={slot.id === session.slot.id}>
					<span>{slot.label}</span>
					<strong>{statusText(slot.status)}</strong>
					<small>{slot.summary}</small>
				</div>
			{/each}
		</div>
	</div>

	<div class="question-card">
		<span class="k">Clarification Seed</span>
		<p>{questionForSlot(session.slot.id)}</p>
	</div>

	<div class="route-card">
		<span class="k">Tool Route Preview</span>
		{#if routeDecisions.length > 0}
			<div class="route-list">
				{#each routeDecisions as decision (decision.node_id)}
					<div class="route-item" class:confirm={decision.needs_confirm}>
						<strong>{decision.tool || '未命中工具'}</strong>
						<span>{decision.node_kind}</span>
						<small>{decision.reason}</small>
					</div>
				{/each}
			</div>
		{:else if session.slot.id === 'input' || session.slot.id === 'output'}
			<p>建议进入 I/O 合约校验，检查上下游、边界条件和可测试性。</p>
		{:else if session.slot.id === 'prototype'}
			<p>建议先澄清验证目标，再决定是否生成独立原型文件。</p>
		{:else}
			<p>建议先分析结构关系，再用最少问题补齐缺失字段。</p>
		{/if}
	</div>

	<div class="fireworks-card">
		<span class="k">Fireworks Tech Graph</span>
		{#if fireworks}
			<div class="fireworks-stage">
				{#each fireworks.nodes as node, index (node.id)}
					<div class="spark {node.type} {node.status}" style:left="{8 + (index % 4) * 28}%" style:top="{18 + Math.floor(index / 4) * 24}%">
						<strong>{node.label}</strong>
						<small>{node.type} · {node.status}</small>
					</div>
				{/each}
				{#each fireworks.edges as edge, index (`${edge.from}-${edge.to}-${index}`)}
					<span class="spark-edge" style:top="{14 + index * 7}%">{edge.label}</span>
				{/each}
			</div>
		{:else}
			<p>正在生成 route preview 和 fireworks 图谱数据…</p>
		{/if}
	</div>
</section>

<style>
	.visual-pane {
		display: flex;
		flex-direction: column;
		gap: 12px;
		height: 100%;
		min-height: 0;
		overflow: auto;
		padding: 14px;
		background:
			linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			#0e1016;
		background-size: 24px 24px;
	}

	.map-card,
	.question-card,
	.route-card,
	.fireworks-card,
	.spec-node,
	.slot-node {
		border: 1px solid #303746;
		border-radius: 16px;
		background: rgba(28, 32, 42, 0.78);
		color: #eef0f5;
	}

	.map-card,
	.question-card,
	.route-card,
	.fireworks-card {
		padding: 14px;
	}

	.map-card.focus {
		border-color: rgba(122, 162, 255, 0.72);
		background: rgba(122, 162, 255, 0.11);
	}

	.k {
		display: block;
		margin-bottom: 7px;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	h3,
	p {
		margin: 0;
	}

	h3 {
		font-size: 16px;
		line-height: 1.25;
	}

	p {
		margin-top: 7px;
		color: #a3abb9;
		font-size: 12px;
		line-height: 1.5;
	}

	code {
		display: block;
		margin-top: 10px;
		color: #858fa1;
		font-size: 10px;
		white-space: normal;
		word-break: break-all;
	}

	.flow {
		display: grid;
		grid-template-columns: minmax(220px, 0.78fr) minmax(280px, 1.22fr);
		gap: 12px;
		min-height: 0;
	}

	.spec-node {
		padding: 16px;
	}

	.slot-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 10px;
	}

	.slot-node {
		padding: 11px;
		min-height: 78px;
	}

	.slot-node.active {
		border-color: rgba(122, 162, 255, 0.88);
		box-shadow: 0 0 0 1px rgba(122, 162, 255, 0.18);
	}

	.slot-node.present {
		border-color: rgba(114, 214, 208, 0.44);
	}

	.slot-node.missing,
	.slot-node.empty {
		border-color: rgba(239, 198, 107, 0.5);
	}

	.slot-node span,
	.slot-node strong,
	.slot-node small {
		display: block;
	}

	.slot-node span {
		color: #eef0f5;
		font-weight: 800;
	}

	.slot-node strong {
		margin-top: 5px;
		color: #72d6d0;
		font-size: 11px;
	}

	.slot-node small {
		margin-top: 6px;
		color: #858fa1;
		font-size: 10px;
		line-height: 1.35;
	}

	@media (max-width: 900px) {
		.flow {
			grid-template-columns: 1fr;
		}
	}

	.route-list {
		display: grid;
		gap: 8px;
		margin-top: 10px;
	}

	.route-item {
		border: 1px solid rgba(114, 214, 208, 0.34);
		border-radius: 12px;
		padding: 10px;
		background: rgba(114, 214, 208, 0.07);
	}

	.route-item.confirm {
		border-color: rgba(239, 198, 107, 0.46);
		background: rgba(239, 198, 107, 0.07);
	}

	.route-item strong,
	.route-item span,
	.route-item small {
		display: block;
	}

	.route-item strong {
		color: #eef0f5;
		font-size: 12px;
	}

	.route-item span {
		margin-top: 4px;
		color: #72d6d0;
		font-family: ui-monospace, monospace;
		font-size: 10px;
	}

	.route-item small {
		margin-top: 6px;
		color: #858fa1;
		font-size: 10px;
		line-height: 1.4;
	}

	.fireworks-stage {
		position: relative;
		height: 260px;
		margin-top: 10px;
		overflow: hidden;
		border: 1px solid #242b38;
		border-radius: 14px;
		background:
			radial-gradient(circle at 30% 18%, rgba(122, 162, 255, 0.15), transparent 22%),
			radial-gradient(circle at 78% 70%, rgba(114, 214, 208, 0.12), transparent 26%),
			rgba(12, 14, 19, 0.58);
	}

	.spark {
		position: absolute;
		width: 132px;
		min-height: 48px;
		transform: translate(-50%, -50%);
		border: 1px solid #465064;
		border-radius: 13px;
		padding: 8px;
		background: rgba(28, 32, 42, 0.9);
		box-shadow: 0 0 24px rgba(122, 162, 255, 0.1);
	}

	.spark.ready {
		border-color: rgba(114, 214, 208, 0.56);
	}

	.spark.confirm {
		border-color: rgba(239, 198, 107, 0.6);
	}

	.spark.missing {
		border-color: rgba(248, 113, 113, 0.55);
	}

	.spark.tool {
		background: rgba(122, 162, 255, 0.13);
	}

	.spark.gate {
		background: rgba(239, 198, 107, 0.1);
	}

	.spark strong,
	.spark small {
		display: block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.spark strong {
		color: #eef0f5;
		font-size: 10px;
	}

	.spark small {
		margin-top: 4px;
		color: #a3abb9;
		font-size: 9px;
	}

	.spark-edge {
		position: absolute;
		left: 50%;
		width: 38%;
		height: 1px;
		overflow: hidden;
		border-top: 1px dashed rgba(122, 162, 255, 0.34);
		color: transparent;
	}
</style>
