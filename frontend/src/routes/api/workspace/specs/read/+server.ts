import { readFile } from 'node:fs/promises';
import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { resolveSafeSpecsPath } from '$lib/server/safe-spec-path';

/** 本地开发：读取 WORKSPACE_ROOT/specs 下 YAML 文本（供总目标图谱等拉取绑定表与 L1） */
export const GET: RequestHandler = async ({ url }) => {
	const rel = url.searchParams.get('path');
	if (!rel) throw error(400, 'missing query path');

	const full = resolveSafeSpecsPath(rel);
	if (!full) throw error(403, 'path not under specs/');

	try {
		const content = await readFile(full, 'utf-8');
		return json({ path: rel, content });
	} catch (e) {
		const msg = e instanceof Error ? e.message : String(e);
		throw error(404, msg);
	}
};
