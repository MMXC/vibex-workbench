/**
 * Wails Runtime TypeScript wrapper
 * Provides typed access to window.runtime APIs and event system.
 * Safe for non-Wails environments (browser dev server, static builds):
 * all functions degrade gracefully instead of throwing.
 */

function getRuntime(): any | null {
	return (window as any).runtime ?? null;
}

/** Returns true when running inside a Wails WebView (runtime available). */
export function isWails(): boolean {
	return getRuntime() !== null;
}

/** Opens a native directory picker dialog.
 *  Priority (Wails WebView2 / Chromium):
 *  1. File System Access API showDirectoryPicker()  — 提供完整路径（WebView2 支持）
 *  2. Wails runtime.OpenDirectoryDialog              — 有时只返回文件夹名（Windows Wails 已知问题）
 *  3. <input type="file" webkitdirectory>            — 通用备选
 *
 *  Returns the selected directory full path, or '' if cancelled.
 */
export async function openDirectoryDialog(): Promise<string> {
	// ── 优先级 1：File System Access API（WebView2 / Chromium — 完整路径）────────
	if ('showDirectoryPicker' in window) {
		try {
			const dirHandle: any = await (window as any).showDirectoryPicker();
			const path: string | undefined = dirHandle.path;
			if (path) return path;
			// 有 handle.name 但无 path → 不可用，抛异常继续往下
			throw new Error('showDirectoryPicker has no path property');
		} catch (e: any) {
			if (e?.name === 'AbortError' || e?.message?.includes('cancelled')) {
				return ''; // 用户取消 → 不继续
			}
			// 其他错误（no path property）→ 继续往下走，不 return
		}
	}

	// ── 优先级 2：Wails runtime OpenDirectoryDialog（已切 sqweek/dialog，返回完整路径）─
	const rt = getRuntime();
	if (rt) {
		try {
			const result: string = await rt.OpenDirectoryDialog();
			if (result && (result.includes('/') || result.includes('\\'))) {
				return result; // 有路径分隔符 = 完整路径
			}
			if (result) {
				console.warn('[openDirectoryDialog] Wails returned folder name only:', result);
				// 只有文件夹名，没有完整路径 → 不接受，继续往下
			}
		} catch (e) {
			console.warn('[openDirectoryDialog] Wails OpenDirectoryDialog failed:', e);
		}
	}

	// ── 优先级 3：<input webkitdirectory> 备选（路径可能不完整）──────────────
	return new Promise<string>((resolve) => {
		const input = document.createElement('input');
		input.type = 'file';
		input.webkitdirectory = '';
		input.style.cssText =
			'position:fixed;top:-9999px;left:-9999px;opacity:0;width:1px;height:1px;';
		input.addEventListener('change', () => {
			const path = input.files?.[0]?.webkitRelativePath ?? '';
			const dir = path.split('/').slice(0, -1).join('/');
			resolve(dir || '');
			input.remove();
		});
		document.body.appendChild(input);
		input.click();
		setTimeout(() => {
			if (document.body.contains(input)) {
				resolve('');
				input.remove();
			}
		}, 5000);
	});
}

/** Minimizes the application window. */
export async function windowMinimize(): Promise<void> {
	const rt = getRuntime();
	if (!rt) return;
	await rt.WindowMinimise();
}

/** Toggles maximize/restore on the application window. */
export async function windowToggleMaximize(): Promise<void> {
	const rt = getRuntime();
	if (!rt) return;
	await rt.WindowToggleMaximise();
}

/** Quits the application. */
export async function windowQuit(): Promise<void> {
	const rt = getRuntime();
	if (!rt) return;
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
