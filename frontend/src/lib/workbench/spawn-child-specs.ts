/**
 * 从发散层（L2 skeleton / L4 feature）YAML 抽取子收敛 spec 候选，并渲染模板、回写父 spec structure.children。
 */
import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';

export type SpawnLayer = 'L2' | 'L4';

export type ChildSpawnCandidate = {
	/** 勾选列表用稳定 id */
	id: string;
	specName: string;
	relativePath: string;
	titleHint: string;
	summaryHint: string;
	/** 已在父 spec structure.children 中声明 */
	alreadyLinked: boolean;
	/** L5：对应 content.behaviors[].id */
	behaviorId?: string;
};

function asRecord(value: unknown): Record<string, unknown> | null {
	return value && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

function parseDoc(yamlText: string): Record<string, unknown> | null {
	try {
		const doc = parseYaml(yamlText) as Record<string, unknown> | null;
		return doc && typeof doc === 'object' ? doc : null;
	} catch {
		return null;
	}
}

/** 与 GenericSpecGraph.getSpecLevelTag 对齐：仅 L2 / L4 可一键生成子收敛 spec */
export function inferSpawnLayer(yamlText: string): SpawnLayer | null {
	const doc = parseDoc(yamlText);
	const raw = String(asRecord(doc?.spec)?.level ?? '').toLowerCase();
	if (raw.includes('1') || raw.includes('3') || raw.includes('5')) return null;
	if (raw.includes('2')) return 'L2';
	if (raw.includes('4')) return 'L4';
	return null;
}

export function extractSpecName(yamlText: string): string | null {
	const doc = parseDoc(yamlText);
	const n = asRecord(doc?.spec)?.name;
	return typeof n === 'string' && n.trim() ? n.trim() : null;
}

export function extractMetaOwner(yamlText: string): string {
	const doc = parseDoc(yamlText);
	const o = asRecord(doc?.meta)?.owner;
	return typeof o === 'string' && o.trim() ? o.trim() : 'unknown';
}

export function extractMetaModule(yamlText: string): string {
	const doc = parseDoc(yamlText);
	const m = asRecord(doc?.meta)?.module;
	return typeof m === 'string' && m.trim() ? m.trim() : 'workspace';
}

function normalizeChildPath(p: string): string {
	return p.replace(/\\/g, '/').trim();
}

/** structure.children 中的相对路径（.vibex/specs/...） */
export function parseExistingChildPaths(yamlText: string): string[] {
	const doc = parseDoc(yamlText);
	const st = asRecord(doc?.structure);
	const ch = st?.children;
	if (!Array.isArray(ch)) return [];
	return ch
		.map(x => (typeof x === 'string' ? normalizeChildPath(x) : ''))
		.filter(Boolean);
}

function moduleKeyToModSpecName(moduleKey: string): string {
	const slug = moduleKey.trim().toLowerCase().replace(/\s+/g, '-');
	return `MOD-${slug}`;
}

function l4StemForSlice(l4SpecName: string): string {
	return l4SpecName.replace(/^FEAT-/i, '').replace(/^MOD-/i, '');
}

function buildSliceName(l4SpecName: string, behaviorId: string): string {
	const stem = l4StemForSlice(l4SpecName);
	const id = behaviorId.trim().toUpperCase().replace(/^ID\s*/i, '');
	return `SLICE-${stem}-${id}`;
}

/**
 * 将模板中 `{key}` 占位符替换为 vars[key]；按 key 长度降序，避免短 key 截断长 key。
 * 不处理 `{{language}}` 这类双花括号（无对应单 `{language}` 片段）。
 */
export function replaceBracePlaceholders(template: string, vars: Record<string, string>): string {
	const keys = Object.keys(vars).sort((a, b) => b.length - a.length);
	let out = template;
	for (const k of keys) {
		const token = `{${k}}`;
		const v = vars[k] ?? '';
		out = out.split(token).join(v);
	}
	return out;
}

/** 自 L2 YAML 抽取 L3 候选：优先 content.l2_l3_lineage.which_modules_become_l3 */
export function collectL3CandidatesFromL2(
	parentYaml: string,
	l2SpecName: string
): ChildSpawnCandidate[] {
	const doc = parseDoc(parentYaml);
	if (!doc) return [];
	const content = asRecord(doc.content);
	const lineage = asRecord(content?.l2_l3_lineage);
	const rows = lineage?.which_modules_become_l3;
	const existing = new Set(parseExistingChildPaths(parentYaml));
	const out: ChildSpawnCandidate[] = [];

	if (Array.isArray(rows)) {
		for (const row of rows) {
			const r = asRecord(row);
			const moduleKey =
				(typeof r?.module === 'string' && r.module.trim()) ||
				(typeof r?.name === 'string' && r.name.trim()) ||
				'';
			if (!moduleKey) continue;
			const specName = moduleKeyToModSpecName(moduleKey);
			const relativePath = `.vibex/specs/L3-module/${specName}.yaml`;
			if (out.some(c => c.specName === specName)) continue;
			const note = compactNote(r?.l3_note ?? r?.note);
			const titleHint = humanizeModule(moduleKey);
			const summaryHint =
				note ||
				`L3 模块 spec：对应 L2「${l2SpecName}」中的模块「${moduleKey}」。请在 content 中补齐 public_api 与状态定义。`;
			const id = `l3:${specName}`;
			out.push({
				id,
				specName,
				relativePath,
				titleHint,
				summaryHint,
				alreadyLinked: existing.has(relativePath),
			});
		}
	}

	// 可选：modules_matrix 下列出 modules 数组（未来 L2 若采用结构化枚举）
	const mm = asRecord(content?.modules_matrix);
	const extraMods = mm?.modules;
	if (Array.isArray(extraMods)) {
		for (const item of extraMods) {
			const r = asRecord(item);
			const key =
				(typeof r?.id === 'string' && r.id.trim()) ||
				(typeof r?.name === 'string' && r.name.trim()) ||
				(typeof r?.module === 'string' && r.module.trim()) ||
				'';
			if (!key) continue;
			const specName = moduleKeyToModSpecName(key);
			const relativePath = `.vibex/specs/L3-module/${specName}.yaml`;
			if (out.some(c => c.relativePath === relativePath)) continue;
			const id = `l3:${specName}`;
			out.push({
				id,
				specName,
				relativePath,
				titleHint: humanizeModule(key),
				summaryHint: `由 modules_matrix 枚举：${key}`,
				alreadyLinked: existing.has(relativePath),
			});
		}
	}

	return out;
}

function compactNote(v: unknown): string {
	if (v == null) return '';
	if (typeof v === 'string') return v.split(/\n/)[0]?.trim() ?? '';
	return '';
}

function humanizeModule(moduleKey: string): string {
	return moduleKey
		.split(/[-_/]+/)
		.filter(Boolean)
		.map(w => w.charAt(0).toUpperCase() + w.slice(1))
		.join(' ');
}

/** 自 L4 YAML 抽取 L5 候选：content.behaviors[].id */
export function collectL5CandidatesFromL4(
	parentYaml: string,
	l4SpecName: string
): ChildSpawnCandidate[] {
	const doc = parseDoc(parentYaml);
	if (!doc) return [];
	const content = asRecord(doc.content);
	const behaviors = content?.behaviors;
	const existing = new Set(parseExistingChildPaths(parentYaml));
	const out: ChildSpawnCandidate[] = [];

	if (Array.isArray(behaviors)) {
		let idx = 0;
		for (const b of behaviors) {
			const r = asRecord(b);
			idx += 1;
			const behaviorId =
				(typeof r?.id === 'string' && r.id.trim()) || `B${idx}`;
			const specName = buildSliceName(l4SpecName, behaviorId);
			const relativePath = `.vibex/specs/L5-slice/${specName}.yaml`;
			if (out.some(c => c.specName === specName)) continue;
			const trigger = typeof r?.trigger === 'string' ? r.trigger.split(/\n/)[0].trim() : '';
			const titleHint = `${behaviorId} · ${trigger.slice(0, 48)}${trigger.length > 48 ? '…' : ''}`;
			const summaryHint =
				trigger ||
				`由 L4「${l4SpecName}」的 behavior ${behaviorId} 生成的实现切片占位，请细化 content.file_path 与验证步骤。`;
			out.push({
				id: `l5:${specName}`,
				specName,
				relativePath,
				titleHint,
				summaryHint,
				alreadyLinked: existing.has(relativePath),
				behaviorId,
			});
		}
	}

	return out;
}

export function appendChildrenPaths(parentYaml: string, newRelativePaths: string[]): string {
	const doc = (parseYaml(parentYaml) ?? {}) as Record<string, unknown>;
	const structure = { ...(asRecord(doc.structure) ?? {}) };
	const prev = Array.isArray(structure.children)
		? (structure.children as unknown[]).map(x => (typeof x === 'string' ? normalizeChildPath(x) : '')).filter(Boolean)
		: [];
	const set = new Set(prev);
	for (const p of newRelativePaths.map(normalizeChildPath)) {
		if (p) set.add(p);
	}
	structure.children = [...set];
	doc.structure = structure;
	return stringifyYaml(doc);
}

export type RenderL3Vars = {
	moduleSpecName: string;
	l2SpecName: string;
	owner: string;
	dateYmd: string;
	titleZh: string;
	summaryLine: string;
	descriptionParagraph: string;
};

/** 使用仓库内 L3-module-template 渲染（传入完整模板文本） */
export function renderL3ModuleFromTemplate(templateYaml: string, v: RenderL3Vars): string {
	const vars: Record<string, string> = {
		'module-name': v.moduleSpecName,
		'l2-spec-name': v.l2SpecName,
		owner: v.owner,
		'YYYY-MM-DD': v.dateYmd,
		中文模块标题: v.titleZh,
		一句话说明该模块的职责和边界: v.summaryLine,
		'面向用户和 agent 的完整模块说明': v.descriptionParagraph,
		'依赖的模块或外部库，可为空': '—',
		'L2 skeleton 中该模块的定义': `见 L2「${v.l2SpecName}」content.modules_matrix / l2_l3_lineage。`,
		'公开 API、状态定义和模块依赖': '待 L3 content.public_api / state_definitions 补全',
		exported_function: 'TBD',
		'func(arg: Type): ReturnType': 'TBD',
		函数作用: '待定义',
		有无副作用: '待评估',
		StateName: 'TBD',
		状态含义: '待定义',
		'from-state': 'idle',
		'to-state': 'ready',
		触发条件: '待定义',
		模块内部组织结构的文字描述: '待补充：目录结构与职责分层。',
		'库/模块名': 'stdlib',
		用途: '待补充',
		'L3 定义接口，L4 使用接口实现功能': 'L3 定义接口，L4 使用接口实现功能',
		'feature-name': 'TBD',
		function: 'TBD',
	};
	return replaceBracePlaceholders(templateYaml, vars);
}

export type RenderL5Vars = {
	sliceSpecName: string;
	l4SpecName: string;
	moduleName: string;
	owner: string;
	dateYmd: string;
	titleZh: string;
	summaryLine: string;
	descriptionParagraph: string;
	behaviorRefLine: string;
};

export function renderL5SliceFromTemplate(templateYaml: string, v: RenderL5Vars): string {
	const vars: Record<string, string> = {
		'slice-name': v.sliceSpecName,
		'l4-spec-name': v.l4SpecName,
		'module-name': v.moduleName,
		owner: v.owner,
		'YYYY-MM-DD': v.dateYmd,
		中文实现切片标题: v.titleZh,
		'一句话说明这个文件/切片要实现什么': v.summaryLine,
		'面向用户和 agent 的完整实现说明': v.descriptionParagraph,
		'依赖的文件或模块，可为空': '—',
		'src/generated/ComponentName.skeleton.svelte': 'TBD — 在 content.file_path 写明真实路径',
		'对应 L4 behavior 或详细 spec': v.behaviorRefLine,
		生成或修改的具体文件: '见 content.file_path',
		如何选择对应模板: '按 file_type 选择对应 gen 模板',
		'spec 数据如何填入模板': '由本 L5 spec 的 content 字段驱动',
		重复生成不破坏已有代码: '仅覆盖 *.Skeleton.* / 生成器声明路径',
		'test/path/to/spec_test.go': 'TBD',
		'覆盖率目标，如 80%': '按模块惯例',
		测试模式描述: '待补充',
		依赖的文件: 'TBD',
		为什么依赖它: '待补充',
		placeholder: '…',
	};
	let out = replaceBracePlaceholders(templateYaml, vars);
	// 模板里的 ```{{language}}``` 保持不动；若仍有裸占位 `{模板内容...}` 简化为提示行
	out = out.replace(
		/\{模板内容[^}]*\}/,
		'# 由 agent / 开发者在本切片 spec 中补充具体模板片段'
	);
	return out;
}

