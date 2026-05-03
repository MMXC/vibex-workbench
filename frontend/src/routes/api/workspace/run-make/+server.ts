import { json } from '@sveltejs/kit';
import { execSync } from 'child_process';
import path from 'path';

export async function POST(event) {
	const body = await event.request.json().catch(() => ({}));
	const target = body.target || 'help';
	const workspaceRoot = path.resolve(body.workspace_root || body.workspaceRoot || process.cwd());

	if (!target) {
		return json({ ok: false, error: 'target required' });
	}

	try {
		const output = execSync(`make ${target}`, {
			cwd: workspaceRoot,
			timeout: 120000,
			env: { ...process.env, WORKSPACE: workspaceRoot },
		});
		return json({ ok: true, output: output.toString() });
	} catch (e: any) {
		const stderr = e.stderr?.toString() || '';
		const stdout = e.stdout?.toString() || '';
		return json({
			ok: false,
			error: e.message,
			output: stdout || stderr,
			exitCode: e.status,
		});
	}
}
