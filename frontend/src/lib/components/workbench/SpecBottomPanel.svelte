<!-- SpecBottomPanel.svelte
Bottom split panel — toggled by StatusBar click.
Three tabs: MF Prototype | DC State | Spec Console
Data from /api/specpilot/* endpoints (proxied through Wails to agent backend 33338).
-->
<script lang="ts">
	import { onDestroy } from 'svelte';
	import { specpilotStatusStore } from '$lib/stores/specpilot-status-store';

	let { workspaceRoot }: { workspaceRoot?: string } = $props();

	// Panel driven by StatusBar store
	let open = $state(false);
	let prevOpen = false;
	const unsub = specpilotStatusStore.subscribe((s) => {
		prevOpen = open;
		open = s?.panelOpen ?? false;
	});

	let activeTab = $state<'mf' | 'dc' | 'console'>('mf');
	let height = $state(280);

	function toggle() {
		open = !open;
		specpilotStatusStore.update((s) => (s ? { ...s, panelOpen: open } : s));
	}

	function collapse() {
		open = false;
		specpilotStatusStore.update((s) => (s ? { ...s, panelOpen: false } : s));
	}

	// ── DC tab ──────────────────────────────────────────
	let dcData = $state<{key: string; value: unknown; source?: string; updated_at?: string; version?: number}[]>([]);
	let dcTotal = $state(0);
	let dcLoading = $state(false);
	let dcError = $state('');

	// ── EC tab ──────────────────────────────────────────
	interface ECEvent { event: string; payload: unknown; emitted_at: string; subscriber_count: number; }
	let ecHistory = $state<ECEvent[]>([]);

	// ── MF tab ──────────────────────────────────────────
	interface MFComponent { name: string; path: string; status: string; contract?: { inputs: Record<string, unknown>; outputs: unknown[]; events: unknown[] }; }
	let mfComponents = $state<MFComponent[]>([]);

	// ── Load ───────────────────────────────────────────
	async function loadAll() {
		await Promise.all([loadDC(), loadECHistory(), loadMFComponents()]);
	}

	async function loadDC() {
		dcLoading = true;
		dcError = '';
		try {
			const res = await fetch('/api/specpilot/dc');
			const r = await res.json();
			if (r.error) { dcError = r.error; return; }
			dcData = Object.entries(r.data ?? {}).map(([key, value]: [string, unknown]) => ({ key, value, source: '', updated_at: '', version: 0 }));
			dcTotal = r.total ?? 0;
		} catch (e: unknown) {
			dcError = String(e);
		} finally {
			dcLoading = false;
		}
	}

	async function loadECHistory() {
		try {
			const res = await fetch('/api/specpilot/ec/history?limit=20');
			const r = await res.json();
			ecHistory = Array.isArray(r) ? r : [];
		} catch { /* ignore */ }
	}

	async function loadMFComponents() {
		try {
			const res = await fetch('/api/specpilot/mf/components');
			const r = await res.json();
			mfComponents = Array.isArray(r.components) ? r.components : [];
		} catch { /* ignore */ }
	}

	$effect(() => {
		if (open) loadAll();
	});

	// ── Helpers ────────────────────────────────────────
	function fmtVal(v: unknown): string {
		if (v == null) return '--';
		if (typeof v === 'object') return JSON.stringify(v)?.slice(0, 60) ?? '--';
		return String(v);
	}

	function fmtTime(iso: string): string {
		try { return new Date(iso).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }); }
		catch { return iso; }
	}

	// ── Resize handle ──────────────────────────────────
	let dragging = false;
	let startY = 0;
	let startHeight = 0;

	function startResize(e: MouseEvent) {
		dragging = true;
		startY = e.clientY;
		startHeight = height;
		window.addEventListener('mousemove', onMove);
		window.addEventListener('mouseup', stopResize);
	}

	function onMove(e: MouseEvent) {
		if (!dragging) return;
		height = Math.max(120, Math.min(600, startHeight + startY - e.clientY));
	}

	function stopResize() {
		dragging = false;
		window.removeEventListener('mousemove', onMove);
		window.removeEventListener('mouseup', stopResize);
	}

	onDestroy(() => {
		unsub();
		window.removeEventListener('mousemove', onMove);
		window.removeEventListener('mouseup', stopResize);
	});

	const TABS = [
		{ id: 'mf' as const, label: 'MF 原型' },
		{ id: 'dc' as const, label: 'DC State' },
		{ id: 'console' as const, label: 'Spec Console' },
	];
