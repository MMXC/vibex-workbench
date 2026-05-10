import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';
import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';
import type { ShellManifestRoute } from '$lib/workbench/prototype-shell-manifest';
import {
	buildFeatureShellRoute,
	isSafeWorkspaceRel,
} from '$lib/workbench/prototype-shell-manifest';

const MANIFEST_REL = path.join('.vibex', 'prototype-manifest.yaml');

function resolveRoot(workspaceRoot: string): string | null {
	const root = path.resolve(workspaceRoot.trim());
	if (!workspaceRoot.trim() || !fs.existsSync(root)) return null;
	return root;
}

function manifestAbs(root: string): string {
	return path.join(root, MANIFEST_REL);
}

function normalizeManifest(raw: unknown): {
	version: number;
	lastGenerated?: string;
	routes: ShellManifestRoute[];
} {
	if (!raw || typeof raw !== 'object') {
		return { version: 1, routes: [] };
	}
	const o = raw as Record<string, unknown>;
	const routesIn = Array.isArray(o.routes) ? o.routes : [];
	const routes: ShellManifestRoute[] = [];
	for (const r of routesIn) {
		if (!r || typeof r !== 'object') continue;
		const x = r as Record<string, unknown>;
		const id = typeof x.id === 'string' ? x.id.trim() : '';
		const pathStr = typeof x.path === 'string' ? x.path.trim() : '';
		const title = typeof x.title === 'string' ? x.title.trim() : '';
		const specRef = typeof x.specRef === 'string' ? x.specRef.trim() : '';
		const entryHtml = typeof x.entryHtml === 'string' ? x.entryHtml.trim() : '';
		const kind = x.kind === 'shell' ? 'shell' : 'feature';
		if (!id || !pathStr || !title || !specRef || !entryHtml) continue;
		if (!isSafeWorkspaceRel(entryHtml)) continue;
		routes.push({ id, path: pathStr, title, specRef, entryHtml, kind });
	}
	return {
		version: typeof o.version === 'number' && Number.isFinite(o.version) ? o.version : 1,
		lastGenerated: typeof o.lastGenerated === 'string' ? o.lastGenerated : undefined,
		routes,
	};
}

function upsertRoute(
	m: { version: number; lastGenerated?: string; routes: ShellManifestRoute[] },
	route: ShellManifestRoute
): { ok: true } | { ok: false; error: string } {
	const byId = m.routes.findIndex(x => x.id === route.id);
	if (byId >= 0) {
		const existing = m.routes[byId];
		if (existing.specRef !== route.specRef) {
			return {
				ok: false,
				error: `路由 id「${route.id}」已被 spec「${existing.specRef}」占用，请改名或手工编辑 .vibex/prototype-manifest.yaml`,
			};
		}
		m.routes[byId] = { ...route };
		return { ok: true };
	}
	const byRef = m.routes.findIndex(x => x.specRef === route.specRef);
	if (byRef >= 0) {
		m.routes[byRef] = { ...route };
		return { ok: true };
	}
	m.routes.push({ ...route });
	return { ok: true };
}

export async function GET(event) {
	const url = event.url;
	const workspaceRoot =
		url.searchParams.get('workspaceRoot') || url.searchParams.get('workspace_root') || '';
	const root = resolveRoot(workspaceRoot);
	if (!root) {
		return json({ ok: false, error: 'workspaceRoot invalid or not found' }, { status: 400 });
	}
	const abs = manifestAbs(root);
	if (!fs.existsSync(abs)) {
		return json({
			ok: true,
			exists: false,
			manifestPath: MANIFEST_REL.replace(/\\/g, '/'),
			data: { version: 1, routes: [] as ShellManifestRoute[] },
		});
	}
	try {
		const raw = fs.readFileSync(abs, 'utf-8');
		const parsed = parseYaml(raw);
		const data = normalizeManifest(parsed);
		return json({
			ok: true,
			exists: true,
			manifestPath: MANIFEST_REL.replace(/\\/g, '/'),
			data,
		});
	} catch (e) {
		return json(
			{
				ok: false,
				error: e instanceof Error ? e.message : String(e),
			},
			{ status: 500 }
		);
	}
}

export async function POST(event) {
	let body: Record<string, unknown> = {};
	try {
		body = (await event.request.json()) as Record<string, unknown>;
	} catch {
		return json({ ok: false, error: 'invalid JSON body' }, { status: 400 });
	}

	const workspaceRoot =
		(typeof body.workspaceRoot === 'string' && body.workspaceRoot) ||
		(typeof body.workspace_root === 'string' && body.workspace_root) ||
		'';
	const root = resolveRoot(workspaceRoot);
	if (!root) {
		return json({ ok: false, error: 'workspaceRoot invalid or not found' }, { status: 400 });
	}

	const action = typeof body.action === 'string' ? body.action : 'register';
	if (action !== 'register') {
		return json({ ok: false, error: 'unsupported action' }, { status: 400 });
	}

	const specName = typeof body.specName === 'string' ? body.specName.trim() : '';
	const specPath = typeof body.specPath === 'string' ? body.specPath.trim() : '';
	const displayTitle = typeof body.displayTitle === 'string' ? body.displayTitle.trim() : '';
	const yamlContent = typeof body.yamlContent === 'string' ? body.yamlContent : '';
	const entryHtmlOverride =
		typeof body.entryHtml === 'string' && body.entryHtml.trim()
			? body.entryHtml.trim()
			: null;

	if (!specName || !specPath) {
		return json({ ok: false, error: 'specName and specPath required' }, { status: 400 });
	}

	const route = buildFeatureShellRoute({
		specName,
		specPath,
		displayTitle: displayTitle || specName,
		yamlContent,
		entryHtmlOverride,
	});

	if (!isSafeWorkspaceRel(route.entryHtml)) {
		return json({ ok: false, error: 'entryHtml 路径非法（需相对工作区根，且不含 ..）' }, { status: 400 });
	}

	const abs = manifestAbs(root);
	let base: ReturnType<typeof normalizeManifest>;
	if (fs.existsSync(abs)) {
		try {
			const raw = fs.readFileSync(abs, 'utf-8');
			base = normalizeManifest(parseYaml(raw));
		} catch (e) {
			return json(
				{ ok: false, error: `读取 manifest 失败：${e instanceof Error ? e.message : String(e)}` },
				{ status: 500 }
			);
		}
	} else {
		base = { version: 1, routes: [] };
	}

	const merged = upsertRoute(base, route);
	if (!merged.ok) {
		return json({ ok: false, error: merged.error }, { status: 409 });
	}

	base.lastGenerated = new Date().toISOString();

	const dir = path.dirname(abs);
	fs.mkdirSync(dir, { recursive: true });

	const out = stringifyYaml(base, { lineWidth: 120 });
	try {
		fs.writeFileSync(abs, out, 'utf-8');
	} catch (e) {
		return json(
			{ ok: false, error: e instanceof Error ? e.message : String(e) },
			{ status: 500 }
		);
	}

	return json({
		ok: true,
		manifestPath: MANIFEST_REL.replace(/\\/g, '/'),
		route,
		data: base,
	});
}
