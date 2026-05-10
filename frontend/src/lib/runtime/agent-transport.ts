/**
 * Agent / vibex-agent HTTP API 基址（VITE_SSE_URL），与同源的 workspace SvelteKit `/api/workspace/*` 分离。
 * 见 specs/L4-feature/FEAT-workbench-transport-adapter.yaml
 */

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
