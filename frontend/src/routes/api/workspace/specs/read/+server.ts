import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';

export async function GET(event) {
	const url = event.url;
	const workspaceRoot = url.searchParams.get('workspaceRoot') || '';
	const filePath = url.searchParams.get('path') || '';

	if (!workspaceRoot || !filePath) {
		return json({ error: 'workspaceRoot and path required' }, { status: 400 });
	}

	const fullPath = path.join(workspaceRoot, filePath);
	// Security: ensure the resolved path is within workspaceRoot
	if (!fullPath.startsWith(path.resolve(workspaceRoot))) {
		return json({ error: 'Invalid path' }, { status: 403 });
	}

	if (!fs.existsSync(fullPath)) {
		return json({ error: 'File not found' }, { status: 404 });
	}

	try {
		const content = fs.readFileSync(fullPath, 'utf-8');
		return json({ content, path: filePath });
	} catch (e: any) {
		return json({ error: e.message });
	}
}
