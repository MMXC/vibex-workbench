<!-- AgentHubPanel.svelte
左侧 Agent Hub 面板。
无会话时：显示会话列表（可新建/删除）。
有会话时：显示 SpecSlotChatPane + 顶部操作栏（Abort/Reset/Compact/Resume）。
原型文件绑定会话：顶部原型路径栏（点击选择 .specpilot/prototypes/ 下的 .html 文件）。
-->
<script lang="ts">
	import { tick } from 'svelte';
	import {
		activeSpecSlotSession,
		specSlotSessionStore,
		type SpecSlotSession,
	} from '$lib/stores/spec-slot-session-store';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import SpecSlotChatPane from '$lib/components/workbench/SpecSlotChatPane.svelte';

	// ── Store (Svelte 5: wrap store in $state for $derived compat) ───────────────
	let store = $state({ activeKey: null as string | null, drawerOpen: false, sessions: {} as Record<string, SpecSlotSession> });
	$effect(() => {
		const unsub = specSlotSessionStore.subscribe(v => {
			store = { activeKey: v.activeKey, drawerOpen: v.drawerOpen, sessions: v.sessions };
		});
		return unsub;
	});

	const activeSession = $derived(store.activeKey ? store.sessions[store.activeKey] ?? null : null);
	const sessionList = $derived(
		Object.values(store.sessions).sort(
			(a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
		)
	);

	// ── Spec Explorer ────────────────────────────────────────────────────────────
	let selectedSpecPath = $derived($specExplorerStore.selectedSpecPath ?? '');
	let selectedSpecMeta = $derived.by(() => {
		const path = selectedSpecPath;
		if (!path) return null;
		const specs = $specExplorerStore.specs ?? [];
		return specs.find(s => s.path === path) ?? null;
	});

	// ── Prototype file picker ───────────────────────────────────────────────────
	let protoList = $state<{ specId: string; path: string; size: number }[]>([]);
	let protoDropdownOpen = $state(false);
	let protoLoading = $state(false);

	async function loadPrototypes() {
		protoLoading = true;
		try {
			const r = await fetch('/api/specpilot/prototypes');
			if (r.ok) {
				const d = await r.json();
				protoList = d.prototypes ?? [];
			}
		} catch {}
		finally { protoLoading = false; }
	}

	// ── New session ────────────────────────────────────────────────────────────
	async function newSession() {
		if (!selectedSpecMeta || !selectedSpecPath) return;
		try {
			const r = await fetch(
				`/api/workspace/specs/read?path=${encodeURIComponent(selectedSpecPath)}` +
				`&workspaceRoot=${encodeURIComponent($specExplorerStore.workspaceRoot ?? '')}`
			);
			if (!r.ok) return;
			const j = await r.json();
			const raw = String(j.content ?? '');
			// Parse slots from spec
			const slots = selectedSpecMeta.slots?.all ?? [];
			if (!slots.length) return;
			const slot = slots.find(s => s.id === 'prototype') ?? slots[0];
			specSlotSessionStore.open({
				spec: selectedSpecMeta,
				slot,
				content: raw,
			});
			void loadPrototypes();
		} catch {}
	}

	// ── Bind prototype path to active session ─────────────────────────────────
	async function bindPrototype(specId: string) {
		if (!activeSession) return;
		const path = `.specpilot/prototypes/${specId}.html`;
		specSlotSessionStore.setPrototypePath(path);
		protoDropdownOpen = false;
	}

	// ── Chat actions ────────────────────────────────────────────────────────────
	function doAbort() { specSlotSessionStore.abortActiveAgent(); }
	function doReset() { specSlotSessionStore.resetActive(); }
	function doCompact() { specSlotSessionStore.compactActive(); }
	function doResume() { specSlotSessionStore.resumeInterruptedAgentTurn(); }
</script>

<div class="ahp">
	{#if activeSession}
		<!-- Active session header + actions -->
		<div class="ahp-header">
			<!-- Spec badge -->
			<div class="ahp-spec-badge" title={activeSession.spec.path}>
				<span class="ahp-spec-name">{activeSession.spec.display.title}</span>
				<span class="ahp-slot-tag">{activeSession.slot.label}</span>
			</div>

			<!-- Prototype path binder -->
			<div class="ahp-proto-wrap">
				<button
					type="button"
					class="ahp-proto-btn"
					onclick={() => { protoDropdownOpen = !protoDropdownOpen; if (!protoList.length) void loadPrototypes(); }}
					title="绑定原型 HTML 到当前会话"
				>
					<span class="ahp-proto-icon">◎</span>
					<span class="ahp-proto-path">
						{activeSession.pptDemoPath
							? activeSession.pptDemoPath.split('/').pop()
							: '绑定原型'}
					</span>
				</button>
				{#if protoDropdownOpen}
					<div class="ahp-proto-dropdown">
						{#if protoLoading}
							<div class="ahp-proto-loading">加载中…</div>
						{:else if protoList.length === 0}
							<div class="ahp-proto-empty">暂无原型文件</div>
						{:else}
							{#each protoList as p}
								<button
									type="button"
									class="ahp-proto-item"
									class:active={activeSession.pptDemoPath?.includes(p.specId)}
									onclick={() => void bindPrototype(p.specId)}
								>
									<span class="ahp-proto-id">{p.specId}</span>
									<span class="ahp-proto-size">{(p.size / 1024).toFixed(1)}KB</span>
								</button>
							{/each}
						{/if}
					</div>
				{/if}
			</div>

			<!-- Action buttons -->
			<div class="ahp-actions">
				{#if activeSession.status === 'running'}
					<button type="button" class="ahp-action-btn danger" onclick={doAbort} title="中止">■ 中止</button>
				{:else if activeSession.pendingResumePrompt}
					<button type="button" class="ahp-action-btn primary" onclick={doResume} title="继续">▶ 继续</button>
				{/if}
				<button type="button" class="ahp-action-btn" onclick={doCompact} title="压缩历史">⊡ 压缩</button>
				<button type="button" class="ahp-action-btn danger" onclick={doReset} title="重置会话">✕ 重置</button>
			</div>
		</div>

		<!-- Chat pane -->
		<div class="ahp-body">
			<SpecSlotChatPane session={activeSession} />
		</div>

	{:else}
		<!-- Empty state: session list -->
		<div class="ahp-empty-state">
			<div class="ahp-empty-title">Agent Hub</div>

			<!-- New session -->
			{#if selectedSpecPath && selectedSpecMeta}
				<button type="button" class="ahp-new-btn" onclick={newSession}>
					<span class="ahp-new-icon">＋</span>
					<div class="ahp-new-info">
						<div class="ahp-new-name">{selectedSpecMeta.display.title}</div>
						<div class="ahp-new-slot">新建会话 · {selectedSpecMeta.slots?.all[0]?.label ?? 'prototype'}</div>
					</div>
				</button>
			{:else}
				<div class="ahp-no-spec-hint">在左侧 Spec Explorer 中选择一个文件，即可新建会话</div>
			{/if}

			<!-- Session list -->
			{#if sessionList.length > 0}
				<div class="ahp-session-list-header">最近的会话</div>
				<div class="ahp-session-list">
					{#each sessionList as sess}
						<button
							type="button"
							class="ahp-session-item"
							onclick={() => specSlotSessionStore.open({ spec: sess.spec, slot: sess.slot, content: sess.content })}
						>
							<div class="ahp-sess-title">{sess.spec.display.title}</div>
							<div class="ahp-sess-meta">
								<span class="ahp-sess-slot">{sess.slot.label}</span>
								<span class="ahp-sess-msg">{sess.messages.length} 条</span>
								<span class="ahp-sess-time">{fmtTime(sess.updatedAt)}</span>
							</div>
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Click outside to close dropdown -->
<svelte:window onclick={(e) => {
	const t = e.target as HTMLElement;
	if (!t.closest('.ahp-proto-wrap')) protoDropdownOpen = false;
}} />

<script lang="ts" module>
	function fmtTime(iso: string): string {
		try {
			const d = new Date(iso);
			const now = new Date();
			const diff = now.getTime() - d.getTime();
			if (diff < 60000) return '刚刚';
			if (diff < 3600000) return `${Math.floor(diff / 60000)}m前`;
			if (diff < 86400000) return `${Math.floor(diff / 3600000)}h前`;
			return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' });
		} catch { return iso; }
	}
</script>

<style>
.ahp {
	display: flex;
	flex-direction: column;
	height: 100%;
	background: #0b0d12;
	overflow: hidden;
	font-size: 12px;
}

/* ── Header (active session) ── */
.ahp-header {
	display: flex;
	align-items: center;
	gap: 8px;
	padding: 6px 10px;
	border-bottom: 1px solid rgba(255,255,255,0.06);
	flex-shrink: 0;
	background: #0e1118;
}

.ahp-spec-badge {
	display: flex;
	align-items: center;
	gap: 4px;
	min-width: 0;
	flex: 1;
}
.ahp-spec-name {
	font-weight: 600;
	color: var(--wb-text, #c0caf5);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	max-width: 120px;
}
.ahp-slot-tag {
	background: rgba(124,58,237,0.25);
	border: 1px solid rgba(124,58,237,0.4);
	color: #a78bfa;
	border-radius: 4px;
	padding: 1px 6px;
	font-size: 10px;
	white-space: nowrap;
	flex-shrink: 0;
}

/* ── Prototype binder ── */
.ahp-proto-wrap { position: relative; flex-shrink: 0; }
.ahp-proto-btn {
	display: flex;
	align-items: center;
	gap: 3px;
	background: rgba(114,214,208,0.08);
	border: 1px solid rgba(114,214,208,0.2);
	border-radius: 4px;
	color: var(--wb-accent, #72d6d0);
	font-size: 10px;
	cursor: pointer;
	padding: 2px 7px;
	white-space: nowrap;
}
.ahp-proto-btn:hover { background: rgba(114,214,208,0.15); }
.ahp-proto-icon { font-size: 11px; }
.ahp-proto-path { max-width: 100px; overflow: hidden; text-overflow: ellipsis; }

.ahp-proto-dropdown {
	position: absolute;
	top: calc(100% + 4px);
	left: 0;
	z-index: 100;
	min-width: 180px;
	background: #1a1d27;
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 6px;
	box-shadow: 0 8px 24px rgba(0,0,0,0.5);
	overflow: hidden;
}
.ahp-proto-loading, .ahp-proto-empty {
	padding: 8px 12px;
	color: var(--wb-text-sec, #787c99);
	font-size: 11px;
	text-align: center;
}
.ahp-proto-item {
	display: flex;
	align-items: center;
	justify-content: space-between;
	width: 100%;
	padding: 6px 10px;
	background: none;
	border: none;
	border-bottom: 1px solid rgba(255,255,255,0.04);
	color: var(--wb-text, #c0caf5);
	font-size: 11px;
	cursor: pointer;
	text-align: left;
}
.ahp-proto-item:last-child { border-bottom: none; }
.ahp-proto-item:hover { background: rgba(255,255,255,0.05); }
.ahp-proto-item.active { color: var(--wb-accent, #72d6d0); }
.ahp-proto-id { font-family: ui-monospace, monospace; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ahp-proto-size { color: var(--wb-text-sec, #787c99); font-size: 10px; flex-shrink: 0; }

/* ── Action buttons ── */
.ahp-actions { display: flex; gap: 4px; flex-shrink: 0; }
.ahp-action-btn {
	background: rgba(255,255,255,0.05);
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 4px;
	color: var(--wb-text-sec, #787c99);
	font-size: 10px;
	cursor: pointer;
	padding: 2px 8px;
	white-space: nowrap;
}
.ahp-action-btn:hover { background: rgba(255,255,255,0.1); color: var(--wb-text, #c0caf5); }
.ahp-action-btn.primary { color: var(--wb-accent, #72d6d0); border-color: rgba(114,214,208,0.3); }
.ahp-action-btn.danger:hover { color: #e16d75; border-color: rgba(225,109,117,0.4); }

/* ── Chat body ── */
.ahp-body { flex: 1; overflow: hidden; position: relative; }

/* ── Empty state ── */
.ahp-empty-state {
	display: flex;
	flex-direction: column;
	align-items: stretch;
	height: 100%;
	overflow-y: auto;
	padding: 16px 12px;
	gap: 8px;
}
.ahp-empty-title {
	font-size: 11px;
	font-weight: 700;
	letter-spacing: 0.08em;
	color: var(--wb-text-sec, #787c99);
	text-transform: uppercase;
	margin-bottom: 4px;
	padding-bottom: 8px;
	border-bottom: 1px solid rgba(255,255,255,0.05);
}

.ahp-new-btn {
	display: flex;
	align-items: center;
	gap: 10px;
	background: rgba(124,58,237,0.12);
	border: 1px solid rgba(124,58,237,0.3);
	border-radius: 8px;
	padding: 10px 12px;
	cursor: pointer;
	text-align: left;
	width: 100%;
	transition: background 0.15s;
}
.ahp-new-btn:hover { background: rgba(124,58,237,0.2); }
.ahp-new-icon {
	font-size: 20px;
	color: #a78bfa;
	flex-shrink: 0;
}
.ahp-new-name { font-size: 13px; font-weight: 600; color: var(--wb-text, #c0caf5); }
.ahp-new-slot { font-size: 10px; color: var(--wb-text-sec, #787c99); margin-top: 2px; }

.ahp-no-spec-hint {
	font-size: 11px;
	color: var(--wb-text-sec, #787c99);
	text-align: center;
	padding: 20px 12px;
	line-height: 1.6;
}

.ahp-session-list-header {
	font-size: 10px;
	font-weight: 700;
	letter-spacing: 0.08em;
	color: #3d4656;
	text-transform: uppercase;
	margin-top: 4px;
}

.ahp-session-list { display: flex; flex-direction: column; gap: 2px; }
.ahp-session-item {
	display: flex;
	flex-direction: column;
	gap: 3px;
	padding: 8px 10px;
	background: rgba(255,255,255,0.02);
	border: 1px solid rgba(255,255,255,0.05);
	border-radius: 6px;
	cursor: pointer;
	text-align: left;
	width: 100%;
	transition: background 0.12s;
}
.ahp-session-item:hover { background: rgba(255,255,255,0.06); border-color: rgba(255,255,255,0.1); }
.ahp-sess-title { font-size: 12px; font-weight: 600; color: var(--wb-text, #c0caf5); }
.ahp-sess-meta { display: flex; gap: 6px; align-items: center; }
.ahp-sess-slot {
	font-size: 10px;
	background: rgba(124,58,237,0.2);
	color: #a78bfa;
	border-radius: 3px;
	padding: 0 4px;
}
.ahp-sess-msg { font-size: 10px; color: var(--wb-text-sec, #787c99); }
.ahp-sess-time { font-size: 10px; color: #3d4656; margin-left: auto; }
</style>
