import { agentApiUrl } from '$lib/runtime/agent-transport';

/** GET `/api/workspace/file/<path>?workspaceRoot=` — 供 iframe 预览 HTML（相对路径使用 `/`）。 */
export function toWorkspaceFileURL(relPath: string, workspaceRoot: string): string {
	const normalized = relPath
		.trim()
		.replace(/^[/\\]+/, '')
		.split(/[/\\]+/)
		.filter(Boolean)
		.map(seg => encodeURIComponent(seg))
		.join('/');
	return `${agentApiUrl('/api/workspace/file/' + normalized)}?workspaceRoot=${encodeURIComponent(workspaceRoot)}`;
}
