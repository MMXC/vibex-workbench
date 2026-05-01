<!-- 选中 spec 后主区：默认图谱，可切 Spec 文本（对齐 R2 center tabs） -->
<script lang="ts">
	import { parse as parseYaml } from 'yaml';
	import GoalSpecCanvas from '$lib/components/workbench/GoalSpecCanvas.svelte';
	import GenericSpecGraph from '$lib/components/workbench/GenericSpecGraph.svelte';
	import MonacoEditor from '$lib/components/workbench/MonacoEditor.svelte';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import {
		type ConventionPayload,
		extractSpecMeta,
		inferParentSpecPath,
		inferSiblingFeaturePath,
		inferSpecTypeId,
		normalizeSpecPath,
		specTypeLabel,
	} from '$lib/workbench/spec-convention';
	import { wailsReadSpecFile, wailsWriteSpecFile } from '$lib/wails-filesystem';
	import { specAgentContextStore } from '$lib/stores/spec-agent-context-store';
	import { extractSpecDisplay } from '$lib/workbench/spec-display';
	import { get } from 'svelte/store';

	let selectedPath = $state<string | null>(null);
	let centerView = $state<'graph' | 'text'>('graph');

	let raw = $state('');
	let loading = $state(false);
	let fetchErr = $state<string | null>(null);
	let isGoalFile = $state(false);
	let convention = $state<ConventionPayload['convention'] | null>(null);

	let typeId = $state<string | null>(null);
	let specParent = $state<string | null>(null);
	let specName = $state<string | null>(null);
	let parentGuessPath = $state<string | null>(null);
	let siblingFeaturePath = $state<string | null>(null);

	// ── edit mode state ───────────────────────────────────────────
	let editMode = $state(false);
	let editContent = $state('');
	let editOriginal = $state('');
	let saveError = $state<string | null>(null);
	let saveSuccess = $state(false);
	let saving = $state(false);

	$effect(() => {
		return specExplorerStore.subscribe(s => {
			selectedPath = s.selectedSpecPath;
			centerView = s.centerView;
		});
	});

	$effect(() => {
		let cancelled = false;
		fetch('/api/workspace/specs/convention')
			.then(r => (r.ok ? r.json() : null))
			.then((j: ConventionPayload | null) => {
				if (!cancelled && j?.convention) convention = j.convention;
			})
			.catch(() => {
				if (!cancelled) convention = null;
			});
		return () => {
			cancelled = true;
		};
	});

	async function resolveIsGoal(goalPath: string): Promise<boolean> {
		try {
			const state = get(specExplorerStore);
			const root = state.workspaceRoot;
			const content = await wailsReadSpecFile(root, 'specs/meta/goal-aspect-bindings.yaml');
			const b = parseYaml(content) as { goal_spec_path?: string };
			const g = b.goal_spec_path?.replace(/\\/g, '/') ?? '';
			const t = goalPath.replace(/\\/g, '/');
			return g !== '' && t === g;
		} catch {
			return false;
		}
	}

	$effect(() => {
		const p = selectedPath;
		if (!p) {
			raw = '';
			fetchErr = null;
			isGoalFile = false;
			typeId = null;
			specParent = null;
			specName = null;
			parentGuessPath = null;
			siblingFeaturePath = null;
			return;
		}
		loading = true;
		fetchErr = null;
		let cancelled = false;
		const state = get(specExplorerStore);
		const root = state.workspaceRoot;

		Promise.all([
			wailsReadSpecFile(root, p).catch(e => { throw e; }),
			resolveIsGoal(p),
		])
			.then(([content, goal]) => {
				if (cancelled) return;
				raw = content;
				isGoalFile = goal;
				fetchErr = null;
			})
			.catch(e => {
				if (cancelled) return;
				fetchErr = e instanceof Error ? e.message : String(e);
				raw = '';
				isGoalFile = false;
				typeId = null;
				specParent = null;
				specName = null;
				parentGuessPath = null;
				siblingFeaturePath = null;
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});

		return () => {
			cancelled = true;
		};
	});

	$effect(() => {
		if (!selectedPath) {
			typeId = null;
			return;
		}
		if (!convention) {
			typeId = null;
			return;
		}
		typeId = inferSpecTypeId(normalizeSpecPath(selectedPath), convention);
	});

	$effect(() => {
		if (!raw) {
			specParent = null;
			specName = null;
			parentGuessPath = null;
			siblingFeaturePath = null;
			return;
		}
		const meta = extractSpecMeta(raw);
		specParent = meta.parent;
		specName = meta.name;
		parentGuessPath = meta.parent ? inferParentSpecPath(meta.parent, convention) : null;
		if (!selectedPath) return;
		const norm = normalizeSpecPath(selectedPath);
		const sib = inferSiblingFeaturePath(norm);
		siblingFeaturePath =
			sib && normalizeSpecPath(sib) !== norm ? sib : null;
	});

	// ── edit mode ───────────────────────────────────────────────
	function startEdit() {
		editContent = raw;
		editOriginal = raw;
		saveError = null;
		saveSuccess = false;
		editMode = true;
	}
	function cancelEdit() {
		editMode = false;
		editContent = '';
		saveError = null;
		saveSuccess = false;
	}
	async function saveEdit() {
		if (!selectedPath || saving) return;
		saving = true;
		saveError = null;
		saveSuccess = false;
		try {
			const state = get(specExplorerStore);
			const root = state.workspaceRoot;
			const normPath = selectedPath.replace(/\\/g, '/');
			const specsIdx = normPath.indexOf('specs/');
			const relPath = specsIdx >= 0 ? normPath.slice(specsIdx) : normPath;
			await wailsWriteSpecFile(root, relPath, editContent);
			saveSuccess = true;
			editOriginal = editContent;
			raw = editContent;
			editMode = false;
			// 刷新 spec 列表（文件已变更）
			specExplorerStore.loadList(root);
		} catch (e) {
			saveError = e instanceof Error ? e.message : String(e);
		} finally {
			saving = false;
		}
	}

	function attachSelectedSpec() {
		if (!selectedPath || !raw) return;
		specAgentContextStore.addSpec(extractSpecDisplay(raw, selectedPath), raw);
	}
	$effect(() => {
		if (!editMode) return;
		const onBeforeunload = (e: BeforeUnloadEvent) => {
			if (editContent !== editOriginal) { e.preventDefault(); e.returnValue = ''; }
		};
		window.addEventListener('beforeunload', onBeforeunload);
		return () => window.removeEventListener('beforeunload', onBeforeunload);
	});
