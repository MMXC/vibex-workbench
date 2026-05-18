<!-- SpecPilotPreview.svelte
中央 SpecPilot 原型预览区。
加载 {workspace}/.specpilot/prototypes/{specId}.html
通过 Go backend HTTP API 获取，无 Python 服务依赖。
-->
<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { onDestroy } from 'svelte';

	let { specId = '' }: { specId?: string } = $props();

	let html = $state('');
	let loading = $state(false);
	let error = $state('');
	let reloadKey = $state(0);

	async function loadPrototype() {
		if (!specId) {
			html = '';
			return;
		}
		loading = true;
		error = '';
		try {
			const r = await fetch(`/api/specpilot/prototype/${specId}`, {
				signal: AbortSignal.timeout(5000),
			});
			if (r.status === 404) {
				error = '暂无原型 — Agent 生成后自动加载';
				html = '';
			} else if (!r.ok) {
				error = `加载失败 (${r.status})`;
				html = '';
			} else {
				html = await r.text();
			}
		} catch (e) {
			error = '原型服务不可用';
			html = '';
		} finally {
			loading = false;
		}
	}

	// Re-load when specId changes or reloadKey changes
	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		specId;
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		reloadKey;
		loadPrototype();
	});

	// Listen for prototype-updated events (from Agent writing new prototype)
	function handleUpdate(e: Event) {
		const detail = (e as CustomEvent).detail;
		if (detail?.specId === specId) {
			reloadKey++;
		}
	}

	// Expose reload for external use (e.g. StatusBar button)
	export function reload() {
		reloadKey++;
	}

	const wsRoot = $derived($specExplorerStore.workspaceRoot ?? '—');
</script>

<div class="spp">
	{#if loading}
		<div class="spp-empty">
			<div class="spp-spinner">◌</div>
			<div class="spp-title">加载原型…</div>
		</div>
	{:else if html}
		<div class="spp-toolbar">
			<span class="spp-badge">{specId || '原型'}</span>
			<span class="spp-path">{wsRoot}/.specpilot/prototypes/</span>
			<button class="spp-reload" onclick={() => reloadKey++} title="刷新原型">↻</button>
		</div>
		<iframe
			srcdoc={html}
			class="spp-iframe"
			title="SpecPilot Prototype: {specId}"
			sandbox="allow-scripts allow-same-origin allow-forms"
		></iframe>
	{:else}
		<div class="spp-empty">
			<div class="spp-icon">⬡</div>
			<div class="spp-title">{specId ? error : '选择 Spec 以预览原型'}</div>
			{#if !specId}
				<div class="spp-hint">在左侧 AgentHub 对话中描述想要的功能<br/>AI Agent 会生成 HTML 原型并自动加载</div>
			{:else if !error}
				<div class="spp-hint">原型文件不存在 · AI 生成后自动出现</div>
			{/if}
		</div>
	{/if}
</div>

<style>
.spp {
	width: 100%;
	height: 100%;
	overflow: hidden;
	background: #0d0d0e;
	display: flex;
	flex-direction: column;
}
.spp-iframe {
	border: none;
	width: 100%;
	flex: 1;
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
.spp-hint { font-size: 12px; color: #3d4656; text-align: center; max-width: 280px; line-height: 1.6; }
.spp-spinner { font-size: 32px; animation: spin 1s linear infinite; color: var(--wb-accent, #72d6d0); }
@keyframes spin { to { transform: rotate(360deg); } }
.spp-toolbar {
	display: flex;
	align-items: center;
	gap: 8px;
	padding: 4px 10px;
	background: #111114;
	border-bottom: 1px solid #1e2030;
	flex-shrink: 0;
}
.spp-badge {
	background: var(--wb-accent, #72d6d0);
	color: #000;
	font-size: 11px;
	font-weight: 700;
	padding: 1px 6px;
	border-radius: 3px;
}
.spp-path { font-size: 11px; color: #3d4656; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.spp-reload {
	background: none;
	border: 1px solid #2a2d3a;
	color: #787c99;
	font-size: 14px;
	padding: 0 6px;
	border-radius: 3px;
	cursor: pointer;
	line-height: 1;
}
.spp-reload:hover { border-color: var(--wb-accent, #72d6d0); color: var(--wb-accent, #72d6d0); }
</style>
