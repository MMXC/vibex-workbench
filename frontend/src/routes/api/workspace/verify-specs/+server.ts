import { json } from '@sveltejs/kit';
import { execSync, spawn } from 'child_process';
import path from 'path';

// GET /api/workspace/verify-specs?workspaceRoot=...&format=summary&checks=...&levels=...
// POST /api/workspace/verify-specs {workspace_root, format, checks, levels}
export async function GET(event) {
	const url = event.url;
	const workspaceRoot = url.searchParams.get('workspaceRoot') || '';
	const format = url.searchParams.get('format') || 'summary';
	const checks = url.searchParams.get('checks') || '';
	const levels = url.searchParams.get('levels') || '';
	return runVerify({ workspaceRoot, format, checks, levels });
}

export async function POST(event) {
	const body = await event.request.json().catch(() => ({}));
	return runVerify({
		workspaceRoot: body.workspace_root || body.workspaceRoot || '',
		format: body.format || 'summary',
		checks: body.checks || '',
		levels: body.levels || '',
	});
}

async function runVerify(opts: { workspaceRoot: string; format: string; checks: string; levels: string }) {
	if (!opts.workspaceRoot) {
		return json({ error: 'workspace_root is required' }, { status: 400 });
	}

	// Find verify_specs binary relative to workspace root (same dir as Makefile)
	const workspaceRoot = path.resolve(opts.workspaceRoot);
	const binary = path.join(workspaceRoot, 'verify_specs');

	// Build args
	const args = ['--workspace', workspaceRoot];
	if (opts.format) args.push('--format', opts.format);
	if (opts.checks) {
		for (const c of opts.checks.split(',').filter(Boolean)) {
			args.push('--check', c.trim());
		}
	}
	if (opts.levels) {
		for (const l of opts.levels.split(',').filter(Boolean)) {
			args.push('--level', l.trim());
		}
	}

	try {
		// Always try JSON first for structured output
		const jsonArgs = removeFormatArg(args, opts.format).concat(['--format', 'json']);
		const result = await new Promise<{ stdout: string; stderr: string; exitCode: number }>((resolve) => {
			const proc = spawn(binary, jsonArgs);
			let stdout = '';
			let stderr = '';
			proc.stdout.on('data', (d) => (stdout += d));
			proc.stderr.on('data', (d) => (stderr += d));
			const timer = setTimeout(() => proc.kill(), 60000);
			proc.on('close', (code) => { clearTimeout(timer); resolve({ stdout, stderr, exitCode: code ?? 0 }); });
		});

		if (result.exitCode === 0) {
			try {
				const parsed = JSON.parse(result.stdout);
				return json(parsed);
			} catch {
				// Not valid JSON, fall through
			}
		}

		// Fall back to summary text
		const summaryResult = await new Promise<{ stdout: string; stderr: string }>((resolve) => {
			const proc = spawn(binary, args);
			let stdout = '';
			let stderr = '';
			proc.stdout.on('data', (d) => (stdout += d));
			proc.stderr.on('data', (d) => (stderr += d));
			const timer = setTimeout(() => proc.kill(), 60000);
			proc.on('close', () => { clearTimeout(timer); resolve({ stdout, stderr }); });
		});

		return json({ output: summaryResult.stdout, stderr: summaryResult.stderr });
	} catch (e: any) {
		// Binary not found — try make verify-specs
		if (e.code === 'ENOENT' || e.message?.includes('ENOENT')) {
			try {
				const output = execSync(
					`make verify-specs WORKSPACE="${workspaceRoot}" 2>&1 || true`,
					{ cwd: workspaceRoot, timeout: 90000 }
				);
				return json({ output: output.toString() });
			} catch (makeErr: any) {
				return json({ error: 'verify_specs binary not found and make verify-specs failed', detail: makeErr.message });
			}
		}
		return json({ error: e.message });
	}
}

function removeFormatArg(args: string[], format: string): string[] {
	const out: string[] = [];
	for (let i = 0; i < args.length; i++) {
		if (args[i] === '--format' && args[i + 1] === format) {
			i++; // skip value
		} else if (args[i] === format && i > 0 && args[i - 1] === '--format') {
			// skip, already handled above
		} else {
			out.push(args[i]);
		}
	}
	return out;
}