</script>

<div class="spec-viewer">
	{#if !selectedPath}
		<div class="empty">
			<p class="t">在左侧资源管理器（📂）中打开 specs 下的 YAML</p>
			<p class="s">
				默认<strong>图谱</strong>；切面/中心卡片详情在<strong>右侧抽屉</strong>（总目标）。
				「Spec 文本」查看完整源码。
			</p>
		</div>
	{:else}
		<div class="spec-tabbar" role="tablist" aria-label="规格主区视图">
			<button
				type="button"
				role="tab"
				class:active={centerView === 'graph'}
				onclick={() => specExplorerStore.setCenterView('graph')}
			>
				图谱
			</button>
			<button
				type="button"
				role="tab"
				class:active={centerView === 'text'}
				onclick={() => specExplorerStore.setCenterView('text')}
			>
				Spec 文本
			</button>
			<span class="spec-tab-spacer"></span>
			<button type="button" class="spec-close" onclick={() => specExplorerStore.selectSpec(null)}>
				返回画布
			</button>
		</div>

		{#if !loading && !fetchErr && raw}
			<div class="spec-meta" aria-label="规格类型与关联">
				{#if typeId}
					<span class="meta-badge" title={convention ? specTypeLabel(convention, typeId) ?? typeId : typeId}
						>{typeId}</span
					>
				{:else}
					<span class="meta-muted">类型未匹配目录约定</span>
				{/if}
				{#if specName}
					<span class="meta-kv"><span class="meta-k">name</span><code>{specName}</code></span>
				{/if}
				{#if specParent}
					<span class="meta-kv"
						><span class="meta-k">parent</span><code>{specParent}</code></span
					>
					{#if parentGuessPath}
						<button
							type="button"
							class="meta-link"
							title={parentGuessPath}
							onclick={() => specExplorerStore.selectSpec(parentGuessPath)}
						>
							打开 parent（推断）
						</button>
					{/if}
				{/if}
				{#if siblingFeaturePath}
					<button
						type="button"
						class="meta-link"
						title={siblingFeaturePath}
						onclick={() => specExplorerStore.selectSpec(siblingFeaturePath)}
					>
						同目录主 feature
					</button>
				{/if}
				<button type="button" class="meta-link" onclick={attachSelectedSpec}>
					添加到 Context
				</button>
			</div>
		{/if}

		<div class="spec-body">
			{#if loading}
				<p class="muted pad">加载 {selectedPath}…</p>
			{:else if fetchErr}
				<p class="err pad">{fetchErr}</p>
			{:else}
				{#if editMode}
					<div class="edit-wrap">
						<MonacoEditor bind:value={editContent} language="yaml" />
					</div>
				{:else}
					<div class="view-toolbar">
						<button type="button" class="btn-edit" onclick={startEdit}>
							✏️ 编辑
						</button>
					</div>
				{/if}
				{#if editMode}
					<div class="edit-toolbar">
						<button type="button" class="btn-save" onclick={saveEdit} disabled={saving}>
							{saving ? '保存中…' : '💾 保存'}
						</button>
						<button type="button" class="btn-cancel" onclick={cancelEdit} disabled={saving}>
							取消
						</button>
						<span class="edit-hint">编辑 {selectedPath}</span>
						{#if saveError}
							<span class="edit-error">❌ {saveError}</span>
						{/if}
						{#if saveSuccess}
							<span class="edit-success">✅ 已保存</span>
						{/if}
					</div>
				{:else if centerView === 'text'}
					<pre class="spec-text">{raw}</pre>
				{:else if isGoalFile}
					<div class="goal-wrap">
						<GoalSpecCanvas />
					</div>
				{:else}
					<GenericSpecGraph specPath={selectedPath} content={raw} />
				{/if}
			{/if}
		</div>
	{/if}
</div>

<style>
	.spec-viewer {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background: #0b0c10;
		color: #eef0f5;
	}

	.empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		text-align: center;
		color: var(--wb-text-sec, #8a8a8e);
	}

	.empty .t {
		margin: 0 0 0.5rem;
		font-size: 14px;
		color: var(--wb-text, #e8e8ed);
	}

	.empty .s {
		margin: 0;
		font-size: 12px;
		max-width: 320px;
		line-height: 1.5;
	}

	.spec-tabbar {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		height: 40px;
		background: rgba(18, 20, 26, 0.92);
		border-bottom: 1px solid #303746;
	}

	.spec-tabbar button {
		display: flex;
		align-items: center;
		padding: 0 14px;
		height: 100%;
		border: none;
		background: transparent;
		color: #a3abb9;
		font-family: inherit;
		font-size: 13px;
		cursor: pointer;
		border-right: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
		position: relative;
	}

	.spec-tabbar button:hover {
		background: rgba(122, 162, 255, 0.08);
		color: #eef0f5;
	}

	.spec-tabbar button.active {
		color: #eef0f5;
		background: #0b0c10;
		margin-bottom: -1px;
		border-bottom: 1px solid var(--wb-base, #0d0d0e);
	}

	.spec-tabbar button.active::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 1px;
		background: #7aa2ff;
	}

	.spec-tab-spacer {
		flex: 1;
	}

	.spec-close {
		margin-right: 8px;
		padding: 0 12px;
		height: calc(100% - 8px);
		align-self: center;
		font-size: 11px;
		border-radius: 5px;
		border: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
		background: rgba(255, 255, 255, 0.04);
		color: var(--wb-text-sec, #8a8a8e);
		cursor: pointer;
	}

	.spec-close:hover {
		color: var(--wb-text, #e8e8ed);
		border-color: rgba(88, 86, 214, 0.35);
	}

	.spec-meta {
		flex-shrink: 0;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
		padding: 6px 14px 8px;
		font-size: 11px;
		color: var(--wb-text-sec, #8a8a8e);
		background: rgba(28, 32, 42, 0.78);
		border-bottom: 1px solid #303746;
	}

	.meta-badge {
		padding: 2px 8px;
		border-radius: 5px;
		font-weight: 600;
		font-size: 10px;
		letter-spacing: 0.02em;
		background: rgba(122, 162, 255, 0.12);
		color: #9fc0ff;
		max-width: 220px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.meta-muted {
		opacity: 0.85;
		font-style: italic;
	}

	.meta-kv {
		display: inline-flex;
		align-items: baseline;
		gap: 6px;
	}

	.meta-k {
		color: var(--wb-muted, #555558);
		font-size: 10px;
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.meta-kv code {
		font-size: 11px;
		color: var(--wb-text-sec, #c4c4cc);
		background: rgba(255, 255, 255, 0.06);
		padding: 2px 6px;
		border-radius: 4px;
	}

	.meta-link {
		padding: 3px 10px;
		border-radius: 5px;
		border: 1px solid rgba(114, 214, 208, 0.45);
		background: rgba(114, 214, 208, 0.1);
		color: #bdf7f3;
		font-size: 11px;
		cursor: pointer;
		font-family: inherit;
	}

	.meta-link:hover {
		background: rgba(114, 214, 208, 0.18);
		color: #eef0f5;
	}

	.spec-body {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.goal-wrap {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.spec-text {
		flex: 1;
		margin: 0;
		padding: 16px 24px;
		overflow: auto;
		font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
		font-size: 12px;
		line-height: 1.7;
		color: var(--wb-text-sec, #8a8a8e);
		white-space: pre-wrap;
		word-break: break-word;
	}

	.muted.pad,
	.err.pad {
		padding: 16px 20px;
		font-size: 13px;
	}

	.err {
		color: #f87171;
	}

	/* ── edit mode ── */
	.view-toolbar {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		padding: 4px 12px;
		background: rgba(0, 0, 0, 0.2);
		border-bottom: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
		gap: 8px;
	}
	.btn-edit {
		padding: 4px 14px;
		border: 1px solid rgba(88, 86, 214, 0.35);
		background: rgba(88, 86, 214, 0.12);
		color: #c4b5fd;
		border-radius: 5px;
		font-size: 12px;
		cursor: pointer;
		font-family: inherit;
	}
	.btn-edit:hover {
		background: rgba(88, 86, 214, 0.22);
		color: var(--wb-text, #e8e8ed);
	}
	.edit-wrap {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}
	.edit-toolbar {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px 12px;
		background: rgba(88, 86, 214, 0.1);
		border-bottom: 1px solid rgba(88, 86, 214, 0.3);
	}
	.btn-save {
		padding: 4px 14px;
		background: rgba(88, 86, 214, 0.6);
		border: 1px solid rgba(88, 86, 214, 0.8);
		color: #fff;
		border-radius: 5px;
		font-size: 12px;
		cursor: pointer;
		font-family: inherit;
	}
	.btn-save:hover:not(:disabled) { background: rgba(88, 86, 214, 0.8); }
	.btn-save:disabled { opacity: 0.5; cursor: not-allowed; }
	.btn-cancel {
		padding: 4px 12px;
		background: transparent;
		border: 1px solid var(--wb-border, rgba(255, 255, 255, 0.1));
		color: var(--wb-muted, #555558);
		border-radius: 5px;
		font-size: 12px;
		cursor: pointer;
		font-family: inherit;
	}
	.btn-cancel:hover { color: var(--wb-text, #e8e8ed); border-color: rgba(255,255,255,0.2); }
	.edit-hint {
		font-size: 11px;
		color: var(--wb-muted, #555558);
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.edit-error { font-size: 11px; color: #f87171; }
	.edit-success { font-size: 11px; color: #6ee7b7; }
</style>
