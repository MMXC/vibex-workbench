<!-- 非 L1 总目标时的轻量「图谱」：中心为 spec，周围为 canonical 槽位卡片 -->
<script lang="ts">
	import { get } from 'svelte/store';
	import { parse as parseYaml } from 'yaml';
	import { specAgentContextStore } from '$lib/stores/spec-agent-context-store';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { wailsReadSpecFile, wailsWriteSpecFile } from '$lib/wails-filesystem';
	import {
		extractSpecDisplay,
		getSpecPptFileFromYaml,
		upsertSpecBlockPptFile,
	} from '$lib/workbench/spec-display';
	import {
		appendChildrenPaths,
		collectL3CandidatesFromL2,
		collectL5CandidatesFromL4,
		extractSpecName,
		inferSpawnLayer,
		parseExistingChildPaths,
		type ChildSpawnCandidate,
	} from '$lib/workbench/spawn-child-specs';
	import {
		collectChildPathsForParent,
		collectParentRefs,
		resolveSpecRefToPath,
	} from '$lib/workbench/spec-lineage';
	import type { ConventionPayload } from '$lib/workbench/spec-convention';
	import { specChildDraftStore } from '$lib/stores/spec-child-draft-store';
	import { agentApiUrl } from '$lib/runtime/agent-transport';
	import { toWorkspaceFileURL } from '$lib/workbench/workspace-file-url';

	let {
		specPath,
		content,
		onSpawnComplete,
		convention = null,
	}: {
		specPath: string;
		content: string;
		onSpawnComplete?: () => void;
		convention?: ConventionPayload['convention'] | null;
	} = $props();

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

	let currentSpecContent = $state('');
	let autoGenerateTriggered = $state(false);
	let lastAutoRefreshKey = $state('');
	let pptPath = $state<string>('');
	let pptSrc = $state<string>('');
	let pptLoading = $state(false);
	let pptError = $state<string | null>(null);
	let pptTaskState = $state<'idle' | 'queued' | 'running'>('idle');
	let pptContainerEl = $state<HTMLElement | null>(null);
	let pptFullscreen = $state(false);
	let copyQuestionTip = $state('');
	let currentPptHtml = $state('');
	let questionMenuOpen = $state(false);

	let spawnErr = $state<string | null>(null);
	let spawnOk = $state<string | null>(null);

	/** 左侧「父链」栏：默认收起 */
	let parentRailExpanded = $state(false);
	/** 右侧「子血缘」栏：默认收起 */
	let lineageRailExpanded = $state(false);

	let explorerSpecs = $state<
		{ path: string; parent?: string | null; display?: { title?: string } }[]
	>([]);

	let mountBusy = $state(false);
	$effect(() => {
		return specExplorerStore.subscribe(s => {
			explorerSpecs = s.specs;
		});
	});

	const yamlForSpawn = $derived.by(() => currentSpecContent || content);

	const spawnLayer = $derived.by(() => inferSpawnLayer(yamlForSpawn));

	const spawnCandidates = $derived.by((): ChildSpawnCandidate[] => {
		const layer = spawnLayer;
		const name = extractSpecName(yamlForSpawn);
		if (!layer || !name) return [];
		if (layer === 'L2') return collectL3CandidatesFromL2(yamlForSpawn, name);
		return collectL5CandidatesFromL4(yamlForSpawn, name);
	});

	/** 与当前父 spec 同级的子节点 spec.name：已挂载 + children 列表中的路径解析名，用于推断列不重名 */
	const inferredChildCandidates = $derived.by((): ChildSpawnCandidate[] => {
		const raw = spawnCandidates.filter(c => !c.alreadyLinked);

		const occupiedNames = new Set<string>();
		for (const c of spawnCandidates) {
			if (c.alreadyLinked) occupiedNames.add(c.specName);
		}
		const childPaths = [...new Set(parseExistingChildPaths(yamlForSpawn))];
		for (const p of childPaths) {
			const hit = explorerSpecs.find(x => normSpecPath(x.path) === normSpecPath(p));
			const nm = (hit?.name ?? '').trim() || specBasename(p);
			if (nm) occupiedNames.add(nm);
		}

		const seenInfer = new Set<string>();
		const out: ChildSpawnCandidate[] = [];
		for (const c of raw) {
			if (occupiedNames.has(c.specName)) continue;
			if (seenInfer.has(c.specName)) continue;
			seenInfer.add(c.specName);
			out.push(c);
		}
		return out;
	});

	function normSpecPath(p: string): string {
		return p.replace(/\\/g, '/').trim();
	}

	function specBasename(path: string): string {
		return (
			path
				.replace(/\\/g, '/')
				.split('/')
				.pop()
				?.replace(/\.ya?ml$/i, '') ?? path
		);
	}

	function lookupSpecTitle(path: string): string {
		const n = normSpecPath(path);
		const hit = explorerSpecs.find(x => normSpecPath(x.path) === n);
		const t = hit?.display?.title?.trim();
		return t || specBasename(path);
	}

	const specNameCatalog = $derived.by(() =>
		explorerSpecs.map(x => ({
			path: x.path,
			name: (x.name ?? '').trim() || specBasename(x.path),
		}))
	);

	const parentCards = $derived.by(() => {
		const refs = collectParentRefs(yamlForSpawn);
		return refs.map(ref => {
			const path = resolveSpecRefToPath(ref, convention ?? null, specNameCatalog);
			return {
				ref,
				path,
				title: path ? lookupSpecTitle(path) : ref,
			};
		});
	});

	const existingChildCards = $derived.by(() => {
		const paths = [...new Set(parseExistingChildPaths(yamlForSpawn))];
		return paths.map(path => ({
			path,
			title: lookupSpecTitle(path),
		}));
	});

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
		return getSpecPptFileFromYaml(yamlText);
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


	function delay(ms: number): Promise<void> {
		return new Promise(resolve => setTimeout(resolve, ms));
	}

	function ensureTag(source: string, needle: string, insertBefore: string): string {
		if (source.includes(needle)) return source;
		const idx = source.indexOf(insertBefore);
		if (idx >= 0) {
			return source.slice(0, idx) + `${needle}\n` + source.slice(idx);
		}
		return `${source}\n${needle}\n`;
	}

	function postValidateAndRepairPptHtml(html: string): { content: string; changed: boolean; notes: string[] } {
		let out = html;
		let changed = false;
		const notes: string[] = [];
		const ensure = (next: string, note: string) => {
			if (next !== out) {
				out = next;
				changed = true;
				notes.push(note);
			}
		};

		ensure(
			ensureTag(out, '<link rel="stylesheet" href="../assets/base.css">', '</head>'),
			'补齐 base.css'
		);
		ensure(
			ensureTag(out, '<link rel="stylesheet" href="../assets/fonts.css">', '</head>'),
			'补齐 fonts.css'
		);
		ensure(
			ensureTag(out, '<script src="../assets/runtime.js"><\\/script>', '</body>'),
			'补齐 runtime.js'
		);
		ensure(
			ensureTag(out, '<script src="../assets/vibex-ppt-nav.js"><\\/script>', '</body>'),
			'补齐 vibex-ppt-nav.js'
		);

		if (!out.includes('<main class="deck">') && out.includes('<section class="slide')) {
			out = out.replace(/<body([^>]*)>/i, '<body$1>\n<main class="deck">');
			if (out.includes('<script src="../assets/runtime.js"><\\/script>')) {
				out = out.replace(
					'<script src="../assets/runtime.js"><\\/script>',
					'</main>\n<script src="../assets/runtime.js"><\\/script>'
				);
			} else {
				out = out.replace('</body>', '</main>\n</body>');
			}
			changed = true;
			notes.push('补齐 .deck 容器');
		}

		return { content: out, changed, notes };
	}

	function buildDivergentQuestionPack(): string {
		return [
			`[Spec] ${specMeta.display.title}`,
			`[Path] ${specPath}`,
			'',
			'请围绕该 spec 做发散式完善，并明确 why / why-not：',
			'1) 这个方案最关键的业务价值是什么？有没有更低成本达到同样价值的路径？',
			'2) 当前 as-is 与 to-be 的最大差距是什么？哪个差距最值得优先投入？为什么？',
			'3) 推荐方案与备选方案的取舍边界是什么？在什么条件下应该切换方案？',
			'4) 对 CTO/开发/测试/用户四类角色，最容易被忽略的风险分别是什么？',
			'5) 如果要把当前 spec 拆成更小的 L5 slices，最合理的切分维度是什么？',
			'6) 验证路径是否足够？缺少哪些可量化验收指标（性能/稳定性/可恢复性）？',
			'7) 若当前实现失败，回滚策略是否可执行？回滚后如何保证数据/状态一致？',
			'8) 还有哪些“现在不做”的内容需要被明确记录，避免范围蔓延？',
		].join('\n');
	}

	function stripTags(input: string): string {
		return input
			.replace(/<style[\s\S]*?<\/style>/gi, '')
			.replace(/<script[\s\S]*?<\/script>/gi, '')
			.replace(/<[^>]+>/g, ' ')
			.replace(/\s+/g, ' ')
			.trim();
	}

	function collectMatches(re: RegExp, text: string, max = 6): string[] {
		const out: string[] = [];
		let m: RegExpExecArray | null = null;
		const rx = new RegExp(re.source, re.flags.includes('g') ? re.flags : re.flags + 'g');
		while ((m = rx.exec(text)) !== null) {
			const v = stripTags(m[1] ?? '');
			if (!v) continue;
			if (!out.includes(v)) out.push(v);
			if (out.length >= max) break;
		}
		return out;
	}

	function buildDynamicQuestionPackFromPptHtml(html: string): string {
		const clean = html || '';
		const titles = collectMatches(/<h[1-3][^>]*>([\s\S]*?)<\/h[1-3]>/gi, clean, 8);
		const strongs = collectMatches(/<strong[^>]*>([\s\S]*?)<\/strong>/gi, clean, 8);
		const codes = collectMatches(/<code[^>]*>([\s\S]*?)<\/code>/gi, clean, 8);
		const riskHints = (clean.match(/risk|风险|回滚|rollback|fallback/gi) || []).length;
		const asisTobeHints = (clean.match(/as-is|to-be|现状|目标|差距/gi) || []).length;
		const audienceHints = (clean.match(/CTO|开发|测试|用户/gi) || []).length;
		const slotsHints = (clean.match(/slot|插槽|implementation|实现/gi) || []).length;

		const focusA = titles[0] || specMeta.display.title;
		const focusB = titles[1] || strongs[0] || '核心方案';
		const keyPath = codes[0] || specPath;
		const doc = (parseYaml(currentSpecContent || '') ?? {}) as Record<string, unknown>;
		const spec = (doc.spec ?? {}) as Record<string, unknown>;
		const io = (spec.io ?? {}) as Record<string, unknown>;
		const ioContract = (spec.io_contract ?? {}) as Record<string, unknown>;
		const constraints = Array.isArray(spec.constraints) ? spec.constraints : [];
		const prototype = (doc.prototype ?? {}) as Record<string, unknown>;
		const slots = ((doc as Record<string, unknown>).slots ??
			(spec.slots ?? (spec.implementation_slots ?? []))) as unknown;
		const slotCount = Array.isArray(slots) ? slots.length : 0;
		const inputHint = String(io.input ?? ioContract.input ?? '').trim() || '（未显式填写）';
		const outputHint = String(io.output ?? ioContract.output ?? '').trim() || '（未显式填写）';
		const boundaryHint = String(io.boundary ?? ioContract.boundary ?? '').trim() || '（未显式填写）';
		const behaviorHint = String(io.behavior ?? ioContract.behavior ?? '').trim() || '（未显式填写）';
		const pptDemo = getSpecPptFileFromYaml(currentSpecContent || '');
		const protoFile = String(prototype.file ?? prototype.path ?? '').trim();
		const protoHint =
			[protoFile ? `prototype.file: ${protoFile}` : '', pptDemo ? `spec.ppt_file（演示）: ${pptDemo}` : '']
				.filter(Boolean)
				.join('；') || '（未显式填写）';

		return [
			`[Spec] ${specMeta.display.title}`,
			`[Path] ${specPath}`,
			`[PPT] ${pptPath || '(unknown)'}`,
			'',
			'以下问题由“当前PPT + 详情抽屉结构字段”联合生成，请在详情抽屉继续和 agent 深挖：',
			'',
			'【输入（Input）】',
			`1) 当前输入定义为：${inputHint}。它是否足以覆盖主流程与异常流程？还缺哪些触发条件或前置状态？`,
			`2) PPT「${focusA}」中的案例与输入定义是否一一对应？哪些输入在 PPT 里被忽略了？`,
			'',
			'【输出（Output）】',
			`3) 当前输出定义为：${outputHint}。输出是否可验证、可观测、可回归？对应验收口径是什么？`,
			`4) 若输出失败，是否有降级输出/错误输出规范？PPT 中是否给出了 why-not 方案与代价？`,
			'',
			'【约束（Constraints）】',
			`5) 当前约束数量：${constraints.length}；边界：${boundaryHint}；行为：${behaviorHint}。这些约束是否可执行而非口号？`,
			`6) 结合 PPT 中 as-is/to-be/gap（命中 ${asisTobeHints} 次），哪些约束会阻塞实现？优先级应如何调整？`,
			`7) 关键风险线索命中 ${riskHints} 次：是否已转化为“可验证约束 + 回滚条件”？`,
			'',
			'【原型（Prototype）】',
			`8) 当前原型字段为：${protoHint}。原型与 spec 的输入/输出/约束是否一致？哪些页面只展示了“是什么”但缺“为什么不那样做”？`,
			`9) 对比线索（如 ${keyPath}）是否应写入 prototype 注释或 spec 追溯字段，便于团队复盘？`,
			'',
			'【实现插槽（Implementation Slots）】',
			`10) 当前实现插槽数量：${slotCount}（PPT 实现相关命中 ${slotsHints} 次）。插槽边界是否清晰到可直接分配给 agent/开发？`,
			`11) 每个插槽是否具备“输入→处理→输出→验证→回滚”闭环？如果没有，优先补哪一段？`,
			`12) 请基于「输入/输出/约束/原型/插槽」五维，给出一版可落地的 spec 调整清单（字段级变更 + 验收标准）。`,
			'',
			`补充视角：四类角色覆盖命中 ${audienceHints} 次，建议分别给出 CTO/开发/测试/用户的最小验收问答。`,
			`对比焦点：围绕「${focusB}」补齐推荐方案 vs 备选方案的 why / why-not 与切换条件。`,
		].join('\n');
	}

	type QuestionSectionKey = 'input' | 'output' | 'constraints' | 'prototype' | 'slots';

	function extractQuestionSections(fullText: string): Record<QuestionSectionKey, string> {
		const lines = fullText.split('\n');
		const sections: Record<QuestionSectionKey, string[]> = {
			input: [],
			output: [],
			constraints: [],
			prototype: [],
			slots: [],
		};

		let active: QuestionSectionKey | null = null;
		for (const raw of lines) {
			const line = raw.trim();
			if (line === '【输入（Input）】') active = 'input';
			else if (line === '【输出（Output）】') active = 'output';
			else if (line === '【约束（Constraints）】') active = 'constraints';
			else if (line === '【原型（Prototype）】') active = 'prototype';
			else if (line === '【实现插槽（Implementation Slots）】') active = 'slots';
			else if (line.startsWith('补充视角：') || line.startsWith('对比焦点：')) active = null;

			if (active && line) sections[active].push(raw);
		}

		const title = `# ${specMeta.display.title}\n# ${specPath}\n`;
		return {
			input: `${title}\n${sections.input.join('\n')}`.trim(),
			output: `${title}\n${sections.output.join('\n')}`.trim(),
			constraints: `${title}\n${sections.constraints.join('\n')}`.trim(),
			prototype: `${title}\n${sections.prototype.join('\n')}`.trim(),
			slots: `${title}\n${sections.slots.join('\n')}`.trim(),
		};
	}

	async function copyDivergentQuestions() {
		const text = buildDivergentQuestionPack();
		try {
			await navigator.clipboard.writeText(text);
			copyQuestionTip = '已复制发散问题';
		} catch {
			copyQuestionTip = '复制失败，请检查剪贴板权限';
		}
		window.setTimeout(() => {
			copyQuestionTip = '';
		}, 1800);
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

	async function copyDynamicQuestionsFromCurrentPpt() {
		const text = buildDynamicQuestionPackFromPptHtml(currentPptHtml);
		try {
			await navigator.clipboard.writeText(text);
			copyQuestionTip = '已复制动态问题清单';
		} catch {
			copyQuestionTip = '复制失败，请检查剪贴板权限';
		}
		window.setTimeout(() => {
			copyQuestionTip = '';
		}, 1800);
	}

	async function copyQuestionSection(key: QuestionSectionKey) {
		const all = buildDynamicQuestionPackFromPptHtml(currentPptHtml);
		const sections = extractQuestionSections(all);
		const text = sections[key];
		if (!text.trim()) {
			copyQuestionTip = '当前PPT未提取到该维度问题';
			window.setTimeout(() => {
				copyQuestionTip = '';
			}, 1800);
			return;
		}
		try {
			await navigator.clipboard.writeText(text);
			const labels: Record<QuestionSectionKey, string> = {
				input: '输入',
				output: '输出',
				constraints: '约束',
				prototype: '原型',
				slots: '实现插槽',
			};
			copyQuestionTip = `已复制${labels[key]}问题`;
			questionMenuOpen = false;
		} catch {
			copyQuestionTip = '复制失败，请检查剪贴板权限';
		}
		window.setTimeout(() => {
			copyQuestionTip = '';
		}, 1800);
	}

	function waitForAgentCompletion(
		threadId: string,
		onState: (state: 'queued' | 'running') => void,
		timeoutMs = 120000
	): Promise<void> {
		return new Promise((resolve, reject) => {
			const es = new EventSource(agentApiUrl(`/api/sse/${encodeURIComponent(threadId)}`));
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
			pptError = '当前 spec 未配置 spec.ppt_file，已尝试自动生成，请点击「生成/重生成」重试。';
			return;
		}
		pptLoading = true;
		pptError = null;
		try {
			const html = await wailsReadSpecFile(wsRoot, configured);
			const content = String(html ?? '');
			currentPptHtml = content;
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
			const templateFile = '.agents/ppt-layer-templates.yaml';
			const prompt = [
				'你是 VibeX Workbench 的原型生成代理（针对当前打开的工作区）。',
				'必须使用 html-ppt skill 生成可运行的 spec 说明型 PPT（HTML）。',
				`spec_path: ${specPath}`,
				`spec_level: ${levelTag}`,
				`visual_theme_primary: ${layerConfig.primaryTheme}`,
				`visual_theme_candidates: ${layerConfig.themeCandidates.join(', ')}`,
				`work_root_dir: ${wsRoot}`,
				`目标输出文件: ${target}`,
				'硬约束：仅写目标文件；保持 html-ppt 分页 deck（6~10 页）；统一主题；不是长文档。',
				'资源硬约束：head 中必须包含 ../assets/base.css 与 ../assets/fonts.css，再加载主题与 animations.css。',
				'HTML 结构硬约束：所有 <section class="slide"> 必须包裹在 <main class="deck">...</main> 中，否则视为不合格。',
				'必须在末尾包含：<script src="../assets/runtime.js"><\\/script> 与 <script src="../assets/vibex-ppt-nav.js"><\\/script>。',
				'讲解硬约束：技术类说明优先使用结构图/流程图、对比表格（方案A/B/不选理由）、思维导图式分层讲解，不要长段落。',
				'决策硬约束：每个关键点必须回答 why（为何做）与 why-not（为何不做其他方案），并标注前提条件与切换条件。',
				'发散引导硬约束：结尾必须给出“可复制的问题清单”（8~12条），用于用户在详情抽屉继续和 agent 对话完善 spec。',
				'内容必须覆盖：as-is 现状、to-be 目标、偏差、优化方向、验证与风险、决策与执行。',
				'受众必须覆盖：CTO/开发/测试/用户。',
				`层级模板规则请读取并遵循：${templateFile}`,
				'当前层级模板摘要：',
				...layerConfig.lines.slice(0, 12).map((line, idx) => `${idx + 1}. ${line}`),
				'若存在 spec.ppt_file（演示 HTML）或 prototype.file 且文件存在，必须加“原型现状”页。',
				'尽量引用真实路径/接口名，便于追溯。',
				'不要依赖本提示内嵌的大段 YAML；请优先用 read_file 读取 spec_path 获取完整内容。',
				'如果文件较大，分段读取并聚焦关键信息（spec/meta/io/io_contract/content/constraints/changelog/prototype；演示稿路径见 spec.ppt_file）。',
				'完成后返回 done。',
				'上下文提示（非权威，仅供快速定位）：',
				`spec_title_hint: ${specMeta.display.title}`,
				`spec_summary_hint: ${specMeta.display.summary}`,
			].join('\n');
			await fetch(agentApiUrl('/api/agent/execute'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					threadId,
					input: prompt,
					workspaceRoot: wsRoot,
					workRootDir: wsRoot,
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
			const repaired = postValidateAndRepairPptHtml(html);
			if (repaired.changed) {
				await wailsWriteSpecFile(wsRoot, target, repaired.content);
				html = repaired.content;
			}
			currentPptHtml = html;
			pptSrc = toWorkspaceFileURL(target, wsRoot);
			const patched = upsertSpecBlockPptFile(currentSpecContent, target);
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

	function navigateToSpec(targetPath: string | null | undefined, labelForErr: string) {
		spawnErr = null;
		if (!targetPath) {
			spawnErr = `无法解析路径：${labelForErr}`;
			return;
		}
		specExplorerStore.selectSpec(normSpecPath(targetPath));
	}

	function relSpecsPathForWrite(p: string): string {
		const norm = p.replace(/\\/g, '/');
		const i = norm.indexOf('specs/');
		return i >= 0 ? norm.slice(i) : norm;
	}

	/** 扫描全库 spec.parent === 当前 spec.name，合并路径到 structure.children */
	async function syncChildrenFromRepoByParent() {
		spawnErr = null;
		spawnOk = null;
		const parentName = extractSpecName(yamlForSpawn);
		if (!parentName) {
			spawnErr = '无法解析当前 spec.name';
			return;
		}
		const wsRoot = get(specExplorerStore).workspaceRoot;
		if (!wsRoot) {
			spawnErr = '未绑定 workspace root';
			return;
		}
		mountBusy = true;
		try {
			await specExplorerStore.loadList(wsRoot);
			const specs = get(specExplorerStore).specs;
			const discovered = collectChildPathsForParent(specs, parentName, {
				excludeSelfPath: specPath,
			});
			if (discovered.length === 0) {
				spawnOk = '未发现 spec.parent 指向当前名称的子 spec（已刷新列表）。';
				return;
			}
			const relParent = relSpecsPathForWrite(specPath);
			const patched = appendChildrenPaths(yamlForSpawn, discovered);
			await wailsWriteSpecFile(wsRoot, relParent, patched);
			currentSpecContent = patched;
			await specExplorerStore.loadList(wsRoot);
			spawnOk = `已合并 ${discovered.length} 条子 spec 路径到 structure.children`;
			onSpawnComplete?.();
		} catch (e) {
			spawnErr = e instanceof Error ? e.message : String(e);
		} finally {
			mountBusy = false;
		}
	}

	async function openChildDraft(c: ChildSpawnCandidate) {
		const layer = spawnLayer;
		if (!layer) return;
		spawnErr = null;
		spawnOk = null;
		try {
			await specChildDraftStore.openFromCandidate({
				parentSpecPath: specPath,
				parentYaml: yamlForSpawn,
				candidate: c,
				layer,
				onDone: () => {
					spawnOk = `已创建 ${c.relativePath}`;
					onSpawnComplete?.();
				},
			});
		} catch (e) {
			spawnErr = e instanceof Error ? e.message : String(e);
		}
	}

	$effect(() => {
		specPath;
		content;
		spawnErr = null;
		spawnOk = null;
	});

	/** 换文件时收起两侧血缘栏 */
	$effect(() => {
		specPath;
		lineageRailExpanded = false;
		parentRailExpanded = false;
	});

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

	<div class="graph-main">
		<aside
			class="parent-rail"
			class:parent-rail--expanded={parentRailExpanded}
			aria-label="父规格血缘侧栏"
		>
			{#if parentRailExpanded}
				<div class="lineage-panel-inner parent-panel-inner">
					<div class="spawn-panel-head">
						<strong>父链（平面）</strong>
						<span class="spawn-badge spawn-badge--muted">上游</span>
					</div>
					<p class="lineage-intro">
						列出 <code>spec.parent</code>、<code>structure.parent</code>、<code>structure.dependencies</code>
						全部引用，不做层级展开。
					</p>
					{#if parentCards.length === 0}
						<p class="spawn-muted">当前 spec 未声明上游引用。</p>
					{:else}
						<div class="lineage-cards">
							{#each parentCards as p (p.ref)}
								<button
									type="button"
									class="lineage-card"
									class:lineage-card--warn={!p.path}
									onclick={() => navigateToSpec(p.path, p.ref)}
								>
									<strong>{p.title}</strong>
									<small>{p.ref}</small>
									{#if !p.path}
										<span class="spawn-tag">路径未解析</span>
									{/if}
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{/if}
			<button
				type="button"
				class="parent-tab"
				onclick={() => (parentRailExpanded = !parentRailExpanded)}
				aria-expanded={parentRailExpanded}
				title={parentRailExpanded ? '收起父链栏' : '展开父链栏'}
			>
				{#if parentRailExpanded}
					<span class="lineage-tab-chev" aria-hidden="true">‹</span>
				{:else}
					<span class="lineage-tab-v" aria-hidden="true">父链</span>
				{/if}
			</button>
		</aside>

		<div class="radial">
		<div class="ppt-center" bind:this={pptContainerEl}>
			<div class="ppt-head">
				<strong>Spec HTML PPT 演示</strong>
				<div class="ppt-actions">
					<button type="button" onclick={refreshPpt} disabled={pptLoading}>刷新</button>
					<button type="button" onclick={copyDivergentQuestions} disabled={pptLoading}>复制发散问题</button>
					<button
						type="button"
						onclick={copyDynamicQuestionsFromCurrentPpt}
						disabled={pptLoading || !currentPptHtml}
					>
						从当前PPT抽取问题
					</button>
					<div class="q-menu-wrap">
						<button
							type="button"
							onclick={() => (questionMenuOpen = !questionMenuOpen)}
							disabled={pptLoading || !currentPptHtml}
						>
							按五维复制
						</button>
						{#if questionMenuOpen}
							<div class="q-menu">
								<button type="button" onclick={() => copyQuestionSection('input')}>输入</button>
								<button type="button" onclick={() => copyQuestionSection('output')}>输出</button>
								<button type="button" onclick={() => copyQuestionSection('constraints')}>约束</button>
								<button type="button" onclick={() => copyQuestionSection('prototype')}>原型</button>
								<button type="button" onclick={() => copyQuestionSection('slots')}>实现插槽</button>
							</div>
						{/if}
					</div>
					<button type="button" class="gen-btn" onclick={generatePpt} disabled={pptLoading}>
						{genButtonLabel}
					</button>
				</div>
			</div>
			{#if copyQuestionTip}
				<small class="ppt-tip">{copyQuestionTip}</small>
			{/if}
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
							Agent 正在生成 spec.ppt_file…
						{:else}
							正在加载 spec.ppt_file…
						{/if}
					</span>
				</div>
			{:else if pptSrc}
				<iframe
					class="ppt-frame"
					title="spec ppt demo preview"
					sandbox="allow-scripts allow-same-origin"
					referrerpolicy="no-referrer"
					src={pptSrc}
				></iframe>
			{:else}
				<p class="ppt-empty">未找到可展示 HTML。</p>
			{/if}
			{#if pptError}
				<p class="ppt-err">{pptError}</p>
			{/if}
		</div>
		</div>

		<aside
			class="lineage-rail"
			class:lineage-rail--expanded={lineageRailExpanded}
			aria-label="规格血缘侧栏"
		>
			<button
				type="button"
				class="lineage-tab"
				onclick={() => (lineageRailExpanded = !lineageRailExpanded)}
				aria-expanded={lineageRailExpanded}
				title={lineageRailExpanded ? '收起子血缘栏' : '展开子血缘栏'}
			>
				{#if lineageRailExpanded}
					<span class="lineage-tab-chev" aria-hidden="true">›</span>
				{:else}
					<span class="lineage-tab-v" aria-hidden="true">子链</span>
				{/if}
			</button>
			{#if lineageRailExpanded}
				<div class="lineage-panel-inner">
					<div class="spawn-panel-head">
						<strong>子链</strong>
						{#if spawnLayer}
							<span class="spawn-badge">{spawnLayer === 'L2' ? 'L2→L3' : 'L4→L5'}</span>
						{:else}
							<span class="spawn-badge spawn-badge--muted">{levelShort(level)}</span>
						{/if}
					</div>
					<p class="lineage-intro">
						已挂载子 spec 可点击跳转；推断项打开抽屉修订 YAML 后落盘。
					</p>

					<div class="mount-toolbar">
						<button
							type="button"
							class="mount-btn"
							onclick={() => void syncChildrenFromRepoByParent()}
							disabled={mountBusy}
						>
							{mountBusy ? '扫描中…' : '按 parent 挂载到 children'}
						</button>
					</div>
					<p class="mount-hint">
						遍历仓库内 <code>spec.parent</code> = 当前 <code>spec.name</code> 的子 spec，合并相对路径到
						<code>structure.children</code>（去重，不删已有项）。
					</p>

					<div class="spawn-section-head">
						<strong>已有子 spec</strong>
					</div>
					{#if existingChildCards.length === 0}
						<p class="spawn-muted"><code>structure.children</code> 为空。</p>
					{:else}
						<div class="lineage-cards">
							{#each existingChildCards as cc (cc.path)}
								<button
									type="button"
									class="lineage-card"
									onclick={() => navigateToSpec(cc.path, cc.path)}
								>
									<strong>{cc.title}</strong>
									<small>{cc.path}</small>
								</button>
							{/each}
						</div>
					{/if}

					{#if spawnLayer}
						<div class="spawn-section-head">
							<strong>推断 · 待创建</strong>
						</div>
						{#if inferredChildCandidates.length === 0}
							<p class="spawn-muted">
								当前 YAML 无未挂载推断项；或需 L2 的 <code>which_modules_become_l3</code> / L4 的
								<code>content.behaviors</code>。
							</p>
						{:else}
							<div class="lineage-cards">
								{#each inferredChildCandidates as c (c.id)}
									<button
										type="button"
										class="lineage-card lineage-card--draft"
										onclick={() => void openChildDraft(c)}
									>
										<span class="draft-badge">推断</span>
										<strong>{c.specName}</strong>
										<small>{c.relativePath}</small>
										<small class="card-hint">{c.summaryHint}</small>
									</button>
								{/each}
							</div>
						{/if}
					{:else}
						<p class="spawn-muted">
							仅 L2 / L4 可从正文推断待创建子 spec；当前为 {levelShort(level)}。
						</p>
					{/if}

					{#if spawnErr}
						<p class="spawn-err">{spawnErr}</p>
					{/if}
					{#if spawnOk}
						<p class="spawn-ok">{spawnOk}</p>
					{/if}
				</div>
			{/if}
		</aside>
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

	.graph-main {
		flex: 1;
		display: flex;
		flex-direction: row;
		align-items: stretch;
		min-height: 0;
		gap: 10px;
	}

	.parent-rail {
		flex-shrink: 0;
		display: flex;
		flex-direction: row;
		align-items: stretch;
		align-self: stretch;
		border-radius: 16px;
		border: 1px solid #303746;
		background: rgba(18, 22, 30, 0.92);
		overflow: hidden;
		transition: width 0.18s ease;
	}

	.parent-rail:not(.parent-rail--expanded) {
		width: 44px;
	}

	.parent-rail.parent-rail--expanded {
		width: min(300px, 36vw);
		max-width: 100%;
	}

	.parent-tab {
		flex-shrink: 0;
		width: 44px;
		border: none;
		border-left: 1px solid #303746;
		background: rgba(28, 32, 42, 0.95);
		color: #c7d2fe;
		cursor: pointer;
		padding: 8px 0;
		font-family: inherit;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.parent-tab:hover {
		background: rgba(122, 162, 255, 0.12);
		color: #eef0f5;
	}

	.parent-panel-inner {
		flex: 1;
		min-width: 0;
	}

	.lineage-cards {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.lineage-card {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 4px;
		text-align: left;
		border: 1px solid #3b465d;
		border-radius: 10px;
		padding: 8px 10px;
		background: rgba(12, 14, 18, 0.65);
		color: #eef0f5;
		cursor: pointer;
		font-family: inherit;
		font-size: 12px;
	}

	.lineage-card:hover {
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.08);
	}

	.lineage-card strong {
		font-size: 12px;
		font-weight: 700;
		color: #eef0f5;
	}

	.lineage-card small {
		font-size: 10px;
		color: #858fa1;
		word-break: break-all;
	}

	.card-hint {
		display: block;
		margin-top: 2px;
		line-height: 1.35;
	}

	.lineage-card--warn {
		opacity: 0.88;
		border-color: #b45309;
	}

	.lineage-card--draft {
		border-style: dashed;
		border-color: rgba(122, 162, 255, 0.55);
	}

	.draft-badge {
		align-self: flex-start;
		font-size: 9px;
		font-weight: 800;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		padding: 2px 6px;
		border-radius: 4px;
		background: rgba(122, 162, 255, 0.18);
		color: #a5c8ff;
	}

	.radial {
		position: relative;
		flex: 1;
		min-width: 0;
		min-height: 420px;
		border: 1px solid #303746;
		border-radius: 16px;
		background: rgba(12, 14, 19, 0.74);
		overflow: hidden;
	}

	.lineage-rail {
		flex-shrink: 0;
		display: flex;
		flex-direction: row;
		align-items: stretch;
		align-self: stretch;
		border-radius: 16px;
		border: 1px solid #303746;
		background: rgba(18, 22, 30, 0.92);
		overflow: hidden;
		transition: width 0.18s ease;
	}

	.lineage-rail:not(.lineage-rail--expanded) {
		width: 44px;
	}

	.lineage-rail.lineage-rail--expanded {
		width: min(340px, 40vw);
		max-width: 100%;
	}

	.lineage-tab {
		flex-shrink: 0;
		width: 44px;
		border: none;
		border-right: 1px solid #303746;
		background: rgba(28, 32, 42, 0.95);
		color: #c7d2fe;
		cursor: pointer;
		padding: 8px 0;
		font-family: inherit;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.lineage-tab:hover {
		background: rgba(122, 162, 255, 0.12);
		color: #eef0f5;
	}

	.lineage-tab-v {
		writing-mode: vertical-rl;
		text-orientation: mixed;
		font-size: 12px;
		font-weight: 700;
		letter-spacing: 0.12em;
	}

	.lineage-tab-chev {
		font-size: 22px;
		line-height: 1;
		font-weight: 300;
		color: #a3abb9;
	}

	.lineage-panel-inner {
		flex: 1;
		min-width: 0;
		overflow: auto;
		padding: 12px 12px 14px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.lineage-intro {
		margin: 0;
		font-size: 11px;
		line-height: 1.45;
		color: #858fa1;
	}

	.mount-toolbar {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		align-items: center;
	}

	.mount-btn {
		border: 1px solid rgba(167, 139, 250, 0.55);
		background: rgba(139, 92, 246, 0.14);
		color: #ede9fe;
		font-size: 11px;
		font-weight: 700;
		padding: 6px 10px;
		border-radius: 8px;
		cursor: pointer;
		font-family: inherit;
	}

	.mount-btn:hover:not(:disabled) {
		background: rgba(139, 92, 246, 0.22);
		border-color: rgba(167, 139, 250, 0.75);
	}

	.mount-btn:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	.mount-hint {
		margin: 0;
		font-size: 10px;
		line-height: 1.45;
		color: #6f7888;
	}

	.mount-hint code {
		font-size: 10px;
		color: #a5b4fc;
	}

	.spawn-section-head {
		margin-top: 4px;
		font-size: 12px;
		color: #d4d4d8;
	}

	.spawn-section-head strong {
		font-weight: 700;
		color: #eef0f5;
	}

	.spawn-badge--muted {
		background: rgba(120, 130, 150, 0.15);
		color: #a3abb9;
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

	.ppt-tip {
		color: #8fd3c8;
		font-size: 11px;
	}

	.q-menu-wrap {
		position: relative;
	}

	.q-menu {
		position: absolute;
		top: calc(100% + 4px);
		right: 0;
		z-index: 10;
		display: flex;
		flex-direction: column;
		gap: 4px;
		min-width: 110px;
		padding: 6px;
		border: 1px solid #3b465d;
		border-radius: 8px;
		background: rgba(10, 12, 17, 0.96);
	}

	.q-menu button {
		text-align: left;
	}

	.ppt-frame {
		flex: 1;
		width: 100%;
		border: 1px solid #303746;
		border-radius: 8px;
		background: #0a0b0f;
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

	.spawn-panel-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		margin-bottom: 10px;
	}

	.spawn-panel-head strong {
		font-size: 13px;
		color: #eef0f5;
	}

	.spawn-badge {
		font-size: 11px;
		padding: 2px 8px;
		border-radius: 999px;
		background: rgba(114, 214, 208, 0.12);
		color: #8fd3c8;
		font-family: 'Cascadia Code', ui-monospace, monospace;
	}

	.spawn-muted {
		margin: 0;
		font-size: 12px;
		line-height: 1.5;
		color: #a3abb9;
	}

	.spawn-muted code {
		font-size: 11px;
		color: #c7d2fe;
	}

	.spawn-toolbar {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-bottom: 10px;
		flex-wrap: wrap;
	}

	.spawn-link {
		border: none;
		background: transparent;
		color: #7aa2ff;
		font-size: 12px;
		cursor: pointer;
		text-decoration: underline;
		padding: 0;
		font-family: inherit;
	}

	.spawn-link:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}

	.spawn-confirm {
		border: 1px solid #4c6ef5;
		background: rgba(76, 110, 245, 0.18);
		color: #eef0f5;
		font-size: 12px;
		padding: 6px 12px;
		border-radius: 8px;
		cursor: pointer;
		font-family: inherit;
	}

	.spawn-confirm:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.spawn-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
		max-height: 220px;
		overflow: auto;
	}

	.spawn-row {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 8px 10px;
		border-radius: 8px;
		background: rgba(12, 14, 18, 0.65);
		border: 1px solid rgba(48, 55, 70, 0.85);
	}

	.spawn-label {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
		font-size: 12px;
		color: #eef0f5;
		cursor: pointer;
	}

	.spawn-path {
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 11px;
		color: #c7d2fe;
		word-break: break-all;
	}

	.spawn-tag {
		font-size: 10px;
		color: #86efac;
		padding: 1px 6px;
		border-radius: 4px;
		background: rgba(34, 197, 94, 0.12);
	}

	.spawn-hint-line {
		display: block;
		margin-left: 22px;
		font-size: 11px;
		color: #858fa1;
		line-height: 1.35;
	}

	.spawn-err {
		margin: 10px 0 0;
		font-size: 12px;
		color: #f87171;
	}

	.spawn-ok {
		margin: 10px 0 0;
		font-size: 12px;
		color: #86efac;
	}
</style>
