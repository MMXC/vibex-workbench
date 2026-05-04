import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';

const DESIGN_REL = '.vibex/design/DESIGN.md';
const PROTOS_REL = '.vibex/prototypes';

export async function GET(event) {
	const url = event.url;
	const workspaceRoot = url.searchParams.get('workspaceRoot') || url.searchParams.get('workspace_root') || '';

	if (!workspaceRoot) {
		return json({ ok: false, error: 'workspaceRoot required' }, { status: 400 });
	}

	const root = path.resolve(workspaceRoot);
	if (!fs.existsSync(root)) {
		return json({ ok: false, error: 'workspace not found' }, { status: 400 });
	}

	const designAbs = path.join(root, '.vibex', 'design', 'DESIGN.md');
	const protosAbs = path.join(root, '.vibex', 'prototypes');

	return json({
		ok: true,
		workspaceRoot: root,
		designMdExists: fs.existsSync(designAbs),
		prototypesDirExists: fs.existsSync(protosAbs),
		designPath: DESIGN_REL,
		prototypesPath: PROTOS_REL,
	});
}
