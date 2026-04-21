import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { parse as parseYaml } from 'yaml';
import { getWorkspaceRoot } from '$lib/server/workspace-root';

const REL = 'specs/meta/spec-directory-convention.yaml';

/** GET：返回 specs/meta/spec-directory-convention.yaml 解析后的 JSON（目录层级 + 关联边） */
export const GET: RequestHandler = async () => {
	const root = getWorkspaceRoot();
	const abs = path.join(root, REL);
	try {
		const raw = await readFile(abs, 'utf-8');
		const doc = parseYaml(raw) as Record<string, unknown>;
		return json({
			source: REL,
			convention: doc,
		});
	} catch (e) {
		const msg = e instanceof Error ? e.message : String(e);
		throw error(500, `cannot read convention: ${msg}`);
	}
};
