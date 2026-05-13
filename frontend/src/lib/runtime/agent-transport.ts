/**
 * Agent / vibex-agent HTTP API 基址（VITE_SSE_URL），与同源的 workspace SvelteKit `/api/workspace/*` 分离。
 * 见 .vibex/specs/L4-feature/FEAT-workbench-transport-adapter.yaml
 */

import { get } from 'svelte/store';
import { specExplorerStore } from '$lib/stores/spec-explorer-store';

export function getAgentApiBase(): string {
	const u = import.meta.env.VITE_SSE_URL;
	if (typeof u === 'string' && u.trim()) {
		return u.trim().replace(/\/$/, '');
	}
	return 'http://localhost:33338';
}

/** 拼接 Agent 侧绝对 URL（路径须以 / 开头）。 */
export function agentApiUrl(path: string): string {
	const base = getAgentApiBase();
	const p = path.startsWith('/') ? path : `/${path}`;
	return `${base}${p}`;
}

/** HTTP(S) → ws(s) 基址（Inspector WebSocket）。 */
export function httpToWsBase(httpUrl: string): string {
	let u = httpUrl.trim().replace(/\/$/, '');
	if (u.startsWith('https://')) return `wss://${u.slice('https://'.length)}`;
	if (u.startsWith('http://')) return `ws://${u.slice('http://'.length)}`;
	return u;
}

export function inspectorWsUrl(): string {
	return `${httpToWsBase(getAgentApiBase())}/api/workbench/inspector/ws`;
}

/** 解析对话使用的 workspace 根路径：优先显式参数，否则用当前 Workbench 已选工作区。 */
export function resolveWorkspaceRootForAgent(override?: string | null): string | null {
	const t = override?.trim();
	if (t) return t;
	const w = get(specExplorerStore).workspaceRoot?.trim();
	return w || null;
}

/**
 * POST /api/chat：合并 `workspaceRoot`（请求体或 store），两者皆空则抛错。
 * 后端亦拒绝无 workspaceRoot 的请求，此处前置避免误写到 agent 默认目录。
 */
export async function postAgentChat(
	body: Record<string, unknown>,
	init?: RequestInit
): Promise<Response> {
	const keys = ['workspaceRoot', 'workspace_root', 'workRootDir', 'work_root_dir'] as const;
	let explicit: string | undefined;
	for (const k of keys) {
		const v = body[k];
		if (typeof v === 'string' && v.trim()) {
			explicit = v.trim();
			break;
		}
	}
	const root = resolveWorkspaceRootForAgent(explicit);
	if (!root) {
		throw new Error('workspaceRoot required：请先在 Workbench 中选择工作区目录');
	}
	const merged = { ...body, workspaceRoot: root };
	const headers = new Headers(init?.headers as HeadersInit | undefined);
	if (!headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}
	return fetch(agentApiUrl('/api/chat'), {
		method: 'POST',
		headers,
		body: JSON.stringify(merged),
		signal: init?.signal,
	});
}
