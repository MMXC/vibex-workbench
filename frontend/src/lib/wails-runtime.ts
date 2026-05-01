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
