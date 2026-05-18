<!-- TopNavBar.svelte
顶部导航栏：窗口控制 | 工作区名 | Agent 切换标签 | 设置
固定高度 42px，背景 #10131a。
Wails frameless 模式下：窗口控制按钮由我们绘制，titlebar 区域标记 draggable。
-->
<script lang="ts">
	import { windowMinimize, windowToggleMaximize, windowIsMaximized, windowQuit } from '$lib/wails-runtime';
	import { openDirectoryNativeFirst } from '$lib/wails-dialogs';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';

	type ViewTab = 'spec' | 'task' | 'ppt';
	let activeView = $state<ViewTab>('spec');
	let isMaximized = $state(false);
	let wsPickerOpen = $state(false);

	const TABS: { id: ViewTab; label: string }[] = [
		{ id: 'spec', label: 'Spec Console' },
		{ id: 'task', label: 'Task Console' },
		{ id: 'ppt', label: 'PPT' },
	];

	// 窗口状态
	$effect(() => {
		void windowIsMaximized().then(v => { isMaximized = v; });
	});

	async function handleMinimize() {
		await windowMinimize();
	}

	async function handleMaximize() {
		await windowToggleMaximize();
		isMaximized = !isMaximized;
	}

	// 打开工作区
	async function handleOpenWorkspace() {
		const dir = await openDirectoryNativeFirst('workspace');
		if (dir && dir.trim()) {
			specExplorerStore.setWorkspaceRoot(dir.trim());
			localStorage.setItem('vibex-workspace-root', dir.trim());
			window.dispatchEvent(new CustomEvent('workspace:selected', { detail: dir.trim() }));
		}
	}
</script>

<div class="tnb">
	<!-- Window controls (drag handle) -->
	<div class="tnb-win-ctrl">
		<button type="button" class="tnb-btn tnb-btn-close" onclick={handleMinimize} title="最小化">─</button>
		<button type="button" class="tnb-btn" onclick={handleMaximize} title={isMaximized ? '还原' : '最大化'}>
			{isMaximized ? '❐' : '□'}
		</button>
		<button type="button" class="tnb-btn tnb-btn-close" onclick={() => windowQuit()} title="关闭">✕</button>
	</div>

	<!-- Drag handle (full titlebar area) -->
	<div class="tnb-drag-area">
		<!-- Left: workspace name + open button -->
		<div class="tnb-left">
			<button
				type="button"
				class="tnb-ws-btn"
				onclick={handleOpenWorkspace}
				title="打开工作区"
			>
				<span class="tnb-logo">⬡</span>
				<span class="tnb-ws">
					{$specExplorerStore.workspaceRoot
						? $specExplorerStore.workspaceRoot.split(/[/\\]/).pop()
						: '选择工作区'}
				</span>
				<span class="tnb-ws-open">📂</span>
			</button>
		</div>

		<!-- Center: view tabs -->
		<div class="tnb-tabs">
			{#each TABS as tab}
			<button
				type="button"
				class="tnb-tab"
				class:active={activeView === tab.id}
				onclick={() => (activeView = tab.id)}
			>{tab.label}</button>
			{/each}
		</div>

		<!-- Right: settings -->
		<div class="tnb-right">
			<button type="button" class="tnb-settings" title="设置">⚙</button>
		</div>
	</div>
</div>

<style>
.tnb {
	display: flex;
	align-items: center;
	height: 42px;
	background: #10131a;
	border-bottom: 1px solid #303746;
	overflow: hidden;
}

/* Window controls — leftmost */
.tnb-win-ctrl {
	display: flex;
	align-items: center;
	flex-shrink: 0;
	height: 100%;
	padding: 0 8px;
	gap: 2px;
	/* Non-draggable — Wails needs this area to have no drag so buttons work */
}
.tnb-btn {
	background: none;
	border: none;
	color: #8a8a8e;
	font-size: 12px;
	cursor: pointer;
	padding: 4px 8px;
	border-radius: 4px;
	line-height: 1;
}
.tnb-btn:hover { background: rgba(255,255,255,0.08); color: #c0caf5; }
.tnb-btn-close:hover { background: rgba(232, 113, 113, 0.7); color: #fff; }

/* Draggable area — everything except window controls */
.tnb-drag-area {
	display: flex;
	align-items: center;
	flex: 1;
	height: 100%;
	/* Wails frameless drag */
	--wails-draggable: drag;
}
.tnb-drag-area button {
	--wails-draggable: no-drag;
}

/* Workspace */
.tnb-left {
	display: flex;
	align-items: center;
	padding: 0 4px;
	flex-shrink: 0;
}
.tnb-ws-btn {
	display: flex;
	align-items: center;
	gap: 6px;
	background: none;
	border: none;
	color: var(--wb-text, #c0caf5);
	font-size: 12px;
	cursor: pointer;
	padding: 4px 8px;
	border-radius: 4px;
	white-space: nowrap;
	max-width: 200px;
	overflow: hidden;
	text-overflow: ellipsis;
}
.tnb-ws-btn:hover { background: rgba(255,255,255,0.06); }
.tnb-logo { font-size: 14px; color: var(--wb-accent, #72d6d0); }
.tnb-ws { overflow: hidden; text-overflow: ellipsis; }
.tnb-ws-open { font-size: 11px; opacity: 0.6; }

/* Tabs */
.tnb-tabs {
	display: flex;
	align-items: center;
	flex: 1;
	justify-content: center;
}
.tnb-tab {
	padding: 0 14px;
	height: 30px;
	background: none;
	border: none;
	border-radius: 6px 6px 0 0;
	color: var(--wb-text-sec, #787c99);
	font-size: 11.5px;
	cursor: pointer;
	transition: color 0.15s, background 0.15s;
}
.tnb-tab.active {
	color: var(--wb-text, #c0caf5);
	background: rgba(255,255,255,0.04);
	border-bottom: 2px solid var(--wb-accent, #72d6d0);
}
.tnb-tab:hover:not(.active) { color: var(--wb-text, #c0caf5); }

/* Settings */
.tnb-right {
	padding: 0 12px 0 4px;
}
.tnb-settings {
	background: none;
	border: none;
	color: var(--wb-text-sec, #787c99);
	font-size: 14px;
	cursor: pointer;
	padding: 4px 6px;
	border-radius: 4px;
}
.tnb-settings:hover { color: var(--wb-text, #c0caf5); background: rgba(255,255,255,0.06); }
</style>
