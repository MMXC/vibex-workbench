<!-- R2 顶栏：菜单 + 居中标题 + 设置 / 窗口控件占位 — prototypes/vibex-ide-chrome-r2.html .titlebar -->
<script lang="ts">
	let { title = 'VibeX Workbench' }: { title?: string } = $props();
	const menus = ['文件', '编辑', '视图', '转到', '运行', '终端', '帮助'] as const;
</script>

<header class="titlebar">
	<div class="menu">
		{#each menus as m (m)}
			<button type="button" class="menu-btn">{m}</button>
		{/each}
	</div>
	<span class="spacer" aria-hidden="true"></span>
	<span class="center-title">{title}</span>
	<div class="trail">
		<button type="button" class="icon-btn" title="设置" aria-label="设置">
			<svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true"
				><path
					d="M12 15a3 3 0 100-6 3 3 0 000 6z"
				/><path
					d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"
				/></svg
			>
		</button>
		<div class="win-controls" aria-hidden="true">
			<span class="win-btn close"></span>
			<span class="win-btn min"></span>
			<span class="win-btn max"></span>
		</div>
	</div>
</header>

<style>
	.titlebar {
		flex-shrink: 0;
		height: var(--titlebar-h, 38px);
		display: flex;
		align-items: center;
		padding: 0 12px;
		background: var(--bg-base, #0d0d0e);
		border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.07));
		gap: 2px;
		position: relative;
		z-index: 100;
		font-family: var(--font-ui, 'Inter', sans-serif);
		font-size: 12.5px;
		color: var(--text-primary, #e8e8ed);
	}

	.menu {
		display: flex;
		gap: 1px;
		flex-shrink: 0;
	}

	.menu-btn {
		background: none;
		border: none;
		color: var(--text-secondary, #8a8a8e);
		font: inherit;
		padding: 5px 8px;
		border-radius: var(--radius-sm, 3px);
		cursor: default;
		transition:
			background 150ms ease,
			color 150ms ease;
	}

	.menu-btn:hover {
		background: var(--bg-hover, rgba(255, 255, 255, 0.05));
		color: var(--text-primary, #e8e8ed);
	}

	.spacer {
		flex: 1;
	}

	.center-title {
		position: absolute;
		left: 50%;
		transform: translateX(-50%);
		font-size: 12px;
		color: var(--text-muted, #555558);
		white-space: nowrap;
		pointer-events: none;
	}

	.trail {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.icon-btn {
		background: none;
		border: none;
		color: var(--text-muted, #555558);
		padding: 4px;
		border-radius: var(--radius-sm, 3px);
		cursor: pointer;
		display: flex;
		align-items: center;
	}

	.icon-btn:hover {
		color: var(--text-secondary, #8a8a8e);
	}

	.icon-btn svg {
		stroke: currentColor;
		fill: none;
		stroke-width: 1.5;
	}

	.win-controls {
		display: flex;
		gap: 7px;
		align-items: center;
	}

	.win-btn {
		width: 12px;
		height: 12px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.win-btn.close {
		background: #ff5f57;
	}
	.win-btn.min {
		background: #febc2e;
	}
	.win-btn.max {
		background: #28c840;
	}
</style>
