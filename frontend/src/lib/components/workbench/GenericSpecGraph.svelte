<!-- 非 L1 总目标时的轻量「图谱」：中心为 spec，周围为 canonical 槽位卡片 -->
<script lang="ts">
	import { get } from 'svelte/store';
	import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';
	import { specAgentContextStore } from '$lib/stores/spec-agent-context-store';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { wailsReadSpecFile, wailsWriteSpecFile } from '$lib/wails-filesystem';
	import { extractSpecDisplay } from '$lib/workbench/spec-display';

	let { specPath, content }: { specPath: string; content: string } = $props();

	const specMeta = $derived.by(() => extractSpecDisplay(content, specPath));

	function attachCurrentSpec() {
		specAgentContextStore.addSpec(specMeta, content);
	}

	const level = $derived.by(() => {
		return specMeta.rawLevel;
	});

	const slotCards = $derived.by(() => specMeta.slots.all);

	function levelShort(raw: string): string {
		if (raw.includes('1')) return 'L1';
		if (raw.includes('2')) return 'L2';
		if (raw.includes('3')) return 'L3';
		if (raw.includes('4')) return 'L4';
		if (raw.includes('5')) return 'L5';
		return specMeta.level === 'UNKNOWN' ? 'SPEC' : specMeta.level;
	}

	function laneLeft(lane: string): number {
		const order = ['L1', 'L2', 'L3', 'L4', 'L5'];
		const idx = Math.max(order.indexOf(lane), 2);
		return 8 + idx * 21;
	}

	const currentLeft = $derived.by(() => laneLeft(levelShort(level)));
	const parentLeft = $derived.by(() => Math.max(8, currentLeft - 21));

	const SSE_URL = import.meta.env.VITE_SSE_URL || 'http://localhost:33338';
	let currentSpecContent = $state('');
	let autoGenerateTriggered = $state(false);
	let lastAutoRefreshKey = $state('');
	let pptPath = $state<string>('');
	let pptSrc = $state<string>('');
	let pptLoading = $state(false);
	let pptError = $state<string | null>(null);
	let pptTaskState = $state<'idle' | 'queued' | 'running'>('idle');
	let pptContainerEl = $state<HTMLElement | null>(null);
	let pptFrameEl = $state<HTMLIFrameElement | null>(null);
	let pptFullscreen = $state(false);

	const hasPptFile = $derived.by(() => getPptFileOnly(currentSpecContent).length > 0);
	const genButtonLabel = $derived.by(() => {
		if (pptTaskState === 'queued') return '队列中';
		if (pptTaskState === 'running') return '生成中';
		return hasPptFile ? '重新生成' : '生成';
	});

	function basenameNoExt(path: string): string {
		const p = path.replace(/\\/g, '/').split('/').pop() ?? 'spec';
		return p.replace(/\.[^.]+$/, '');
	}

	function getPptFileOnly(yamlText: string): string {
		try {
			const doc = parseYaml(yamlText) as Record<string, unknown> | null;
			const proto = (doc?.prototype ?? {}) as Record<string, unknown>;
			const explicit = String(proto.ppt_file ?? '').trim();
			if (explicit.toLowerCase().endsWith('.html')) return explicit;
			return '';
		} catch {
			return '';
		}
	}

	function getSpecLevelTag(yamlText: string): 'L1' | 'L2' | 'L3' | 'L4' | 'L5' | 'UNK' {
		try {
			const doc = parseYaml(yamlText) as Record<string, unknown> | null;
			const spec = (doc?.spec ?? {}) as Record<string, unknown>;
			const raw = String(spec.level ?? '').toLowerCase();
			if (raw.includes('1')) return 'L1';
			if (raw.includes('2')) return 'L2';
			if (raw.includes('3')) return 'L3';
			if (raw.includes('4')) return 'L4';
			if (raw.includes('5')) return 'L5';
			return 'UNK';
		} catch {
			return 'UNK';
		}
	}

	function getLayerSlideTemplate(levelTag: 'L1' | 'L2' | 'L3' | 'L4' | 'L5' | 'UNK'): string[] {
		const common = [
			'统一风格要求：theme 使用 blueprint/tokyo-night/corporate-clean 之一；保持统一 header/footer/页码与键盘导航。',
			'统一内容要求：每页 1 个主标题 + 3~5 条要点，避免长段落；所有要点要可追溯到当前 spec YAML 字段。',
			'统一目标：同时覆盖「现有实现现状(as-is)」与「spec目标实现(to-be)」，并给出差距与可调点。',
		];
		const byLevel: Record<string, string[]> = {
			L1: [
				'L1 模板（目标层）：封面 -> 业务目标与成功指标 -> 范围边界(in/out) -> 顶层能力地图 -> 主要风险与假设 -> 决策与下一步。',
				'聚焦“为什么做/做到什么算成功”，弱化实现细节，强调对下层 spec 的约束与验收口径。',
			],
			L2: [
				'L2 模板（能力分解层）：封面 -> 能力分解树 -> 用户旅程/主流程 -> 模块职责分配 -> 依赖与接口边界 -> 风险与调整建议。',
				'聚焦“把目标拆成可交付能力”，明确每块能力的 owner、输入输出与完成定义。',
			],
			L3: [
				'L3 模板（模块/接口层）：封面 -> 模块定位 -> 对外接口与契约 -> 关键数据流与状态 -> 异常路径与恢复策略 -> 验证计划。',
				'聚焦“模块怎么协作”，强调接口稳定性、状态机和故障处理。',
			],
			L4: [
				'L4 模板（特性层）：封面 -> 特性价值与触发场景 -> as-is vs to-be 差异 -> 关键交互/流程图 -> 受影响文件与改动面 -> 风险与回滚。',
				'聚焦“这个 feature 如何落地”，强调跨模块影响、改动边界、验证入口。',
			],
			L5: [
				'L5 模板（实现切片层）：封面 -> 实现目标与完成标准 -> 输入/输出/boundary 细化 -> 代码改动点与调用链 -> 测试验证与观测信号 -> 已知问题与调参建议。',
				'聚焦“具体实现怎么做”，必须给出可执行验证步骤与排错线索，方便 agent/用户快速修订描述。',
			],
			UNK: [
				'未知层级模板：封面 -> 背景 -> IO契约 -> 结构位置 -> 实现与风险 -> 总结。',
			],
		};
		return [...common, ...(byLevel[levelTag] ?? byLevel.UNK)];
	}

	type LayerPromptConfig = {
		lines: string[];
		primaryTheme: string;
		themeCandidates: string[];
	};

	function defaultThemeByLevel(levelTag: 'L1' | 'L2' | 'L3' | 'L4' | 'L5' | 'UNK'): string {
		switch (levelTag) {
			case 'L1':
				return 'corporate-clean';
			case 'L2':
				return 'swiss-grid';
			case 'L3':
				return 'tokyo-night';
			case 'L4':
				return 'blueprint';
			case 'L5':
				return 'engineering-whiteprint';
			default:
				return 'blueprint';
		}
	}

	async function loadLayerSlideTemplate(
		workspaceRoot: string,
		levelTag: 'L1' | 'L2' | 'L3' | 'L4' | 'L5' | 'UNK'
	): Promise<LayerPromptConfig> {
		const fallback: LayerPromptConfig = {
			lines: getLayerSlideTemplate(levelTag),
			primaryTheme: defaultThemeByLevel(levelTag),
			themeCandidates: ['blueprint', 'tokyo-night', 'corporate-clean'],
		};
		try {
			const raw = String(
				await wailsReadSpecFile(workspaceRoot, '.agents/ppt-layer-templates.yaml')
			);
			const doc = parseYaml(raw) as Record<string, unknown> | null;
			const style = (doc?.style ?? {}) as Record<string, unknown>;
			const themeCandidates = Array.isArray(style.theme_candidates)
				? style.theme_candidates.map(v => String(v)).filter(Boolean)
				: fallback.themeCandidates;
			const themeByLevel = (style.theme_by_level ?? {}) as Record<string, unknown>;
			const primaryTheme =
				String(themeByLevel[levelTag] ?? '').trim() ||
				String(themeByLevel.UNK ?? '').trim() ||
				fallback.primaryTheme;
			const rules = Array.isArray(style.rules)
				? style.rules.map(v => String(v)).filter(Boolean)
				: [];
			const layers = (doc?.layers ?? {}) as Record<string, unknown>;
			const layerItems = Array.isArray(layers[levelTag])
				? (layers[levelTag] as unknown[]).map(v => String(v)).filter(Boolean)
				: [];
			const fallbackItems = Array.isArray(layers.UNK)
				? (layers.UNK as unknown[]).map(v => String(v)).filter(Boolean)
				: [];
			const merged = [...rules, ...(layerItems.length > 0 ? layerItems : fallbackItems)];
			if (merged.length > 0) {
				return {
					lines: merged,
					primaryTheme,
					themeCandidates,
				};
			}
		} catch {
			// fallback to builtin template
		}
		return fallback;
	}

	function upsertPrototypePptFile(yamlText: string, nextPath: string): string {
		const doc = (parseYaml(yamlText) ?? {}) as Record<string, unknown>;
		const proto = ((doc.prototype ?? {}) as Record<string, unknown>) || {};
		proto.ppt_file = nextPath;
		doc.prototype = proto;
		return stringifyYaml(doc);
	}

	function delay(ms: number): Promise<void> {
		return new Promise(resolve => setTimeout(resolve, ms));
	}

	function toWorkspaceFileURL(path: string, workspaceRoot: string): string {
		const safe = path
			.split('/')
			.map(seg => encodeURIComponent(seg))
			.join('/');
		return `${SSE_URL}/api/workspace/file/${safe}?workspaceRoot=${encodeURIComponent(workspaceRoot)}`;
	}

	async function togglePptFullscreen() {
		try {
			if (!document.fullscreenElement) {
				await pptContainerEl?.requestFullscreen();
			} else {
				await document.exitFullscreen();
			}
		} catch (e) {
			pptError = e instanceof Error ? e.message : String(e);
		}
	}

	function gotoPptSlide(direction: 'prev' | 'next') {
		const frameWin = pptFrameEl?.contentWindow;
		if (!frameWin) return;
		try {
			const key = direction === 'next' ? 'ArrowRight' : 'ArrowLeft';
			frameWin.focus();
			frameWin.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
			frameWin.document.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
			return;
		} catch {
			// fallback to hash navigation when keyboard injection is blocked
		}
		try {
			const loc = frameWin.location;
			const match = /^#\/(\d+)$/.exec(loc.hash || '');
			const current = match ? Math.max(1, Number(match[1])) : 1;
			const next = direction === 'next' ? current + 1 : Math.max(1, current - 1);
			loc.hash = `#/${next}`;
		} catch {
			// ignore
		}
	}

	function waitForAgentCompletion(
		threadId: string,
		onState: (state: 'queued' | 'running') => void,
		timeoutMs = 120000
	): Promise<void> {
		return new Promise((resolve, reject) => {
			const es = new EventSource(`${SSE_URL}/api/sse/${encodeURIComponent(threadId)}`);
			const timer = window.setTimeout(() => {
				es.close();
				reject(new Error('生成超时：未收到完成事件'));
			}, timeoutMs);
			const done = () => {
				window.clearTimeout(timer);
				es.close();
				resolve();
			};
			const fail = (msg: string) => {
				window.clearTimeout(timer);
				es.close();
				reject(new Error(msg));
			};
			es.addEventListener('ppt.queue', () => onState('queued'));
			es.addEventListener('run.started', () => onState('running'));
			es.addEventListener('run.completed', () => done());
			es.addEventListener('run.failed', (ev) => {
				try {
					const data = JSON.parse((ev as MessageEvent).data) as { error?: string };
					fail(data.error || 'run.failed');
				} catch {
					fail('run.failed');
				}
			});
			es.addEventListener('error', () => {
				// keep waiting; network hiccups are possible on SSE
			});
		});
	}

	async function refreshPpt() {
		if (pptLoading) return;
		const wsRoot = get(specExplorerStore).workspaceRoot;
		const configured = getPptFileOnly(currentSpecContent);
		pptPath = configured;
		if (!configured) {
			pptSrc = '';
			pptError = null;
			if (!autoGenerateTriggered) {
				autoGenerateTriggered = true;
				void generatePpt();
				return;
			}
			pptError = '当前 spec 未配置 prototype.ppt_file，已尝试自动生成，请点击「生成/重生成」重试。';
			return;
		}
		pptLoading = true;
		pptError = null;
		try {
			const html = await wailsReadSpecFile(wsRoot, configured);
			const content = String(html ?? '');
			pptSrc = toWorkspaceFileURL(configured, wsRoot);
			if (!content) {
				pptError = 'ppt_file 指向文件为空。';
			}
		} catch (e) {
			pptSrc = '';
			pptError = e instanceof Error ? e.message : String(e);
		} finally {
			pptLoading = false;
		}
	}

	async function generatePpt() {
		if (pptLoading) return;
		const wsRoot = get(specExplorerStore).workspaceRoot;
		const existing = getPptFileOnly(currentSpecContent);
		const target = existing || `.vibex/ppt/${basenameNoExt(specPath)}.html`;
		pptLoading = true;
		pptTaskState = 'queued';
		pptError = null;
		pptPath = target;
		try {
			const threadId = `ppt-${basenameNoExt(specPath)}-${Date.now()}`;
			const levelTag = getSpecLevelTag(currentSpecContent);
			const layerConfig = await loadLayerSlideTemplate(wsRoot, levelTag);
			const prompt = [
				'你是 vibex-workbench 的原型生成代理。',
				'必须使用 html-ppt skill 生成一个可运行 HTML 演示文件（spec 说明型 PPT）。',
				`spec_path: ${specPath}`,
				`spec_level: ${levelTag}`,
				`visual_theme_primary: ${layerConfig.primaryTheme}`,
				`visual_theme_candidates: ${layerConfig.themeCandidates.join(', ')}`,
				`目标输出文件: ${target}`,
				'要求：只写入该目标文件；不要写其它路径。',
				'允许使用 html-ppt 外链资源（如 ../assets/themes/*.css 和 ../assets/runtime.js），优先保持标准 html-ppt deck 结构。',
				'视觉要求：优先使用 visual_theme_primary；如不适配可在 visual_theme_candidates 中选择，但必须保持统一风格。',
				'必须是“分页演示稿”而不是长文档：每个 section.slide 占满一屏，默认仅显示 1 页。',
				'必须提供交互：←/→ 或 PgUp/PgDn 切页、底部页码、目录页跳转。',
				'页面结构建议：封面、问题背景、IO契约、在总spec中的位置、上下游关系、验证与风险、总结（6~8页）。',
				'以下是按当前 spec 层级的定制模板（必须遵循）：',
				...layerConfig.lines.map((line, idx) => `${idx + 1}. ${line}`),
				'内容表达要求：每页仅 1 个主标题 + 3~5 条要点，避免大段文本堆砌。',
				'内容目标：用于让用户快速理解该 spec 的功能与作用，不是页面原型实现图。',
				'必须包含：',
				'1) 该 spec 的核心功能与问题背景',
				'2) input/output/boundary 的摘要',
				'3) 它在总 spec（父/子/依赖）中的位置与作用',
				'4) 与相邻 spec 的关系（上游输入、下游影响）',
				'5) 验证方式与关键风险',
				'6) 现有实现(as-is)与目标实现(to-be)差距，并给出可调整的 spec 描述建议（优先高风险/高收益项）',
				'输出完成后返回 done。',
				'下面是当前 spec YAML：',
				currentSpecContent,
			].join('\n');
			await fetch(`${SSE_URL}/api/agent/execute`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					threadId,
					input: prompt,
					workspaceRoot: wsRoot,
					agent_profile: 'ppt-generator',
				}),
			});
			await waitForAgentCompletion(threadId, (next) => {
				pptTaskState = next;
			});
			let html = '';
			for (let i = 0; i < 8; i++) {
				try {
					html = String(await wailsReadSpecFile(wsRoot, target) ?? '');
					if (html.trim().length > 0) break;
				} catch {
					// 生成中常见 404，轮询等待
				}
				await delay(800);
			}
			if (!html.trim()) {
				throw new Error(`生成后未检测到文件: ${target}`);
			}
			pptSrc = toWorkspaceFileURL(target, wsRoot);
			const patched = upsertPrototypePptFile(currentSpecContent, target);
			await wailsWriteSpecFile(wsRoot, specPath, patched);
			currentSpecContent = patched;
		} catch (e) {
			pptError = e instanceof Error ? e.message : String(e);
		} finally {
			pptLoading = false;
			pptTaskState = 'idle';
			void refreshPpt();
		}
	}

	$effect(() => {
		currentSpecContent = content;
		autoGenerateTriggered = false;
		lastAutoRefreshKey = '';
	});

	$effect(() => {
		const key = `${specPath}::${currentSpecContent}`;
		if (key === lastAutoRefreshKey) return;
		lastAutoRefreshKey = key;
		void refreshPpt();
	});

	$effect(() => {
		const onFsChange = () => {
			pptFullscreen = !!document.fullscreenElement;
		};
		document.addEventListener('fullscreenchange', onFsChange);
		return () => document.removeEventListener('fullscreenchange', onFsChange);
	});
