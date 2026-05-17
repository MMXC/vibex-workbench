<!-- StatusBar — 底部状态栏：workspace 路径 + backend 连接状态 + spec 状态
     开发者维护，gen.py 永不覆盖。
-->
<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { specSlotSessionStore } from '$lib/stores/spec-slot-session-store';

	interface Props {
		workspaceRoot?: string;
		backendStatus?: 'connecting' | 'ready' | 'error';
		workspaceState?: 'empty' | 'partial' | 'ready';
		canvasAutoUiEnabled?: boolean;
		ontoggleCanvasAutoUi?: () => void;
	}

	let {
		workspaceRoot = '—',
		backendStatus = 'connecting',
		workspaceState = 'empty',
		canvasAutoUiEnabled = true,
		ontoggleCanvasAutoUi,
	}: Props = $props();

	// SpecPilot service status — polled every 10s
	interface SpStatus { installed: boolean; dcRunning: boolean; mfRunning: boolean; dcPort: number; mfPort: number; }
	let spStatus: SpStatus | null = $state(null);
	let spPollTimer: ReturnType<typeof setInterval> | null = null;
	let spBootstrapping = $state(false);
	let spError = $state('');

	async function pollSpStatus() {
		try {
			// Poll the Go backend HTTP endpoint
			const r = await fetch('http://localhost:33338/api/specpilot/status', { signal: AbortSignal.timeout(3000) });
			if (r.ok) {
				const d = await r.json();
				spStatus = {
					installed: d.installed ?? false,
					dcRunning: d.dcRunning ?? false,
					mfRunning: d.mfRunning ?? false,
					dcPort: d.dcPort ?? 7890,
					mfPort: d.mfPort ?? 5177,
				};
				spError = '';
				return;
			}
		} catch {}
		// Fallback: try DC health directly
		try {
			const r = await fetch('http://127.0.0.1:7890/api/health', { signal: AbortSignal.timeout(2000) });
			if (r.ok) {
				const d = await r.json();
				spStatus = { installed: true, dcRunning: true, mfRunning: true, dcPort: d.port ?? 7890, mfPort: d.mfPort ?? 5177 };
				spError = '';
				return;
			}
		} catch {}
		spStatus = null;
	}

	async function bootstrap() {
		spBootstrapping = true;
		spError = '';
		try {
			const r = await fetch('http://localhost:33338/api/specpilot/bootstrap', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ component: 'Dashboard' }),
			});
			const d = await r.json();
			if (d.error) {
				spError = d.error;
			} else {
				// Inject structured bootstrap result as a tool message in the active session
				const payload = {
					type: 'specpilot_bootstrap',
					dcPort: d.dcPort,
					mfPort: d.mfPort,
					dcUrl: d.dcUrl,
					mfUrl: d.mfUrl,
					mfRemoteUrl: d.mfRemoteUrl,
					message: d.message ?? 'SpecPilot 服务已启动',
				};
				specSlotSessionStore.injectToolResult(JSON.stringify(payload));
				// Ensure the drawer is open so the user can see the result card
				specSlotSessionStore.openDrawer();
				await pollSpStatus();
			}
		} catch (e: any) {
			spError = e?.message ?? '启动失败';
		} finally {
			spBootstrapping = false;
		}
	}

	onMount(() => {
		pollSpStatus();
		spPollTimer = setInterval(pollSpStatus, 10000);
	});

	onDestroy(() => {
		if (spPollTimer) clearInterval(spPollTimer);
	});

	const stateLabels = {
		empty: '空仓库',
		partial: '半成品',
		ready: '就绪',
	} as const;

	const stateColors = {
		empty: 'var(--accent-orange, #f09a6a)',
		partial: 'var(--accent-yellow, #efc66b)',
		ready: 'var(--accent-green, #87cf8a)',
	} as const;

	const backendLabels = {
		connecting: '启动中…',
		ready: '后端就绪',
		error: '后端错误',
	} as const;

	const backendColors = {
		connecting: 'var(--accent-yellow, #efc66b)',
		ready: 'var(--accent-green, #87cf8a)',
		error: 'var(--accent-red, #e16d75)',
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

	<!-- 右区：SpecPilot 服务 + backend 状态 + 版本 -->
	<div class="sb-right">
		<button
			type="button"
			class="sb-toggle"
			class:off={!canvasAutoUiEnabled}
			title="通用对话结束后自动触发 Canvas 专用 Agent"
			onclick={() => ontoggleCanvasAutoUi?.()}
		>
			Canvas Auto UI: {canvasAutoUiEnabled ? 'On' : 'Off'}
		</button>
		<span class="sb-sep" aria-hidden="true">|</span>

		<!-- SpecSlot 手动打开 -->
		<button
			type="button"
			class="sb-sp-btn"
			title="打开 Spec 槽位抽屉"
			onclick={() => specSlotSessionStore.openDrawer()}
		>
			<span class="sp-dot" style="color: var(--accent-green, #87cf8a)">▣</span>
			SpecSlot
		</button>
		<span class="sb-sep" aria-hidden="true">|</span>

		<!-- SpecPilot DC status -->
		{#if spStatus}
			<span class="sb-sp" title="SpecPilot DataCenter: {spStatus.dcPort}">
				<span class="sp-dot" style="color: var(--accent-green, #87cf8a)">●</span>
				DC:{spStatus.dcPort}
			</span>
			<span class="sb-sep" aria-hidden="true">|</span>
			<span class="sb-sp" title="SpecPilot MF Server: {spStatus.mfPort}">
				<span class="sp-dot" style="color: var(--accent-green, #87cf8a)">●</span>
				MF:{spStatus.mfPort}
			</span>
			<button
				type="button"
				class="sb-sp-btn"
				title="在浏览器新标签打开 SpecPilot 原型预览"
				onclick={() => window.open(`http://localhost:${spStatus.mfPort}/preview`, '_blank')}
			>
				预览
			</button>
		{:else}
			<span class="sb-sp sb-sp-off" title="SpecPilot 未启动">
				<span class="sp-dot" style="color: var(--accent-orange, #f09a6a)">○</span>
				DC:—
			</span>
			<span class="sb-sep" aria-hidden="true">|</span>
			<button
				type="button"
				class="sb-sp-btn"
				disabled={spBootstrapping}
				title={spBootstrapping ? '启动中…' : spError || '启动 SpecPilot DC+MF 服务'}
				onclick={bootstrap}
			>
				{#if spBootstrapping}
					<span class="sp-dot" style="color: var(--accent-yellow, #efc66b)">⟳</span> 启动中…
				{:else if spError}
					<span class="sp-dot" style="color: var(--accent-red, #e16d75)">✗</span> 重试
				{:else}
					<span class="sp-dot" style="color: var(--accent-orange, #f09a6a)">▶</span> SpecPilot
				{/if}
			</button>
		{/if}

		<span class="sb-sep" aria-hidden="true">|</span>
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
		background: var(--statusbar-bg, #10131a);
		border-top: 1px solid var(--wb-border, #303746);
		flex-shrink: 0;
		font-family: var(--font-ui, 'Inter', sans-serif);
		font-size: 12px;
		color: var(--wb-text-sec, #a3abb9);
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
		color: var(--wb-text-sec, #a3abb9);
		font-size: 11.5px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		max-width: 320px;
	}

	.sb-sep {
		color: var(--wb-muted, #6f7888);
		font-size: 11px;
	}

	.sb-meta {
		color: var(--wb-muted, #6f7888);
		font-size: 11px;
		white-space: nowrap;
	}

	.sb-sp {
		display: flex;
		align-items: center;
		gap: 2px;
		font-size: 11.5px;
		font-weight: 500;
		color: var(--accent-green, #87cf8a);
		white-space: nowrap;
	}

	.sb-sp-off {
		color: var(--wb-muted, #555558);
	}

	.sp-dot {
		font-size: 10px;
		line-height: 1;
	}

	.sb-sp-btn {
		border: none;
		background: transparent;
		color: var(--accent-orange, #f09a6a);
		font-size: 11.5px;
		font-weight: 500;
		padding: 2px 6px;
		border-radius: 4px;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 2px;
		white-space: nowrap;
		transition: background 0.15s;
	}

	.sb-sp-btn:hover:not(:disabled) {
		background: rgba(240, 154, 106, 0.12);
	}

	.sb-sp-btn:disabled {
		cursor: default;
		opacity: 0.7;
	}

	.sb-toggle {
		border: 1px solid #3d557a;
		background: #11203a;
		color: #cfe0ff;
		font-size: 11px;
		padding: 2px 8px;
		border-radius: 999px;
		cursor: pointer;
	}
	.sb-toggle.off {
		border-color: #5a3a3a;
		background: #2a1717;
		color: #f4c5c5;
	}
</style>
