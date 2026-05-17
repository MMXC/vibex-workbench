<script lang="ts">
	import {
		activeSpecSlotSession,
		specSlotSessionStore,
		type SpecSlotSession,
	} from '$lib/stores/spec-slot-session-store';
	import SpecSlotChatPane from '$lib/components/workbench/SpecSlotChatPane.svelte';
	import SpecSlotVisualPane from '$lib/components/workbench/SpecSlotVisualPane.svelte';

	// Props
	let { workspaceRoot = '—' }: { workspaceRoot?: string } = $props();

	let state = $state<{
		activeKey: string | null;
		drawerOpen: boolean;
		sessions: Record<string, SpecSlotSession>;
	}>({
		activeKey: null,
		drawerOpen: false,
		sessions: {},
	});

	$effect(() => {
		const unsub = specSlotSessionStore.subscribe(value => {
			// Reconstruct drawerOpen-equivalent: drawer is open when there's an active session
			state = value;
		});
		return unsub;
	});

	const session = $derived.by(() => activeSpecSlotSession(state));

	// SpecPilot status — same polling as StatusBar
	interface SpStatus {
		installed: boolean;
		dcRunning: boolean;
		mfRunning: boolean;
		dcPort: number;
		mfPort: number;
	}
	let spStatus: SpStatus | null = $state(null);
	let spPollTimer: ReturnType<typeof setInterval> | null = null;

	async function pollSpStatus() {
		try {
			const r = await fetch('http://localhost:33338/api/specpilot/status', {
				signal: AbortSignal.timeout(3000),
			});
			if (r.ok) {
				const d = (await r.json()) as Record<string, unknown>;
				spStatus = {
					installed: (d.installed as boolean) ?? false,
					dcRunning: (d.dcRunning as boolean) ?? false,
					mfRunning: (d.mfRunning as boolean) ?? false,
					dcPort: (d.dcPort as number) ?? 7890,
					mfPort: (d.mfPort as number) ?? 5177,
				};
				// Expose port globally so MF iframe can use it
				(window as unknown as Record<string, unknown>).__specpilotMfPort = spStatus.mfPort;
				return;
			}
		} catch {
			// fallback: DC health directly
		}
		try {
			const r = await fetch('http://127.0.0.1:7890/api/health', {
				signal: AbortSignal.timeout(2000),
			});
			if (r.ok) {
				spStatus = { installed: true, dcRunning: true, mfRunning: false, dcPort: 7890, mfPort: 5177 };
				(window as unknown as Record<string, unknown>).__specpilotMfPort = 5177;
			}
		} catch {
			spStatus = { installed: false, dcRunning: false, mfRunning: false, dcPort: 7890, mfPort: 5177 };
		}
	}

	// Derive the SpecPilot preview URL
	const previewUrl = $derived.by(() => {
		const port = spStatus?.mfPort ?? 5177;
		const sp = spStatus;
		if (sp && (sp.mfRunning || sp.installed)) {
			return `http://127.0.0.1:${port}/preview`;
		}
		return null; // not started yet
	});

	// Polling
	$effect(() => {
		pollSpStatus();
		spPollTimer = setInterval(pollSpStatus, 10000);
		return () => {
			if (spPollTimer) clearInterval(spPollTimer);
		};
	});
</script>

