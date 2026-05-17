<script lang="ts">
	import { specpilotStatusStore } from '$lib/stores/specpilot-status-store';

	let panelOpen = $state(false);
	let mfPort = $state(5177);
	let mounted = $state(false);

	$effect(() => {
		const unsub = specpilotStatusStore.subscribe((s) => {
			if (s) {
				panelOpen = s.panelOpen;
				mfPort = s.mfPort;
				mounted = true;
			}
		});
		return unsub;
	});

	const previewUrl = $derived(
		mounted ? `http://localhost:${mfPort}/preview` : null
	);

	function close() {
		specpilotStatusStore.update((s) => (s ? { ...s, panelOpen: false } : s));
	}
</script>

{#if panelOpen}
<div class="sp-panel" role="region" aria-label="SpecPilot 原型预览">
	<div class="sp-panel-head">
		<span class="sp-panel-title">SpecPilot 原型</span>
		<div class="sp-panel-actions">
			{#if previewUrl}
				<button type="button" class="sp-btn" onclick={() => window.open(previewUrl, '_blank')}>
					↗ 全屏
				</button>
			{/if}
			<button type="button" class="sp-btn close" onclick={close}>关闭</button>
		</div>
	</div>
	<div class="sp-panel-body">
		{#if previewUrl}
			<iframe
				src={previewUrl}
				title="SpecPilot Preview"
				class="sp-iframe"
				sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
			></iframe>
		{:else}
			<div class="sp-panel-empty">
				<p>SpecPilot 未启动</p>
				<p class="sp-hint">点击底部状态栏「SpecPilot」按钮启动服务</p>
			</div>
		{/if}
	</div>
</div>
{/if}

<style>
	.sp-panel {
		position: fixed;
		bottom: var(--statusbar-h, 24px);
		left: 0;
		right: 0;
		height: 280px;
		z-index: 90;
		display: flex;
		flex-direction: column;
		background: #0b0d12;
		border-top: 1px solid rgba(122, 162, 255, 0.3);
		box-shadow: 0 -8px 32px rgba(0, 0, 0, 0.4);
	}

	.sp-panel-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 6px 14px;
		border-bottom: 1px solid rgba(255, 255, 255, 0.07);
		flex-shrink: 0;
	}

	.sp-panel-title {
		font-size: 11px;
		font-weight: 600;
		color: #8ab;
		letter-spacing: 0.05em;
		text-transform: uppercase;
	}

	.sp-panel-actions {
		display: flex;
		gap: 6px;
	}

	.sp-btn {
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 5px;
		background: rgba(255, 255, 255, 0.05);
		color: #889;
		font-size: 11px;
		padding: 3px 9px;
		cursor: pointer;
	}
	.sp-btn:hover { background: rgba(255, 255, 255, 0.1); color: #aab; }
	.sp-btn.close { color: #e16d75; border-color: rgba(225, 109, 117, 0.3); }
	.sp-btn.close:hover { background: rgba(225, 109, 117, 0.15); }

	.sp-panel-body {
		flex: 1;
		min-height: 0;
		overflow: hidden;
	}

	.sp-iframe {
		width: 100%;
		height: 100%;
		border: none;
		background: #0f0f1a;
	}

	.sp-panel-empty {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 6px;
		color: #556;
	}
	.sp-panel-empty p { margin: 0; font-size: 13px; }
	.sp-hint { font-size: 11px; color: #445; }
</style>
