import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';
import { CANONICAL_SPEC_SUBDIRS } from '$lib/workbench/spec-layout-dirs';

/** Dev fallback: create canonical specs dirs locally (same contract as Go speclayout). */
export async function POST(event) {
	const body = await event.request.json().catch(() => ({}));
	const workspaceRoot = String(body.workspace_root ?? body.workspaceRoot ?? '').trim();
	if (!workspaceRoot) {
		return json({ ok: false, error: 'workspace_root required' }, { status: 400 });
	}

	const created: string[] = [];
	const skipped: string[] = [];

	for (const rel of CANONICAL_SPEC_SUBDIRS) {
		const full = path.join(workspaceRoot, ...rel.split('/'));
		try {
			if (fs.existsSync(full)) {
				const st = fs.statSync(full);
				if (st.isDirectory()) {
					skipped.push(rel);
					continue;
				}
				return json(
					{ ok: false, error: `path exists but is not a directory: ${rel}` },
					{ status: 409 }
				);
			}
			fs.mkdirSync(full, { recursive: true });
			created.push(rel);
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

	return json({ ok: true, created, skipped });
}
