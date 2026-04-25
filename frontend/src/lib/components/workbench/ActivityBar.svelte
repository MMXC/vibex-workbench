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

<nav class="activity-bar" aria-label="活动栏">
	<div class="activity-top">
		<button
			type="button"
			class:active={act === 'explorer'}
			title="资源管理器"
			onclick={() => setAct('explorer')}
		>
			<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
				<path d="M3 7.5A2.5 2.5 0 015.5 5H10l2 2h6.5A2.5 2.5 0 0121 9.5v7A2.5 2.5 0 0118.5 19h-13A2.5 2.5 0 013 16.5z" />
			</svg>
		</button>
		<button type="button" class:active={act === 'search'} title="搜索" onclick={() => setAct('search')}>
			<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
				<circle cx="11" cy="11" r="7" />
				<path d="M20 20l-4.2-4.2" />
			</svg>
		</button>
		<button type="button" class:active={act === 'git'} title="源代码管理" onclick={() => setAct('git')}>
			<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
				<circle cx="18" cy="18" r="2.7" />
				<circle cx="6" cy="6" r="2.7" />
				<circle cx="6" cy="18" r="2.7" />
				<path d="M6 9v6M8.5 6.8A9 9 0 0118 15.3" />
			</svg>
			<span class="activity-badge">8</span>
		</button>
		<button
			type="button"
			class:active={act === 'extensions'}
			title="扩展与自定义"
			onclick={() => setAct('extensions')}
		>
			<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
				<rect x="4" y="4" width="7" height="7" rx="1" />
				<rect x="13" y="4" width="7" height="7" rx="1" />
				<rect x="13" y="13" width="7" height="7" rx="1" />
				<rect x="4" y="13" width="7" height="7" rx="1" />
			</svg>
		</button>
	</div>
	<div class="activity-bottom">
		<button type="button" title="账户">
			<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
				<circle cx="12" cy="8" r="4" />
				<path d="M4 21a8 8 0 0116 0" />
			</svg>
		</button>
		<button type="button" title="管理">
			<svg class="ab-svg" viewBox="0 0 24 24" aria-hidden="true">
				<path d="M12 15.5a3.5 3.5 0 100-7 3.5 3.5 0 000 7z" />
				<path d="M19.4 15a8 8 0 000-6l2-1.6-2-3.4-2.4 1a8 8 0 00-5.2-3L11.5 0h-4l-.3 2a8 8 0 00-5.2 3l-2.4-1-2 3.4L.6 9a8 8 0 000 6l-2 1.6 2 3.4 2.4-1a8 8 0 005.2 3l.3 2h4l.3-2a8 8 0 005.2-3l2.4 1 2-3.4z" />
			</svg>
		</button>
	</div>
</nav>

<style>
	.activity-bar {
		width: 48px;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: space-between;
		padding: 6px 0;
		background: #181818;
		border-right: 1px solid #2b2b2b;
	}

	.activity-top,
	.activity-bottom {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
	}

	.activity-bar button {
		width: 48px;
		height: 44px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: none;
		background: none;
		color: #858585;
		cursor: pointer;
		border-radius: 0;
		position: relative;
		transition:
			background 150ms ease,
			color 150ms ease;
	}

	.activity-bar button:hover {
		background: #2a2d2e;
		color: #cccccc;
	}

	.activity-bar button.active {
		color: #ffffff;
	}

	.activity-bar button.active::before {
		content: '';
		position: absolute;
		left: 0;
		top: 50%;
		transform: translateY(-50%);
		width: 2px;
		height: 28px;
		background: #007acc;
		border-radius: 0 2px 2px 0;
	}

	.activity-badge {
		position: absolute;
		right: 7px;
		bottom: 7px;
		min-width: 15px;
		height: 15px;
		padding: 0 3px;
		border-radius: 999px;
		background: #007acc;
		color: #fff;
		font-size: 9px;
		font-weight: 700;
		line-height: 15px;
		text-align: center;
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
