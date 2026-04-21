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
	<button
		type="button"
		class:active={act === 'explorer'}
		title="资源管理器（规格文件）"
		onclick={() => setAct('explorer')}
	>
		<span class="ico" aria-hidden="true">📂</span>
	</button>
	<button
		type="button"
		class:active={act === 'git'}
		title="源代码管理"
		onclick={() => setAct('git')}
	>
		<span class="ico" aria-hidden="true">⑂</span>
	</button>
	<button
		type="button"
		class:active={act === 'search'}
		title="搜索"
		onclick={() => setAct('search')}
	>
		<span class="ico" aria-hidden="true">🔍</span>
	</button>
	<button
		type="button"
		class:active={act === 'extensions'}
		title="扩展"
		onclick={() => setAct('extensions')}
	>
		<span class="ico" aria-hidden="true">🧩</span>
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

	.ico {
		font-size: 18px;
		line-height: 1;
	}
</style>
