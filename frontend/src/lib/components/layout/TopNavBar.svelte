<script lang="ts">
	// Top navigation bar — workspace + view tabs + settings
	let { workspaceRoot = '—' }: { workspaceRoot?: string } = $props();

	let activeView = $state<'spec' | 'task' | 'ppt'>('spec');

	const views = [
		{ id: 'spec' as const, label: 'Spec Console' },
		{ id: 'task' as const, label: 'Task Console' },
		{ id: 'ppt' as const, label: 'PPT' },
	];

	function shortPath(p: string) {
		if (!p || p === '—') return '—';
		const parts = p.replace(/\\/g, '/').split('/');
		return parts.length <= 2 ? p : '…/' + parts.slice(-2).join('/');
	}
</script>

<nav class="topbar" role="navigation">
	<!-- Left: workspace -->
	<div class="tb-left">
		<span class="tb-ws" title={workspaceRoot}>
			<span class="ws-dot">●</span>
			{shortPath(workspaceRoot)}
		</span>
	</div>

	<!-- Center: view tabs -->
	<div class="tb-center">
		{#each views as v}
			<button
				type="button"
				class="tb-tab"
				class:active={activeView === v.id}
				onclick={() => (activeView = v.id)}
			>
				{v.label}
			</button>
		{/each}
	</div>

	<!-- Right: settings -->
	<div class="tb-right">
		<button type="button" class="tb-icon-btn" title="设置">⚙</button>
	</div>
</nav>

<style>
	.topbar {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		height: 42px;
		z-index: 100;
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: #10131a;
		border-bottom: 1px solid #1e2030;
		padding: 0 12px;
		gap: 12px;
	}

	.tb-left {
		display: flex;
		align-items: center;
		min-width: 0;
		flex: 1;
	}
	.tb-ws {
		font-size: 12px;
		color: #8ab;
		display: flex;
		align-items: center;
		gap: 5px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		max-width: 280px;
	}
	.ws-dot { color: #4ade80; font-size: 8px; }

	.tb-center {
		display: flex;
		align-items: center;
		gap: 2px;
		flex-shrink: 0;
	}
	.tb-tab {
		border: none;
		border-radius: 6px;
		background: transparent;
		color: #556;
		font-size: 12px;
		font-family: inherit;
		padding: 5px 12px;
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
	}
	.tb-tab:hover { background: rgba(255,255,255,0.06); color: #99a; }
	.tb-tab.active {
		background: rgba(122,162,255,0.15);
		color: #aab7ff;
		font-weight: 600;
	}

	.tb-right {
		display: flex;
		align-items: center;
		flex: 1;
		justify-content: flex-end;
	}
	.tb-icon-btn {
		border: none;
		background: transparent;
		color: #556;
		font-size: 14px;
		cursor: pointer;
		padding: 5px 8px;
		border-radius: 5px;
		transition: background 0.15s, color 0.15s;
	}
	.tb-icon-btn:hover { background: rgba(255,255,255,0.07); color: #99a; }
</style>
