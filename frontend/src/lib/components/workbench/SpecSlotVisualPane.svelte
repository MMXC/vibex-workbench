<script lang="ts">
	import { specSlotSessionStore } from '$lib/stores/spec-slot-session-store';
	import type { SpecSlotSession } from '$lib/stores/spec-slot-session-store';
	import SpecSlotA2UIStage from '$lib/components/workbench/SpecSlotA2UIStage.svelte';
	import SpecSlotPrototypeKitBar from '$lib/components/workbench/SpecSlotPrototypeKitBar.svelte';

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
</script>

<section class="visual-pane" aria-label="槽位可视化与 A2UI">
	<header class="focus-strip">
		<span class="slot-pill">{session.slot.label}</span>
		<span class="dot">·</span>
		<span class="spec-title">{session.spec.display.title}</span>
		<span class="status-pill">{statusText(session.slot.status)}</span>
	</header>

	<div class="a2ui-main">
		<SpecSlotA2UIStage {session} />
	</div>

	<details class="drawer context-drawer">
		<summary class="drawer-sum">槽位上下文 · 图谱与路由预览</summary>
		<div class="drawer-body">
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

			<div class="route-card compact">
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
				{:else}
					<p class="route-fallback">路由预览加载后显示各节点命中工具；详见 A2UI 区。</p>
				{/if}
			</div>
		</div>
	</details>

	{#if session.slot.id === 'prototype'}
		<details class="drawer kit-drawer">
			<summary class="drawer-sum">设计物料库 · DESIGN.md / 从页面提取</summary>
			<div class="drawer-body kit-body">
				<SpecSlotPrototypeKitBar {session} />
			</div>
		</details>
	{/if}

	<div class="action-zone">
		<button type="button" class="btn-confirm" onclick={() => specSlotSessionStore.confirmActiveA2UI()}>
			确认
		</button>
		<details class="drawer action-drawer">
			<summary class="drawer-sum action-sum">更多操作</summary>
			<div class="action-grid">
				<button type="button" onclick={() => specSlotSessionStore.tuneActiveA2UI()}>微调</button>
				<button type="button" onclick={() => specSlotSessionStore.draftActiveA2UI()}>草稿</button>
				<button type="button" onclick={() => specSlotSessionStore.cancelActiveA2UI()}>取消</button>
				<button type="button" onclick={() => specSlotSessionStore.regenerateActiveA2UI()}>
					重新生成
				</button>
			</div>
		</details>
	</div>
</section>

<style>
	.visual-pane {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background:
			linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			#0e1016;
		background-size: 24px 24px;
	}

	.focus-strip {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
		padding: 8px 14px;
		border-bottom: 1px solid #2a3140;
		background: rgba(12, 14, 19, 0.75);
		font-size: 12px;
		color: #c8ced9;
	}

	.slot-pill {
		font-weight: 800;
		color: #7aa2ff;
	}
	.dot {
		opacity: 0.45;
	}
	.spec-title {
		color: #eef0f5;
		font-weight: 600;
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.status-pill {
		font-size: 10px;
		padding: 2px 8px;
		border-radius: 999px;
		border: 1px solid rgba(114, 214, 208, 0.35);
		color: #72d6d0;
	}

	.a2ui-main {
		flex: 1;
		min-height: 0;
		padding: 8px 14px 0;
		display: flex;
		flex-direction: column;
	}

	.drawer {
		flex-shrink: 0;
		border-top: 1px solid #2a3140;
		background: rgba(12, 14, 19, 0.55);
	}

	.drawer-sum {
		cursor: pointer;
		list-style: none;
		padding: 10px 14px;
		font-size: 11px;
		font-weight: 800;
		color: #94a3b8;
		letter-spacing: 0.04em;
		user-select: none;
	}
	.drawer-sum::-webkit-details-marker {
		display: none;
	}
	.drawer-sum::before {
		content: '▸ ';
		opacity: 0.5;
	}
	details[open] > .drawer-sum::before {
		content: '▾ ';
	}

	.drawer-body {
		padding: 0 14px 12px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		max-height: 40vh;
		overflow: auto;
	}

	.kit-body {
		max-height: 48vh;
		padding-top: 4px;
	}

	.map-card,
	.question-card,
	.route-card,
	.spec-node,
	.slot-node {
		border: 1px solid #303746;
		border-radius: 16px;
		background: rgba(28, 32, 42, 0.78);
		color: #eef0f5;
	}

	.map-card,
	.question-card {
		padding: 12px 14px;
	}

	.route-card.compact {
		padding: 12px 14px;
	}

	.map-card.focus {
		border-color: rgba(122, 162, 255, 0.72);
		background: rgba(122, 162, 255, 0.11);
	}

	.k {
		display: block;
		margin-bottom: 6px;
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
		font-size: 15px;
		line-height: 1.25;
	}

	p {
		margin-top: 6px;
		color: #a3abb9;
		font-size: 11px;
		line-height: 1.45;
	}

	.route-fallback {
		margin-top: 6px;
	}

	code {
		display: block;
		margin-top: 8px;
		color: #858fa1;
		font-size: 10px;
		white-space: normal;
		word-break: break-all;
	}

	.flow {
		display: grid;
		grid-template-columns: minmax(200px, 0.78fr) minmax(260px, 1.22fr);
		gap: 10px;
	}

	.spec-node {
		padding: 12px;
	}

	.slot-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 8px;
	}

	.slot-node {
		padding: 9px;
		min-height: 68px;
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
		font-size: 11px;
	}

	.slot-node strong {
		margin-top: 4px;
		color: #72d6d0;
		font-size: 10px;
	}

	.slot-node small {
		margin-top: 4px;
		color: #858fa1;
		font-size: 9px;
		line-height: 1.35;
	}

	@media (max-width: 900px) {
		.flow {
			grid-template-columns: 1fr;
		}
	}

	.route-list {
		display: grid;
		gap: 6px;
		margin-top: 8px;
		max-height: 120px;
		overflow: auto;
	}

	.route-item {
		border: 1px solid rgba(114, 214, 208, 0.34);
		border-radius: 10px;
		padding: 8px;
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
		font-size: 11px;
	}

	.route-item span {
		margin-top: 3px;
		color: #72d6d0;
		font-family: ui-monospace, monospace;
		font-size: 9px;
	}

	.route-item small {
		margin-top: 4px;
		color: #858fa1;
		font-size: 9px;
		line-height: 1.35;
	}

	.action-zone {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 10px;
		flex-wrap: wrap;
		padding: 10px 14px 12px;
		border-top: 1px solid #303746;
		background: rgba(12, 14, 19, 0.92);
	}

	.btn-confirm {
		border: 1px solid rgba(114, 214, 208, 0.55);
		border-radius: 999px;
		background: rgba(114, 214, 208, 0.12);
		color: #bdf7f3;
		padding: 8px 18px;
		font-size: 12px;
		font-weight: 800;
		cursor: pointer;
	}
	.btn-confirm:hover {
		border-color: #72d6d0;
		background: rgba(114, 214, 208, 0.2);
	}

	.action-drawer {
		flex: 1;
		min-width: 140px;
		border: none;
		background: transparent;
	}

	.action-sum {
		padding: 8px 0;
		color: #7aa2ff;
	}

	.action-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		padding-bottom: 4px;
	}

	.action-grid button {
		border: 1px solid #465064;
		border-radius: 999px;
		background: rgba(28, 32, 42, 0.85);
		color: #d4d8e3;
		padding: 7px 13px;
		font-size: 11px;
		font-weight: 800;
		cursor: pointer;
	}

	.action-grid button:hover {
		border-color: #7aa2ff;
		color: #eef0f5;
	}
</style>
