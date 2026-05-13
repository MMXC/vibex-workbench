/**
 * 规格血缘：平面列出全部上游引用（多继承 / 多依赖），不做链式级联。
 */
import { parse as parseYaml } from 'yaml';
import type { ConventionPayload } from '$lib/workbench/spec-convention';
import { inferParentSpecPath, normalizeSpecPath } from '$lib/workbench/spec-convention';

/**
 * 遍历 specs 列表，找出 spec.parent === parentSpecName 的条目，返回规范化的相对路径（去重、排序）。
 * 用于将「已存在的子 spec 文件」批量写入当前父 spec 的 structure.children。
 */
export function collectChildPathsForParent(
	specs: { path: string; parent?: string | null }[],
	parentSpecName: string,
	opts?: { excludeSelfPath?: string | null }
): string[] {
	const want = parentSpecName.trim();
	if (!want) return [];

	const selfNorm = opts?.excludeSelfPath
		? normalizeSpecPath(opts.excludeSelfPath.replace(/\\/g, '/'))
		: '';

	const seen = new Set<string>();
	const out: string[] = [];

	for (const s of specs) {
		const par = (s.parent ?? '').trim();
		if (par !== want) continue;

		const rel = normalizeSpecPath(s.path.replace(/\\/g, '/'));
		if (!/\.ya?ml$/i.test(rel)) continue;

		if (selfNorm && normalizeSpecPath(rel) === selfNorm) continue;

		if (!seen.has(rel)) {
			seen.add(rel);
			out.push(rel);
		}
	}

	out.sort();
	return out;
}

function asRecord(value: unknown): Record<string, unknown> | null {
	return value && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

/** 从 YAML 收集所有「上游」引用字符串（去重、保持顺序） */
export function collectParentRefs(yamlText: string): string[] {
	let doc: Record<string, unknown> | null = null;
	try {
		doc = parseYaml(yamlText) as Record<string, unknown>;
	} catch {
		return [];
	}
	if (!doc) return [];

	const seen = new Set<string>();
	const add = (raw: unknown) => {
		if (typeof raw !== 'string') return;
		const t = raw.trim();
		if (!t || seen.has(t)) return;
		seen.add(t);
	};

	const spec = asRecord(doc.spec);
	add(spec?.parent);

	const structure = asRecord(doc.structure);
	add(structure?.parent);

	const deps = structure?.dependencies;
	if (Array.isArray(deps)) {
		for (const d of deps) {
			if (typeof d === 'string') add(d);
			else if (d && typeof d === 'object') {
				const o = d as Record<string, unknown>;
				if (typeof o.path === 'string') add(o.path);
				if (typeof o.spec === 'string') add(o.spec);
			}
		}
	}

	return [...seen];
}

/** 全库 spec 列表项：用于将父引用与 `spec.name` 做精确匹配 */
export type SpecNameCatalogEntry = { path: string; name: string };

/**
 * 将单个引用解析为仓库内 specs 相对路径（用于跳转）。
 * - 已是 `.vibex/specs/...yaml` 则直接规范化
 * - 若提供 catalog：按 **spec.name 与 ref 全字匹配**（trim 后严格相等）优先，避免启发式错指目录
 * - 否则走 convention + inferParentSpecPath + 目录启发式
 */
export function resolveSpecRefToPath(
	ref: string,
	convention: ConventionPayload['convention'] | null | undefined,
	catalog?: SpecNameCatalogEntry[] | null
): string | null {
	const r = ref.trim().replace(/\\/g, '/');
	if (!r) return null;
	if (r.startsWith('.vibex/specs/') && /\.ya?ml$/i.test(r)) {
		return normalizeSpecPath(r);
	}

	const refName = ref.trim();
	if (catalog?.length) {
		const hit = catalog.find(e => (e.name ?? '').trim() === refName);
		if (hit?.path) return normalizeSpecPath(hit.path.replace(/\\/g, '/'));
	}

	const mapped = inferParentSpecPath(ref, convention ?? null);
	if (mapped) return normalizeSpecPath(mapped);

	const vx = '.vibex/specs';
	if (/^MOD-[A-Za-z0-9_-]+$/.test(r)) return `${vx}/L3-module/${r}.yaml`;
	if (/^FEAT-[A-Za-z0-9_-]+$/.test(r)) return `${vx}/L4-feature/${r}.yaml`;
	if (/^SLICE-[A-Za-z0-9_-]+$/.test(r)) return `${vx}/L5-slice/${r}.yaml`;

	if (r.endsWith('-skeleton') || r.includes('skeleton')) return `${vx}/L2-skeleton/${r}.yaml`;
	if (r.endsWith('-mvp') || r.endsWith('mvp')) return `${vx}/L1-goal/${r}.yaml`;

	return null;
}
