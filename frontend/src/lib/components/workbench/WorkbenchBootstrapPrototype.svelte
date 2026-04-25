<script lang="ts">
	type StageState = 'done' | 'active' | 'blocked' | 'next';

	const stages: { id: string; title: string; state: StageState; detail: string }[] = [
		{
			id: 'open',
			title: '打开仓库',
			state: 'done',
			detail: 'Wails 菜单与 workspace root 已进入产品主路径',
		},
		{
			id: 'detect',
			title: '三态探测',
			state: 'done',
			detail: 'empty / half / ready 由 state_detector.py 输出',
		},
		{
			id: 'clarify',
			title: '澄清目标',
			state: 'active',
			detail: '图谱页确认 MVP 边界、前后端栈与生成器规格',
		},
		{
			id: 'scaffold',
			title: '初始化脚手架',
			state: 'blocked',
			detail: 'scaffold API 还需清掉硬编码脚本路径',
		},
		{
			id: 'validate',
			title: '校验 / 生成',
			state: 'next',
			detail: 'make validate 已通过，下一步闭合 run-make 与桌面后端',
		},
	];

	const specTree = [
		{ level: 'L1', name: 'vibex-workbench-mvp', note: '空仓库到可运行 MVP 的目标' },
		{ level: 'L2', name: 'vibex-workbench-skeleton', note: '工作区、脚手架、编辑器、构建面板' },
		{ level: 'L3', name: 'MOD-workspace-root', note: '仓库根绑定与桌面选择器' },
		{ level: 'L3', name: 'MOD-state-detection', note: '空 / 半 / 就绪探测' },
		{ level: 'L3', name: 'MOD-scaffolding', note: 'spec-templates 落盘' },
		{ level: 'L4', name: 'FEAT-workspace-selector', note: '用户确认目录后进入 Workbench' },
	];

	const currentFacts = [
		'最新 validator 已支持新 spec 格式与 L3-module 布局',
		'make validate 当前通过，parent chain 已闭合',
		'Wails shell 可编译并进入 dev server',
		'backend 自动拉起仍卡在 Windows 产物路径',
	];

	const blockers = [
		{
			title: 'agent-build 输出路径跑偏',
			body: 'Go 在 agent/cmd/web 下输出了 backend/vibex-backend，Wails 期待仓库根 backend/vibex-backend.exe。',
		},
		{
			title: 'scaffold API 仍有测试路径',
			body: 'workspaceScaffoldHandler 还会查 /root/v-test/generators/scaffolder.py，需要改成仓库相对解析。',
		},
		{
			title: '图谱确认流未成型',
			body: '目标澄清、生成器规格确认、写入 L1/L2 的 UI 还只是产品边界，未形成完整 wizard。',
		},
	];

	const stateTone: Record<StageState, string> = {
		done: '已具备',
		active: '评审中',
		blocked: '阻塞',
		next: '下一步',
	};
</script>

