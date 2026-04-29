<!-- StatusBar — 底部状态栏：workspace 路径 + backend 连接状态 + spec 状态
     开发者维护，gen.py 永不覆盖。
-->
<script lang="ts">
	interface Props {
		workspaceRoot?: string;
		backendStatus?: 'connecting' | 'ready' | 'error';
		workspaceState?: 'empty' | 'partial' | 'ready';
	}

	let {
		workspaceRoot = '—',
		backendStatus = 'connecting',
		workspaceState = 'empty',
	}: Props = $props();

	const stateLabels = {
		empty: '空仓库',
		partial: '半成品',
		ready: '就绪',
	} as const;

	const stateColors = {
		empty: '#f97316',
		partial: '#eab308',
		ready: '#22c55e',
	} as const;

	const backendLabels = {
		connecting: '启动中…',
		ready: '后端就绪',
		error: '后端错误',
	} as const;

	const backendColors = {
		connecting: '#eab308',
		ready: '#22c55e',
		error: '#ef4444',
	} as const;

	/** 截短长路径用于显示 */
	function shortPath(p: string): string {
		if (p === '—' || !p) return '—';
		const parts = p.replace(/\\/g, '/').split('/');
		if (parts.length <= 3) return p;
		return '…/' + parts.slice(-2).join('/');
	}
</script>

<div class="statusbar" role="status" aria-label="状态栏">
	<!-- 左区：workspace 状态 + 路径 -->
	<div class="sb-left">
		<span
			class="sb-badge"
			style:color={stateColors[workspaceState]}
			title="仓库状态：{stateLabels[workspaceState]}"
		>
			● {stateLabels[workspaceState]}
		</span>
		<span class="sb-sep" aria-hidden="true">|</span>
		<span class="sb-path" title={workspaceRoot}>
			{shortPath(workspaceRoot)}
		</span>
	</div>

	<!-- 右区：backend 状态 + 版本 -->
	<div class="sb-right">
		<span
			class="sb-badge"
			style:color={backendColors[backendStatus]}
			title="Backend: {backendLabels[backendStatus]}"
		>
			{backendLabels[backendStatus]}
		</span>
		<span class="sb-sep" aria-hidden="true">|</span>
		<span class="sb-meta">VibeX Workbench</span>
	</div>
</div>

<style>
	.statusbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: var(--statusbar-h, 24px);
		padding: 0 10px;
		background: var(--statusbar-bg, #007acc);
		flex-shrink: 0;
		font-family: var(--font-ui, 'Inter', sans-serif);
		font-size: 12px;
		color: #fff;
		user-select: none;
		gap: 8px;
	}

	.sb-left,
	.sb-right {
		display: flex;
		align-items: center;
		gap: 6px;
		min-width: 0;
	}

	.sb-right {
		margin-left: auto;
	}

	.sb-badge {
		font-size: 11.5px;
		font-weight: 500;
		letter-spacing: 0.01em;
		white-space: nowrap;
	}

	.sb-path {
		color: rgba(255, 255, 255, 0.8);
		font-size: 11.5px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		max-width: 320px;
	}

	.sb-sep {
		color: rgba(255, 255, 255, 0.3);
		font-size: 11px;
	}

	.sb-meta {
		color: rgba(255, 255, 255, 0.55);
		font-size: 11px;
		white-space: nowrap;
	}
</style>
