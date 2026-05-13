/**
 * 扩展侧 → 本机 Go Agent（默认 http://127.0.0.1:33338）的 HTTP 客户端。
 * 需 manifest host_permissions 覆盖 Agent 基址；当前 wxt 已声明 <all_urls>。
 */

const STORAGE_AGENT_BASE = 'vibexAgentBaseUrl';
const STORAGE_WORKSPACE_ROOT = 'vibexWorkspaceRoot';

export const DEFAULT_AGENT_BASE_URL = 'http://127.0.0.1:33338';

export async function getAgentBaseUrl(): Promise<string> {
	const { [STORAGE_AGENT_BASE]: v } = await browser.storage.local.get(STORAGE_AGENT_BASE);
	const s = typeof v === 'string' && v.trim() ? v.trim() : DEFAULT_AGENT_BASE_URL;
	return s.replace(/\/+$/, '');
}

export async function setAgentBaseUrl(url: string): Promise<void> {
	await browser.storage.local.set({ [STORAGE_AGENT_BASE]: url.trim() });
}

export async function getWorkspaceRootForAgent(): Promise<string> {
	const { [STORAGE_WORKSPACE_ROOT]: v } = await browser.storage.local.get(STORAGE_WORKSPACE_ROOT);
	return typeof v === 'string' ? v.trim() : '';
}

export async function setWorkspaceRootForAgent(path: string): Promise<void> {
	await browser.storage.local.set({ [STORAGE_WORKSPACE_ROOT]: path.trim() });
}

export type HealthResult = {
	ok: boolean;
	json?: Record<string, unknown>;
	error?: string;
	status?: number;
};

/** GET /health — 探测 Agent 是否在线及当前 workspace_dir。 */
export async function fetchAgentHealth(): Promise<HealthResult> {
	const base = await getAgentBaseUrl();
	try {
		const r = await fetch(`${base}/health`, { method: 'GET' });
		const text = await r.text();
		let json: Record<string, unknown> | undefined;
		try {
			json = JSON.parse(text) as Record<string, unknown>;
		} catch {
			/* 非 JSON */
		}
		if (!r.ok) return { ok: false, error: text || r.statusText, status: r.status };
		return { ok: true, json, status: r.status };
	} catch (e) {
		return { ok: false, error: e instanceof Error ? e.message : String(e) };
	}
}

export type ChatPostResult = {
	ok: boolean;
	threadId?: string;
	error?: string;
	status?: number;
};

/**
 * POST /api/chat — 投递一轮对话（异步；SSE 在 Workbench 或另接 EventSource）。
 * workspace_root 必填，与 Agent chatHandler 校验一致。
 */
export async function postAgentChat(
	input: string,
	opts?: { threadId?: string; agentProfile?: string }
): Promise<ChatPostResult> {
	const base = await getAgentBaseUrl();
	const workspace = await getWorkspaceRootForAgent();
	if (!workspace) {
		return { ok: false, error: '请先在侧栏填写「工作区根路径」（本机打开的项目目录）' };
	}
	const threadId = (opts?.threadId?.trim() || `ext-${Date.now()}`).slice(0, 120);
	const body: Record<string, string> = {
		threadId,
		input,
		workspace_root: workspace,
	};
	if (opts?.agentProfile?.trim()) {
		body.agent_profile = opts.agentProfile.trim();
	}
	try {
		const r = await fetch(`${base}/api/chat`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(body),
		});
		const text = await r.text();
		if (!r.ok) {
			return { ok: false, error: text || r.statusText, status: r.status };
		}
		let j: { threadId?: string } = {};
		try {
			j = JSON.parse(text) as { threadId?: string };
		} catch {
			/* ignore */
		}
		return { ok: true, threadId: j.threadId ?? threadId, status: r.status };
	} catch (e) {
		return { ok: false, error: e instanceof Error ? e.message : String(e) };
	}
}