<section class="prototype-shell" aria-label="VibeX Workbench 自举原型">
	<div class="hero">
		<div>
			<p class="eyebrow">SELF-BOOTSTRAP MVP</p>
			<h1>从空仓库长出项目</h1>
			<p class="lead">
				这版原型把截图里的 IDE 布局转成产品主路径：打开仓库、探测状态、澄清目标、
				确认生成器规格，然后由 spec 维护脚手架与生成命令。
			</p>
		</div>

		<div class="health-card">
			<span class="health-kicker">当前判断</span>
			<strong>spec 自举层已通过</strong>
			<p>运行层剩余关键缺口：桌面后端自动启动与脚手架 API 路径。</p>
		</div>
	</div>

	<div class="stage-rail" aria-label="MVP 链路">
		{#each stages as stage, index (stage.id)}
			<article class="stage-card {stage.state}">
				<div class="stage-topline">
					<span class="stage-index">{String(index + 1).padStart(2, '0')}</span>
					<span class="stage-state">{stateTone[stage.state]}</span>
				</div>
				<h2>{stage.title}</h2>
				<p>{stage.detail}</p>
			</article>
		{/each}
	</div>

	<div class="workspace-grid">
		<section class="panel large">
			<div class="panel-title">
				<span>图谱页需要承载的最小确认流</span>
				<small>human-in-the-loop</small>
			</div>
			<div class="flow-board">
				<div class="flow-node primary">
					<b>用户目标</b>
					<span>我要在这个空仓库做什么产品？</span>
				</div>
				<div class="flow-node">
					<b>生成器规格</b>
					<span>前端框架、后端框架、包管理器、是否带 agent</span>
				</div>
				<div class="flow-node">
					<b>Spec 真源</b>
					<span>L1/L2 写入 specs/，脚本落到 generators/ 与 Makefile</span>
				</div>
				<div class="flow-node accent">
					<b>可运行证明</b>
					<span>validate / generate / dev 至少一条路径成功</span>
				</div>
			</div>
		</section>

		<section class="panel">
			<div class="panel-title">
				<span>当前事实</span>
				<small>pulled latest</small>
			</div>
			<ul class="fact-list">
				{#each currentFacts as fact}
					<li>{fact}</li>
				{/each}
			</ul>
		</section>
	</div>

	<div class="workspace-grid bottom">
		<section class="panel">
			<div class="panel-title">
				<span>Spec 骨架</span>
				<small>L1-L4</small>
			</div>
			<div class="spec-list">
				{#each specTree as item}
					<div class="spec-row">
						<span>{item.level}</span>
						<div>
							<b>{item.name}</b>
							<small>{item.note}</small>
						</div>
					</div>
				{/each}
			</div>
		</section>

		<section class="panel large">
			<div class="panel-title">
				<span>阻塞项</span>
				<small>need fix before demo</small>
			</div>
			<div class="blocker-stack">
				{#each blockers as blocker}
					<article class="blocker">
						<b>{blocker.title}</b>
						<p>{blocker.body}</p>
					</article>
				{/each}
			</div>
		</section>
	</div>
</section>

<style>
	.prototype-shell {
		min-height: 100%;
		overflow: auto;
		padding: 22px;
		box-sizing: border-box;
		color: #e6edf3;
		background:
			radial-gradient(circle at 12% 4%, rgba(37, 99, 235, 0.16), transparent 32%),
			radial-gradient(circle at 92% 12%, rgba(20, 184, 166, 0.12), transparent 28%),
			linear-gradient(135deg, #0a0f17 0%, #0d1117 45%, #11100d 100%);
		font-family:
			'IBM Plex Sans',
			'Microsoft YaHei',
			system-ui,
			sans-serif;
	}

	.hero {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 280px;
		gap: 18px;
		align-items: stretch;
		margin-bottom: 18px;
	}

	.eyebrow {
		margin: 0 0 8px;
		color: #7dd3fc;
		font-size: 11px;
		letter-spacing: 0.18em;
		font-weight: 700;
	}

	h1 {
		margin: 0;
		font-size: clamp(28px, 4vw, 48px);
		line-height: 0.98;
		letter-spacing: -0.05em;
	}

	.lead {
		max-width: 760px;
		margin: 14px 0 0;
		color: #9aa7b5;
		font-size: 14px;
		line-height: 1.7;
	}

	.health-card,
	.panel,
	.stage-card {
		border: 1px solid rgba(148, 163, 184, 0.18);
		background: rgba(13, 18, 28, 0.72);
		box-shadow: 0 18px 60px rgba(0, 0, 0, 0.28);
		backdrop-filter: blur(18px);
	}

	.health-card {
		padding: 18px;
		border-radius: 18px;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}

	.health-kicker,
	.panel-title small {
		color: #64748b;
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.12em;
	}

	.health-card strong {
		margin-top: 14px;
		font-size: 22px;
		letter-spacing: -0.03em;
	}

	.health-card p {
		margin: 12px 0 0;
		color: #93a4b8;
		font-size: 12px;
		line-height: 1.6;
	}

	.stage-rail {
		display: grid;
		grid-template-columns: repeat(5, minmax(150px, 1fr));
		gap: 10px;
		margin-bottom: 14px;
	}

	.stage-card {
		position: relative;
		padding: 14px;
		border-radius: 16px;
		overflow: hidden;
	}

	.stage-card::before {
		content: '';
		position: absolute;
		inset: 0 auto 0 0;
		width: 3px;
		background: #475569;
	}

	.stage-card.done::before {
		background: #22c55e;
	}

	.stage-card.active::before {
		background: #38bdf8;
	}

	.stage-card.blocked::before {
		background: #f97316;
	}

	.stage-card.next::before {
		background: #a78bfa;
	}

	.stage-topline {
		display: flex;
		justify-content: space-between;
		gap: 10px;
		color: #64748b;
		font-size: 11px;
	}

	.stage-state {
		color: #cbd5e1;
	}

	.stage-card h2 {
		margin: 12px 0 6px;
		font-size: 15px;
	}

	.stage-card p,
	.blocker p {
		margin: 0;
		color: #8fa1b5;
		font-size: 12px;
		line-height: 1.55;
	}

	.workspace-grid {
		display: grid;
		grid-template-columns: minmax(0, 1.55fr) minmax(260px, 0.9fr);
		gap: 14px;
		margin-bottom: 14px;
	}

	.workspace-grid.bottom {
		grid-template-columns: minmax(260px, 0.92fr) minmax(0, 1.58fr);
	}

	.panel {
		border-radius: 18px;
		padding: 16px;
		min-height: 210px;
	}

	.panel-title {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		margin-bottom: 14px;
	}

	.panel-title span {
		font-size: 13px;
		font-weight: 700;
	}

	.flow-board {
		display: grid;
		grid-template-columns: repeat(4, minmax(130px, 1fr));
		gap: 10px;
		height: calc(100% - 34px);
	}

	.flow-node {
		position: relative;
		padding: 14px;
		border-radius: 15px;
		border: 1px solid rgba(148, 163, 184, 0.14);
		background: rgba(15, 23, 42, 0.72);
	}

	.flow-node::after {
		content: '';
		position: absolute;
		top: 50%;
		right: -10px;
		width: 10px;
		height: 1px;
		background: rgba(125, 211, 252, 0.45);
	}

	.flow-node:last-child::after {
		display: none;
	}

	.flow-node.primary {
		background: linear-gradient(145deg, rgba(14, 165, 233, 0.22), rgba(15, 23, 42, 0.8));
	}

	.flow-node.accent {
		background: linear-gradient(145deg, rgba(34, 197, 94, 0.18), rgba(15, 23, 42, 0.82));
	}

	.flow-node b,
	.blocker b,
	.spec-row b {
		display: block;
		font-size: 13px;
		color: #f8fafc;
	}

	.flow-node span {
		display: block;
		margin-top: 10px;
		color: #91a4b8;
		font-size: 12px;
		line-height: 1.55;
	}

	.fact-list {
		margin: 0;
		padding: 0;
		list-style: none;
		display: grid;
		gap: 10px;
	}

	.fact-list li {
		padding: 10px 10px 10px 28px;
		position: relative;
		border-radius: 12px;
		background: rgba(15, 23, 42, 0.62);
		color: #aebccd;
		font-size: 12px;
		line-height: 1.45;
	}

	.fact-list li::before {
		content: '';
		position: absolute;
		left: 11px;
		top: 15px;
		width: 7px;
		height: 7px;
		border-radius: 999px;
		background: #22c55e;
		box-shadow: 0 0 18px rgba(34, 197, 94, 0.75);
	}

	.spec-list,
	.blocker-stack {
		display: grid;
		gap: 10px;
	}

	.spec-row {
		display: grid;
		grid-template-columns: 44px minmax(0, 1fr);
		gap: 10px;
		align-items: start;
	}

	.spec-row > span {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		height: 28px;
		border-radius: 10px;
		background: rgba(56, 189, 248, 0.12);
		color: #7dd3fc;
		font-size: 11px;
		font-weight: 800;
	}

	.spec-row small {
		display: block;
		margin-top: 4px;
		color: #74869b;
		font-size: 11px;
		line-height: 1.45;
	}

	.blocker {
		padding: 13px;
		border-radius: 14px;
		border: 1px solid rgba(249, 115, 22, 0.22);
		background: linear-gradient(135deg, rgba(249, 115, 22, 0.12), rgba(15, 23, 42, 0.74));
	}

	.blocker p {
		margin-top: 7px;
	}

	@media (max-width: 1100px) {
		.hero,
		.workspace-grid,
		.workspace-grid.bottom {
			grid-template-columns: 1fr;
		}

		.stage-rail,
		.flow-board {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
</style>
