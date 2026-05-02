/**
 * Wails Filesystem Bridge — Go filesystem binding wrappers
 *
 * Production (Wails): calls window.go.main.App.* directly (zero HTTP overhead)
 * Development (Vite): falls back to HTTP via Vite proxy
 *
 * 调用约定：
 *   1. isWails() → 确认在 Wails WebView 内
 *   2. rt.ListSpecs 存在 → 确认 binding 方法已注册
 *   3. 调用之，catch → 打出诊断信息，不静默 fallback
 */

import { extractSpecDisplay, type SpecDisplay, type SpecSlotModel } from './workbench/spec-display';
import { isWails } from './wails-runtime';

// ── Types ─────────────────────────────────────────────────────

export interface WailsSpecFile {
	path: string;   // 相对路径
	level: number;  // 1-5
	name: string;    // frontmatter.name 或文件名
	status: string; // frontmatter.status
	display?: SpecDisplay;
	slots?: SpecSlotModel;
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

function getGoApp(): any | null {
	return (window as any).go?.main?.App ?? null;
}

function isLikelyFullPath(p: string): boolean {
	if (!p) return false;
	return p.includes('/') || p.includes('\\');
}

/**
 * 检查 Wails binding 方法是否存在。
 * 在 Wails runtime 完全 init 前，window.go.main.App.* 方法尚不存在。
 */
function hasBinding(name: string): boolean {
	const rt = getRuntime();
	return !!(rt && (rt as any).ListSpecs !== undefined);
}

function levelTokenToNumber(level: string): number {
	const match = level.match(/^L([1-5])$/);
	return match ? Number(match[1]) : 0;
}

function inferLevelFromPath(path: string): number {
	const match = path.match(/(?:^|\/)L([1-5])[-_]/i);
	return match ? Number(match[1]) : 0;
}

async function enrichSpecFiles(root: string, files: WailsSpecFile[]): Promise<WailsSpecFile[]> {
	return Promise.all(
		files.map(async file => {
			try {
				const content = await wailsReadSpecFile(root, file.path);
				const meta = extractSpecDisplay(content, file.path);
				return {
					...file,
					level: file.level || levelTokenToNumber(meta.level) || inferLevelFromPath(file.path),
					name: meta.name || file.name,
					status: meta.status || file.status,
					display: meta.display,
					slots: meta.slots,
				};
			} catch {
				return {
					...file,
					level: file.level || inferLevelFromPath(file.path),
				};
			}
		})
	);
}

// ── ListSpecs ──────────────────────────────────────────────────

/**
 * 列出 {root}/specs/ 下所有 .yaml 文件的元信息。
 * 生产用 Wails binding，开发用 HTTP fallback。
 */
export async function wailsListSpecs(root: string): Promise<WailsSpecFile[]> {
	if (!root) return [];
	if (!isLikelyFullPath(root)) {
		throw new Error(`Invalid workspace root (not full path): ${root}`);
	}

	const app = getGoApp();
	// 优先用 Wails binding（仅在方法已注册时）
	if (isWails() && app && typeof (app as any).ListSpecs === 'function') {
		try {
			console.log('[wails-filesystem] ListSpecs via Wails, root=', root);
			const files = (await (app as any).ListSpecs(root)) as WailsSpecFile[];
			return enrichSpecFiles(root, files);
		} catch (e) {
			console.error('[wails-filesystem] ListSpecs Wails call failed:', e);
			// Wails 调用失败时不应静默降级到 HTTP（在 Wails 环境里 HTTP 走不通）
			throw e;
		}
	}
	if (isWails()) {
		throw new Error('Wails binding missing: App.ListSpecs');
	}

	// HTTP fallback for browser dev
	console.log('[wails-filesystem] ListSpecs via HTTP, root=', root);
	const r = await fetch(
		`/api/workspace/specs/list?workspaceRoot=${encodeURIComponent(root)}`
	);
	if (!r.ok) return [];
	const j = await r.json();
	const files = (j.paths ?? []).map((p: string) => ({
		path: p,
		level: inferLevelFromPath(p),
		name: p.split('/').pop() ?? p,
		status: 'active',
	}));
	return enrichSpecFiles(root, files);
}

// ── ReadSpecFile ───────────────────────────────────────────────

/**
 * 读取单个 spec 文件内容。
 */
export async function wailsReadSpecFile(
	root: string,
	path: string
): Promise<string> {
	if (!root || !path) throw new Error('root and path required');
	if (!isLikelyFullPath(root)) {
		throw new Error(`Invalid workspace root (not full path): ${root}`);
	}

	const app = getGoApp();
	if (isWails() && app && typeof (app as any).ReadSpecFile === 'function') {
		try {
			return (await (app as any).ReadSpecFile(root, path)) as string;
		} catch (e) {
			console.error('[wails-filesystem] ReadSpecFile Wails call failed:', e);
			throw e;
		}
	}
	if (isWails()) {
		throw new Error('Wails binding missing: App.ReadSpecFile');
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
	if (!root || !path) throw new Error('root and path required');
	if (!isLikelyFullPath(root)) {
		throw new Error(`Invalid workspace root (not full path): ${root}`);
	}

	const app = getGoApp();
	if (isWails() && app && typeof (app as any).WriteSpecFile === 'function') {
		try {
			await (app as any).WriteSpecFile(root, path, content);
			return;
		} catch (e) {
			console.error('[wails-filesystem] WriteSpecFile Wails call failed:', e);
			throw e;
		}
	}
	if (isWails()) {
		throw new Error('Wails binding missing: App.WriteSpecFile');
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
	if (!root) return { state: 'error', signals: [], suggestions: ['workspace root 为空'] };
	if (!isLikelyFullPath(root)) {
		throw new Error(`Invalid workspace root (not full path): ${root}`);
	}

	const app = getGoApp();
	if (isWails() && app && typeof (app as any).DetectWorkspaceState === 'function') {
		try {
			return (await (app as any).DetectWorkspaceState(root)) as WailsWorkspaceState;
		} catch (e) {
			console.error('[wails-filesystem] DetectWorkspaceState Wails call failed:', e);
			throw e;
		}
	}
	if (isWails()) {
		throw new Error('Wails binding missing: App.DetectWorkspaceState');
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
 * Wails 环境专用（HTTP fallback 无意义）。
 */
export async function wailsRunMake(
	target: string,
	workspace: string
): Promise<{ ok: boolean; output: string }> {
	if (!isWails()) throw new Error('wailsRunMake requires Wails mode');
	const app = getGoApp();
	if (!app || typeof (app as any).RunMake !== 'function') {
		throw new Error('Wails binding missing: App.RunMake');
	}
	const result = await (app as any).RunMake(target, workspace);
	return result as { ok: boolean; output: string };
}
