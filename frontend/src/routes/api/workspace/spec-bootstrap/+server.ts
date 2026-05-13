import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';
import { execFileSync } from 'child_process';

function findWorkbenchRoot(): string {
	const env = process.env.VIBEX_WORKBENCH_ROOT?.trim();
	if (env && fs.existsSync(path.join(env, '.vibex', 'agents', 'skills', 'workspace-bootstrap', 'scripts', 'execute.py'))) {
		return path.resolve(env);
	}
	if (env && fs.existsSync(path.join(env, 'generators', 'spec_workspace_bootstrap.py'))) {
		return path.resolve(env);
	}
	let d = process.cwd();
	for (let i = 0; i < 10; i++) {
		const script = path.join(d, '.vibex', 'agents', 'skills', 'workspace-bootstrap', 'scripts', 'execute.py');
		if (fs.existsSync(script)) {
			return d;
		}
		const legacyScript = path.join(d, 'generators', 'spec_workspace_bootstrap.py');
		if (fs.existsSync(legacyScript)) {
			return d;
		}
		const up = path.dirname(d);
		if (up === d) break;
		d = up;
	}
	return '';
}

function resolveRunnerScripts(vxRoot: string): { preferred: string; legacy: string } {
	return {
		preferred: path.join(vxRoot, '.vibex', 'agents', 'skills', 'workspace-bootstrap', 'scripts', 'execute.py'),
		legacy: path.join(vxRoot, 'generators', 'spec_workspace_bootstrap.py'),
	};
}

export async function POST(event) {
	const body = await event.request.json().catch(() => ({}));
	const workspaceRoot = path.resolve(body.workspace_root || body.workspaceRoot || '');
	const confirm = body.confirm === true;
	const overwrite = body.overwrite === true;
	const projectSlug = String(body.project_slug || body.projectSlug || '').trim();
	const owner = String(body.owner || 'user').trim() || 'user';

	if (!workspaceRoot) {
		return json({ ok: false, error: 'workspace_root required' }, { status: 400 });
	}
	if (!fs.existsSync(workspaceRoot)) {
		return json({ ok: false, error: 'Directory does not exist' }, { status: 400 });
	}
	if (!confirm) {
		return json({
			ok: false,
			error: 'spec-bootstrap requires confirm=true',
			hint: '从 spec-templates 生成 L1–L5 占位链（workspace_bootstrap_contract）',
		});
	}

	const vxRoot = findWorkbenchRoot();
	const scripts = resolveRunnerScripts(vxRoot);
	const usePreferred = vxRoot && fs.existsSync(scripts.preferred);
	const scriptPath = usePreferred ? scripts.preferred : scripts.legacy;
	if (!vxRoot || !fs.existsSync(scriptPath)) {
		return json({
			ok: false,
			error:
				'未找到 workspace-bootstrap skill execute（且 legacy script 缺失）。请在 VibeX Workbench 安装/源码根目录启动 dev，或设置 VIBEX_WORKBENCH_ROOT 指向该根目录。',
		});
	}

	const args = [scriptPath];
	if (usePreferred) {
		args.push('--workspace-root', workspaceRoot, '--json', '--owner', owner, '--confirm');
		if (projectSlug) args.push('--project-slug', projectSlug);
		if (overwrite) args.push('--overwrite');
	} else {
		args.push(workspaceRoot, '--json', '--owner', owner);
		if (projectSlug) args.push('--project-slug', projectSlug);
		if (overwrite) args.push('--overwrite');
	}

	const pyCandidates =
		process.platform === 'win32' ? ['python', 'python3'] : ['python3', 'python'];

	let lastErr: unknown;
	for (const py of pyCandidates) {
		try {
			const out = execFileSync(py, args, {
				cwd: vxRoot,
				encoding: 'utf-8',
				maxBuffer: 8 * 1024 * 1024,
				windowsHide: true,
			});
			const result = JSON.parse(out.trim()) as Record<string, unknown>;
			return json(result);
		} catch (e: unknown) {
			lastErr = e;
			continue;
		}
	}

	const err = lastErr as { status?: number; stdout?: Buffer; stderr?: Buffer; message?: string } | undefined;
	if (!err) {
		return json({ ok: false, error: '未找到可用的 python / python3' }, { status: 500 });
	}
	let parsed: Record<string, unknown> | null = null;
	const raw = (err.stdout?.toString('utf-8') || err.stderr?.toString('utf-8') || '').trim();
	try {
		if (raw) {
			parsed = JSON.parse(raw) as Record<string, unknown>;
		}
	} catch {
		/* ignore */
	}
	if (parsed) {
		return json({ ...parsed, _spawn_message: err.message });
	}
	return json(
		{
			ok: false,
			error: err.message || 'spec_workspace_bootstrap failed',
			raw: raw.slice(0, 4000),
		},
		{ status: 500 },
	);
}