<!-- 始终显示，不依赖 drawerOpen — 只要 workspace 打开就有右侧预览区 -->
<div class="drawer-backdrop" role="presentation">
	<section class="slot-drawer" aria-label="Spec 原型抽屉">

		{#if session}
			<!-- 有 spec session 时：左侧澄清 + 右侧 SpecSlotVisualPane -->
			<header class="drawer-head">
				<div class="title">
					<span class="eyebrow">Spec Slot Workspace</span>
					<h2>{session.spec.display.title}</h2>
					<p>{session.slot.label} · {session.slot.status} · {session.spec.path}</p>
				</div>
				<div class="tools">
					<button type="button" onclick={() => specSlotSessionStore.compactActive()}>Compact</button>
					<button type="button" onclick={() => specSlotSessionStore.resetActive()}>Reset</button>
					<button type="button" class="primary" onclick={() => specSlotSessionStore.close()}>Close</button>
				</div>
			</header>
			<div class="drawer-body">
				<div class="chat-wrap">
					<SpecSlotChatPane {session} />
				</div>
				<div class="visual-wrap">
					<SpecSlotVisualPane {session} />
				</div>
			</div>
		{:else}
			<!-- 无 spec session：左侧空聊天 + 右侧 SpecPilot 原型预览 -->
			<header class="drawer-head">
				<div class="title">
					<span class="eyebrow">SpecPilot</span>
					<h2>原型预览</h2>
					<p class="hint">
						{workspaceRoot !== '—' ? workspaceRoot : '未绑定 workspace'}
					</p>
				</div>
				<div class="tools">
					<span class="sp-status-badge" class:ready={spStatus?.mfRunning} class:stopped={!spStatus?.mfRunning}>
						{spStatus?.mfRunning ? '🟢 Running' : spStatus?.installed ? '🟡 Stopped' : '⚪ Not Init'}
					</span>
					{#if previewUrl}
						<button
							type="button"
							class="primary"
							onclick={() => window.open(previewUrl, '_blank')}
						>
							↗ 全屏
						</button>
					{/if}
				</div>
			</header>
			<div class="drawer-body">
				<!-- Left: placeholder chat area -->
				<div class="chat-wrap">
					<div class="spec-pilot-placeholder">
						<div class="sp-hint">
							<span class="sp-logo">🚀</span>
							<p>SpecPilot 原型预览区</p>
							<p class="sp-sub">选择左侧 spec 节点后在此澄清并预览，或直接通过 Agent 驱动</p>
							<div class="sp-cmd-list">
								<div class="sp-cmd-item"><code>specpilot init</code> — 初始化工作区</div>
								<div class="sp-cmd-item"><code>specpilot start</code> — 启动服务</div>
								<div class="sp-cmd-item"><code>specpilot generate &lt;spec&gt;</code> — 生成原型</div>
							</div>
						</div>
					</div>
				</div>
				<!-- Right: SpecPilot preview iframe (always visible) -->
				<div class="visual-wrap">
					{#if previewUrl}
						<iframe
							src={previewUrl}
							title="SpecPilot Preview"
							class="specpilot-iframe"
							allow="cross-origin-isolated"
							sandbox="allow-scripts allow-same-origin allow-forms"
						></iframe>
					{:else}
						<div class="sp-empty-preview">
							<div class="sp-empty-icon">🚀</div>
							<p>SpecPilot 未启动</p>
							<p class="sp-empty-hint">Agent 将在需要时自动启动</p>
							{#if workspaceRoot !== '—'}
								<div class="sp-ws-info">
									<span>工作区：</span><code>{workspaceRoot}</code>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			</div>
		{/if}

	</section>
</div>

<style>
	.drawer-backdrop {
		position: fixed;
		inset: 42px 18px 24px 74px;
		z-index: 80;
		display: flex;
		pointer-events: none;
	}
	.slot-drawer {
		pointer-events: auto;
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100%;
		min-height: 0;
		border: 1px solid rgba(122, 162, 255, 0.42);
		border-radius: 20px;
		background: #0b0d12;
		box-shadow: 0 26px 90px rgba(0, 0, 0, 0.58);
		overflow: hidden;
	}
	.drawer-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 18px;
		border-bottom: 1px solid rgba(255, 255, 255, 0.07);
		background: rgba(255, 255, 255, 0.02);
		flex-shrink: 0;
	}
	.title { flex: 1; min-width: 0; }
	.eyebrow { font-size: 10px; color: rgba(122, 162, 255, 0.7); text-transform: uppercase; letter-spacing: 1px; }
	h2 { font-size: 14px; font-weight: 600; color: #dde; margin: 3px 0 1px; }
	p { font-size: 11px; color: #667; }
	p.hint { font-size: 11px; color: #556; font-family: monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.tools { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
	button {
		padding: 5px 11px;
		border-radius: 7px;
		border: 1px solid rgba(255, 255, 255, 0.1);
		background: rgba(255, 255, 255, 0.05);
		color: #99a;
		font-size: 11px;
		cursor: pointer;
		transition: all 0.15s;
	}
	button:hover { background: rgba(255, 255, 255, 0.1); color: #ccd; }
	button.primary { background: rgba(122, 162, 255, 0.2); border-color: rgba(122, 162, 255, 0.4); color: #aab; }
	button.primary:hover { background: rgba(122, 162, 255, 0.3); }

	.sp-status-badge {
		font-size: 10px;
		padding: 3px 8px;
		border-radius: 8px;
		font-weight: 600;
	}
	.sp-status-badge.ready { background: rgba(5, 150, 105, 0.2); color: #34d399; }
	.sp-status-badge.stopped { background: rgba(255, 255, 255, 0.05); color: #667; }

	.drawer-body {
		display: grid;
		grid-template-columns: minmax(320px, 0.8fr) minmax(400px, 1.2fr);
		flex: 1;
		min-height: 0;
		overflow: hidden;
	}
	.chat-wrap {
		border-right: 1px solid rgba(255, 255, 255, 0.06);
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
	.visual-wrap {
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	/* SpecPilot placeholder (left panel, no session) */
	.spec-pilot-placeholder {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 20px;
	}
	.sp-hint { text-align: center; color: #556; }
	.sp-logo { font-size: 36px; display: block; margin-bottom: 12px; }
	.sp-hint p { font-size: 13px; color: #778; margin-bottom: 4px; }
	.sp-sub { font-size: 11px; color: #556; margin-bottom: 16px; }
	.sp-cmd-list { text-align: left; display: inline-block; }
	.sp-cmd-item { font-size: 11px; color: #667; margin: 5px 0; }
	.sp-cmd-item code {
		background: rgba(255, 255, 255, 0.06);
		padding: 2px 6px;
		border-radius: 4px;
		font-size: 10px;
		color: #8ab;
	}

	/* SpecPilot preview iframe (right panel) */
	.specpilot-iframe {
		width: 100%;
		height: 100%;
		border: none;
		background: #0f0f1a;
	}

	/* Empty preview state (SpecPilot not started) */
	.sp-empty-preview {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		color: #445;
	}
	.sp-empty-icon { font-size: 40px; margin-bottom: 8px; }
	.sp-empty-preview p { font-size: 14px; color: #556; margin: 0; }
	.sp-empty-hint { font-size: 11px; color: #445; }
	.sp-ws-info { margin-top: 12px; font-size: 10px; color: #556; }
	.sp-ws-info code {
		background: rgba(255, 255, 255, 0.05);
		padding: 2px 6px;
		border-radius: 4px;
		color: #8ab;
		font-size: 9px;
		max-width: 300px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		display: inline-block;
		vertical-align: middle;
	}
</style>
