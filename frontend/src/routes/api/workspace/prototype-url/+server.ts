import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';

const PROTOTYPE_URL_FILE = '.vibex/.vibex-prototype-url';

export async function GET(event) {
	const workspaceRoot = event.url.searchParams.get('workspaceRoot')
		|| event.url.searchParams.get('workspace_root')
		|| '';

	if (!workspaceRoot) {
		return json({ url: '' });
	}

	const urlFile = path.join(workspaceRoot, PROTOTYPE_URL_FILE);
	let url = '';
	if (fs.existsSync(urlFile)) {
		url = fs.readFileSync(urlFile, 'utf-8').trim();
	}

	return json({ url });
}
