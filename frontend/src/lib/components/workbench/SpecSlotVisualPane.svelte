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
	const verificationPlan = $derived.by(() => session.strongValidationPlan ?? null);
	const verificationItems = $derived.by(() => verificationPlan?.items ?? []);
	const verificationRuns = $derived.by(() => session.validationRuns ?? {});
	const verificationTrace = $derived.by(() => session.traceEvents ?? []);
	const verificationRequired = $derived.by(
		() =>
			(session.slot.id === 'prototype' || session.slot.id === 'implementation') &&
			verificationItems.length > 0
	);
	const verificationReady = $derived.by(() =>
		verificationRequired ? !!session.verificationSubmission : true
	);
	const iterNodes = $derived.by(() => session.iterationNodes ?? []);
	const iterEdges = $derived.by(() => session.iterationEdges ?? []);
	const iterCursorId = $derived.by(() => session.iterationCursorId);
	let expandIterHistory = $state(false);
	let selectedIterNodeId: string | null = $state(null);

	type IterLayoutNode = {
		id: string;
		label: string;
		type: 'run' | 'plan' | 'tool' | 'result';
		phase: 'init' | 'plan' | 'execute' | 'validate' | 'repair' | 'complete';
		status: 'running' | 'done' | 'error';
		detail?: string;
		x: number;
		y: number;
	};

	type IterLayoutEdge = {
		from: string;
		to: string;
		label?: string;
		x1: number;
		y1: number;
		x2: number;
		y2: number;
		failed: boolean;
		active: boolean;
	};

	const iterLayout = $derived.by(() => {
		/** 泳道：禁止用 detail 中 “Please repair…” 等筛子提示做子串匹配，否则会几乎把所有 tool 划进 repair。 */
		const inferPhase = (
			node: Pick<IterLayoutNode, 'type' | 'label' | 'status' | 'detail'>
		): IterLayoutNode['phase'] => {
			const label = (node.label ?? '').toLowerCase();
			const detail = (node.detail ?? '').toLowerCase();

			if (node.type === 'run') return 'init';
			if (node.type === 'result') {
				if (label.includes('failed') || node.status === 'error') return 'repair';
				return 'complete';
			}

			if (node.type === 'plan') {
				if (label === 'repair.decision' || label.startsWith('repair:')) return 'repair';
				return 'plan';
			}

			if (node.type === 'tool') {
				if (node.status === 'error') return 'repair';
				const head = (detail.trimStart().split('\n')[0] ?? '');
				if (
					head.startsWith('filter_rejected:') ||
					head.startsWith('error:') ||
					head.startsWith('blocked:') ||
					head.startsWith('unsupported tool')
				) {
					return 'repair';
				}
				return 'execute';
			}

			if (label.includes('validate') || detail.trimStart().startsWith('validate')) return 'validate';
			return 'execute';
		};
		const laneOrder: Record<IterLayoutNode['phase'], number> = {
			init: 0,
			plan: 1,
			execute: 2,
			validate: 3,
			repair: 4,
			complete: 5,
		};
		const laneCount = new Map<number, number>();
		const nodes: IterLayoutNode[] = iterNodes.map(node => {
			const phase = inferPhase(node);
			const lane = laneOrder[phase] ?? 0;
			const idx = laneCount.get(lane) ?? 0;
			laneCount.set(lane, idx + 1);
			return {
				...node,
				phase,
				x: 70 + lane * 118,
				y: 44 + idx * 70,
			};
		});
		const index = new Map(nodes.map(n => [n.id, n]));
		const edges: IterLayoutEdge[] = iterEdges
			.map(edge => {
				const from = index.get(edge.from);
				const to = index.get(edge.to);
				if (!from || !to) return null;
				return {
					...edge,
					x1: from.x,
					y1: from.y,
					x2: to.x,
					y2: to.y,
					failed: (edge.label ?? '').toLowerCase().includes('error'),
					active: edge.to === iterCursorId,
				};
			})
			.filter((e): e is IterLayoutEdge => !!e);
		const height = Math.max(220, ...nodes.map(n => n.y + 44));
		const width = 760;
		return { nodes, edges, width, height };
	});

	const iterVisible = $derived.by(() => {
		const maxKeep = 8;
		if (expandIterHistory || iterLayout.nodes.length <= maxKeep) {
			const idSet = new Set(iterLayout.nodes.map(n => n.id));
			return {
				nodes: iterLayout.nodes,
				edges: iterLayout.edges.filter(e => idSet.has(e.from) && idSet.has(e.to)),
				hiddenCount: 0,
			};
		}
		const nodes = iterLayout.nodes.slice(-maxKeep);
		const idSet = new Set(nodes.map(n => n.id));
		return {
			nodes,
			edges: iterLayout.edges.filter(e => idSet.has(e.from) && idSet.has(e.to)),
			hiddenCount: iterLayout.nodes.length - nodes.length,
		};
	});

	const iterLaneStats = $derived.by(() => {
		const acc = { init: 0, plan: 0, execute: 0, validate: 0, repair: 0, complete: 0 };
		for (const node of iterLayout.nodes) acc[node.phase] += 1;
		return acc;
	});

	const selectedIterNode = $derived.by(
		() =>
			iterLayout.nodes.find(n => n.id === (selectedIterNodeId ?? iterCursorId ?? '')) ??
			iterLayout.nodes.at(-1) ??
			null
	);

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

			<div class="route-card compact">
				<span class="k">Iteration Graph（动态）</span>
				<details class="iter-collapsible">
					<summary class="iter-collapsible-sum">查看运行泳道与节点轨迹</summary>
				{#if iterNodes.length > 0}
					<div class="iter-toolbar">
						<div class="iter-lanes">
							<span>init {iterLaneStats.init}</span>
							<span>plan {iterLaneStats.plan}</span>
							<span>execute {iterLaneStats.execute}</span>
							<span>validate {iterLaneStats.validate}</span>
							<span>repair {iterLaneStats.repair}</span>
							<span>complete {iterLaneStats.complete}</span>
						</div>
						{#if iterVisible.hiddenCount > 0}
							<button
								type="button"
								class="iter-toggle"
								onclick={() => (expandIterHistory = !expandIterHistory)}
							>
								{expandIterHistory
									? '收起历史'
									: `展开历史（+${iterVisible.hiddenCount}）`}
							</button>
						{/if}
					</div>
					<div class="iter-graph-viewport">
						<svg
							class="iter-svg"
							width={iterLayout.width}
							height={iterLayout.height}
							viewBox={`0 0 ${iterLayout.width} ${iterLayout.height}`}
							preserveAspectRatio="xMinYMin meet"
							aria-label="iteration graph"
						>
							{#each iterVisible.edges as e (`${e.from}-${e.to}`)}
								<line
									x1={e.x1}
									y1={e.y1}
									x2={e.x2}
									y2={e.y2}
									class="iter-line"
									class:error={e.failed}
									class:active={e.active}
								/>
							{/each}
							{#each iterVisible.nodes as n (n.id)}
								<circle
									cx={n.x}
									cy={n.y}
									r={n.id === iterCursorId ? 11 : 8}
									class="iter-dot"
									class:run={n.type === 'run'}
									class:plan={n.type === 'plan'}
									class:tool={n.type === 'tool'}
									class:result={n.type === 'result'}
									class:error={n.status === 'error'}
									class:active={n.id === iterCursorId}
								/>
							{/each}
						</svg>
					</div>
					<div class="iter-graph">
						{#each iterVisible.nodes as n (n.id)}
							<button
								type="button"
								class="iter-node {n.type} {n.status}"
								class:active={n.id === iterCursorId}
								class:selected={n.id === (selectedIterNodeId ?? iterCursorId)}
								onclick={() => (selectedIterNodeId = n.id)}
							>
								<strong>{n.label}</strong>
								<small>{n.type} · {n.status}</small>
								{#if n.detail}
									<small class="detail">{n.detail}</small>
								{/if}
							</button>
						{/each}
					</div>
					{#if selectedIterNode}
						<div class="iter-detail">
							<strong>当前节点详情</strong>
							<small>{selectedIterNode.label}</small>
							<small>{selectedIterNode.type} · {selectedIterNode.status}</small>
							{#if selectedIterNode.detail}
								<code>{selectedIterNode.detail}</code>
							{/if}
						</div>
					{/if}
					{#if iterVisible.edges.length > 0}
						<div class="iter-edges">
							{#each iterVisible.edges as e, i (`${e.from}-${e.to}-${i}`)}
								<div class="iter-edge" class:error={e.failed} class:active={e.active}>
									<code>{e.from.split(':')[0]}</code> → <code>{e.to.split(':')[0]}</code>
									{#if e.label}<span>{e.label}</span>{/if}
								</div>
							{/each}
						</div>
					{/if}
				{:else}
					<p class="route-fallback">提交后会实时显示 run/planning/tool 调用迭代轨迹。</p>
				{/if}
				</details>
			</div>
		</div>
	</details>

	{#if session.slot.id === 'prototype'}
		<details class="drawer kit-drawer" open>
			<summary class="drawer-sum">设计物料库 · DESIGN.md / 从页面提取 · Shell 路由</summary>
			<div class="drawer-body kit-body">
				<SpecSlotPrototypeKitBar {session} />
			</div>
		</details>
	{/if}

	<details class="drawer verify-drawer" open={verificationRequired}>
		<summary class="drawer-sum">强校验 · Verification Bar</summary>
		<div class="drawer-body verify-body">
			{#if verificationItems.length === 0}
				<p class="route-fallback">当前槽位暂无 StrongValidationPlan 条目。</p>
			{:else}
				<div class="verify-items">
					{#each verificationItems as item (item.id)}
						<div class="verify-item">
							<div class="verify-meta">
								<strong>{item.label || item.id}</strong>
								<small>{item.tool_call_template || item.command || 'no command/template'}</small>
								{#if item.timeout_sec || item.expect_signal}
									<small>
										{item.timeout_sec ? `${item.timeout_sec}s` : ''}
										{item.expect_signal ? ` · expect ${item.expect_signal}` : ''}
									</small>
								{/if}
							</div>
							<div class="verify-actions">
								<button type="button" onclick={() => specSlotSessionStore.runValidationItem(item.id)}>
									运行
								</button>
								{#if verificationRuns[item.id]}
									<span class="verify-run {verificationRuns[item.id].ok ? 'ok' : 'fail'}">
										{verificationRuns[item.id].ok
											? 'OK'
											: verificationRuns[item.id].not_implemented
												? '待实现'
												: 'FAIL'}
									</span>
									{#if verificationRuns[item.id].source}
										<span class="verify-source">
											{verificationRuns[item.id].source === 'custom_agent_flow'
												? 'Custom Agent Flow'
												: verificationRuns[item.id].source === 'legacy_qa'
													? 'Legacy QA'
													: 'Builtin'}
										</span>
									{/if}
								{/if}
							</div>
						</div>
					{/each}
				</div>
				<div class="verify-submit">
					<button type="button" class="verify-green" onclick={() => specSlotSessionStore.submitVerificationOutcome('passed')}>
						标记 GREEN
					</button>
					<button type="button" class="verify-red" onclick={() => specSlotSessionStore.submitVerificationOutcome('failed')}>
						标记 RED
					</button>
					{#if session.verificationSubmission}
						<small class="verify-sub">
							已提交：{session.verificationSubmission.outcome} · {session.verificationSubmission.submission_id}
						</small>
					{/if}
				</div>
			{/if}
			{#if verificationTrace.length > 0}
				<details>
					<summary class="trace-sum">trace · {verificationTrace.length}</summary>
					<div class="trace-list">
						{#each verificationTrace.slice(-10).reverse() as t, i (`${t.node.node_id}-${i}`)}
							<div class="trace-item">
								<code>{t.phase}</code>
								<span>{t.node.kind || t.node.node_id}</span>
								<small>{t.outcome_summary || '—'}</small>
							</div>
						{/each}
					</div>
				</details>
			{/if}
		</div>
	</details>

	<div class="action-zone">
		<button
			type="button"
			class="btn-confirm"
			disabled={!verificationReady}
			title={!verificationReady ? '请先提交本槽位验证结果（GREEN/RED）' : undefined}
			onclick={() => specSlotSessionStore.confirmActiveA2UI()}
		>
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

	.iter-graph-viewport {
		margin-top: 8px;
		border: 1px solid #2f384a;
		border-radius: 10px;
		background:
			radial-gradient(circle at 20% 20%, rgba(122, 162, 255, 0.09), transparent 40%),
			rgba(12, 16, 26, 0.8);
		overflow: auto;
		max-height: min(520px, 58vh);
	}

	.iter-toolbar {
		margin-top: 8px;
		display: flex;
		justify-content: space-between;
		gap: 8px;
		align-items: center;
		flex-wrap: wrap;
	}

	.iter-lanes {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.iter-lanes span {
		font-size: 10px;
		color: #9fb2d1;
		border: 1px solid #374861;
		border-radius: 999px;
		padding: 2px 8px;
		background: rgba(30, 41, 59, 0.5);
	}

	.iter-toggle {
		border: 1px solid #3d4d68;
		background: rgba(30, 41, 59, 0.55);
		color: #d1def5;
		border-radius: 999px;
		font-size: 10px;
		font-weight: 700;
		padding: 4px 10px;
		cursor: pointer;
	}

	.iter-svg {
		display: block;
		max-width: 100%;
		height: auto;
		min-height: 220px;
		vertical-align: top;
	}

	.iter-line {
		stroke: rgba(122, 162, 255, 0.36);
		stroke-width: 1.5;
		stroke-dasharray: 6 6;
		animation: iter-flow 1.8s linear infinite;
	}

	.iter-line.error {
		stroke: rgba(248, 113, 113, 0.75);
	}

	.iter-line.active {
		stroke-width: 2.6;
		stroke: rgba(114, 214, 208, 0.85);
	}

	.iter-dot {
		fill: #7aa2ff;
		stroke: rgba(226, 232, 240, 0.8);
		stroke-width: 1;
	}

	.iter-dot.plan {
		fill: #72d6d0;
	}

	.iter-dot.tool {
		fill: #a8b6d1;
	}

	.iter-dot.result {
		fill: #e8c94b;
	}

	.iter-dot.error {
		fill: #f87171;
	}

	.iter-dot.active {
		filter: drop-shadow(0 0 6px rgba(114, 214, 208, 0.75));
	}

	.iter-graph {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
		gap: 8px;
		margin-top: 8px;
	}

	.iter-node {
		border: 1px solid #334155;
		border-radius: 10px;
		padding: 8px;
		background: rgba(15, 20, 30, 0.55);
		display: flex;
		flex-direction: column;
		gap: 3px;
		text-align: left;
		cursor: pointer;
	}

	.iter-node.run {
		border-color: rgba(122, 162, 255, 0.45);
	}

	.iter-node.plan {
		border-color: rgba(114, 214, 208, 0.45);
	}

	.iter-node.tool {
		border-color: rgba(148, 163, 184, 0.45);
	}

	.iter-node.result {
		border-color: rgba(236, 201, 75, 0.5);
	}

	.iter-node.done {
		box-shadow: inset 0 0 0 1px rgba(114, 214, 208, 0.35);
	}

	.iter-node.error {
		box-shadow: inset 0 0 0 1px rgba(248, 113, 113, 0.4);
	}

	.iter-node.active {
		box-shadow:
			inset 0 0 0 1px rgba(114, 214, 208, 0.45),
			0 0 0 1px rgba(114, 214, 208, 0.25);
	}

	.iter-node.selected {
		outline: 1px solid rgba(122, 162, 255, 0.6);
	}

	.iter-node strong {
		font-size: 11px;
		color: #e2e8f0;
		word-break: break-word;
	}

	.iter-node small {
		font-size: 10px;
		color: #94a3b8;
	}

	.iter-node .detail {
		color: #cbd5e1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.iter-edges {
		margin-top: 8px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		font-size: 10px;
		color: #9aa7bb;
	}

	.iter-detail {
		margin-top: 8px;
		border: 1px solid rgba(122, 162, 255, 0.35);
		border-radius: 10px;
		background: rgba(16, 22, 34, 0.8);
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.iter-detail strong {
		font-size: 11px;
		color: #dbeafe;
	}

	.iter-detail small {
		font-size: 10px;
		color: #9fb2d1;
	}

	.iter-detail code {
		font-size: 10px;
		color: #dbeafe;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.iter-edge {
		display: flex;
		gap: 6px;
		align-items: center;
	}

	.iter-edge.error {
		color: #fda4af;
	}

	.iter-edge.active {
		color: #72d6d0;
	}

	.iter-edge code {
		font-size: 10px;
		color: #cbd5e1;
	}

	@keyframes iter-flow {
		to {
			stroke-dashoffset: -12;
		}
	}

	.verify-body {
		max-height: 36vh;
	}
	.verify-items {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.verify-item {
		display: flex;
		justify-content: space-between;
		gap: 10px;
		padding: 8px 10px;
		border: 1px solid #303746;
		border-radius: 10px;
		background: rgba(18, 21, 28, 0.9);
	}
	.verify-meta {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.verify-meta small {
		color: #9aa3b5;
		font-size: 11px;
	}
	.verify-actions {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.verify-actions button,
	.verify-submit button {
		border: 1px solid #465064;
		border-radius: 999px;
		background: rgba(28, 32, 42, 0.9);
		color: #e7ebf5;
		padding: 6px 11px;
		font-size: 11px;
		font-weight: 700;
		cursor: pointer;
	}
	.verify-run {
		font-size: 10px;
		font-weight: 800;
		padding: 2px 8px;
		border-radius: 999px;
		border: 1px solid #475569;
	}
	.verify-run.ok {
		color: #86efac;
		border-color: rgba(134, 239, 172, 0.45);
	}
	.verify-run.fail {
		color: #fda4af;
		border-color: rgba(253, 164, 175, 0.45);
	}
	.verify-submit {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.verify-source {
		display: inline-flex;
		align-items: center;
		padding: 2px 8px;
		border-radius: 999px;
		border: 1px solid #3d557a;
		background: rgba(122, 162, 255, 0.16);
		color: #cfe0ff;
		font-size: 10px;
		font-weight: 700;
	}
	.verify-green {
		border-color: rgba(134, 239, 172, 0.45) !important;
		color: #86efac !important;
	}
	.verify-red {
		border-color: rgba(253, 164, 175, 0.45) !important;
		color: #fda4af !important;
	}
	.verify-sub {
		font-size: 11px;
		color: #9aa3b5;
	}
	.trace-sum {
		font-size: 11px;
		color: #72d6d0;
	}
	.trace-list {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-top: 6px;
	}
	.trace-item {
		display: flex;
		gap: 8px;
		align-items: center;
		font-size: 11px;
		color: #cdd6e5;
	}
	.trace-item small {
		color: #9aa3b5;
	}

	.iter-collapsible {
		margin-top: 8px;
		border-top: 1px dashed #3b4455;
		padding-top: 8px;
	}
	.iter-collapsible-sum {
		cursor: pointer;
		font-size: 11px;
		color: #9aa3b5;
		font-weight: 700;
	}
	.iter-collapsible-sum::-webkit-details-marker {
		display: none;
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
	.btn-confirm:disabled {
		opacity: 0.5;
		cursor: not-allowed;
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
