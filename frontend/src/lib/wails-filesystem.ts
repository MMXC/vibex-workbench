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
	/** spec.parent（便于血缘挂载等） */
	parent?: string | null;
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

/** 同步探测：当前 WebView 是否具备 Go App 及 disk 相关 binding（不发磁盘请求）。 */
export type WailsFilesystemBindingStatus = {
	isWailsRuntime: boolean;
	hasGoApp: boolean;
	bindings: {
		ListSpecs: boolean;
		ReadSpecFile: boolean;
		WriteSpecFile: boolean;
		DetectWorkspaceState: boolean;
	};
};

export function getWailsFilesystemBindingStatus(): WailsFilesystemBindingStatus {
	const app = getGoApp();
	const fn = (n: string) => {
		if (!app) return false;
		return typeof (app as Record<string, unknown>)[n] === 'function';
	};
	return {
		isWailsRuntime: isWails(),
		hasGoApp: !!app,
		bindings: {
			ListSpecs: fn('ListSpecs'),
			ReadSpecFile: fn('ReadSpecFile'),
			WriteSpecFile: fn('WriteSpecFile'),
			DetectWorkspaceState: fn('DetectWorkspaceState'),
		},
	};
}

/**
 * 烟雾测试：使用与业务相同的 `wailsListSpecs` 路径验证 disk 通路。
 * - 浏览器 dev：期望走 HTTP 回落且返回 spec 列表。
 * - Wails：期望 ListSpecs binding 存在且调用成功后再考虑把槽位 specs I/O 从 Agent HTTP 迁到本模块。
 */
export async function probeWailsFilesystemListSpecs(workspaceRoot: string): Promise<{
	ok: boolean;
	specCount: number;
	channel: 'wails_binding' | 'http_fallback';
	error?: string;
}> {
	const before = getWailsFilesystemBindingStatus();
	try {
		const files = await wailsListSpecs(workspaceRoot);
		const channel =
			before.isWailsRuntime && before.bindings.ListSpecs ? 'wails_binding' : 'http_fallback';
		return { ok: true, specCount: files.length, channel };
	} catch (e) {
		const channel =
			before.isWailsRuntime && before.bindings.ListSpecs ? 'wails_binding' : 'http_fallback';
		return {
			ok: false,
			specCount: 0,
			channel,
			error: e instanceof Error ? e.message : String(e),
		};
	}
}

function levelTokenToNumber(level: string): number {
	const match = level.match(/^L([1-5])$/);
	return match ? Number(match[1]) : 0;
}

function inferLevelFromPath(path: string): number {
	const match = path.match(/(?:^|\/)L([1-5])[-_]/i);
	return match ? Number(match[1]) : 0;
}

function normalizeSpecPath(path: string): string {
	const normalized = path.replace(/\\/g, '/').replace(/^\/+/, '');
	return normalized.startsWith('specs/') ? normalized : `specs/${normalized}`;
}

function parseSpecListPayload(payload: unknown): WailsSpecFile[] {
	if (!payload || typeof payload !== 'object') return [];
	const record = payload as Record<string, unknown>;
	const raw = Array.isArray(record.paths)
		? record.paths
		: Array.isArray(record.specs)
			? record.specs
			: Array.isArray(record.files)
				? record.files
				: [];

	return raw
		.map(item => {
			if (typeof item === 'string') {
				const path = normalizeSpecPath(item);
				return {
					path,
					level: inferLevelFromPath(path),
					name: path.split('/').pop() ?? path,
					status: 'active',
				};
			}
			if (item && typeof item === 'object') {
				const spec = item as Partial<WailsSpecFile> & { filePath?: string; filepath?: string };
				const rawPath = spec.path ?? spec.filePath ?? spec.filepath;
				if (typeof rawPath !== 'string') return null;
				const path = normalizeSpecPath(rawPath);
				return {
					path,
					level: spec.level || inferLevelFromPath(path),
					name: spec.name || path.split('/').pop() || path,
					status: spec.status || 'active',
					display: spec.display,
					slots: spec.slots,
				};
			}
			return null;
		})
		.filter((item): item is WailsSpecFile => item !== null);
}

