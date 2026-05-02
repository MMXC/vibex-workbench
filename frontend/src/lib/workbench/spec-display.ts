import { parse as parseYaml } from 'yaml';

export type SpecLevelToken = 'L1' | 'L2' | 'L3' | 'L4' | 'L5' | 'IMPL' | 'UNKNOWN';
export type SpecSlotStatus = 'present' | 'empty' | 'missing' | 'na';

export type SpecDisplay = {
	title: string;
	summary: string;
	description: string;
};

export type SpecSlotSummary = {
	id: 'structure' | 'input' | 'output' | 'constraints' | 'prototype' | 'implementation';
	label: string;
	status: SpecSlotStatus;
	count: number;
	summary: string;
};

export type SpecSlotModel = {
	structure: SpecSlotSummary;
	input: SpecSlotSummary;
	output: SpecSlotSummary;
	constraints: SpecSlotSummary;
	prototype: SpecSlotSummary;
	implementation: SpecSlotSummary;
	all: SpecSlotSummary[];
};

export type SpecDisplayMeta = {
	path: string;
	name: string;
	parent: string | null;
	level: SpecLevelToken;
	rawLevel: string;
	status: string;
	module: string;
	display: SpecDisplay;
	slots: SpecSlotModel;
};

function asRecord(value: unknown): Record<string, unknown> | null {
	return value && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

function asString(value: unknown): string | null {
	return typeof value === 'string' && value.trim() ? value.trim() : null;
}

function compactText(value: unknown): string {
	if (value == null) return '';
	if (typeof value === 'string') return value.trim();
	if (typeof value === 'number' || typeof value === 'boolean') return String(value);
	if (Array.isArray(value)) {
		return value.map(compactText).filter(Boolean).join('；');
	}
	if (typeof value === 'object') {
		return Object.entries(value as Record<string, unknown>)
			.filter(([, v]) => compactText(v))
			.map(([k, v]) => `${k}: ${compactText(v)}`)
			.join('；');
	}
	return '';
}

function firstLine(value: unknown): string {
	const text = compactText(value);
	for (const line of text.split(/\r?\n/)) {
		const cleaned = line.replace(/^[-\s]+/, '').trim();
		if (cleaned) return cleaned;
	}
	return '';
}

function toItems(value: unknown): string[] {
	if (value == null) return [];
	if (Array.isArray(value)) {
		return value.map(compactText).map(v => v.trim()).filter(Boolean);
	}
	const text = compactText(value);
	return text ? [text] : [];
}

function countItems(value: unknown): number {
	if (value == null) return 0;
	if (Array.isArray(value)) return value.filter(item => compactText(item)).length;
	if (typeof value === 'object') {
		return Object.values(value as Record<string, unknown>).filter(item => compactText(item)).length;
	}
	return compactText(value) ? 1 : 0;
}

function slotStatus(fieldExists: boolean, count: number, summary: string): SpecSlotStatus {
	const text = summary.toLowerCase();
	if (text.includes('不适用') || text.includes('not applicable')) return 'na';
	if (count > 0) return 'present';
	return fieldExists ? 'empty' : 'missing';
}

function slot(
	id: SpecSlotSummary['id'],
	label: string,
	fieldExists: boolean,
	values: unknown,
	fallbackSummary: string,
	emptyLabel = '无'
): SpecSlotSummary {
	const count = countItems(values);
	const summary = firstLine(values) || fallbackSummary || (fieldExists ? emptyLabel : '待补充');
	return {
		id,
		label,
		status: slotStatus(fieldExists, count, summary),
		count,
		summary,
	};
}

function mergeArrayValues(...values: unknown[]): string[] {
	const seen = new Set<string>();
	const result: string[] = [];
	for (const value of values) {
		for (const item of toItems(value)) {
			if (!seen.has(item)) {
				seen.add(item);
				result.push(item);
			}
		}
	}
	return result;
}

function extractImpactedFiles(structure: Record<string, unknown> | null, content: Record<string, unknown> | null): string[] {
	const lineage = asRecord(content?.l4_l5_lineage);
	const whichFiles = Array.isArray(lineage?.which_files)
		? lineage.which_files.map(item => asRecord(item)?.file ?? asRecord(item)?.path ?? item)
		: [];
	return mergeArrayValues(
		structure?.impacted_files,
		content?.file_path,
		whichFiles
	);
}

function extractSpecSlots(doc: Record<string, unknown> | null, specParent: string | null): SpecSlotModel {
	const content = asRecord(doc?.content);
	const structure = asRecord(doc?.structure);
	const canonicalIo = asRecord(doc?.io);
	const legacyIo = asRecord(doc?.io_contract);
	const constraints = asRecord(doc?.constraints);
	const prototype = asRecord(doc?.prototype);
	const implementationBoundary = asRecord(content?.implementation_boundary);

	const dependencies = mergeArrayValues(structure?.dependencies, content?.dependencies);
	const impactedFiles = extractImpactedFiles(structure, content);
	const input = mergeArrayValues(canonicalIo?.input, legacyIo?.input);
	const output = mergeArrayValues(canonicalIo?.output, legacyIo?.output);
	const rules = mergeArrayValues(constraints?.rules, content?.constraints);
	const forbidden = mergeArrayValues(constraints?.forbidden, implementationBoundary?.forbidden);
	const prototypeFile = asString(prototype?.file);
	const prototypeStatus = asString(prototype?.status);

	const structureValues = [
		specParent ? `parent: ${specParent}` : '',
		...dependencies.map(item => `dep: ${item}`),
		...impactedFiles.map(item => `file: ${item}`),
	].filter(Boolean);
	const prototypeValues = [
		prototypeFile ? `file: ${prototypeFile}` : '',
		prototypeStatus && prototypeStatus !== 'none' ? `status: ${prototypeStatus}` : '',
		...toItems(prototype?.validates).map(item => `validates: ${item}`),
	].filter(Boolean);

	const model = {
		structure: slot('structure', '结构', !!structure, structureValues, '未定义结构'),
		input: slot('input', '输入', !!canonicalIo || !!legacyIo, input, '待补充输入'),
		output: slot('output', '输出', !!canonicalIo || !!legacyIo, output, '待补充输出'),
		constraints: slot('constraints', '约束', !!constraints, [...rules, ...forbidden], '无约束'),
		prototype: slot('prototype', '原型', !!prototype, prototypeValues, '无原型'),
		implementation: slot('implementation', '实现', !!structure || !!content, impactedFiles, '无实现文件'),
	};
	return {
		...model,
		all: [
			model.structure,
			model.input,
			model.output,
			model.constraints,
			model.prototype,
			model.implementation,
		],
	};
}

export function normalizeSpecLevel(raw: unknown, name = ''): SpecLevelToken {
	const value = String(raw ?? '').toLowerCase();
	if (value.includes('1_') || value.includes('project-goal')) return 'L1';
	if (value.includes('2_') || value.includes('skeleton')) return 'L2';
	if (value.includes('3_') || value.includes('module')) return 'L3';
	if (value.includes('4_') || value.includes('feature')) return 'L4';
	if (value.includes('5_implementation')) return 'IMPL';
	if (value.includes('5_') || value.includes('slice')) return 'L5';
	if (name.endsWith('_L5')) return 'IMPL';
	if (name.startsWith('SLICE-')) return 'L5';
	if (name.startsWith('FEAT-')) return 'L4';
	if (name.startsWith('MOD-')) return 'L3';
	return 'UNKNOWN';
}

export function fallbackDisplayTitle(name: string): string {
	return name
		.replace(/^(FEAT|MOD|SLICE)-/, '')
		.replace(/_L5$/, '')
		.split(/[-_]+/)
		.filter(Boolean)
		.slice(0, 5)
		.join(' ');
}

export function extractSpecDisplay(content: string, path: string): SpecDisplayMeta {
	let doc: Record<string, unknown> | null = null;
	try {
		doc = parseYaml(content) as Record<string, unknown>;
	} catch {
		doc = null;
	}

	const spec = asRecord(doc?.spec);
	const meta = asRecord(doc?.meta);
	const display = asRecord(doc?.display);
	const canonicalIo = asRecord(doc?.io);
	const legacyIo = asRecord(doc?.io_contract);

	const fallbackName = path.split('/').pop()?.replace(/\.ya?ml$/, '') ?? path;
	const name = asString(spec?.name) ?? fallbackName;
	const rawLevel = asString(spec?.level) ?? '';
	const summary =
		asString(display?.summary) ||
		firstLine(canonicalIo?.output) ||
		firstLine(legacyIo?.output) ||
		'缺少 display.summary，agent 应在下次修改 spec 时补齐。';
	const parent = asString(spec?.parent);

	return {
		path,
		name,
		parent,
		level: normalizeSpecLevel(rawLevel, name),
		rawLevel,
		status: asString(spec?.status) ?? asString(asRecord(doc?.lifecycle)?.current) ?? 'proposal',
		module: asString(meta?.module) ?? '',
		display: {
			title: asString(display?.title) ?? fallbackDisplayTitle(name),
			summary,
			description: asString(display?.description) ?? summary,
		},
		slots: extractSpecSlots(doc, parent),
	};
}
