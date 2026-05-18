<!-- ProtoDesignPanel.svelte
右侧 ProtoDesign 面板：5 个 Tab — Spec / DataCenter / EventCenter / MF / Adapter
-->
<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { dslCanvasStore } from '$lib/stores/dsl-canvas-store';
	type TabId = 'spec' | 'dc' | 'ec' | 'prototype';
	let activeTab = $state<TabId>('spec');

	const TABS: { id: TabId; label: string }[] = [
		{ id: 'spec', label: 'Spec' },
		{ id: 'dc', label: 'DataCenter' },
		{ id: 'ec', label: 'EventCenter' },
		{ id: 'prototype', label: 'Prototypes' },
	];

	// Spec tab: current spec info
	let currentSpecPath = $derived($specExplorerStore.selectedSpecPath ?? '');
	let currentSpecName = $derived(
		currentSpecPath ? (currentSpecPath.split('/').pop()?.replace(/\.ya?ml$/, '') ?? '—') : '—'
	);
	let workspaceRoot = $derived($specExplorerStore.workspaceRoot ?? '—');

	// Prototype tab: list from Go backend
	interface ProtoEntry { specId: string; path: string; size: number; updatedAt: string; }
	let prototypes = $state<ProtoEntry[]>([]);
	let protoLoading = $state(false);

	async function loadPrototypes() {
		protoLoading = true;
		try {
			const r = await fetch('/api/specpilot/prototypes');
			const d = await r.json();
			prototypes = d.prototypes ?? [];
		} catch { prototypes = []; }
		finally { protoLoading = false; }
	}

	// DC tab
	interface DCEntry { key: string; value: unknown; }
	let dcData = $state<{ key: string; value: unknown }[]>([]);
	let dcLoading = $state(false);
	let dcError = $state('');
	let editingKey = $state('');
	let editingVal = $state('');
	let newKey = $state('');
	let newVal = $state('');

	async function loadDC() {
		dcLoading = true;
		dcError = '';
		try {
			const res = await fetch('/api/specpilot/dc');
			const r = await res.json();
			dcData = Object.entries(r.data ?? {}).map(([key, value]) => ({ key, value }));
		} catch (e: unknown) {
			dcError = String(e);
		} finally {
			dcLoading = false;
		}
	}

	async function setDCKey(key: string, value: unknown) {
		try {
			await fetch('/api/specpilot/dc/set', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ key, value }),
			});
			await loadDC();
		} catch {}
	}

	async function addDCKey() {
		if (!newKey.trim()) return;
		await setDCKey(newKey.trim(), newVal);
		newKey = '';
		newVal = '';
	}

	function startEdit(key: string, value: unknown) {
		editingKey = key;
		editingVal = JSON.stringify(value);
	}
	async function commitEdit() {
		if (!editingKey) return;
		try {
			const v = JSON.parse(editingVal);
			await setDCKey(editingKey, v);
		} catch { /* ignore invalid JSON */ }
		editingKey = '';
		editingVal = '';
	}
	function cancelEdit() { editingKey = ''; editingVal = ''; }

	// ── EC tab ─────────────────────────────────────────────
	interface ECEvent { event: string; payload: unknown; emitted_at: string; }
	let ecHistory = $state<ECEvent[]>([]);

	async function loadEC() {
		try {
			const res = await fetch('/api/specpilot/ec/history?limit=20');
			const r = await res.json();
			ecHistory = Array.isArray(r) ? r : [];
		} catch {}
	}

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		activeTab;
		if (activeTab === 'dc') loadDC();
		else if (activeTab === 'ec') loadEC();
		else if (activeTab === 'prototype') loadPrototypes();
	});

	function fmtVal(v: unknown): string {
		if (v == null) return '--';
		if (typeof v === 'object') return JSON.stringify(v)?.slice(0, 40) ?? '--';
		return String(v);
	}
	function fmtTime(iso: string): string {
		try { return new Date(iso).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }); }
		catch { return iso; }
	}
