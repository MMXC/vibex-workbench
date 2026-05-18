<!-- SpecPilotPreview.svelte
中央 SpecPilot 原型预览区。
全屏 iframe 加载 MF preview URL。
-->
<script lang="ts">
	import { specpilotStatusStore } from '$lib/stores/specpilot-status-store';

	let mfUrl = $derived(
		$specpilotStatusStore?.mfRunning && $specpilotStatusStore?.mfPort
			? `http://localhost:${$specpilotStatusStore.mfPort}/preview`
			: ''
	);
</script>

<div class="spp">
	{#if mfUrl}
	<iframe
		src={mfUrl}
		class="spp-iframe"
		title="SpecPilot Prototype Preview"
		allow="fullscreen"
	></iframe>
	{:else}
	<div class="spp-empty">
		<div class="spp-icon">⬡</div>
		<div class="spp-title">SpecPilot 未启动</div>
		<div class="spp-hint">点击底部状态栏 <strong>SpecPilot</strong> 按钮启动服务</div>
	</div>
	{/if}
</div>

<style>
.spp {
	width: 100%;
	height: 100%;
	overflow: hidden;
	background: #0d0d0e;
}
.spp-iframe {
	border: none;
	width: 100%;
	height: 100%;
	display: block;
}
.spp-empty {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	height: 100%;
	gap: 10px;
}
.spp-icon { font-size: 48px; opacity: 0.2; color: var(--wb-accent, #72d6d0); }
.spp-title { font-size: 16px; font-weight: 600; color: var(--wb-text-sec, #787c99); }
.spp-hint { font-size: 12px; color: #3d4656; text-align: center; max-width: 280px; }
.spp-hint strong { color: var(--wb-accent, #72d6d0); }
</style>
