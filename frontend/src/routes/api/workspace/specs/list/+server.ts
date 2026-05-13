import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';

export async function GET(event) {
	const url = event.url;
	const workspaceRoot = url.searchParams.get('workspaceRoot') || url.searchParams.get('workspace_root') || '';

	if (!workspaceRoot) {
		return json({ error: 'workspaceRoot required' }, { status: 400 });
	}

	const specsDir = path.join(workspaceRoot, '.vibex', 'specs');
	const paths: string[] = [];

	if (fs.existsSync(specsDir)) {
		function walk(dir: string) {
			for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
				const full = path.join(dir, entry.name);
				const rel = path.relative(workspaceRoot, full);
				if (entry.isDirectory()) {
					walk(full);
				} else if (entry.name.endsWith('.yaml') || entry.name.endsWith('.yml')) {
					paths.push(rel.split(path.sep).join('/'));
				}
			}
		}
		walk(specsDir);
	}

	paths.sort();
	return json({ paths });
}
