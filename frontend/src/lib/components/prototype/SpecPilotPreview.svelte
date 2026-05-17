<script lang="ts">
	import { specpilotStatusStore } from '$lib/stores/specpilot-status-store';

	let spStatus = $state<{ mfPort: number; dcPort: number } | null>(null);

	$effect(() => {
		const unsub = specpilotStatusStore.subscribe(s => {
			if (s) spStatus = s;
		});
		return unsub;
	});

	const previewUrl = $derived(
		spStatus ? `http://localhost:${spStatus.mfPort}/preview` : null
	);

	let key = $state(0); // force iframe reload
	function refresh() { key++; }
</script>

<div class="preview-wrap">
	{#if previewUrl}
		<div class="preview-toolbar">
			<span class="pt-label">SpecPilot Preview</span>
			<button type="button" class="pt-btn" onclick={refresh} title="刷新">↻</button>
			<button type="button" class="pt-btn" onclick={() => window.open(previewUrl, '_blank')} title="新窗口打开">↗</button>
			<span class="pt-url">{previewUrl}</span>
		</div>
		<iframe
			src="{previewUrl}?k={key}"
			title="SpecPilot 原型预览"
			class="preview-iframe"
			sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
		></iframe>
	{:else}
		<div class="preview-empty">
			<div class="pe-icon">🚀</div>
			<p class="pe-title">SpecPilot 未启动</p>
			<p class="pe-hint">点击底部状态栏「SpecPilot」按钮启动服务</p>
			<p class="pe-hint2">启动后，原型将在此全屏显示</p>
		</div>
	{/if}
</div>

<style>
	.preview-wrap {
		width: 100%;
		height: 100%;
		display: flex;
		flex-direction: column;
		background: #0a0b10;
	}

	.preview-toolbar {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 5px 12px;
		background: #10131a;
		border-bottom: 1px solid #1e2030;
		flex-shrink: 0;
	}
	.pt-label {
		font-size: 11px;
		color: #556;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		flex-shrink: 0;
	}
	.pt-btn {
		border: none;
		background: transparent;
		color: #556;
		font-size: 13px;
		cursor: pointer;
		padding: 2px 5px;
		border-radius: 4px;
	}
	.pt-btn:hover { background: rgba(255,255,255,0.08); color: #99a; }
	.pt-url {
		font-size: 10px;
		color: #334;
		font-family: 'Cascadia Code', monospace;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.preview-iframe {
		flex: 1;
		border: none;
		width: 100%;
		height: 100%;
		background: #0f1018;
	}

	.preview-empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
	}
	.pe-icon { font-size: 48px; opacity: 0.3; }
	.pe-title { font-size: 16px; color: #556; margin: 0; }
	.pe-hint { font-size: 13px; color: #334; margin: 0; }
	.pe-hint2 { font-size: 12px; color: #2a2d3a; margin: 0; }
</style>
