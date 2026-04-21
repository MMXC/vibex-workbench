import { readdir } from 'node:fs/promises';
import path from 'node:path';
import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getWorkspaceRoot } from '$lib/server/workspace-root';

async function collectYaml(dir: string, root: string, out: string[]): Promise<void> {
	const entries = await readdir(dir, { withFileTypes: true });
	for (const e of entries) {
		const full = path.join(dir, e.name);
		if (e.isDirectory()) {
			await collectYaml(full, root, out);
		} else if (/\.ya?ml$/i.test(e.name)) {
			const rel = path.relative(root, full).replace(/\\/g, '/');
			out.push(rel);
		}
	}
}

/** 列出仓库 specs/ 下所有 .yaml/.yml（相对仓库根的路径，如 specs/feature/foo.yaml） */
export const GET: RequestHandler = async () => {
	const root = getWorkspaceRoot();
	const specsDir = path.join(root, 'specs');
	try {
		const paths: string[] = [];
		await collectYaml(specsDir, root, paths);
		paths.sort((a, b) => a.localeCompare(b));
		return json({ paths });
	} catch (e) {
		const msg = e instanceof Error ? e.message : String(e);
		throw error(500, msg);
	}
};
