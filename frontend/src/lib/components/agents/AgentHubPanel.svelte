<!-- AgentHubPanel.svelte
左侧 Agent Hub 面板。
承载 SpecSlot 聊天能力（SpecSlotChatPane）。
Phase 2：多会话 tab 支持。
-->
<script lang="ts">
	import {
		activeSpecSlotSession,
		specSlotSessionStore,
		type SpecSlotSession,
		type SpecSlotSessionState,
	} from '$lib/stores/spec-slot-session-store';
	import SpecSlotChatPane from '$lib/components/workbench/SpecSlotChatPane.svelte';

	let storeState = $state<SpecSlotSessionState>({
		activeKey: null,
		drawerOpen: false,
		sessions: {},
	});

	$effect(() => {
		const unsub = specSlotSessionStore.subscribe(value => {
			storeState = value;
		});
		return unsub;
	});

	const activeSession = $derived.by(() => activeSpecSlotSession(storeState));

	function openDrawer() {
		specSlotSessionStore.openDrawer();
	}
</script>

<div class="ahp">
	<div class="ahp-body">
		{#if activeSession}
			<SpecSlotChatPane session={activeSession} />
		{:else}
		<div class="ahp-empty">
			<div class="ahp-icon">⬡</div>
			<div class="ahp-hint">暂无活跃会话</div>
			<button type="button" class="ahp-start-btn" onclick={openDrawer}>
				打开 SpecSlot
			</button>
		</div>
		{/if}
	</div>
</div>

<style>
.ahp {
	display: flex;
	flex-direction: column;
	height: 100%;
	background: #0b0d12;
	overflow: hidden;
}
.ahp-body {
	flex: 1;
	overflow: hidden;
	position: relative;
}
.ahp-empty {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	height: 100%;
	gap: 12px;
	padding: 24px;
}
.ahp-icon { font-size: 40px; opacity: 0.25; color: var(--wb-accent, #72d6d0); }
.ahp-hint { font-size: 13px; color: var(--wb-text-sec, #787c99); }
.ahp-start-btn {
	background: rgba(114,214,208,0.12);
	border: 1px solid rgba(114,214,208,0.3);
	border-radius: 6px;
	color: var(--wb-accent, #72d6d0);
	font-size: 12px;
	cursor: pointer;
	padding: 6px 16px;
}
.ahp-start-btn:hover { background: rgba(114,214,208,0.2); }
</style>
