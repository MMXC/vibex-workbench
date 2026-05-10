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
	import { evaluatePrototypeGate } from '$lib/workbench/ui-workflow-gate';
	import {
		buildFeatureShellRoute,
		deriveEntryHtmlForFeature,
	} from '$lib/workbench/prototype-shell-manifest';
	import {
		fetchPrototypeManifest,
		registerPrototypeManifestRoute,
	} from '$lib/services/prototype-shell-manifest-client';

	let { session }: { session: SpecSlotSession } = $props();

	let status = $state<DesignKitStatus | null>(null);
	let busy = $state('');
	let err = $state<string | null>(null);
	let designBody = $state<string | null>(null);
	let designOpen = $state(false);
	let shellManifestExists = $state(false);
	let shellManifestPath = $state<string | null>(null);
	let shellRouteCount = $state(0);
	let shellBusy = $state('');
	let shellErr = $state<string | null>(null);
	let shellEntryOverride = $state('');
	/** 可由 Agent 根据 `.vibex/graph/` 回填；留空时请先跑「预填：页面/CSS 图谱分析」 */
	let sourcePath = $state('');
	let lastSpecSnippet = $state<string | null>(null);

	const root = $derived($specExplorerStore.workspaceRoot?.trim() ?? '');
	const gate = $derived(evaluatePrototypeGate(session.content ?? ''));
	const shellPreview = $derived(
		buildFeatureShellRoute({
			specName: session.spec.name,
			specPath: session.spec.path,
			displayTitle: session.spec.display?.title ?? session.spec.name,
			yamlContent: session.content ?? '',
			entryHtmlOverride: shellEntryOverride.trim() || null,
		})
	);
	/** 仅「extract 写入 prototypes」需 Gate；初始化 DESIGN/README 不阻塞 */
	const canExtractWrite = $derived(gate.canCommitPrototype);

	async function refreshShellManifest() {
		shellErr = null;
		if (!root) {
			shellManifestExists = false;
			shellManifestPath = null;
			shellRouteCount = 0;
			return;
		}
		shellBusy = 'manifest';
		try {
			const m = await fetchPrototypeManifest(root);
			if (!m.ok) {
				shellErr = m.error ?? 'manifest read failed';
				return;
			}
			shellManifestExists = !!m.exists;
			shellManifestPath = m.manifestPath ?? '.vibex/prototype-manifest.yaml';
			shellRouteCount = m.data?.routes?.length ?? 0;
		} finally {
			shellBusy = '';
		}
	}

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
			await refreshShellManifest();
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
				err =
					(r as { gateFailure?: { next_action?: string } }).gateFailure?.next_action ??
					r.error ??
					'scaffold failed';
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
				specYaml: session.content,
			});
			if (!r.ok) {
				err =
					(r as { gateFailure?: { next_action?: string } }).gateFailure?.next_action ??
					r.error ??
					'extract failed';
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

	function onPrefillGateTemplate() {
		specSlotSessionStore.prefillActiveChat(gate.nextAction);
	}

	/** 预填「图谱 + 源码」分析任务，供 Agent 结合 graphifyy / 代码图与实际文件后再 extract */
	async function onRegisterShellManifest() {
		if (!root) return;
		shellBusy = 'register';
		shellErr = null;
		try {
			const r = await registerPrototypeManifestRoute({
				workspaceRoot: root,
				specName: session.spec.name,
				specPath: session.spec.path,
				displayTitle: session.spec.display?.title ?? session.spec.name,
				yamlContent: session.content ?? '',
				entryHtml: shellEntryOverride.trim() || null,
			});
			if (!r.ok) {
				shellErr = r.error ?? 'register failed';
				return;
			}
			await refreshShellManifest();
		} finally {
			shellBusy = '';
		}
	}

	function onPrefillShellManifest() {
		const entry = deriveEntryHtmlForFeature({
			specPath: session.spec.path,
			yamlContent: session.content ?? '',
			override: shellEntryOverride.trim() || null,
		});
		specSlotSessionStore.prefillActiveChat(
			[
				'请将当前功能注册到 App Shell 路由 manifest（若尚未创建则从示例复制 `.vibex/prototype-manifest.example.yaml` 为 `.vibex/prototype-manifest.yaml`）。',
				`建议条目：id=${shellPreview.id} · path=${shellPreview.path} · specRef=${session.spec.name} · entryHtml=${entry}`,
				'Shell 消费侧实现见 specs/L4-feature/FEAT-spec-prototype-shell-deck.yaml。',
			].join('\n')
		);
	}

	function onPrefillGraphAnalysis() {
		const hint = sourcePath.trim();
		specSlotSessionStore.prefillActiveChat(
			[
				'请基于工作区源码与代码图谱，分析页面边界、样式入口与可剥离为 Design Kit 原型的范围。',
				'',
				`**路径来源（优先）**：不要仅依赖聊天里手工粘贴的路径（易过时）。请按 \`graphify-page-analysis\` skill：用 PyPI 包 \`graphifyy\` 的 CLI \`graphify\` 在工作区根生成图谱（默认输出 \`graphify-out/graph.json\`；若团队同步到 \`.vibex/graph/\` 也可读），再根据图谱或 \`graphify query\` 结果定位入口 Svelte/CSS 相对路径，再 \`read_file\` 精读。`,
				`**工作区根（解析基准）**：${root}`,
				hint
					? `（可选线索，若与图谱冲突以图谱为准）相对路径参考：${hint}`
					: '（左侧输入框未填路径 — 请完全由图谱解析后回填再读文件。）',
				'',
				'**精读**：对源文件及其 import 链上的 CSS/组件做交叉核对（read_file / bash+grep）；勿仅凭推测写 prototypes。',
				'**输出**：①页面职责；②CSS 与组件依赖摘要；③建议的 `.vibex/prototypes/` 文件名草案（默认不自动物料写盘，除非我确认）。',
				'若已加载 skills，可先 `skill_load graphify-page-analysis`（默认从仓库 `skills/graphify-page-analysis/` 加载）。',
			].join('\n')
		);
	}
</script>

<div class="kit-bar" aria-label="原型物料库工具">
	<span class="k">Design Kit</span>
	{#if !canExtractWrite}
		<p class="gate-hint">
			Prototype Gate：阶段 <strong>{gate.stage}</strong> — 补齐 Intent / UI Spec 后才能「剥离并写入 prototypes」（可先「初始化物料库」）。
			<button type="button" class="linkish" onclick={() => onPrefillGateTemplate()}>预填模板与缺口说明</button>
		</p>
	{/if}
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
				<span>从页面提取（相对路径，可先空着由图谱分析回填）</span>
				<input
					type="text"
					bind:value={sourcePath}
					placeholder="例：frontend/src/routes/workbench/+page.svelte（或由 Agent 按图谱填写）"
					spellcheck="false"
				/>
			</label>
			<button
				type="button"
				class="primary"
				disabled={!!busy || !sourcePath.trim() || !canExtractWrite}
				onclick={() => onExtract()}
			>
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
			<button type="button" disabled={!!busy} onclick={() => onPrefillGraphAnalysis()} title="预填对话：图谱 + 源码分析（graphifyy / 缺失则生成）">
				预填：页面/CSS 图谱分析
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

		<div class="shell-block" aria-label="App Shell 路由 manifest">
			<span class="k">App Shell · prototype-manifest</span>
			<p class="shell-hint">
				写入工作区 <code>{shellManifestPath ?? '.vibex/prototype-manifest.yaml'}</code>
				（与 <code>.vibex/prototypes/manifest.json</code> 设计物料登记不同）。
			</p>
			<div class="row">
				<span class="pill" class:ok={shellManifestExists} class:miss={!shellManifestExists}>
					manifest {shellManifestExists ? '已有' : '未创建'}
				</span>
				<span class="pill ok">routes {shellRouteCount}</span>
			</div>
			<label class="shell-entry">
				<span>entryHtml 覆盖（可选，默认取 prototype.file 或 .vibex/prototypes/&lt;spec&gt;.html）</span>
				<input
					type="text"
					bind:value={shellEntryOverride}
					placeholder={shellPreview.entryHtml}
					spellcheck="false"
				/>
			</label>
			<div class="actions">
				<button
					type="button"
					class="primary"
					disabled={!!busy || !!shellBusy}
					onclick={() => onRegisterShellManifest()}
				>
					{shellBusy === 'register' ? '写入中…' : '注册到 Shell manifest'}
				</button>
				<button type="button" disabled={!!busy || !!shellBusy} onclick={() => refreshShellManifest()}>
					{shellBusy === 'manifest' ? '…' : '刷新状态'}
				</button>
				<button type="button" disabled={!!busy} onclick={() => onPrefillShellManifest()}>
					预填对话（说明 manifest）
				</button>
			</div>
			{#if shellErr}
				<p class="err">{shellErr}</p>
			{/if}
			<pre class="snippet shell-snippet" title="即将写入的路由预览">{JSON.stringify(shellPreview, null, 2)}</pre>
		</div>
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

	.gate-hint {
		margin: 0;
		font-size: 11px;
		line-height: 1.45;
		color: #fcd34d;
	}
	.gate-hint strong {
		color: #fde68a;
	}
	button.linkish {
		display: inline;
		border: none;
		background: none;
		padding: 0;
		margin-left: 6px;
		color: #7aa2ff;
		text-decoration: underline;
		cursor: pointer;
		font-size: inherit;
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

	.shell-block {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 10px 12px;
		margin: 10px 0 0;
		border: 1px solid rgba(122, 162, 255, 0.35);
		border-radius: 14px;
		background: rgba(14, 18, 28, 0.95);
	}

	.shell-hint {
		margin: 0;
		font-size: 10px;
		line-height: 1.45;
		color: #858fa1;
	}

	.shell-hint code {
		font-size: 10px;
		color: #a5b4fc;
	}

	.shell-entry {
		display: flex;
		flex-direction: column;
		gap: 4px;
		font-size: 10px;
		color: #858fa1;
	}

	.shell-entry input {
		border: 1px solid #303746;
		border-radius: 10px;
		padding: 7px 10px;
		background: #0a0c10;
		color: #e2e8f0;
		font-size: 11px;
	}

	.shell-snippet {
		max-height: 140px;
	}
</style>