/** 为抽屉草稿生成完整 YAML（单候选） */
export function draftYamlForChildCandidate(
	layer: SpawnLayer,
	templateYaml: string,
	parentYaml: string,
	candidate: ChildSpawnCandidate,
	dateYmd: string
): string {
	const parentName = extractSpecName(parentYaml) ?? '';
	const owner = extractMetaOwner(parentYaml);
	const moduleName = extractMetaModule(parentYaml);
	if (layer === 'L2') {
		return renderL3ModuleFromTemplate(templateYaml, {
			moduleSpecName: candidate.specName,
			l2SpecName: parentName,
			owner,
			dateYmd,
			titleZh: candidate.titleHint,
			summaryLine: candidate.summaryHint,
			descriptionParagraph: `${candidate.summaryHint}\n\n（草稿：在抽屉中修订后写入磁盘。）`,
		});
	}
	return renderL5SliceFromTemplate(templateYaml, {
		sliceSpecName: candidate.specName,
		l4SpecName: parentName,
		moduleName,
		owner,
		dateYmd,
		titleZh: candidate.titleHint,
		summaryLine: candidate.summaryHint,
		descriptionParagraph: `${candidate.summaryHint}\n\n（草稿：在抽屉中修订后写入磁盘。）`,
		behaviorRefLine: `L4「${parentName}」content.behaviors · ${candidate.behaviorId ?? '—'}`,
	});
}
