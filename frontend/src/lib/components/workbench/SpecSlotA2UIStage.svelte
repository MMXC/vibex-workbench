<script lang="ts">
	import type { SpecSlotSession } from '$lib/stores/spec-slot-session-store';

	let { session }: { session: SpecSlotSession } = $props();

	const model = $derived.by(() => session.a2uiModel);
	const fireworks = $derived.by(() => session.fireworksGraph);
	const showFw = $derived.by(
		() =>
			!!model?.showFireworks &&
			!!fireworks &&
			fireworks.nodes.length > 0 &&
			session.a2uiStatus === 'ready'
	);
	const layout = $derived.by(() => model?.primaryStage ?? 'split');

	const hasPrimaryVisual = $derived.by(() => {
		if (!model) return false;
		const proto = !!model.prototype?.html;
		const fw = showFw;
		return proto || fw;
	});

	/** 本地工作台可信内容：保留脚本以支持交互原型；iframe 仍用 sandbox 限制弹窗/顶层导航等。 */
	const prototypeSrcdoc = $derived.by(() => (model?.prototype?.html ?? '').trim());
</script>

<div class="a2ui-stage" aria-label="A2UI 确认区" class:hero-layout={hasPrimaryVisual}>
	{#if session.a2uiStatus === 'loading'}
		<p class="loading">正在生成工具路由与 A2UI 确认组件…</p>
	{/if}

	{#if model}
		{#if model.uiWorkflowGate}
			<details class="gate-drawer" open={!model.uiWorkflowGate.canCommitPrototype}>
				<summary>
					Prototype Gate · {model.uiWorkflowGate.stage}
					{#if model.uiWorkflowGate.canCommitPrototype}
						<span class="gate-ok">已通过</span>
					{:else}
						<span class="gate-warn">待补齐</span>
					{/if}
				</summary>
				<ul class="gate-checks">
					{#each model.uiWorkflowGate.checks as c (c.id)}
						<li class:pass={c.passed} class:fail={!c.passed}>
							<span class="mark">{c.passed ? '✓' : '✗'}</span>
							{c.label}{c.detail ? ` — ${c.detail}` : ''}
						</li>
					{/each}
				</ul>
				{#if !model.uiWorkflowGate.canCommitPrototype}
					<pre class="gate-next">{model.uiWorkflowGate.nextAction}</pre>
				{/if}
			</details>
		{/if}
		{#if hasPrimaryVisual}
			<div class="head-min">
				<h3>{model.headline}</h3>
				{#if model.componentHints.length}
					<details class="hints-mini">
						<summary>映射提示</summary>
						<ul>
							{#each model.componentHints as hint (hint)}
								<li>{hint}</li>
							{/each}
						</ul>
					</details>
				{/if}
			</div>

			<div class="stage-stack stage-visual-priority">
				{#if model.prototype?.html}
					<div class="proto-wrap proto-hero">
						<span class="k">{model.prototype.caption ?? 'Prototype'}</span>
						{#if prototypeSrcdoc}
							<iframe
								class="proto-frame"
								title="prototype preview"
								sandbox="allow-scripts allow-same-origin"
								referrerpolicy="no-referrer"
								srcdoc={prototypeSrcdoc}
							></iframe>
						{:else}
							<p class="muted proto-fallback">暂无原型 HTML，请在对话或设计物料中生成内容。</p>
						{/if}
					</div>
				{/if}

				{#if showFw && fireworks}
					<div class="fw-wrap" class:fw-emphasis={layout === 'fireworks'}>
						<span class="k">Fireworks Tech Graph</span>
						<div class="fireworks-stage">
							{#each fireworks.nodes as node, index (node.id)}
								<div
									class="spark {node.type} {node.status}"
									style:left="{8 + (index % 4) * 28}%"
									style:top="{18 + Math.floor(index / 4) * 24}%"
								>
									<strong>{node.label}</strong>
									<small>{node.type} · {node.status}</small>
								</div>
							{/each}
							{#each fireworks.edges as edge, index (`${edge.from}-${edge.to}-${index}`)}
								<span class="spark-edge" style:top="{14 + index * 7}%">{edge.label}</span>
							{/each}
						</div>
					</div>
				{/if}

				<details class="cards-drawer">
					<summary class="cards-drawer-sum">说明与路由卡片 · {model.cards.length}</summary>
					<div class="cards-column">
						{#each model.cards as card (card.id)}
							<article
								class="a2-card"
								class:warn={card.emphasis === 'warning'}
								class:confirm={card.emphasis === 'confirm'}
							>
								<h4>{card.title}</h4>
								<p>{card.body}</p>
								{#if card.bullets?.length}
									<ul>
										{#each card.bullets as b}
											<li>{b}</li>
										{/each}
									</ul>
								{/if}
							</article>
						{/each}
					</div>
				</details>
			</div>
		{:else}
			<header class="head">
				<span class="k">A2UI</span>
				<h3>{model.headline}</h3>
				{#if model.componentHints.length}
					<details class="hints">
						<summary>组件映射提示（outputs）</summary>
						<ul>
							{#each model.componentHints as hint (hint)}
								<li>{hint}</li>
							{/each}
						</ul>
					</details>
				{/if}
			</header>

			<div class="stage-stack">
				<div class="cards-column">
					{#each model.cards as card (card.id)}
						<article
							class="a2-card"
							class:warn={card.emphasis === 'warning'}
							class:confirm={card.emphasis === 'confirm'}
						>
							<h4>{card.title}</h4>
							<p>{card.body}</p>
							{#if card.bullets?.length}
								<ul>
									{#each card.bullets as b}
										<li>{b}</li>
									{/each}
								</ul>
							{/if}
						</article>
					{/each}
				</div>

				{#if model.prototype?.html}
					<div class="proto-wrap">
						<span class="k">{model.prototype.caption ?? 'Prototype'}</span>
						{#if prototypeSrcdoc}
							<iframe
								class="proto-frame"
								title="prototype preview"
								sandbox="allow-scripts allow-same-origin"
								referrerpolicy="no-referrer"
								srcdoc={prototypeSrcdoc}
							></iframe>
						{:else}
							<p class="muted proto-fallback">暂无原型 HTML。</p>
						{/if}
					</div>
				{/if}

				{#if showFw && fireworks}
					<div class="fw-wrap" class:fw-emphasis={layout === 'fireworks'}>
						<span class="k">Fireworks Tech Graph</span>
						<div class="fireworks-stage">
							{#each fireworks.nodes as node, index (node.id)}
								<div
									class="spark {node.type} {node.status}"
									style:left="{8 + (index % 4) * 28}%"
									style:top="{18 + Math.floor(index / 4) * 24}%"
								>
									<strong>{node.label}</strong>
									<small>{node.type} · {node.status}</small>
								</div>
							{/each}
							{#each fireworks.edges as edge, index (`${edge.from}-${edge.to}-${index}`)}
								<span class="spark-edge" style:top="{14 + index * 7}%">{edge.label}</span>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		{/if}
	{:else if session.a2uiStatus !== 'loading'}
		<p class="muted">
			暂无 A2UI 模型。若路由 API 不可用，仍可通过左侧对话澄清；或点击「重新生成」。
		</p>
	{/if}
</div>

<style>
	.a2ui-stage {
		display: flex;
		flex-direction: column;
		gap: 10px;
		min-height: 0;
		flex: 1;
		overflow: hidden;
		padding: 2px 2px 4px;
	}

	.a2ui-stage.hero-layout {
		min-height: 0;
	}

	.loading {
		margin: 0;
		color: #72d6d0;
		font-size: 12px;
	}

	.muted {
		margin: 0;
		color: #858fa1;
		font-size: 12px;
		line-height: 1.5;
	}

	.head-min {
		flex-shrink: 0;
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 8px 12px;
	}

	.head-min h3 {
		margin: 0;
		font-size: 14px;
		color: #eef0f5;
		line-height: 1.25;
		flex: 1;
		min-width: 0;
	}

	.hints-mini {
		font-size: 10px;
		color: #a3abb9;
	}
	.hints-mini summary {
		cursor: pointer;
		color: #7aa2ff;
		font-weight: 700;
	}
	.hints-mini ul {
		margin: 6px 0 0;
		padding-left: 16px;
	}

	.gate-drawer {
		flex-shrink: 0;
		border: 1px solid rgba(239, 198, 107, 0.35);
		border-radius: 12px;
		padding: 8px 10px;
		background: rgba(239, 198, 107, 0.06);
		font-size: 11px;
		color: #d4d8e3;
	}
	.gate-drawer summary {
		cursor: pointer;
		color: #fcd34d;
		font-weight: 700;
		list-style-position: outside;
	}
	.gate-ok {
		margin-left: 8px;
		font-size: 10px;
		color: #72d6d0;
		font-weight: 800;
	}
	.gate-warn {
		margin-left: 8px;
		font-size: 10px;
		color: #fca5a5;
		font-weight: 800;
	}
	.gate-checks {
		margin: 8px 0 0;
		padding-left: 18px;
		line-height: 1.45;
	}
	.gate-checks li.pass {
		color: #a7f3d0;
	}
	.gate-checks li.fail {
		color: #fecaca;
	}
	.gate-checks .mark {
		display: inline-block;
		width: 1.2em;
		font-weight: 800;
	}
	.gate-next {
		margin: 8px 0 0;
		padding: 8px 10px;
		border-radius: 10px;
		background: #0a0c10;
		border: 1px solid #242b38;
		font-size: 10px;
		color: #a3abb9;
		white-space: pre-wrap;
		max-height: 160px;
		overflow: auto;
	}

	.head h3 {
		margin: 4px 0 0;
		font-size: 15px;
		color: #eef0f5;
		line-height: 1.25;
	}

	.k {
		display: block;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	.hints {
		margin-top: 8px;
		font-size: 11px;
		color: #a3abb9;
	}

	.hints summary {
		cursor: pointer;
		color: #7aa2ff;
	}

	.hints ul {
		margin: 6px 0 0;
		padding-left: 18px;
	}

	.stage-stack {
		display: flex;
		flex-direction: column;
		gap: 12px;
		min-height: 0;
		flex: 1;
		overflow: auto;
	}

	.stage-visual-priority {
		flex: 1;
		min-height: 0;
	}

	.cards-drawer {
		flex-shrink: 0;
		border: 1px solid #2a3140;
		border-radius: 12px;
		background: rgba(18, 21, 28, 0.65);
		overflow: hidden;
	}

	.cards-drawer-sum {
		cursor: pointer;
		list-style: none;
		padding: 10px 12px;
		font-size: 11px;
		font-weight: 800;
		color: #94a3b8;
		user-select: none;
	}
	.cards-drawer-sum::-webkit-details-marker {
		display: none;
	}
	.cards-drawer-sum::before {
		content: '▸ ';
		opacity: 0.5;
	}
	.cards-drawer[open] > .cards-drawer-sum::before {
		content: '▾ ';
	}

	.cards-drawer .cards-column {
		padding: 0 10px 10px;
		border-top: 1px solid #2a3140;
	}

	.cards-column {
		display: flex;
		flex-direction: column;
		gap: 10px;
		min-width: 0;
	}

	.a2-card {
		border: 1px solid #303746;
		border-radius: 14px;
		padding: 12px 14px;
		background: rgba(28, 32, 42, 0.78);
		color: #eef0f5;
	}

	.a2-card.confirm {
		border-color: rgba(239, 198, 107, 0.55);
		background: rgba(239, 198, 107, 0.07);
	}

	.a2-card.warn {
		border-color: rgba(248, 113, 113, 0.45);
		background: rgba(248, 113, 113, 0.06);
	}

	.a2-card h4 {
		margin: 0 0 6px;
		font-size: 13px;
	}

	.a2-card p {
		margin: 0;
		font-size: 12px;
		color: #a3abb9;
		line-height: 1.45;
	}

	.a2-card ul {
		margin: 8px 0 0;
		padding-left: 18px;
		font-size: 11px;
		color: #858fa1;
	}

	.proto-wrap {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
		flex-shrink: 0;
	}

	.proto-hero {
		flex: 1;
		min-height: 220px;
	}

	.proto-frame {
		width: 100%;
		min-height: 200px;
		height: 280px;
		border: 1px solid #242b38;
		border-radius: 12px;
		background: #0a0c10;
	}

	.proto-hero .proto-frame {
		flex: 1;
		min-height: min(52vh, 480px);
		height: auto;
	}

	.fw-wrap {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
		flex-shrink: 0;
	}

	.fw-wrap.fw-emphasis .fireworks-stage {
		min-height: 300px;
	}

	.fireworks-stage {
		position: relative;
		height: 260px;
		min-height: 200px;
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
		width: 128px;
		min-height: 46px;
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