</script>

{#if open}
<div
	class="sp-handle"
	onmousedown={startResize}
	role="separator"
	aria-orientation="horizontal"
	title="拖动调整高度"
></div>

<div class="sp-panel" style="height:{height}px">
	<div class="sp-tabs">
		{#each TABS as tab}
			<button
				type="button"
				class="sp-tab"
				class:active={activeTab === tab.id}
				onclick={() => { activeTab = tab.id; }}
			>{tab.label}</button>
		{/each}
		<div class="sp-tabs-right">
			<span class="sp-total">{dcTotal} keys</span>
			<button type="button" class="sp-icon" onclick={loadAll} title="刷新">↻</button>
			<button type="button" class="sp-icon" onclick={collapse} title="收起">▬</button>
		</div>
	</div>

	<div class="sp-body">

		{#if activeTab === 'mf'}
		<div class="sp-mf">
			{#if mfComponents.length === 0}
			<div class="sp-empty">
				<div class="sp-empty-icon">⬡</div>
				<div>暂无注册的 MF 组件</div>
				<div class="sp-empty-sub">MF app bootstrap 后自动注册</div>
			</div>
			{:else}
			<div class="sp-mf-grid">
				{#each mfComponents as comp}
				<div class="sp-card" class:pending={comp.status === 'pending'}>
					<div class="sp-comp-name">{comp.name}</div>
					<div class="sp-comp-path">{comp.path}</div>
					<div class="sp-comp-status" class:ready={comp.status === 'ready'}>
						{comp.status === 'ready' ? '● 就绪' : '◌ 等待中'}
					</div>
					{#if comp.contract}
					<div class="sp-comp-tags">
						{#if comp.contract.inputs}
						<span class="sp-tag">IN {Object.keys(comp.contract.inputs).length}</span>
						{/if}
						{#if comp.contract.outputs}
						<span class="sp-tag">OUT {comp.contract.outputs.length}</span>
						{/if}
						{#if comp.contract.events}
						<span class="sp-tag">EVT {comp.contract.events.length}</span>
						{/if}
					</div>
					{/if}
				</div>
				{/each}
			</div>
			{/if}
		</div>

		{:else if activeTab === 'dc'}
		<div class="sp-dc">
			{#if dcError}
			<div class="sp-error">{dcError}</div>
			{/if}
			<div class="sp-dc-entries">
				{#if dcLoading}
				<div class="sp-loading">加载中…</div>
				{:else if dcData.length === 0}
				<div class="sp-empty">
					<div class="sp-empty-icon">◈</div>
					<div>DC 中暂无数据</div>
					<div class="sp-empty-sub">SpecPilot bootstrap 后自动灌入</div>
				</div>
				{:else}
				{#each dcData as entry}
				<div class="sp-dc-row">
					<code class="sp-dc-key">{entry.key}</code>
					<code class="sp-dc-val">{fmtVal(entry.value)}</code>
					<code class="sp-dc-source">{entry.source}</code>
				</div>
				{/each}
				{/if}
			</div>
			{#if ecHistory.length > 0}
			<div class="sp-ev-block">
				<span class="sp-section-label">EVENT HISTORY</span>
				{#each ecHistory.slice(-10) as ev}
				<div class="sp-ev-row">{fmtTime(ev.emitted_at)} {ev.event}</div>
				{/each}
			</div>
			{/if}
		</div>

		{:else if activeTab === 'console'}
		<div class="sp-console">
			<div class="sp-empty">
				<div class="sp-empty-icon">📋</div>
				<div>Spec Console</div>
				<div class="sp-empty-sub">spec 校验结果将实时展示在此</div>
			</div>
		</div>
		{/if}

	</div>
</div>
{/if}

<style>
.sp-panel {
	display: flex;
	flex-direction: column;
	background: var(--wb-bg-panel, #0f1117);
	border-top: 1px solid var(--wb-border, rgba(255,255,255,0.07));
	flex-shrink: 0;
}
.sp-handle {
	height: 4px;
	background: var(--wb-border, rgba(255,255,255,0.05));
	cursor: ns-resize;
	flex-shrink: 0;
}
.sp-handle:hover { background: var(--wb-accent, #72d6d0); }
.sp-tabs {
	display: flex;
	align-items: center;
	padding: 0 8px;
	border-bottom: 1px solid var(--wb-border, rgba(255,255,255,0.07));
	flex-shrink: 0;
	height: 32px;
	gap: 0;
}
.sp-tab {
	padding: 0 12px;
	height: 32px;
	background: none;
	border: none;
	border-bottom: 2px solid transparent;
	color: var(--wb-text-sec, #787c99);
	font-size: 11px;
	cursor: pointer;
	transition: color 0.15s;
}
.sp-tab.active {
	color: var(--wb-text, #c0caf5);
	border-bottom-color: var(--wb-accent, #72d6d0);
}
.sp-tab:hover { color: var(--wb-text, #c0caf5); }
.sp-tabs-right {
	margin-left: auto;
	display: flex;
	align-items: center;
	gap: 6px;
}
.sp-total { font-size: 10px; color: var(--wb-text-sec, #787c99); }
.sp-icon {
	background: none;
	border: none;
	color: var(--wb-text-sec, #787c99);
	cursor: pointer;
	font-size: 13px;
	padding: 0 2px;
}
.sp-icon:hover { color: var(--wb-text, #c0caf5); }
.sp-body {
	flex: 1;
	overflow: auto;
	padding: 8px 12px;
}

/* MF tab */
.sp-mf-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
	gap: 8px;
}
.sp-card {
	background: var(--wb-bg, #0d1117);
	border: 1px solid var(--wb-border, rgba(255,255,255,0.07));
	border-radius: 8px;
	padding: 10px 12px;
}
.sp-card.pending { opacity: 0.5; }
.sp-comp-name { font-size: 12px; font-weight: 600; color: var(--wb-text, #c0caf5); font-family: ui-monospace, monospace; }
.sp-comp-path { font-size: 10px; color: var(--wb-text-sec, #787c99); margin-top: 2px; font-family: ui-monospace, monospace; word-break: break-all; }
.sp-comp-status { font-size: 10px; margin-top: 4px; color: var(--accent-yellow, #efc66b); }
.sp-comp-status.ready { color: var(--accent-green, #87cf8a); }
.sp-comp-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 6px; }
.sp-tag {
	font-size: 9px;
	padding: 1px 5px;
	background: rgba(114,214,208,0.1);
	border: 1px solid rgba(114,214,208,0.2);
	border-radius: 4px;
	color: var(--wb-accent, #72d6d0);
	font-family: ui-monospace, monospace;
}

/* DC tab */
.sp-dc-entries { display: flex; flex-direction: column; gap: 1px; }
.sp-dc-row {
	display: grid;
	grid-template-columns: 1fr 1fr 60px;
	gap: 8px;
	padding: 3px 6px;
	border-radius: 4px;
	font-size: 11px;
	font-family: ui-monospace, monospace;
}
.sp-dc-row:hover { background: rgba(255,255,255,0.03); }
.sp-dc-key { color: var(--wb-accent, #72d6d0); overflow: hidden; text-overflow: ellipsis; }
.sp-dc-val { color: var(--wb-text-sec, #787c99); overflow: hidden; text-overflow: ellipsis; }
.sp-dc-source { color: #3d4656; font-size: 9px; text-align: right; }
.sp-ev-block { margin-top: 12px; border-top: 1px solid var(--wb-border, rgba(255,255,255,0.07)); padding-top: 8px; }
.sp-section-label { display: block; font-size: 9px; font-weight: 700; letter-spacing: 0.1em; color: var(--wb-text-sec, #787c99); margin-bottom: 4px; }
.sp-ev-row { font-size: 10px; color: var(--wb-text-sec, #787c99); padding: 2px 4px; font-family: ui-monospace, monospace; }

/* Shared */
.sp-empty {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: 6px;
	min-height: 80px;
	color: var(--wb-text-sec, #787c99);
	font-size: 12px;
}
.sp-empty-icon { font-size: 24px; opacity: 0.4; }
.sp-empty-sub { font-size: 10px; color: #3d4656; }
.sp-error { font-size: 11px; color: var(--accent-red, #e16d75); padding: 4px 8px; background: rgba(225,109,117,0.1); border-radius: 4px; margin-bottom: 8px; }
.sp-loading { font-size: 11px; color: var(--wb-text-sec, #787c99); text-align: center; padding: 16px; }
</style>
