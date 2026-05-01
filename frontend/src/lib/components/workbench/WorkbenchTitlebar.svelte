<!-- R2 顶栏：品牌 Logo + 居中标题 + 设置
     窗口控件（最小化/最大化/关闭）由系统原生标题栏提供，不再在 WebView 内渲染。
     顶部菜单栏由 Wails 原生 MenuSetApplicationMenu 提供，不在此组件内。
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { eventsEmit } from '$lib/wails-runtime';
	import { openDirectoryNativeFirst } from '$lib/wails-dialogs';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';

	let { title = 'VibeX Workbench' }: { title?: string } = $props();

	let fileMenuOpen = $state(false);
	let menuWrapperEl = $state<HTMLElement | null>(null);

	async function handleMinimize() {
		await (window as any).runtime?.WindowMinimise();
	}
	async function handleMaximize() {
		await (window as any).runtime?.WindowToggleMaximise();
	}
	async function handleClose() {
		await (window as any).runtime?.Quit();
	}

	/** 打开项目：弹目录选择 → 同步 store → 触发事件 → 跳转 */
	async function openProject() {
		console.warn('[titlebar] openProject clicked');
		fileMenuOpen = false;
		const dir = await openDirectoryNativeFirst('titlebar');
		console.warn('[titlebar] openProject result:', dir);
		if (!dir) return;
		localStorage.setItem('vibex-workspace-root', dir);
		// 1. 直接更新 store → SpecExplorer 的 $effect 立即触发刷新
		specExplorerStore.setWorkspaceRoot(dir);
		// 2. 发出事件 → workbench/+page.svelte 同步状态 + detectWorkspaceState
		eventsEmit('workspace:selected', dir);
		goto('/workbench');
	}

	function toggleMenu(e: MouseEvent) {
		e.stopPropagation();
		fileMenuOpen = !fileMenuOpen;
	}

	function handleClickOutside(e: MouseEvent) {
		if (fileMenuOpen && menuWrapperEl && !menuWrapperEl.contains(e.target as Node)) {
			fileMenuOpen = false;
		}
	}

	onMount(() => {
		document.addEventListener('click', handleClickOutside);
		return () => document.removeEventListener('click', handleClickOutside);
	});
</script>

