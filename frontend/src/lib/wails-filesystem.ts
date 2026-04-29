/**
 * Wails Filesystem Bridge — Go filesystem binding wrappers
 *
 * Production (Wails): calls window.go.main.App.* directly (zero HTTP overhead)
 * Development (Vite): falls back to HTTP via Vite proxy
 */

import { isWails } from './wails-runtime';

// ── Types ─────────────────────────────────────────────────────

export interface WailsSpecFile {
	path: string;   // 相对路径
	level: number;  // 1-5
	name: string;    // frontmatter.name 或文件名
	status: string; // frontmatter.status
}

export interface WailsWorkspaceState {
	state: 'empty' | 'half' | 'ready' | 'error';
	signals: { path: string; exists: boolean; reason: string }[];
	suggestions: string[];
}

// ── Helpers ─────────────────────────────────────────────────────

function getRuntime(): any | null {
	return (window as any).runtime ?? null;
}

// ── ListSpecs ──────────────────────────────────────────────────

/**
 * 列出 {root}/specs/ 下所有 .yaml 文件的元信息。
 * 生产用 Wails binding，开发用 HTTP fallback。
 */
export async function wailsListSpecs(root: string): Promise<WailsSpecFile[]> {
	const rt = getRuntime();
	if (rt) {
		try {
			return (await rt.ListSpecs(root)) as WailsSpecFile[];
		} catch (e) {
			console.warn('[wails-filesystem] ListSpecs fallback to HTTP:', e);
		}
	}
	// HTTP fallback for browser dev
	const r = await fetch(
		`/api/workspace/specs/list?workspaceRoot=${encodeURIComponent(root)}`
	);
	if (!r.ok) return [];
	const j = await r.json();
	// HTTP API 返回 { paths: string[] }，转成 WailsSpecFile[]
	return (j.paths ?? []).map((p: string) => ({
		path: p,
		level: 0,
		name: p.split('/').pop() ?? p,
		status: 'active',
	}));
}

// ── ReadSpecFile ───────────────────────────────────────────────

/**
 * 读取单个 spec 文件内容。
 */
export async function wailsReadSpecFile(
	root: string,
	path: string
): Promise<string> {
	const rt = getRuntime();
	if (rt) {
		try {
			return (await rt.ReadSpecFile(root, path)) as string;
		} catch (e) {
			console.warn('[wails-filesystem] ReadSpecFile fallback to HTTP:', e);
		}
	}
	// HTTP fallback
	const r = await fetch(
		`/api/workspace/specs/read?workspaceRoot=${encodeURIComponent(root)}&path=${encodeURIComponent(path)}`
	);
	if (!r.ok) throw new Error(await r.text());
	const j = await r.json();
	return (j.content ?? '') as string;
}

// ── WriteSpecFile ──────────────────────────────────────────────

/**
 * 写入 spec 文件（自动创建中间目录）。
 */
export async function wailsWriteSpecFile(
	root: string,
	path: string,
	content: string
): Promise<void> {
	const rt = getRuntime();
	if (rt) {
		try {
			await rt.WriteSpecFile(root, path, content);
			return;
		} catch (e) {
			console.warn('[wails-filesystem] WriteSpecFile fallback to HTTP:', e);
		}
	}
	// HTTP fallback
	const r = await fetch('/api/workspace/specs/write', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ workspaceRoot: root, path, content }),
	});
	if (!r.ok) throw new Error(await r.text());
}

// ── DetectWorkspaceState ────────────────────────────────────────

/**
 * 检测工作区状态（empty / half / ready）。
 */
export async function wailsDetectWorkspaceState(
	root: string
): Promise<WailsWorkspaceState> {
	const rt = getRuntime();
	if (rt) {
		try {
			return (await rt.DetectWorkspaceState(root)) as WailsWorkspaceState;
		} catch (e) {
			console.warn('[wails-filesystem] DetectWorkspaceState fallback to HTTP:', e);
		}
	}
	// HTTP fallback
	const r = await fetch(
		`/api/workspace/detect-state?workspaceRoot=${encodeURIComponent(root)}`
	);
	if (!r.ok) {
		return { state: 'error', signals: [], suggestions: ['无法检测工作区状态'] };
	}
	return (await r.json()) as WailsWorkspaceState;
}

// ── RunMake ────────────────────────────────────────────────────

/**
 * 在 workspace 执行 make target。
 */
export async function wailsRunMake(
	target: string,
	workspace: string
): Promise<{ ok: boolean; output: string }> {
	const rt = getRuntime();
	if (!rt) throw new Error('RunMake requires Wails mode');
	const result = await rt.RunMake(target, workspace);
	return result as { ok: boolean; output: string };
}