</script>

<div class="pdp">
	<!-- Tab bar -->
	<div class="pdp-tabs">
		{#each TABS as tab}
		<button
			type="button"
			class="pdp-tab"
			class:active={activeTab === tab.id}
			onclick={() => (activeTab = tab.id)}
		>{tab.label}</button>
		{/each}
	</div>

	<!-- Tab content -->
	<div class="pdp-body">

		<!-- Spec tab -->
		{:else if activeTab === 'spec'}
		<div class="pdp-spec">
			{#if currentSpecPath}
				<div class="pdp-spec-card">
					<div class="pdp-spec-name">{currentSpecName}</div>
					<div class="pdp-spec-path">{currentSpecPath}</div>
					<div class="pdp-spec-hint">原型路径：.specpilot/prototypes/{currentSpecName}.html</div>
				</div>
			{:else}
			<div class="pdp-placeholder">
				<div class="pdp-icon">📋</div>
				<div>未选中 Spec</div>
				<div class="pdp-sub">在 Spec Explorer 中选择一个文件</div>
			</div>
			{/if}
		</div>

		<!-- DataCenter tab -->
		{:else if activeTab === 'dc'}
		<div class="pdp-dc">
			<div class="pdp-dc-header">
				<span class="pdp-label">DATACENTER</span>
				<button type="button" class="pdp-btn-sm" onclick={loadDC}>↻</button>
			</div>
			{#if dcError}
			<div class="pdp-error">{dcError}</div>
			{/if}
			<div class="pdp-dc-toolbar">
				<input
					class="pdp-input"
					placeholder="key"
					bind:value={newKey}
					onkeydown={(e) => e.key === 'Enter' && addDCKey()}
				/>
				<input
					class="pdp-input"
					placeholder="value (JSON)"
					bind:value={newVal}
					onkeydown={(e) => e.key === 'Enter' && addDCKey()}
				/>
				<button type="button" class="pdp-btn" onclick={addDCKey}>+</button>
			</div>
			<div class="pdp-dc-list">
				{#if dcLoading}
				<div class="pdp-loading">加载中…</div>
				{:else if dcData.length === 0}
				<div class="pdp-empty">DC 中暂无数据</div>
				{:else}
				{#each dcData as entry}
				<div class="pdp-dc-row">
					{#if editingKey === entry.key}
					<code class="pdp-dc-key">{entry.key}</code>
					<input
						class="pdp-input pdp-dc-val-input"
						bind:value={editingVal}
						onkeydown={(e) => e.key === 'Enter' && commitEdit()}
					/>
					<button type="button" class="pdp-icon-btn" onclick={commitEdit}>✓</button>
					<button type="button" class="pdp-icon-btn" onclick={cancelEdit}>✗</button>
					{:else}
					<code class="pdp-dc-key">{entry.key}</code>
					<code class="pdp-dc-val">{fmtVal(entry.value)}</code>
					<button type="button" class="pdp-icon-btn" onclick={() => startEdit(entry.key, entry.value)} title="编辑">✎</button>
					{/if}
				</div>
				{/each}
				{/if}
			</div>
		</div>

		<!-- EventCenter tab -->
		{:else if activeTab === 'ec'}
		<div class="pdp-ec">
			<div class="pdp-ec-header">
				<span class="pdp-label">EVENT HISTORY</span>
				<button type="button" class="pdp-btn-sm" onclick={loadEC}>↻</button>
			</div>
			{#if ecHistory.length === 0}
			<div class="pdp-empty">暂无事件记录</div>
			{:else}
			{#each ecHistory.slice().reverse() as ev}
			<div class="pdp-ev-row">
				<span class="pdp-ev-time">{fmtTime(ev.emitted_at)}</span>
				<span class="pdp-ev-name">{ev.event}</span>
				<code class="pdp-ev-payload">{JSON.stringify(ev.payload)?.slice(0, 30)}</code>
			</div>
			{/each}
			{/if}
		</div>

		<!-- Prototype tab -->
		{:else if activeTab === 'prototype'}
		<div class="pdp-prototypes">
			<div class="pdp-prototypes-header">
				<span class="pdp-label">PROTOTYPES · {workspaceRoot}/.specpilot/prototypes/</span>
				<button type="button" class="pdp-btn-sm" onclick={loadPrototypes}>↻</button>
			</div>
			{#if protoLoading}
				<div class="pdp-loading">加载中…</div>
			{:else if prototypes.length === 0}
				<div class="pdp-empty">暂无原型文件<br/><span class="pdp-sub">Agent 生成 HTML 原型后自动出现</span></div>
			{:else}
				{#each prototypes as p}
				<div class="pdp-prototype-row">
					<span class="pdp-proto-id">{p.specId}</span>
					<span class="pdp-proto-size">{(p.size / 1024).toFixed(1)}KB</span>
					<span class="pdp-proto-time">{fmtTime(p.updatedAt)}</span>
				</div>
				{/each}
			{/if}
		</div>
		{/if}

	</div>
</div>

<style>
.pdp {
	display: flex;
	flex-direction: column;
	height: 100%;
	background: var(--wb-bg-panel, #10131a);
}
.pdp-tabs {
	display: flex;
	align-items: center;
	padding: 0 8px;
	border-bottom: 1px solid var(--wb-border, rgba(255,255,255,0.07));
	flex-shrink: 0;
	height: 36px;
	gap: 0;
}
.pdp-tab {
	padding: 0 10px;
	height: 36px;
	background: none;
	border: none;
	border-bottom: 2px solid transparent;
	color: var(--wb-text-sec, #787c99);
	font-size: 11px;
	cursor: pointer;
	transition: color 0.15s;
	white-space: nowrap;
}
.pdp-tab.active {
	color: var(--wb-text, #c0caf5);
	border-bottom-color: #7c3aed;
}
.pdp-tab:hover:not(.active) { color: var(--wb-text, #c0caf5); }
.pdp-body {
	flex: 1;
	overflow: auto;
	padding: 8px 10px;
}

/* Shared */
.pdp-placeholder {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	height: 120px;
	gap: 6px;
	color: var(--wb-text-sec, #787c99);
	font-size: 12px;
}
.pdp-icon { font-size: 24px; opacity: 0.3; }
.pdp-sub { font-size: 10px; color: #3d4656; text-align: center; }
.pdp-error { font-size: 11px; color: var(--accent-red, #e16d75); padding: 4px 8px; background: rgba(225,109,117,0.1); border-radius: 4px; margin-bottom: 8px; }
.pdp-loading, .pdp-empty { font-size: 11px; color: var(--wb-text-sec, #787c99); text-align: center; padding: 16px 0; }

/* DC tab */
.pdp-dc-toolbar { display: flex; gap: 4px; margin-bottom: 8px; }
.pdp-input {
	flex: 1;
	background: var(--wb-bg, #0d0d0e);
	border: 1px solid var(--wb-border, rgba(255,255,255,0.07));
	border-radius: 4px;
	color: var(--wb-text, #c0caf5);
	font-size: 11px;
	padding: 3px 6px;
	font-family: ui-monospace, monospace;
	outline: none;
}
.pdp-input:focus { border-color: var(--wb-accent, #72d6d0); }
.pdp-btn {
	background: rgba(114,214,208,0.15);
	border: 1px solid rgba(114,214,208,0.3);
	border-radius: 4px;
	color: var(--wb-accent, #72d6d0);
	font-size: 13px;
	cursor: pointer;
	padding: 2px 8px;
}
.pdp-btn:hover { background: rgba(114,214,208,0.25); }
.pdp-btn-sm {
	background: none;
	border: 1px solid var(--wb-border, rgba(255,255,255,0.07));
	border-radius: 4px;
	color: var(--wb-text-sec, #787c99);
	font-size: 11px;
	cursor: pointer;
	padding: 1px 6px;
}
.pdp-btn-sm:hover { color: var(--wb-text, #c0caf5); }
.pdp-dc-list { display: flex; flex-direction: column; gap: 1px; }
.pdp-dc-row {
	display: flex;
	align-items: center;
	gap: 4px;
	padding: 3px 4px;
	border-radius: 4px;
	font-size: 11px;
	font-family: ui-monospace, monospace;
}
.pdp-dc-row:hover { background: rgba(255,255,255,0.03); }
.pdp-dc-key { color: var(--wb-accent, #72d6d0); flex-shrink: 0; max-width: 120px; overflow: hidden; text-overflow: ellipsis; }
.pdp-dc-val { color: var(--wb-text-sec, #787c99); flex: 1; overflow: hidden; text-overflow: ellipsis; }
.pdp-dc-val-input { flex: 1; min-width: 0; }
.pdp-icon-btn {
	background: none;
	border: none;
	color: var(--wb-text-sec, #787c99);
	font-size: 11px;
	cursor: pointer;
	padding: 0 2px;
	flex-shrink: 0;
}
.pdp-icon-btn:hover { color: var(--wb-text, #c0caf5); }

/* EC tab */
.pdp-ec-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.pdp-label { font-size: 9px; font-weight: 700; letter-spacing: 0.1em; color: var(--wb-text-sec, #787c99); }
.pdp-ev-row { display: flex; gap: 6px; align-items: center; padding: 2px 4px; border-radius: 4px; font-size: 10px; font-family: ui-monospace, monospace; }
.pdp-ev-row:hover { background: rgba(255,255,255,0.03); }
.pdp-ev-time { color: #3d4656; flex-shrink: 0; }
.pdp-ev-name { color: var(--wb-text, #c0caf5); flex-shrink: 0; }
.pdp-ev-payload { color: var(--wb-text-sec, #787c99); overflow: hidden; text-overflow: ellipsis; }

/* MF tab */
.pdp-mf-card { background: var(--wb-bg, #0d0d0e); border: 1px solid var(--wb-border, rgba(255,255,255,0.07)); border-radius: 8px; padding: 8px 10px; }
.pdp-mf-name { font-size: 12px; font-weight: 600; color: var(--wb-text, #c0caf5); font-family: ui-monospace, monospace; }
.pdp-mf-path { font-size: 10px; color: var(--wb-text-sec, #787c99); margin-top: 2px; font-family: ui-monospace, monospace; word-break: break-all; }
.pdp-mf-status { font-size: 10px; margin-top: 4px; color: var(--accent-yellow, #efc66b); }
.pdp-mf-status.ready { color: var(--accent-green, #87cf8a); }

/* Spec tab */
.pdp-spec-card { background: var(--wb-bg, #0d0d0e); border: 1px solid var(--wb-border, rgba(255,255,255,0.07)); border-radius: 8px; padding: 12px 14px; }
.pdp-spec-name { font-size: 14px; font-weight: 700; color: var(--wb-text, #c0caf5); font-family: ui-monospace, monospace; }
.pdp-spec-path { font-size: 10px; color: var(--wb-text-sec, #787c99); margin-top: 6px; word-break: break-all; font-family: ui-monospace, monospace; }
.pdp-spec-hint { font-size: 10px; color: var(--wb-accent, #72d6d0); margin-top: 8px; font-family: ui-monospace, monospace; }

/* Prototype tab */
.pdp-prototypes-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.pdp-prototype-row { display: flex; gap: 8px; align-items: center; padding: 5px 4px; border-radius: 4px; font-size: 11px; }
.pdp-prototype-row:hover { background: rgba(255,255,255,0.03); }
.pdp-proto-id { color: var(--wb-accent, #72d6d0); flex: 1; font-family: ui-monospace, monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pdp-proto-size { color: var(--wb-text-sec, #787c99); font-size: 10px; flex-shrink: 0; }
.pdp-proto-time { color: #3d4656; font-size: 10px; flex-shrink: 0; }
.pdp-dc-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }

