import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';

export async function GET(event) {
	const url = event.url;
	const workspaceRoot = url.searchParams.get('workspaceRoot') || url.searchParams.get('workspace_root') || '';

	if (!workspaceRoot) {
		return json({ error: 'workspaceRoot required' }, { status: 400 });
	}

	const specsDir = path.join(workspaceRoot, 'specs');
	if (!fs.existsSync(specsDir)) {
		return json({ paths: [], error: 'specs/ directory not found' });
	}

	const paths: string[] = [];

	function walk(dir: string, base: string) {
		for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
			const full = path.join(dir, entry.name);
			const rel = path.relative(base, full);
			if (entry.isDirectory()) {
				walk(full, base);
			} else if (entry.name.endsWith('.yaml') || entry.name.endsWith('.yml')) {
				paths.push(rel);
			}
		}
	}

	walk(specsDir, specsDir);
	paths.sort();
	return json({ paths });
}
