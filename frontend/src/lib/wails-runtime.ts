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
			// Chromium-based browsers expose 'path' on FileSystemDirectoryHandle
			const path: string | undefined = dirHandle.path;
			if (path) return path;
			// Fallback：handle.name 至少是文件夹名（不完美，但比空字符串好）
			const name = dirHandle.name as string;
			if (name) {
				console.warn('[openDirectoryDialog] showDirectoryPicker no path property, got name:', name);
				// 尝试用 name 拼一个绝对路径（最后手段）
				// 不要返回 name 本身（会导致 CWD 错误），抛异常走备选
				throw new Error('no path property on directory handle');
			}
		} catch (e: any) {
			if (e?.name === 'AbortError' || e?.message?.includes('cancelled') || e?.message?.includes('no path')) {
				// 用户取消或无 path 属性 → 继续往下走
			}
		}
	}

	// ── 优先级 2：Wails runtime OpenDirectoryDialog ──────────────────────────
	const rt = getRuntime();
	if (rt) {
		try {
			const result: string = await rt.OpenDirectoryDialog();
			if (result) {
				// Wails 返回的路径可能只有文件夹名（Windows 已知问题）。
				// 检测：如果返回值不含路径分隔符，尝试追加 CWD。
				if (result && !result.includes('/') && !result.includes('\\')) {
					// 只有文件夹名，尝试用当前 document URL 推断
					// WebView2 下 document.title 可能是完整路径的 hint
					const cwd = (document as any).currentScript?.src ?? '';
					console.warn('[openDirectoryDialog] Wails returned folder name only:', result, 'cwd hint:', cwd);
				}
				return result;
			}
		} catch (e) {
			console.warn('[openDirectoryDialog] Wails OpenDirectoryDialog failed:', e);
		}
	}

	// ── 优先级 3：<input webkitdirectory> 备选 ───────────────────────────────
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