</script>

<div class="generic-graph">
	<div class="graph-head">
		<div>
			<span class="eyebrow">Layered DAG Canvas</span>
			<h2>{specMeta.display.title}</h2>
			<p>{specMeta.display.summary}</p>
		</div>
		<div class="head-actions">
			<span class="mode-badge">{levelShort(level)}</span>
			<button type="button" class="attach-head" onclick={attachCurrentSpec}>Add to Context</button>
		</div>
	</div>
	<div class="radial">
		{#if specMeta.parent}
			<div class="relation-card parent-card" style:left="{parentLeft}%" style:top="50%">
				<span class="k">parent</span>
				<strong>{specMeta.parent}</strong>
			</div>
			<span class="edge parent-edge" style:left="{parentLeft + 9}%" style:width="{currentLeft - parentLeft - 13}%"></span>
		{/if}

		<div class="ppt-center" bind:this={pptContainerEl}>
			<div class="ppt-head">
				<strong>Spec HTML PPT 演示</strong>
				<div class="ppt-actions">
					<button type="button" onclick={refreshPpt} disabled={pptLoading}>刷新</button>
					<button type="button" class="gen-btn" onclick={generatePpt} disabled={pptLoading}>
						{genButtonLabel}
					</button>
				</div>
			</div>
			<button
				type="button"
				class="fs-fab"
				title={pptFullscreen ? '退出全屏' : '全屏演示'}
				onclick={togglePptFullscreen}
			>
				{pptFullscreen ? '⤫' : '⛶'}
			</button>
			{#if pptPath}
				<small class="ppt-path">{pptPath}</small>
			{/if}
			{#if pptLoading}
				<div class="ppt-loading">
					<span class="spinner"></span>
					<span>
						{#if pptTaskState === 'queued'}
							任务队列中，等待执行…
						{:else if pptTaskState === 'running'}
							Agent 正在生成 prototype.ppt_file…
						{:else}
							正在加载 prototype.ppt_file…
						{/if}
					</span>
				</div>
			{:else if pptSrc}
				<iframe
					class="ppt-frame"
					bind:this={pptFrameEl}
					title="spec ppt demo preview"
					sandbox="allow-scripts allow-same-origin"
					referrerpolicy="no-referrer"
					src={pptSrc}
				></iframe>
				<div class="ppt-nav-fab">
					<button type="button" title="上一页" onclick={() => gotoPptSlide('prev')}>◀</button>
					<button type="button" title="下一页" onclick={() => gotoPptSlide('next')}>▶</button>
				</div>
			{:else}
				<p class="ppt-empty">未找到可展示 HTML。</p>
			{/if}
			{#if pptError}
				<p class="ppt-err">{pptError}</p>
			{/if}
		</div>
	</div>
	<div class="graph-foot">
		<div><strong>{slotCards.filter(slot => slot.status === 'present').length}</strong><span>ready slots</span></div>
		<div><strong>{specMeta.status}</strong><span>status</span></div>
		<div><strong>{specMeta.parent ?? 'root'}</strong><span>parent</span></div>
	</div>
</div>

<style>
	.generic-graph {
		height: 100%;
		min-height: 280px;
		display: flex;
		flex-direction: column;
		padding: 12px;
		background:
			linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px),
			#0e1016;
		background-size: 26px 26px;
		color: #eef0f5;
	}

	.graph-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		margin-bottom: 12px;
		padding: 13px 15px;
		border: 1px solid #303746;
		border-radius: 16px;
		background: rgba(28, 32, 42, 0.78);
	}

	.graph-head h2 {
		margin: 0 0 5px;
		font-size: 15px;
		font-weight: 840;
		letter-spacing: -0.02em;
	}

	.graph-head p {
		margin: 0;
		color: #a3abb9;
		font-size: 12px;
		line-height: 1.45;
	}

	.eyebrow {
		display: block;
		margin-bottom: 6px;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	.mode-badge {
		flex-shrink: 0;
		border: 1px solid #465064;
		border-radius: 999px;
		padding: 5px 10px;
		color: #7aa2ff;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 11px;
		font-weight: 800;
		text-transform: uppercase;
	}

	.head-actions {
		display: inline-flex;
		align-items: center;
		gap: 8px;
	}

	.attach-head {
		border: 1px solid rgba(114, 214, 208, 0.5);
		border-radius: 999px;
		background: rgba(114, 214, 208, 0.12);
		color: #dffcf9;
		font-size: 11px;
		font-weight: 700;
		padding: 6px 10px;
		cursor: pointer;
	}

	.radial {
		position: relative;
		flex: 1;
		min-height: 420px;
		border: 1px solid #303746;
		border-radius: 16px;
		background: rgba(12, 14, 19, 0.74);
		overflow: hidden;
	}

	.edge {
		position: absolute;
		height: 1.5px;
		background: #465064;
		transform: translateY(-50%);
		pointer-events: none;
	}

	.edge::after {
		content: '';
		position: absolute;
		right: -1px;
		top: 50%;
		width: 0;
		height: 0;
		border-top: 4px solid transparent;
		border-bottom: 4px solid transparent;
		border-left: 7px solid #465064;
		transform: translateY(-50%);
	}

	.parent-edge {
		top: 50%;
	}

	.relation-card {
		position: absolute;
		transform: translate(-50%, -50%);
		width: 142px;
		padding: 0.48rem 0.62rem;
		font-size: 10px;
		line-height: 1.2;
		border-radius: 12px;
		border: 1px solid #465064;
		background: #171b24;
		color: #d4d4d8;
		text-align: left;
		pointer-events: none;
	}

	.ppt-center {
		position: absolute;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		width: min(90%, 980px);
		height: min(84%, 620px);
		padding: 12px;
		border-radius: 16px;
		border: 1px solid #3b465d;
		background: rgba(10, 12, 17, 0.94);
		box-shadow: 0 20px 70px rgba(0, 0, 0, 0.45);
		display: flex;
		flex-direction: column;
		gap: 8px;
		z-index: 2;
	}

	.fs-fab {
		position: absolute;
		top: 10px;
		right: 10px;
		width: 30px;
		height: 30px;
		border-radius: 999px;
		border: 1px solid rgba(122, 162, 255, 0.65);
		background: rgba(12, 14, 19, 0.7);
		color: #e8f0ff;
		font-size: 15px;
		line-height: 1;
		cursor: pointer;
		z-index: 4;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.fs-fab:hover {
		background: rgba(122, 162, 255, 0.2);
	}

	.ppt-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}

	.ppt-actions {
		display: inline-flex;
		gap: 6px;
	}

	.ppt-actions button {
		padding: 3px 8px;
		border-radius: 6px;
		border: 1px solid rgba(114, 214, 208, 0.45);
		background: rgba(114, 214, 208, 0.12);
		color: #dffcf9;
		font-size: 11px;
		cursor: pointer;
	}

	.ppt-actions button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.ppt-actions .gen-btn {
		border-color: rgba(122, 162, 255, 0.7);
		background: rgba(122, 162, 255, 0.2);
		color: #e8f0ff;
		font-weight: 800;
	}

	.ppt-path {
		color: #8aa4c7;
		font-size: 10px;
		font-family: ui-monospace, monospace;
		word-break: break-all;
	}

	.ppt-frame {
		flex: 1;
		width: 100%;
		border: 1px solid #303746;
		border-radius: 8px;
		background: #0a0b0f;
	}

	.ppt-nav-fab {
		position: absolute;
		right: 14px;
		bottom: 12px;
		z-index: 4;
		display: inline-flex;
		gap: 8px;
	}

	.ppt-nav-fab button {
		width: 34px;
		height: 34px;
		border-radius: 999px;
		border: 1px solid rgba(122, 162, 255, 0.65);
		background: rgba(12, 14, 19, 0.75);
		color: #e8f0ff;
		cursor: pointer;
		font-size: 14px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.ppt-nav-fab button:hover {
		background: rgba(122, 162, 255, 0.2);
	}

	.ppt-loading {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: 12px;
		color: #a3abb9;
	}

	.spinner {
		width: 12px;
		height: 12px;
		border: 2px solid rgba(122, 162, 255, 0.25);
		border-top-color: #7aa2ff;
		border-radius: 50%;
		animation: spin 0.9s linear infinite;
	}

	.ppt-empty {
		margin: 0;
		color: #858fa1;
		font-size: 12px;
	}

	.ppt-err {
		margin: 0;
		color: #f87171;
		font-size: 12px;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.relation-card {
		border-style: dashed;
		background: rgba(23, 27, 36, 0.72);
	}

	.relation-card .k {
		display: block;
		margin-bottom: 4px;
		color: #6f7888;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 9px;
		text-transform: uppercase;
	}

	.relation-card strong {
		display: block;
		color: #eef0f5;
		font-size: 11px;
		font-weight: 800;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.graph-foot {
		flex-shrink: 0;
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 10px;
		margin-top: 12px;
	}

	.graph-foot div {
		min-width: 0;
		border: 1px solid #303746;
		border-radius: 12px;
		padding: 10px;
		background: rgba(28, 32, 42, 0.75);
	}

	.graph-foot strong,
	.graph-foot span {
		display: block;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.graph-foot strong {
		color: #eef0f5;
		font-size: 12px;
		margin-bottom: 4px;
	}

	.graph-foot span {
		color: #a3abb9;
		font-size: 11px;
	}
</style>
