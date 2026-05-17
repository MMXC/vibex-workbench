<script lang="ts">
	import {
		activeSpecSlotSession,
		specSlotSessionStore,
	} from '$lib/stores/spec-slot-session-store';
	import SpecSlotChatPane from '$lib/components/workbench/SpecSlotChatPane.svelte';
	import SpecSlotVisualPane from '$lib/components/workbench/SpecSlotVisualPane.svelte';

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
			state = value;
		});
		return unsub;
	});

	const session = $derived.by(() => activeSpecSlotSession(state));
</script>

<div class="agent-hub">
	{#if session}
		<!-- 有 spec session: 上下布局：聊天 + 原型 -->
		<div class="hub-header">
			<span class="hub-eyebrow">Spec Slot</span>
			<span class="hub-title">{session.spec.display.title}</span>
			<span class="hub-meta">{session.slot.label} · {session.slot.status}</span>
			<div class="hub-actions">
				<button type="button" onclick={() => specSlotSessionStore.compactActive()}>Compact</button>
				<button type="button" onclick={() => specSlotSessionStore.resetActive()}>Reset</button>
			</div>
		</div>
		<div class="hub-body">
			<div class="hub-chat">
				<SpecSlotChatPane {session} />
			</div>
			<div class="hub-visual">
				<SpecSlotVisualPane {session} />
			</div>
		</div>
	{:else}
		<!-- 无 session: 引导页 -->
		<div class="hub-empty">
			<div class="hub-empty-icon">📋</div>
			<p class="hub-empty-title">Spec Slot</p>
			<p class="hub-empty-hint">在 Spec Console 中选择一个 spec 节点<br>即可在此与 Agent 澄清并预览</p>
		</div>
	{/if}
</div>

<style>
	.agent-hub {
		width: 100%;
		height: 100%;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		background: #0b0d12;
	}

	.hub-header {
		padding: 12px 14px;
		border-bottom: 1px solid #1e2030;
		flex-shrink: 0;
	}
	.hub-eyebrow {
		font-size: 10px;
		color: #72d6d0;
		font-weight: 800;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.hub-title {
		display: block;
		font-size: 14px;
		font-weight: 600;
		color: #eef0f5;
		margin: 3px 0 1px;
	}
	.hub-meta {
		font-size: 11px;
		color: #556;
	}
	.hub-actions {
		display: flex;
		gap: 6px;
		margin-top: 8px;
	}
	.hub-actions button {
		padding: 4px 10px;
		border-radius: 6px;
		border: 1px solid #303746;
		background: transparent;
		color: #778;
		font-size: 11px;
		cursor: pointer;
	}
	.hub-actions button:hover { border-color: #7aa2ff; color: #aab; }

	.hub-body {
		flex: 1;
		min-height: 0;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
	.hub-chat { flex: 1; min-height: 0; overflow: hidden; }
	.hub-visual { height: 220px; flex-shrink: 0; border-top: 1px solid #1e2030; overflow: hidden; }

	.hub-empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		padding: 32px;
	}
	.hub-empty-icon { font-size: 36px; opacity: 0.5; }
	.hub-empty-title { font-size: 15px; color: #778; margin: 0; }
	.hub-empty-hint { font-size: 12px; color: #445; text-align: center; line-height: 1.6; margin: 0; }
</style>
