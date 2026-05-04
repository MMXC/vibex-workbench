<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { specSlotSessionStore } from '$lib/stores/spec-slot-session-store';
	import type { SpecSlotSession } from '$lib/stores/spec-slot-session-store';
	import {
		extractPrototypeFromSource,
		fetchDesignKitStatus,
		scaffoldDesignKit,
		type DesignKitStatus,
	} from '$lib/services/design-kit-client';
	import { wailsReadSpecFile } from '$lib/wails-filesystem';

	let { session }: { session: SpecSlotSession } = $props();

	let status = $state<DesignKitStatus | null>(null);
	let busy = $state('');
	let err = $state<string | null>(null);
	let designBody = $state<string | null>(null);
	let designOpen = $state(false);
	let sourcePath = $state('frontend/src/routes/workbench/+page.svelte');
	let lastSpecSnippet = $state<string | null>(null);

	const root = $derived($specExplorerStore.workspaceRoot?.trim() ?? '');

	async function refresh() {
		err = null;
		if (!root) {
			status = null;
			return;
		}
		busy = 'status';
		try {
			status = await fetchDesignKitStatus(root);
			if (!status.ok) err = status.error ?? 'status failed';
		} finally {
			busy = '';
		}
	}

	$effect(() => {
		void root;
		void refresh();
	});

	async function onScaffold() {
		if (!root) return;
		busy = 'scaffold';
		err = null;
		try {
			const r = await scaffoldDesignKit(root);
			if (!r.ok) {
				err = r.error ?? 'scaffold failed';
				return;
			}
			await refresh();
		} finally {
			busy = '';
		}
	}

	async function onOpenDesign() {
		if (!root) return;
		busy = 'design';
		err = null;
		const path = status?.designPath ?? '.vibex/design/DESIGN.md';
		try {
			designBody = await wailsReadSpecFile(root, path);
			designOpen = true;
		} catch (e) {
			designBody = null;
			err = e instanceof Error ? e.message : String(e);
		} finally {
			busy = '';
		}
	}

	async function onExtract() {
		if (!root || !sourcePath.trim()) return;
		busy = 'extract';
		err = null;
		lastSpecSnippet = null;
		try {
			const r = await extractPrototypeFromSource({
				workspaceRoot: root,
				sourcePath: sourcePath.trim(),
			});
			if (!r.ok) {
				err = r.error ?? 'extract failed';
				return;
			}
			lastSpecSnippet = r.specSnippet ?? null;
			await refresh();
		} finally {
			busy = '';
		}
	}

	function onPrefillLinkSpec() {
		const specPath = session.spec.path;
		const snippet =
			lastSpecSnippet ??
			`prototype:
  file: .vibex/prototypes/<your-file>.html
  status: draft`;
		const text = `请将以下片段合并到当前 spec（${specPath}）的 YAML 中，保持缩进与父级字段一致；若已有 prototype 块请只做增量修改：\n\n${snippet}`;
		specSlotSessionStore.prefillActiveChat(text);
	}

	function onPrefillDesignReminder() {
		specSlotSessionStore.prefillActiveChat(
			`生成或调整原型前请先阅读并遵守工作区 ${status?.designPath ?? '.vibex/design/DESIGN.md'} 的栈、色板与组件约定；产物写入 ${status?.prototypesPath ?? '.vibex/prototypes/'} 并在本 spec 的 prototype.file 引用相对路径。`
		);
	}
</script>

