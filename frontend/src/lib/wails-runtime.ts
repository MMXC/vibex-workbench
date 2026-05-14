/**
 * Wails Runtime TypeScript wrapper
 * Provides typed access to window.runtime APIs and event system.
 * Safe for non-Wails environments (browser dev server, static builds):
 * all functions degrade gracefully instead of throwing.
 */

function getRuntime(): any | null {
	return (window as any).runtime ?? null;
}

const NATIVE_WINDOW_ACTION_PATH = '/__vibex/native/window-action';
const NATIVE_WINDOW_STATE_PATH = '/__vibex/native/window-state';

type NativeWindowState = {
	size?: { w: number; h: number };
	position?: { x: number; y: number };
	maximized?: boolean;
};

function isWailsHost(): boolean {
	return window.location.hostname === 'wails.localhost' || window.location.port === '34115';
}

function nativeWindowCandidates(path: string): string[] {
	if (isWailsHost()) return [path];
	const candidates = [path];
	const current = window.location.origin;
	for (const origin of ['http://wails.localhost:34115', 'http://localhost:34115']) {
		if (origin !== current) candidates.push(`${origin}${path}`);
	}
	return candidates;
}

async function fetchNativeWindowState(): Promise<NativeWindowState | null> {
	for (const url of nativeWindowCandidates(NATIVE_WINDOW_STATE_PATH)) {
		try {
			const res = await fetch(url);
			if (!res.ok) continue;
			return (await res.json()) as NativeWindowState;
		} catch {
			// Try the next native host candidate.
		}
	}
	return null;
}

async function postNativeWindowAction(action: string, payload: Record<string, unknown> = {}): Promise<void> {
	for (const url of nativeWindowCandidates(NATIVE_WINDOW_ACTION_PATH)) {
		try {
			const res = await fetch(url, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ action, ...payload }),
			});
			if (res.ok) return;
		} catch {
			// Try the next native host candidate.
		}
	}
}

/** Returns true when running inside a Wails WebView (runtime available). */
export function isWails(): boolean {
	return getRuntime() !== null;
}

export function hasNativeWindowHost(): boolean {
	return isWails() || isWailsHost();
}

/**
 * 在系统浏览器（而非 WebView 内部）中打开 URL。
 * - Wails 环境：通过 runtime.BrowserOpenURL 调系统默认浏览器。
 * - 浏览器开发环境：直接 window.open（Chrome DevTools 可用扩展）。
 */
export async function browserOpenUrl(url: string): Promise<void> {
	if (!url) return;
	if (isWails()) {
		try {
			await (window as any).runtime?.BrowserOpenURL?.(url);
			return;
		} catch {
			// fall through to window.open
		}
	}
	// 浏览器环境：window.open 产生的标签在 DevTools 可控，Chrome 扩展可用
	window.open(url, '_blank', 'noopener,noreferrer');
}

/** Browser-dev directory picker fallback.
 *  Wails desktop must use window.go.main.App.OpenDirectoryDialog via wails-dialogs.ts.
 *  This wrapper intentionally does not use <input webkitdirectory>: it returns only
 *  relative paths and opens a misleading file picker on Windows WebView2.
 */
export async function openDirectoryDialog(): Promise<string> {
	if ('showDirectoryPicker' in window) {
		try {
			const dirHandle: any = await (window as any).showDirectoryPicker();
			const path: string | undefined = dirHandle.path;
			if (path) return path;
			console.warn('[openDirectoryDialog] showDirectoryPicker returned no path; browser fallback unavailable');
			return '';
		} catch (e: any) {
			if (e?.name === 'AbortError' || e?.message?.includes('cancelled')) {
				return '';
			}
			console.warn('[openDirectoryDialog] showDirectoryPicker failed:', e);
			return '';
		}
	}
	console.warn('[openDirectoryDialog] no browser directory picker with full path is available');
	return '';
}

/** Minimizes the application window. */
export async function windowMinimize(): Promise<void> {
	const rt = getRuntime();
	if (!rt) {
		await postNativeWindowAction('minimize');
		return;
	}
	await rt.WindowMinimise();
}

/** Toggles maximize/restore on the application window. */
export async function windowToggleMaximize(): Promise<void> {
	const rt = getRuntime();
	if (!rt) {
		await postNativeWindowAction('toggle-maximize');
		return;
	}
	await rt.WindowToggleMaximise();
}

export async function windowIsMaximized(): Promise<boolean> {
	const rt = getRuntime();
	if (!rt || typeof rt.WindowIsMaximised !== 'function') {
		const state = await fetchNativeWindowState();
		return !!state?.maximized;
	}
	return await rt.WindowIsMaximised();
}

export async function windowGetSize(): Promise<{ w: number; h: number } | null> {
	const rt = getRuntime();
	if (!rt || typeof rt.WindowGetSize !== 'function') {
		const state = await fetchNativeWindowState();
		return state?.size ?? null;
	}
	return await rt.WindowGetSize();
}

export async function windowSetSize(width: number, height: number): Promise<void> {
	const rt = getRuntime();
	if (!rt || typeof rt.WindowSetSize !== 'function') {
		await postNativeWindowAction('set-size', { width, height });
		return;
	}
	await rt.WindowSetSize(width, height);
}

export async function windowGetPosition(): Promise<{ x: number; y: number } | null> {
	const rt = getRuntime();
	if (!rt || typeof rt.WindowGetPosition !== 'function') {
		const state = await fetchNativeWindowState();
		return state?.position ?? null;
	}
	return await rt.WindowGetPosition();
}

export async function windowSetPosition(x: number, y: number): Promise<void> {
	const rt = getRuntime();
	if (!rt || typeof rt.WindowSetPosition !== 'function') {
		await postNativeWindowAction('set-position', { x, y });
		return;
	}
	await rt.WindowSetPosition(x, y);
}

export async function windowResizeTo(x: number, y: number, width: number, height: number): Promise<void> {
	const rt = getRuntime();
	if (!rt || typeof rt.WindowSetPosition !== 'function' || typeof rt.WindowSetSize !== 'function') {
		await postNativeWindowAction('resize', { x, y, width, height });
		return;
	}
	await rt.WindowSetPosition(x, y);
	await rt.WindowSetSize(width, height);
}

/** Quits the application. */
export async function windowQuit(): Promise<void> {
	const rt = getRuntime();
	if (!rt) {
		await postNativeWindowAction('quit');
		return;
	}
	await rt.Quit();
}

/**
 * Subscribe to a Wails event.
 * No-op when runtime is unavailable (e.g. browser dev server).
 * @param event  the event name
 * @param callback called with event payload args
 */
export function eventsOn(event: string, callback: (...args: any[]) => void): void {
	const rt = getRuntime();
	if (!rt) return;
	if (typeof rt.EventsOn !== 'function') return;
	rt.EventsOn(event, callback);
}

/**
 * Emit a Wails event with optional data.
 * No-op when runtime is unavailable.
 * @param event the event name
 * @param data  payload(s) forwarded to listeners
 */
export function eventsEmit(event: string, ...data: any[]): void {
	const rt = getRuntime();
	if (!rt) return;
	rt.EventsEmit(event, ...data);
}
