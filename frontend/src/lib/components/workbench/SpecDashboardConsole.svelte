<script lang="ts">
	import {
		specExplorerStore,
		type SpecDashboardLevel,
	} from '$lib/stores/spec-explorer-store';
	import { specAgentContextStore } from '$lib/stores/spec-agent-context-store';
	import { fallbackDisplayTitle, type SpecSlotModel } from '$lib/workbench/spec-display';

	type DashboardSpec = {
		path: string;
		level: number;
		name: string;
		status: string;
		display?: { title: string; summary: string; description: string };
		slots?: SpecSlotModel;
	};

	type LevelCard = {
		id: SpecDashboardLevel;
		level: number;
		label: string;
		en: string;
		desc: string;
		addLabel: string;
	};

	const levels: LevelCard[] = [
		{ id: 'goal', level: 1, label: '目标', en: 'Goal', desc: '项目目标与验收边界', addLabel: '新增 Goal' },
		{ id: 'skeleton', level: 2, label: '骨架', en: 'Skeleton', desc: '系统骨架与主干流程', addLabel: '新增 Skeleton' },
		{ id: 'module', level: 3, label: '模块', en: 'Module', desc: '领域模块与职责边界', addLabel: '新增 Module' },
		{ id: 'feature', level: 4, label: '功能', en: 'Feature', desc: '用户功能与交互闭环', addLabel: '新增 Feature' },
		{ id: 'slice', level: 5, label: '实现', en: 'Slice', desc: '实现切片与代码落点', addLabel: '新增 Slice' },
	];

	const activeCard = $derived.by(() => levels.find(level => level.id === $specExplorerStore.dashboardLevel) ?? levels[0]);
	const specs = $derived.by(() => $specExplorerStore.specs as DashboardSpec[]);
	const activeSpecs = $derived.by(() => specs.filter(spec => spec.level === activeCard.level));
	const unknownSpecs = $derived.by(() => specs.filter(spec => spec.level < 1 || spec.level > 5));

	function levelSpecs(level: number): DashboardSpec[] {
		return specs.filter(spec => spec.level === level);
	}

	function titleOf(spec: DashboardSpec): string {
		return spec.display?.title || fallbackDisplayTitle(spec.name);
	}

	function summaryOf(spec: DashboardSpec): string {
		return spec.display?.summary || spec.path.replace(/^specs\//, '').replace(/\.ya?ml$/, '');
	}

	function slotStats(specList: DashboardSpec[]) {
		let present = 0;
		let total = 0;
		let missing = 0;
		for (const spec of specList) {
			const slots = spec.slots?.all ?? [];
			total += slots.length;
			present += slots.filter(slot => slot.status === 'present').length;
			missing += slots.filter(slot => slot.status === 'missing' || slot.status === 'empty').length;
		}
		return { present, total, missing };
	}

	function statusCount(specList: DashboardSpec[], status: string): number {
		return specList.filter(spec => spec.status === status).length;
	}

	function openSpec(path: string) {
		specExplorerStore.selectSpec(path, 'dashboard');
	}

	function createIntent(card: LevelCard) {
		const parentHint = card.level === 1
			? '这是 L1 根目标，不需要 parent。'
			: `请先根据现有 L${card.level - 1} spec 推断或询问 parent。`;
		specAgentContextStore.prefillCommand(
			`/add "新增 ${card.label} ${card.en} spec。层级 L${card.level}。${parentHint} 先生成可确认草案，不要直接写文件。"`
		);
	}

</script>

<section class="dashboard" aria-label="Spec 总控制台">
	<header class="hero">
		<div>
			<span class="eyebrow">Spec Dashboard</span>
			<h1>规格总控制台</h1>
			<p>按目标、骨架、模块、功能、实现切片管理 specs；左侧资源树保留为完整文件索引。</p>
		</div>
		<div class="hero-count">
			<strong>{specs.length}</strong>
			<span>total specs</span>
		</div>
	</header>

	<div class="level-grid">
		{#each levels as card (card.id)}
			{@const items = levelSpecs(card.level)}
			{@const slots = slotStats(items)}
			<button
				type="button"
				class="level-card"
				class:active={$specExplorerStore.dashboardLevel === card.id}
				onclick={() => specExplorerStore.setDashboardLevel(card.id)}
			>
				<span class="topline">
					<span class="level-code">L{card.level}</span>
					<strong>{items.length}</strong>
				</span>
				<span class="name">{card.label} <small>{card.en}</small></span>
				<span class="desc">{card.desc}</span>
				<span class="badges">
					<span>{statusCount(items, 'active')} active</span>
					<span>{slots.present}/{slots.total || 0} slots</span>
					<span>{slots.missing} 待补</span>
				</span>
			</button>
		{/each}
	</div>

	<div class="list-head">
		<div>
			<h2>{activeCard.label} {activeCard.en}</h2>
			<p>L{activeCard.level} · {activeSpecs.length} 个 spec</p>
		</div>
		<button type="button" class="add-btn" onclick={() => createIntent(activeCard)}>{activeCard.addLabel}</button>
	</div>

	{#if $specExplorerStore.specsLoading}
		<div class="state">加载 specs…</div>
	{:else if $specExplorerStore.specsError}
		<div class="state err">{$specExplorerStore.specsError}</div>
	{:else if activeSpecs.length === 0}
		<div class="state">当前层级暂无 spec。可以点击右上角新增入口生成创建意图。</div>
	{:else}
		<div class="spec-list">
			{#each activeSpecs as spec (spec.path)}
				{@const slots = spec.slots?.all ?? []}
				{@const present = slots.filter(slot => slot.status === 'present').length}
				<article class="spec-card">
					<div class="spec-main">
						<span class="path">{spec.path}</span>
						<h3>{titleOf(spec)}</h3>
						<p>{summaryOf(spec)}</p>
						<div class="slot-line">
							{#each slots as slot (slot.id)}
								<span class:ok={slot.status === 'present'}>{slot.label} {slot.status === 'present' ? '✓' : '待补'}</span>
							{/each}
						</div>
					</div>
					<div class="spec-side">
						<span class="status">{spec.status}</span>
						<span class="ratio">{present}/{slots.length || 0}</span>
						<button type="button" onclick={() => openSpec(spec.path)}>详情</button>
					</div>
				</article>
			{/each}
		</div>
	{/if}

	{#if unknownSpecs.length > 0}
		<div class="unknown">
			<strong>未分类 specs</strong>
			<span>{unknownSpecs.length} 个文件未匹配 L1-L5 层级，可从左侧资源树查看。</span>
		</div>
	{/if}
</section>

<style>
	.dashboard {
		flex: 1;
		min-height: 0;
		overflow: auto;
		padding: 18px;
		background:
			radial-gradient(circle at 12% 0%, rgba(122, 162, 255, 0.12), transparent 28%),
			#0b0c10;
		color: #eef0f5;
	}

	.hero {
		display: flex;
		align-items: stretch;
		justify-content: space-between;
		gap: 18px;
		margin-bottom: 16px;
		padding: 18px;
		border: 1px solid #303746;
		border-radius: 18px;
		background: rgba(28, 32, 42, 0.78);
	}

	.eyebrow {
		display: block;
		margin-bottom: 7px;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.14em;
		text-transform: uppercase;
	}

	h1,
	h2,
	h3,
	p {
		margin: 0;
	}

	h1 {
		font-size: 22px;
		letter-spacing: -0.03em;
	}

	.hero p,
	.list-head p,
	.spec-card p,
	.unknown span {
		margin-top: 6px;
		color: #a3abb9;
		font-size: 12px;
		line-height: 1.5;
	}

	.hero-count {
		min-width: 128px;
		display: grid;
		place-items: center;
		border: 1px solid rgba(122, 162, 255, 0.35);
		border-radius: 16px;
		background: rgba(122, 162, 255, 0.1);
	}

	.hero-count strong {
		font-size: 30px;
		line-height: 1;
	}

	.hero-count span {
		color: #a3abb9;
		font-size: 10px;
		text-transform: uppercase;
	}

	.level-grid {
		display: grid;
		grid-template-columns: repeat(5, minmax(150px, 1fr));
		gap: 10px;
		margin-bottom: 16px;
	}

	.level-card {
		min-height: 142px;
		padding: 13px;
		border: 1px solid #303746;
		border-radius: 16px;
		background: rgba(21, 24, 32, 0.86);
		color: #eef0f5;
		text-align: left;
		cursor: pointer;
	}

	.level-card:hover,
	.level-card.active {
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.12);
	}

	.topline,
	.badges,
	.list-head,
	.spec-card,
	.spec-side {
		display: flex;
		align-items: center;
	}

	.topline,
	.list-head,
	.spec-card {
		justify-content: space-between;
	}

	.level-code,
	.status,
	.ratio,
	.badges span {
		border: 1px solid #465064;
		border-radius: 999px;
		padding: 3px 7px;
		color: #a3abb9;
		font-family: ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
	}

	.topline strong {
		font-size: 24px;
	}

	.name {
		display: block;
		margin-top: 12px;
		font-size: 15px;
		font-weight: 850;
	}

	.name small {
		color: #72d6d0;
		font-size: 11px;
	}

	.desc {
		display: block;
		margin-top: 7px;
		min-height: 34px;
		color: #a3abb9;
		font-size: 11px;
		line-height: 1.45;
	}

	.badges {
		flex-wrap: wrap;
		gap: 5px;
		margin-top: 10px;
	}

	.list-head {
		margin-bottom: 10px;
		padding: 12px 14px;
		border: 1px solid #303746;
		border-radius: 15px;
		background: rgba(12, 14, 19, 0.58);
	}

	h2 {
		font-size: 16px;
	}

	.add-btn,
	.spec-side button {
		border: 1px solid rgba(114, 214, 208, 0.5);
		border-radius: 999px;
		background: rgba(114, 214, 208, 0.1);
		color: #bdf7f3;
		padding: 7px 12px;
		font-size: 12px;
		font-weight: 800;
		cursor: pointer;
	}

	.add-btn:hover,
	.spec-side button:hover {
		background: rgba(114, 214, 208, 0.18);
		color: #eef0f5;
	}

	.spec-list {
		display: grid;
		gap: 9px;
	}

	.spec-card {
		gap: 14px;
		padding: 13px 14px;
		border: 1px solid #303746;
		border-radius: 15px;
		background: rgba(28, 32, 42, 0.62);
	}

	.spec-main {
		min-width: 0;
	}

	.path {
		color: #6f7888;
		font-family: ui-monospace, monospace;
		font-size: 10px;
	}

	.spec-card h3 {
		margin-top: 5px;
		font-size: 14px;
	}

	.slot-line {
		display: flex;
		flex-wrap: wrap;
		gap: 5px;
		margin-top: 9px;
	}

	.slot-line span {
		border: 1px solid #303746;
		border-radius: 999px;
		padding: 2px 7px;
		color: #858fa1;
		font-size: 10px;
		font-weight: 700;
	}

	.slot-line span.ok {
		border-color: rgba(114, 214, 208, 0.42);
		color: #72d6d0;
	}

	.spec-side {
		flex-shrink: 0;
		flex-direction: column;
		gap: 8px;
	}

	.state,
	.unknown {
		padding: 16px;
		border: 1px dashed #465064;
		border-radius: 14px;
		color: #a3abb9;
		background: rgba(28, 32, 42, 0.42);
	}

	.state.err {
		color: #f87171;
		border-color: rgba(248, 113, 113, 0.4);
	}

	.unknown {
		margin-top: 14px;
		display: grid;
		gap: 4px;
	}

	@media (max-width: 1100px) {
		.level-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
</style>