<header class="titlebar">
	<div class="lead">
		<a class="brand" href="/workbench" title="VibeX Workbench" aria-label="VibeX Workbench">
			<img class="brand-logo" src="/vibex-logo.svg" alt="" width="26" height="26" />
		</a>
		<span class="workspace-mark">vibex-workbench</span>
		<nav class="menu-strip" aria-label="WebView 内层菜单">
			<div class="menu-item-wrapper" bind:this={menuWrapperEl}>
				<button
					type="button"
					class="menu-btn"
					class:active={fileMenuOpen}
					onclick={toggleMenu}
				>文件</button>
				{#if fileMenuOpen}
					<div class="dropdown" role="menu">
						<button type="button" class="dropdown-item" role="menuitem" onclick={openProject}>
							<span class="item-icon">📂</span>打开项目…
							<span class="shortcut">⌘O</span>
						</button>
					</div>
				{/if}
			</div>
			<button type="button" class="menu-btn" disabled>编辑</button>
			<button type="button" class="menu-btn" disabled>视图</button>
			<button type="button" class="menu-btn" disabled>终端</button>
			<button type="button" class="menu-btn" disabled>帮助</button>
		</nav>
	</div>

	<div class="command-center" aria-label="命令中心">
		<span class="search-icon" aria-hidden="true">⌕</span>
		<span>{title} · self-bootstrap workspace</span>
	</div>

	<div class="trail">
		<span class="run-pill">make validate ✓</span>
		<span class="run-pill warn">backend path</span>
		<button type="button" class="icon-btn" title="设置" aria-label="设置">
			<svg class="ico-svg" viewBox="0 0 24 24" aria-hidden="true">
				<path d="M12 15a3 3 0 100-6 3 3 0 000 6z" />
				<path
					d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"
				/>
			</svg>
		</button>
		<div class="window-controls" role="toolbar" aria-label="窗口">
			<button type="button" class="win win-min" title="最小化" aria-label="最小化" onclick={handleMinimize}>
				<svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
					<path d="M2 6h8" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
				</svg>
			</button>
			<button type="button" class="win win-max" title="最大化" aria-label="最大化" onclick={handleMaximize}>
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
			<button type="button" class="win win-close" title="关闭" aria-label="关闭" onclick={handleClose}>
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
		height: var(--titlebar-h, 34px);
		display: flex;
		align-items: center;
		padding: 0 0 0 6px;
		background: var(--wb-bg-base, #0b0c10);
		border-bottom: 1px solid var(--wb-border, #303746);
		position: relative;
		z-index: 100;
		font-family: var(--font-ui, 'Segoe UI', 'Microsoft YaHei', system-ui, sans-serif);
		font-size: 12px;
		color: var(--wb-text-sec, #a3abb9);
		user-select: none;
		--wails-draggable: drag;
	}

	.lead {
		display: flex;
		align-items: center;
		flex-shrink: 0;
		gap: 4px;
		min-width: 0;
	}

	.brand {
		display: flex;
		align-items: center;
		justify-content: center;
		text-decoration: none;
		color: inherit;
		flex-shrink: 0;
		width: 32px;
		height: 32px;
		border-radius: 5px;
		transition: background 150ms ease;
		--wails-draggable: no-drag;
	}

	.brand:hover {
		background: var(--wb-bg-panel-2, #1c202a);
	}

	.brand-logo {
		display: block;
		flex-shrink: 0;
	}

	.workspace-mark {
		max-width: 210px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: var(--wb-text, #eef0f5);
		font-size: 12px;
		font-weight: 500;
	}

	.menu-strip {
		display: flex;
		align-items: center;
		height: 100%;
		gap: 1px;
		margin-left: 4px;
		--wails-draggable: no-drag;
	}

	.menu-strip button {
		height: 26px;
		padding: 0 8px;
		border: 0;
		border-radius: 4px;
		background: transparent;
		color: var(--wb-text-sec, #a3abb9);
		font: inherit;
		cursor: pointer;
	}

	.menu-btn {
		height: 26px;
		padding: 0 8px;
		border: 0;
		border-radius: 4px;
		background: transparent;
		color: var(--wb-text-sec, #a3abb9);
		font: inherit;
		font-size: 12px;
		cursor: pointer;
	}

	.menu-btn:hover:not(:disabled),
	.menu-btn.active {
		background: var(--wb-bg-panel-2, #1c202a);
	}

	.menu-btn:disabled {
		opacity: 0.4;
		cursor: default;
	}

	.menu-item-wrapper {
		position: relative;
	}

	.dropdown {
		position: absolute;
		top: calc(100% + 2px);
		left: 0;
		min-width: 200px;
		background: var(--wb-bg-panel, #151820);
		border: 1px solid var(--wb-border-2, #465064);
		border-radius: 6px;
		padding: 4px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
		z-index: 9999;
	}

	.dropdown-item {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 7px 10px;
		border: 0;
		border-radius: 4px;
		background: transparent;
		color: var(--wb-text, #eef0f5);
		font: inherit;
		font-size: 13px;
		text-align: left;
		cursor: pointer;
		box-sizing: border-box;
		gap: 8px;
	}

	.dropdown-item:hover {
		background: rgba(122, 162, 255, 0.16);
	}

	.dropdown-item .item-icon {
		font-size: 14px;
	}

	.dropdown-item .shortcut {
		margin-left: auto;
		font-size: 11px;
		color: var(--wb-muted, #6f7888);
	}

	.command-center {
		position: absolute;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		display: flex;
		align-items: center;
		gap: 8px;
		width: min(42vw, 520px);
		height: 24px;
		padding: 0 12px;
		box-sizing: border-box;
		border: 1px solid var(--wb-border-2, #465064);
		border-radius: 6px;
		background: var(--wb-bg-panel-2, #1c202a);
		color: var(--wb-text-sec, #a3abb9);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
		--wails-draggable: no-drag;
	}

	.command-center span:last-child {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.search-icon {
		color: var(--wb-muted, #6f7888);
		font-size: 13px;
	}

	.trail {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 6px;
		margin-left: auto;
		height: 100%;
	}

	.run-pill {
		display: inline-flex;
		align-items: center;
		height: 20px;
		padding: 0 8px;
		border-radius: 999px;
		background: rgba(135, 207, 138, 0.12);
		color: var(--accent-green, #87cf8a);
		border: 1px solid rgba(135, 207, 138, 0.25);
		font-size: 11px;
		white-space: nowrap;
	}

	.run-pill.warn {
		background: rgba(239, 198, 107, 0.1);
		color: var(--accent-yellow, #efc66b);
		border-color: rgba(239, 198, 107, 0.24);
	}

	.icon-btn {
		background: none;
		border: none;
		color: var(--wb-muted, #6f7888);
		padding: 0 10px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition:
			background 120ms ease,
			color 120ms ease;
		--wails-draggable: no-drag;
	}

	.icon-btn:hover {
		color: var(--wb-text, #eef0f5);
		background: var(--wb-bg-panel-2, #1c202a);
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
		--wails-draggable: no-drag;
	}

	.win {
		box-sizing: border-box;
		width: 46px;
		min-height: var(--titlebar-h, 34px);
		padding: 0;
		border: none;
		background: transparent;
		color: var(--wb-text-sec, #a3abb9);
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition:
			background 120ms ease,
			color 120ms ease;
	}

	.win:hover {
		background: var(--wb-bg-panel-2, #1c202a);
		color: var(--wb-text, #eef0f5);
	}

	.win-close:hover {
		background: var(--accent-red, #e16d75);
		color: #12080a;
	}

	.win svg {
		flex-shrink: 0;
		opacity: 0.95;
	}

	@media (max-width: 980px) {
		.command-center {
			display: none;
		}

		.run-pill {
			display: none;
		}

		.menu-strip {
			display: none;
		}
	}
</style>
