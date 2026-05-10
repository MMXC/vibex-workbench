import { parse as parseYaml } from 'yaml';

/** App Shell manifest 中单条 feature 路由（见 `.vibex/prototype-manifest.yaml`） */
export type ShellManifestRoute = {
	id: string;
	path: string;
	title: string;
	specRef: string;
	entryHtml: string;
	kind: 'feature' | 'shell';
};

export function slugFromSpecName(name: string): string {
	const s = name
		.trim()
		.toLowerCase()
		.replace(/[^a-z0-9_-]+/g, '-')
		.replace(/^-+|-+$/g, '');
	return s || 'spec-route';
}

export function basenameNoYaml(path: string): string {
	const p = path.replace(/\\/g, '/').split('/').pop() ?? 'spec';
	return p.replace(/\.ya?ml$/i, '');
}

/** 从 spec YAML 读取 `prototype.file`（相对工作区根） */
export function getPrototypeFileRel(yamlContent: string): string | null {
	try {
		const doc = parseYaml(yamlContent) as Record<string, unknown>;
		const proto = doc?.prototype as Record<string, unknown> | undefined;
		const f = proto?.file;
		return typeof f === 'string' && f.trim() ? f.trim().replace(/\\/g, '/') : null;
	} catch {
		return null;
	}
}

/** 禁止脱离工作区（不含 ..） */
export function isSafeWorkspaceRel(rel: string): boolean {
	if (!rel || rel.trim() === '') return false;
	const norm = rel.replace(/\\/g, '/').replace(/^[/\\]+/, '');
	if (!norm || norm.includes('..')) return false;
	return true;
}

export function deriveEntryHtmlForFeature(opts: {
	specPath: string;
	yamlContent: string;
	override?: string | null;
}): string {
	const o = opts.override?.trim();
	if (o && isSafeWorkspaceRel(o)) return o.replace(/\\/g, '/');
	const fromYaml = getPrototypeFileRel(opts.yamlContent);
	if (fromYaml && isSafeWorkspaceRel(fromYaml)) return fromYaml;
	const stem = basenameNoYaml(opts.specPath);
	return `.vibex/prototypes/${stem}.html`;
}

export function buildFeatureShellRoute(opts: {
	specName: string;
	specPath: string;
	displayTitle: string;
	yamlContent: string;
	entryHtmlOverride?: string | null;
}): ShellManifestRoute {
	const id = slugFromSpecName(opts.specName);
	const entryHtml = deriveEntryHtmlForFeature({
		specPath: opts.specPath,
		yamlContent: opts.yamlContent,
		override: opts.entryHtmlOverride,
	});
	return {
		id,
		path: `/proto/${id}`,
		title: (opts.displayTitle || opts.specName).trim() || opts.specName,
		specRef: opts.specName,
		entryHtml,
		kind: 'feature',
	};
}
