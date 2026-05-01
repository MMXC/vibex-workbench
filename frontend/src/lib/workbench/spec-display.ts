import { parse as parseYaml } from 'yaml';

export type SpecLevelToken = 'L1' | 'L2' | 'L3' | 'L4' | 'L5' | 'IMPL' | 'UNKNOWN';

export type SpecDisplay = {
	title: string;
	summary: string;
	description: string;
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
};

function asRecord(value: unknown): Record<string, unknown> | null {
	return value && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

function asString(value: unknown): string | null {
	return typeof value === 'string' && value.trim() ? value.trim() : null;
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
	const io = asRecord(doc?.io_contract);

	const fallbackName = path.split('/').pop()?.replace(/\.ya?ml$/, '') ?? path;
	const name = asString(spec?.name) ?? fallbackName;
	const rawLevel = asString(spec?.level) ?? '';
	const summary =
		asString(display?.summary) ??
		asString(io?.output)?.split('\n').map(line => line.replace(/^[-\s]+/, '').trim()).find(Boolean) ??
		'缺少 display.summary，agent 应在下次修改 spec 时补齐。';

	return {
		path,
		name,
		parent: asString(spec?.parent),
		level: normalizeSpecLevel(rawLevel, name),
		rawLevel,
		status: asString(spec?.status) ?? asString(asRecord(doc?.lifecycle)?.current) ?? 'proposal',
		module: asString(meta?.module) ?? '',
		display: {
			title: asString(display?.title) ?? fallbackDisplayTitle(name),
			summary,
			description: asString(display?.description) ?? summary,
		},
	};
}
