<!-- 活动栏（对齐 R2 / VS Code），切换左侧一级视图 -->
<script lang="ts">
	import { specExplorerStore, type LeftActivity } from '$lib/stores/spec-explorer-store';

	function setAct(a: LeftActivity): void {
		specExplorerStore.setLeftActivity(a);
	}

	let act = $state<LeftActivity>('explorer');
	$effect(() => {
		return specExplorerStore.subscribe(s => {
			act = s.leftActivity;
		});
	});
</script>

<!-- 图标与 prototypes/vibex-ide-chrome-r2.html .ab-icon 一致（描边、22×22） -->
<nav class="activity-bar" aria-label="活动栏">
	<button
		type="button"
		class:active={act === 'explorer'}
		title="工作区（规格文件）"
		onclick={() => setAct('explorer')}
	>
		<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
			<path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
			<polyline points="9 22 9 12 15 12 15 22" />
		</svg>
	</button>
	<button type="button" class:active={act === 'git'} title="源代码管理" onclick={() => setAct('git')}>
		<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
			<circle cx="18" cy="18" r="3" />
			<circle cx="6" cy="6" r="3" />
			<path d="M6 21V9a9 9 0 009 9" />
		</svg>
	</button>
	<button type="button" class:active={act === 'search'} title="搜索" onclick={() => setAct('search')}>
		<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
			<circle cx="11" cy="11" r="8" />
			<path d="M21 21l-4.35-4.35" />
		</svg>
	</button>
	<button
		type="button"
		class:active={act === 'extensions'}
		title="扩展与自定义"
		onclick={() => setAct('extensions')}
	>
		<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
			<rect x="3" y="3" width="8" height="8" rx="1" />
			<rect x="13" y="3" width="8" height="8" rx="1" />
			<rect x="13" y="13" width="8" height="8" rx="1" />
			<rect x="3" y="13" width="8" height="8" rx="1" />
		</svg>
	</button>
</nav>

<style>
	.activity-bar {
		width: 48px;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 8px 0 4px;
		gap: 2px;
		background: var(--wb-base, #0d0d0e);
		border-right: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
	}

	.activity-bar button {
		width: 48px;
		height: 48px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: none;
		background: none;
		color: var(--wb-muted, #555558);
		cursor: pointer;
		border-radius: 0;
		position: relative;
		transition:
			background 150ms ease,
			color 150ms ease;
	}

	.activity-bar button:hover {
		background: rgba(255, 255, 255, 0.05);
		color: var(--wb-text-sec, #8a8a8e);
	}

	.activity-bar button.active {
		color: var(--wb-text, #e8e8ed);
	}

	.activity-bar button.active::before {
		content: '';
		position: absolute;
		left: 0;
		top: 50%;
		transform: translateY(-50%);
		width: 2px;
		height: 22px;
		background: var(--wb-brand, #5856d6);
		border-radius: 0 2px 2px 0;
	}

	.ab-svg {
		width: 22px;
		height: 22px;
		flex-shrink: 0;
		stroke: currentColor;
		fill: none;
		stroke-width: 1.5;
		stroke-linecap: round;
		stroke-linejoin: round;
	}
</style>