async function enrichSpecFiles(root: string, files: WailsSpecFile[]): Promise<WailsSpecFile[]> {
	return Promise.all(
		files.map(async file => {
			const path = normalizeSpecPath(file.path);
			try {
				const content = await wailsReadSpecFile(root, path);
				const meta = extractSpecDisplay(content, path);
				return {
					...file,
					path,
					level: file.level || levelTokenToNumber(meta.level) || inferLevelFromPath(path),
					name: meta.name || file.name,
					status: meta.status || file.status,
					parent: meta.parent ?? undefined,
					display: meta.display,
					slots: meta.slots,
				};
			} catch {
				return {
					...file,
					path,
					level: file.level || inferLevelFromPath(path),
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
	if (root && !isLikelyFullPath(root)) {
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
	const query = root ? `?workspaceRoot=${encodeURIComponent(root)}` : '';
	const r = await fetch(`/api/workspace/specs/list${query}`);
	if (!r.ok) return [];
	const j = await r.json();
	const files = parseSpecListPayload(j);
	if (files.length === 0) {
		console.warn('[wails-filesystem] ListSpecs returned no parsable spec files:', j);
	}
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
	if (!path) throw new Error('path required');
	if (root && !isLikelyFullPath(root)) {
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
	const params = new URLSearchParams({ path });
	if (root) params.set('workspaceRoot', root);
	const r = await fetch(`/api/workspace/specs/read?${params.toString()}`);
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

export type SpecsInitLayoutResult = {
	ok: boolean;
	created?: string[];
	skipped?: string[];
	error?: string;
};

/** Create canonical specs/L1-goal … / L5-slice / _governance dirs under workspace (idempotent). */
export async function wailsInitSpecsLayout(root: string): Promise<SpecsInitLayoutResult> {
	if (!root || !isLikelyFullPath(root)) {
		throw new Error(`Invalid workspace root (not full path): ${root}`);
	}

	const app = getGoApp();
	if (isWails() && app && typeof (app as any).InitSpecsLayout === 'function') {
		try {
			return (await (app as any).InitSpecsLayout(root)) as SpecsInitLayoutResult;
		} catch (e) {
			console.error('[wails-filesystem] InitSpecsLayout Wails call failed:', e);
			throw e;
		}
	}
	if (isWails()) {
		throw new Error('Wails binding missing: App.InitSpecsLayout');
	}

	const r = await fetch('/api/workspace/specs/init-layout', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ workspace_root: root }),
	});
	let j: SpecsInitLayoutResult;
	try {
		j = (await r.json()) as SpecsInitLayoutResult;
	} catch {
		return { ok: false, error: await r.text() };
	}
	if (!r.ok && !j?.error) {
		return { ok: false, error: `HTTP ${r.status}` };
	}
	return j;
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

// ── VerifySpecs ─────────────────────────────────────────────────

export type VerifySpecsOptions = {
	format?: 'summary' | 'json' | 'short';
	checks?: string;       // comma-separated: file_existence,parent_chain,completeness,behaviors
	levels?: string;       // comma-separated: 4_feature,5_slice
	show_pass?: string;    // 'true' | 'false'
};

/**
 * Run spec → code alignment verification via the Go verify package.
 * Returns human-readable text (summary/short) or raw JSON.
 */
export async function wailsVerifySpecs(
	workspace: string,
	opts?: VerifySpecsOptions
): Promise<string> {
	if (!isWails()) throw new Error('wailsVerifySpecs requires Wails mode');
	const app = getGoApp();
	if (!app || typeof (app as any).VerifySpecs !== 'function') {
		throw new Error('Wails binding missing: App.VerifySpecs');
	}
	const result = await (app as any).VerifySpecs(workspace, opts ?? {});
	return result as string;
}