<div class="kit-bar" aria-label="原型物料库工具">
	<span class="k">Design Kit</span>
	{#if !root}
		<p class="hint">请在资源管理器中选择工作区根目录后使用物料库。</p>
	{:else}
		<div class="row">
			<span class="pill" class:ok={status?.designMdExists} class:miss={!status?.designMdExists}>
				DESIGN.md {status?.designMdExists ? '已有' : '缺失'}
			</span>
			<span class="pill" class:ok={status?.prototypesDirExists} class:miss={!status?.prototypesDirExists}>
				prototypes {status?.prototypesDirExists ? '已有' : '缺失'}
			</span>
		</div>
		<div class="actions">
			<button type="button" disabled={!!busy} onclick={() => onScaffold()}>
				{busy === 'scaffold' ? '…' : '初始化物料库'}
			</button>
			<button type="button" disabled={!!busy || !status?.designMdExists} onclick={() => onOpenDesign()}>
				{busy === 'design' ? '…' : '查看 DESIGN.md'}
			</button>
			<button type="button" disabled={!!busy} onclick={() => onPrefillDesignReminder()}>注入设计门禁提示</button>
		</div>
		<div class="extract">
			<label>
				<span>从页面提取（工作区相对路径）</span>
				<input type="text" bind:value={sourcePath} placeholder="frontend/src/..." spellcheck="false" />
			</label>
			<button type="button" class="primary" disabled={!!busy || !sourcePath.trim()} onclick={() => onExtract()}>
				{busy === 'extract' ? '提取中…' : '剥离并写入 prototypes'}
			</button>
			<button
				type="button"
				disabled={!!busy}
				onclick={() => onPrefillLinkSpec()}
				title="将 YAML 片段预填到左侧对话，便于合并到当前 spec"
			>
				关联到当前 spec（预填）
			</button>
		</div>
		{#if err}
			<p class="err">{err}</p>
		{/if}
		{#if lastSpecSnippet}
			<pre class="snippet" title="最近一次提取生成的 prototype 片段">{lastSpecSnippet}</pre>
		{/if}
		{#if designOpen && designBody != null}
			<details class="design-panel" open>
				<summary>DESIGN.md 预览 · 点击折叠</summary>
				<pre>{designBody}</pre>
			</details>
		{/if}
	{/if}
</div>

<style>
	.kit-bar {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 10px 12px;
		margin: 0 0 10px;
		border: 1px solid #303746;
		border-radius: 14px;
		background: rgba(18, 21, 28, 0.92);
	}

	.k {
		display: block;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.pill {
		font-size: 10px;
		padding: 3px 8px;
		border-radius: 999px;
		border: 1px solid #465064;
		color: #a3abb9;
	}
	.pill.ok {
		border-color: rgba(114, 214, 208, 0.45);
		color: #bdf7f3;
	}
	.pill.miss {
		border-color: rgba(239, 198, 107, 0.46);
		color: #fcd34d;
	}

	.actions,
	.extract {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		gap: 8px;
	}

	.extract label {
		display: flex;
		flex-direction: column;
		gap: 4px;
		flex: 1 1 220px;
		font-size: 10px;
		color: #858fa1;
	}

	.extract input {
		border: 1px solid #303746;
		border-radius: 10px;
		padding: 7px 10px;
		background: #0a0c10;
		color: #e2e8f0;
		font-size: 11px;
	}

	button {
		border: 1px solid #465064;
		border-radius: 999px;
		background: rgba(28, 32, 42, 0.85);
		color: #d4d8e3;
		padding: 7px 12px;
		font-size: 11px;
		font-weight: 700;
		cursor: pointer;
	}
	button:hover:not(:disabled) {
		border-color: #7aa2ff;
		color: #eef0f5;
	}
	button:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}
	button.primary {
		border-color: rgba(122, 162, 255, 0.55);
		color: #cfe0ff;
	}

	.hint,
	.err {
		margin: 0;
		font-size: 11px;
		line-height: 1.4;
	}
	.hint {
		color: #858fa1;
	}
	.err {
		color: #fca5a5;
	}

	.snippet {
		margin: 0;
		padding: 8px 10px;
		border-radius: 10px;
		background: #0a0c10;
		border: 1px solid #242b38;
		font-size: 10px;
		color: #a3abb9;
		white-space: pre-wrap;
		max-height: 120px;
		overflow: auto;
	}

	.design-panel {
		font-size: 11px;
		color: #a3abb9;
	}
	.design-panel summary {
		cursor: pointer;
		color: #7aa2ff;
		margin-bottom: 6px;
	}
	.design-panel pre {
		margin: 0;
		max-height: 200px;
		overflow: auto;
		padding: 10px;
		border-radius: 10px;
		background: #0a0c10;
		border: 1px solid #242b38;
		white-space: pre-wrap;
	}
</style>
