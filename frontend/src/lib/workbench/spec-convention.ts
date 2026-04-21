import { parse as parseYaml } from 'yaml';

/** GET /api/workspace/specs/convention */
export type ConventionPayload = {
	source: string;
	convention: {
		spec_types?: Array<{
			id: string;
			vibex_spec_level?: string;
			directory?: string;
			notes?: string;
		}>;
		parent_resolve?: {
			entries?: Array<{ parent: string; target_path: string }>;
		};
		path_inference?: {
			rules?: Array<
				| { if_path_prefix: string; spec_type_id: string }
				| { if_path_match: string; spec_type_id: string }
			>;
		};
	};
};

/** norm 为正斜杠、无首尾空白 */
export function normalizeSpecPath(path: string): string {
	return path.replace(/\\/g, '/').replace(/^\.\/+/, '').trim();
}

// Glob with single-segment stars (e.g. specs/feature/<seg>/<name>_feature.yaml) → anchored regex
export function globPathMatchToRegex(glob: string): RegExp {
	let pattern = '';
	for (let i = 0; i < glob.length; i++) {
		const c = glob[i];
		if (c === '*') pattern += '[^/]+';
		else if ('.+?^${}()[]|\\'.includes(c)) pattern += '\\' + c;
		else pattern += c;
	}
	return new RegExp(`^${pattern}$`);
}

export function inferSpecTypeId(path: string, convention: ConventionPayload['convention']): string | null {
	const norm = normalizeSpecPath(path);
	const rules = convention.path_inference?.rules ?? [];
	const prefixRules = rules.filter(
		(r): r is { if_path_prefix: string; spec_type_id: string } => 'if_path_prefix' in r
	);
	const matchRules = rules.filter(
		(r): r is { if_path_match: string; spec_type_id: string } => 'if_path_match' in r
	);

	/* 先匹配更具体的 glob，再前缀（长前缀优先） */
	const sortedPrefix = [...prefixRules].sort((a, b) => b.if_path_prefix.length - a.if_path_prefix.length);
	const sortedMatch = [...matchRules].sort((a, b) => b.if_path_match.length - a.if_path_match.length);

	for (const r of sortedMatch) {
		try {
			const re = globPathMatchToRegex(normalizeSpecPath(r.if_path_match));
			if (re.test(norm)) return r.spec_type_id;
		} catch {
			continue;
		}
	}
	for (const r of sortedPrefix) {
		if (norm.startsWith(normalizeSpecPath(r.if_path_prefix))) return r.spec_type_id;
	}
	return null;
}

export function specTypeLabel(
	convention: ConventionPayload['convention'],
	specTypeId: string | null
): string | null {
	if (!specTypeId) return null;
	const t = convention.spec_types?.find(x => x.id === specTypeId);
	return t?.vibex_spec_level ?? specTypeId;
}

export function extractSpecMeta(content: string): { parent: string | null; name: string | null } {
	try {
		const doc = parseYaml(content) as { spec?: { parent?: string; name?: string } };
		const parent = typeof doc.spec?.parent === 'string' ? doc.spec.parent : null;
		const name = typeof doc.spec?.name === 'string' ? doc.spec.name : null;
		return { parent, name };
	} catch {
		return { parent: null, name: null };
	}
}

/**
 * 将 spec.parent（聚合名）解析为仓库内 YAML 路径：优先 parent_resolve 表，否则启发式。
 */
export function inferParentSpecPath(
	parent: string,
	convention?: ConventionPayload['convention'] | null
): string | null {
	const p = parent.trim();
	if (!p) return null;

	const entries = convention?.parent_resolve?.entries ?? [];
	const mapped = entries.find(e => e.parent === p)?.target_path;
	if (mapped) return normalizeSpecPath(mapped);

	if (p === 'vibex-workbench-goal') return 'specs/project-goal/vibex-workbench-goal.yaml';
	if (p.endsWith('-skeleton') || p === 'vibex-workbench-skeleton')
		return `specs/architecture/${p}.yaml`;
	if (p.startsWith('MOD-')) return `specs/module/${p}_module.yaml`;
	if (/^[a-z][a-z0-9_-]*$/i.test(p)) {
		return `specs/feature/${p}/${p}_feature.yaml`;
	}
	return null;
}

/** 同 feature 目录下的主 feature（用于 L5 跳转） */
export function inferSiblingFeaturePath(currentPath: string): string | null {
	const norm = normalizeSpecPath(currentPath);
	const m = norm.match(/^specs\/feature\/([^/]+)\//);
	if (!m) return null;
	const slug = m[1];
	return `specs/feature/${slug}/${slug}_feature.yaml`;
}
