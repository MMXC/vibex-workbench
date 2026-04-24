<!-- R2 顶栏：品牌 Logo + 居中标题 + 设置
     窗口控件（最小化/最大化/关闭）由系统原生标题栏提供，不再在 WebView 内渲染。
     顶部菜单栏由 Wails 原生 MenuSetApplicationMenu 提供，不在此组件内。
-->
<script lang="ts">
	let { title = 'VibeX Workbench' }: { title?: string } = $props();
	const menus = ['文件', '编辑', '视图', '终端', '帮助'] as const;

	/** Wails 桌面：调用 window.runtime 窗口管理 */
	async function handleMinimize() {
		await (window as any).runtime?.WindowMinimise();
	}
	async function handleMaximize() {
		await (window as any).runtime?.WindowToggleMaximise();
	}
	async function handleClose() {
		await (window as any).runtime?.Quit();
	}
</script>

<header class="titlebar">
	<div class="lead">
		<a class="brand" href="/workbench" title="VibeX Workbench" aria-label="VibeX Workbench">
			<img class="brand-logo" src="/vibex-logo.svg" alt="" width="26" height="26" />
		</a>
	</div>

	<span class="spacer" aria-hidden="true"></span>
	<span class="center-title">{title}</span>

	<div class="trail">
		<button type="button" class="icon-btn" title="设置" aria-label="设置">
			<svg class="ico-svg" viewBox="0 0 24 24" aria-hidden="true">
				<path d="M12 15a3 3 0 100-6 3 3 0 000 6z" />
				<path
					d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"
				/>
			</svg>
		</button>

		<!-- 桌面模式：显示真实窗口控件（浏览器模式下这些按钮仍保留用于占位）
		     浏览器端：noop，无实际效果
		     桌面端（Wails）：调用 window.runtime -->
		<div class="window-controls" role="toolbar" aria-label="窗口">
			<button
				type="button"
				class="win win-min"
				title="最小化"
				aria-label="最小化"
				onclick={handleMinimize}
			>
				<svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
					<path d="M2 6h8" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
				</svg>
			</button>
			<button
				type="button"
				class="win win-max"
				title="最大化"
				aria-label="最大化"
				onclick={handleMaximize}
			>
				<svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
					<rect
						x="2"
						y="2"
						width="8"
						height="8"
						rx="0.5"
						stroke="currentColor"
						stroke-width="1.2"
						fill="none"
					/>
				</svg>
			</button>
			<button
				type="button"
				class="win win-close"
				title="关闭"
				aria-label="关闭"
				onclick={handleClose}
			>
				<svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
					<path d="M2 2l8 8M10 2L2 10" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
				</svg>
			</button>
		</div>
	</div>
</header>

<style>
	.titlebar {
		flex-shrink: 0;
		height: var(--titlebar-h, 38px);
		display: flex;
		align-items: stretch;
		padding: 0;
		background: var(--bg-base, #0d0d0e);
		border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.07));
		position: relative;
		z-index: 100;
		font-family: var(--font-ui, 'Inter', sans-serif);
		font-size: 12.5px;
		color: var(--text-primary, #e8e8ed);
		user-select: none;
		/* 桌面端：系统标题栏已提供窗口控件，
		   此处 win-min/win-max/win-close 可保留作备选/辅助，
		   在 macOS 上可通过 Accessibility 使用系统原生按钮。 */
	}

	.lead {
		display: flex;
		align-items: center;
		flex-shrink: 0;
		gap: 10px;
		padding-left: 10px;
		min-width: 0;
	}

	.brand {
		display: flex;
		align-items: center;
		justify-content: center;
		text-decoration: none;
		color: inherit;
		flex-shrink: 0;
		padding: 4px 4px;
		border-radius: var(--radius-sm, 3px);
		transition: background 150ms ease;
	}

	.brand:hover {
		background: var(--bg-hover, rgba(255, 255, 255, 0.05));
	}

	.brand-logo {
		display: block;
		flex-shrink: 0;
	}

	.spacer {
		flex: 1;
		min-width: 0;
	}

	.center-title {
		position: absolute;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		font-size: 12px;
		color: var(--text-muted, #555558);
		white-space: nowrap;
		pointer-events: none;
	}

	.trail {
		flex-shrink: 0;
		display: flex;
		align-items: stretch;
		margin-left: auto;
	}

	.icon-btn {
		background: none;
		border: none;
		color: var(--text-muted, #555558);
		padding: 0 10px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition:
			background 120ms ease,
			color 120ms ease;
	}

	.icon-btn:hover {
		color: var(--text-secondary, #8a8a8e);
		background: rgba(255, 255, 255, 0.06);
	}

	.ico-svg {
		width: 15px;
		height: 15px;
		stroke: currentColor;
		fill: none;
		stroke-width: 1.5;
	}

	.window-controls {
		display: flex;
		align-items: stretch;
		height: 100%;
		margin-right: 0;
	}

	.win {
		box-sizing: border-box;
		width: 46px;
		min-height: var(--titlebar-h, 38px);
		padding: 0;
		border: none;
		background: transparent;
		color: var(--text-secondary, #cccccc);
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: background 120ms ease, color 120ms ease;
	}

	.win:hover {
		background: rgba(255, 255, 255, 0.08);
		color: var(--text-primary, #e8e8ed);
	}

	.win-close:hover {
		background: #e81123;
		color: #fff;
	}

	.win svg {
		flex-shrink: 0;
		opacity: 0.95;
	}
</style>
