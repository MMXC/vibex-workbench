<script lang="ts">
	import {
		activeSpecSlotSession,
		specSlotSessionStore,
		type SpecSlotSession,
	} from '$lib/stores/spec-slot-session-store';
	import SpecSlotChatPane from '$lib/components/workbench/SpecSlotChatPane.svelte';
	import SpecSlotVisualPane from '$lib/components/workbench/SpecSlotVisualPane.svelte';

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
			state = value;
		});
		return unsub;
	});

	const session = $derived.by(() => activeSpecSlotSession(state));

	async function openMfInTab() {
		const port = (window as any).__specpilotMfPort ?? 5177;
		window.open(`http://localhost:${port}/preview`, '_blank');
	}

	async function openMfInDrawer() {
		const port = (window as any).__specpilotMfPort ?? 5177;
		const payload = {
			type: 'specpilot_bootstrap',
			mfPort: port,
			dcPort: 7890,
			mfUrl: `http://localhost:${port}/preview`,
			mfRemoteUrl: `http://localhost:${port}/preview`,
			message: 'SpecPilot 原型已加载，点击「新窗口打开」可全屏查看',
		};
		specSlotSessionStore.injectToolResult(JSON.stringify(payload));
	}
</script>

{#if state.drawerOpen}
<div class="drawer-backdrop" role="presentation">
	<section class="slot-drawer" aria-label="Spec 槽位澄清抽屉">

		{#if session}
		<!-- 有 spec session：左侧澄清 + 右侧 SpecSlotVisualPane -->
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
		<!-- 无 spec session：显示引导页 -->
		<header class="drawer-head">
			<div class="title">
				<span class="eyebrow">Spec Slot</span>
				<h2>原型插槽</h2>
				<p>在中央面板选择一个 spec 节点后，可在此处与 Agent 澄清并预览原型</p>
			</div>
			<div class="tools">
				<button type="button" class="primary" onclick={() => specSlotSessionStore.close()}>关闭</button>
			</div>
		</header>
		<div class="drawer-empty">
			<div class="empty-icon">📋</div>
			<p class="empty-title">暂无激活的 Spec 槽位</p>
			<p class="empty-hint">请在左侧或中央面板选择一个 spec 节点，<br>然后点击「打开槽位」按钮</p>
			<div class="empty-actions">
				<button type="button" class="empty-btn secondary" onclick={openMfInTab}>
					↗ 新窗口打开 SpecPilot 原型
				</button>
			</div>
		</div>
		{/if}

	</section>
</div>
{/if}

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

	.drawer-empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		padding: 32px;
	}
	.empty-icon { font-size: 36px; margin-bottom: 4px; }
	.empty-title { font-size: 15px; color: #778; margin: 0; }
	.empty-hint { font-size: 12px; color: #556; text-align: center; margin: 0; }
	.empty-actions { display: flex; gap: 10px; margin-top: 8px; }
	button.empty-btn {
		padding: 8px 16px;
		border-radius: 9px;
		border: 1px solid rgba(122, 162, 255, 0.3);
		background: rgba(122, 162, 255, 0.1);
		color: #8ab;
		font-size: 12px;
		cursor: pointer;
	}
	button.empty-btn:hover { background: rgba(122, 162, 255, 0.18); color: #aac; }
	button.secondary {
		background: rgba(255, 255, 255, 0.05);
		border-color: rgba(255, 255, 255, 0.12);
		color: #889;
	}
	button.secondary:hover { background: rgba(255, 255, 255, 0.1); }
</style>
