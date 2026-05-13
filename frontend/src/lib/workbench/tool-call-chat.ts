/** 将 agent 工具调用格式化为聊天块，并解析可在主区打开的工作区相对路径 */

const PATH_KEYS = [
	'path',
	'file_path',
	'filePath',
	'target_path',
	'targetPath',
	'filepath',
	'relativePath',
	'file',
];

function isPlausibleRelPath(p: string): boolean {
	const t = p.trim();
	if (!t || t.length > 512) return false;
	if (/^https?:\/\//i.test(t) || t.startsWith('data:')) return false;
	if (t.includes('\n') || t.includes('\r')) return false;
	return true;
}

export function extractToolWorkspacePath(toolName: string, args: unknown): string | null {
	if (!args || typeof args !== 'object') return null;
	const o = args as Record<string, unknown>;
	for (const k of PATH_KEYS) {
		const v = o[k];
		if (typeof v === 'string' && isPlausibleRelPath(v)) {
			return v.trim().replace(/^[/\\]+/, '').replace(/\\/g, '/');
		}
	}
	if (toolName === 'bash' && typeof o.command === 'string') {
		const m = o.command.match(/(?:^|\s)([\w./-]+\.(?:ya?ml|md|html|ts|tsx|js|jsx|go|rs|json))\b/);
		if (m?.[1] && isPlausibleRelPath(m[1])) return m[1].replace(/\\/g, '/');
	}
	return null;
}

export function summarizeToolArgs(args: Record<string, unknown> | undefined, maxKeys = 5): string {
	if (!args) return '';
	const keys = Object.keys(args).slice(0, maxKeys);
	return keys.map(key => `${key}=${String(args[key]).slice(0, 96)}`).join(' · ');
}

export function formatToolCalledBody(toolName: string, args: unknown): string {
	const o = args && typeof args === 'object' ? (args as Record<string, unknown>) : undefined;
	const path = extractToolWorkspacePath(toolName, args);
	const lines = [`▸ ${toolName}`];
	if (path) lines.push(`path: ${path}`);
	const sum = summarizeToolArgs(o);
	if (sum) lines.push(sum);
	return lines.join('\n');
}

export function formatToolCompletedFooter(result: string, failed: boolean): string {
	const head = failed ? '✗ 工具结束（异常）' : '✓ 工具结束';
	const body = (result ?? '').trim();
	const cap = body.slice(0, 2400);
	const tail = body.length > 2400 ? '\n…（截断）' : '';
	return `\n\n${head}\n${cap}${tail}`;
}
