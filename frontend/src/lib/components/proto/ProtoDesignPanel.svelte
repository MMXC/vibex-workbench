<!-- ProtoDesignPanel.svelte
右侧 ProtoDesign 面板：Spec / DataCenter / EventCenter / Prototypes
ProtoDesign Spec = 四栏数据聚合视图
-->
<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { specSlotSessionStore } from '$lib/stores/spec-slot-session-store';
	import { dslCanvasStore } from '$lib/stores/dsl-canvas-store';

	type TabId = 'spec' | 'dc' | 'ec' | 'prototype';
	let activeTab = $state<TabId>('spec');

	const TABS: { id: TabId; label: string }[] = [
		{ id: 'spec', label: 'Spec' },
		{ id: 'dc', label: 'DataCenter' },
		{ id: 'ec', label: 'EventCenter' },
		{ id: 'prototype', label: 'Prototypes' },
	];

	// ── Spec Explorer ────────────────────────────────────────────────────────────
	let currentSpecPath = $derived($specExplorerStore.selectedSpecPath ?? '');
	let currentSpecName = $derived(
		currentSpecPath ? (currentSpecPath.split('/').pop()?.replace(/\.ya?ml$/, '') ?? '—') : '—'
	);
	let workspaceRoot = $derived($specExplorerStore.workspaceRoot ?? '—');

	// ── Session state ──────────────────────────────────────────────────────────
	let sessionState = $state({ activeKey: null as string | null, sessions: {} as Record<string, { spec?: { display?: { title?: string } }; slot?: { label?: string }; pptDemoPath?: string | null; messages?: unknown[] }> });
	$effect(() => {
		const unsub = specSlotSessionStore.subscribe(v => {
			sessionState = { activeKey: v.activeKey, sessions: v.sessions };
		});
		return unsub;
	});
	let activeSession = $derived(
		sessionState.activeKey ? sessionState.sessions[sessionState.activeKey] ?? null : null
	);
	let boundProtoPath = $derived(activeSession?.pptDemoPath ?? '');
	let boundProtoName = $derived(
		boundProtoPath ? (boundProtoPath.split('/').pop()?.replace(/\.html$/, '') ?? '—') : '—'
	);
	let sessionSpecTitle = $derived(activeSession?.spec?.display?.title ?? currentSpecName);
	let sessionSlotLabel = $derived(activeSession?.slot?.label ?? 'prototype');
	let sessionMsgCount = $derived(activeSession?.messages?.length ?? 0);

	// ── Spec tab: aggregated data ─────────────────────────────────────────────
	// DC top keys (for Spec tab overview)
	interface DCEntry { key: string; value: unknown; }
	let dcTopKeys = $state<DCEntry[]>([]);
	let dcTotal = $state(0);

	// EC recent events (for Spec tab overview)
	interface ECEvent { event: string; payload?: unknown; emitted_at?: string; }
	let ecRecent = $state<ECEvent[]>([]);

	async function loadDCTopKeys() {
		try {
			const r = await fetch('/api/specpilot/dc');
			if (r.ok) {
				const d = await r.json();
				const entries = Object.entries(d.data ?? {});
				dcTotal = entries.length;
				dcTopKeys = entries.slice(0, 5).map(([key, value]) => ({ key, value }));
			}
		} catch {}
	}

	async function loadECRecent() {
		try {
			const r = await fetch('/api/specpilot/ec/events?limit=5');
			if (r.ok) {
				const d = await r.json();
				ecRecent = (d.events ?? []).slice(0, 5);
			}
		} catch {}
	}

	// Reload on tab change
	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		activeTab;
		if (activeTab === 'spec') {
			void loadDCTopKeys();
			void loadECRecent();
		}
	});

	// Prototype tab
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
	let dcData = $state<DCEntry[]>([]);
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
		} catch (e: unknown) { dcError = String(e); }
		finally { dcLoading = false; }
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
		} catch {}
		editingKey = '';
		editingVal = '';
	}

	function cancelEdit() { editingKey = ''; editingVal = ''; }

	// EC tab
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

	<!-- Tab content rendered via snippet -->
	<div class="pdp-body">
		{#if activeTab === 'spec'}
			{@render specTab()}
		{:else if activeTab === 'dc'}
			{@render dcTab()}
		{:else if activeTab === 'ec'}
			{@render ecTab()}
		{:else if activeTab === 'prototype'}
			{@render protoTab()}
		{/if}
	</div>
</div>

{#snippet specTab()}
<div class="pdp-spec">
	<!-- Session overview header -->
	<div class="pdp-spec-header">
		<div class="pdp-spec-title">{sessionSpecTitle}</div>
		<div class="pdp-spec-tags">
			<span class="pdp-tag pdp-tag-slot">{sessionSlotLabel}</span>
			{#if activeSession}
				<span class="pdp-tag pdp-tag-msgs">{sessionMsgCount} 条</span>
			{:else}
				<span class="pdp-tag pdp-tag-idle">空闲</span>
			{/if}
		</div>
	</div>

	<!-- 1. Prototype bound -->
	<div class="pdp-spec-section">
		<div class="pdp-spec-section-label">◎ Prototype</div>
		{#if boundProtoPath}
			<div class="pdp-spec-row">
				<code class="pdp-spec-path">{boundProtoPath}</code>
			</div>
		{:else}
			<div class="pdp-spec-hint-row">左侧会话绑定原型文件后显示</div>
		{/if}
	</div>

	<!-- 2. DataCenter summary -->
	<div class="pdp-spec-section">
		<div class="pdp-spec-section-label">◈ DataCenter · {dcTotal} keys</div>
		{#if dcTopKeys.length > 0}
			{#each dcTopKeys as e}
				<div class="pdp-dc-mini-row">
					<code class="pdp-dc-key-sm">{e.key}</code>
					<code class="pdp-dc-val-sm">{fmtVal(e.value)}</code>
				</div>
			{/each}
			{#if dcTotal > 5}
				<button type="button" class="pdp-spec-link" onclick={() => (activeTab = 'dc')}>
					+ {dcTotal - 5} more →
				</button>
			{/if}
		{:else}
			<div class="pdp-spec-hint-row">DC 中暂无数据</div>
		{/if}
	</div>

	<!-- 3. EventCenter summary -->
	<div class="pdp-spec-section">
		<div class="pdp-spec-section-label">◈ EventCenter</div>
		{#if ecRecent.length > 0}
			{#each ecRecent as ev}
				<div class="pdp-ec-mini-row">
					<span class="pdp-ev-name">{ev.event}</span>
					{#if ev.emitted_at}
						<span class="pdp-ev-time">{fmtTime(ev.emitted_at)}</span>
					{/if}
				</div>
			{/each}
			<button type="button" class="pdp-spec-link" onclick={() => (activeTab = 'ec')}>
				查看全部 →
			</button>
		{:else}
			<div class="pdp-spec-hint-row">EC 中暂无事件</div>
		{/if}
	</div>

	<!-- 4. Spec Explorer path -->
	{#if currentSpecPath}
	<div class="pdp-spec-section">
		<div class="pdp-spec-section-label">◈ Spec 文件</div>
		<code class="pdp-spec-path pdp-spec-path-sm">{currentSpecPath}</code>
	</div>
	{/if}
</div>
{/snippet}

{#snippet dcTab()}
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
{/snippet}

{#snippet ecTab()}
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
{/snippet}

{#snippet protoTab()}
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
{/snippet}

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
	color: var(--wb-text-sec, #787c99);
	font-size: 11px;
	cursor: pointer;
	border-bottom: 2px solid transparent;
	transition: color 0.15s;
}
.pdp-tab:hover { color: var(--wb-text, #c0caf5); }
.pdp-tab.active {
	color: var(--wb-text, #c0caf5);
	border-bottom-color: var(--wb-accent, #72d6d0);
}
.pdp-body {
	flex: 1;
	overflow-y: auto;
	padding: 10px;
}
/* Spec tab */
.pdp-placeholder { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 120px; color: var(--wb-text-sec, #787c99); gap: 6px; }
.pdp-icon { font-size: 28px; }
.pdp-sub { font-size: 10px; color: var(--wb-text-sec, #787c99); }
.pdp-spec-card { background: var(--wb-bg, #0d0d0e); border: 1px solid var(--wb-border, rgba(255,255,255,0.07)); border-radius: 8px; padding: 12px 14px; }
.pdp-spec-name { font-size: 14px; font-weight: 700; color: var(--wb-text, #c0caf5); font-family: ui-monospace, monospace; }
.pdp-spec-path { font-size: 10px; color: var(--wb-text-sec, #787c99); margin-top: 6px; word-break: break-all; font-family: ui-monospace, monospace; }
.pdp-spec-hint { font-size: 10px; color: var(--wb-accent, #72d6d0); margin-top: 8px; font-family: ui-monospace, monospace; }
/* DC tab */
.pdp-dc { display: flex; flex-direction: column; height: 100%; }
.pdp-dc-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.pdp-label { font-size: 9px; font-weight: 700; letter-spacing: 0.1em; color: var(--wb-text-sec, #787c99); }
.pdp-dc-toolbar { display: flex; gap: 4px; margin-bottom: 8px; }
.pdp-input { background: var(--wb-bg, #0d0d0e); border: 1px solid var(--wb-border, rgba(255,255,255,0.07)); border-radius: 4px; padding: 4px 8px; color: var(--wb-text, #c0caf5); font-size: 11px; flex: 1; min-width: 0; }
.pdp-dc-val-input { flex: 1; }
.pdp-btn { background: var(--wb-accent, #72d6d0); color: #000; border: none; border-radius: 4px; padding: 4px 10px; cursor: pointer; font-size: 12px; }
.pdp-btn-sm { background: none; border: none; color: var(--wb-text-sec, #787c99); cursor: pointer; font-size: 12px; padding: 0 4px; }
.pdp-btn-sm:hover { color: var(--wb-text, #c0caf5); }
.pdp-dc-list { flex: 1; overflow-y: auto; }
.pdp-dc-row { display: flex; gap: 4px; align-items: center; padding: 4px 2px; border-radius: 4px; }
.pdp-dc-row:hover { background: rgba(255,255,255,0.03); }
.pdp-dc-key { color: var(--wb-accent, #72d6d0); font-size: 11px; min-width: 80px; flex-shrink: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pdp-dc-val { color: var(--wb-text-sec, #787c99); font-size: 11px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pdp-icon-btn { background: none; border: none; color: var(--wb-text-sec, #787c99); cursor: pointer; font-size: 11px; padding: 2px 4px; flex-shrink: 0; }
.pdp-icon-btn:hover { color: var(--wb-text, #c0caf5); }
.pdp-error { background: rgba(239,68,68,0.1); border: 1px solid rgba(239,68,68,0.3); border-radius: 4px; padding: 6px 8px; color: #ef4444; font-size: 11px; margin-bottom: 8px; }
.pdp-loading, .pdp-empty { color: var(--wb-text-sec, #787c99); font-size: 11px; text-align: center; padding: 20px 0; }
/* EC tab */
.pdp-ec-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.pdp-ev-row { display: flex; gap: 6px; align-items: center; padding: 2px 4px; border-radius: 4px; font-size: 10px; font-family: ui-monospace, monospace; }
.pdp-ev-row:hover { background: rgba(255,255,255,0.03); }
.pdp-ev-time { color: #3d4656; flex-shrink: 0; }
.pdp-ev-name { color: var(--wb-text, #c0caf5); flex-shrink: 0; }
.pdp-ev-payload { color: var(--wb-text-sec, #787c99); overflow: hidden; text-overflow: ellipsis; }
/* Prototype tab */
.pdp-prototypes-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.pdp-prototype-row { display: flex; gap: 8px; align-items: center; padding: 5px 4px; border-radius: 4px; font-size: 11px; }
.pdp-prototype-row:hover { background: rgba(255,255,255,0.03); }
.pdp-proto-id { color: var(--wb-accent, #72d6d0); flex: 1; font-family: ui-monospace, monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pdp-proto-size { color: var(--wb-text-sec, #787c99); font-size: 10px; flex-shrink: 0; }
.pdp-proto-time { color: #3d4656; font-size: 10px; flex-shrink: 0; }
/* ProtoDesign Spec tab */
.pdp-spec { display: flex; flex-direction: column; gap: 0; padding: 8px; }
.pdp-spec-header { padding: 0 0 8px 0; border-bottom: 1px solid rgba(255,255,255,0.06); margin-bottom: 4px; }
.pdp-spec-title { font-size: 13px; font-weight: 700; color: var(--wb-text, #c0caf5); margin-bottom: 4px; }
.pdp-spec-tags { display: flex; gap: 4px; flex-wrap: wrap; }
.pdp-tag { font-size: 10px; border-radius: 3px; padding: 1px 5px; }
.pdp-tag-slot { background: rgba(124,58,237,0.2); color: #a78bfa; }
.pdp-tag-msgs { background: rgba(114,214,208,0.12); color: var(--wb-accent, #72d6d0); }
.pdp-tag-idle { background: rgba(255,255,255,0.05); color: var(--wb-text-sec, #787c99); }
.pdp-spec-section { padding: 6px 0; border-bottom: 1px solid rgba(255,255,255,0.04); }
.pdp-spec-section:last-child { border-bottom: none; }
.pdp-spec-section-label { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; color: var(--wb-text-sec, #787c99); text-transform: uppercase; margin-bottom: 4px; }
.pdp-spec-row { display: flex; align-items: center; gap: 4px; }
.pdp-spec-path { font-size: 10px; color: var(--wb-accent, #72d6d0); font-family: ui-monospace, monospace; word-break: break-all; }
.pdp-spec-path-sm { font-size: 9px; color: var(--wb-text-sec, #787c99); }
.pdp-spec-hint-row { font-size: 10px; color: var(--wb-text-sec, #787c99); font-style: italic; }
.pdp-spec-link { background: none; border: none; color: var(--wb-accent, #72d6d0); font-size: 10px; cursor: pointer; padding: 0; text-align: left; }
.pdp-spec-link:hover { text-decoration: underline; }
.pdp-dc-mini-row { display: flex; gap: 6px; align-items: baseline; padding: 1px 0; font-size: 10px; }
.pdp-dc-key-sm { color: var(--wb-text, #c0caf5); font-family: ui-monospace, monospace; flex-shrink: 0; max-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pdp-dc-val-sm { color: var(--wb-text-sec, #787c99); font-family: ui-monospace, monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pdp-ec-mini-row { display: flex; gap: 6px; align-items: center; padding: 1px 0; font-size: 10px; }
.pdp-ev-name { color: var(--wb-accent, #72d6d0); font-family: ui-monospace, monospace; }
.pdp-ev-time { color: #3d4656; font-size: 9px; margin-left: auto; }
</style>
