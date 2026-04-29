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
 *  Falls back to a hidden <input type="file" webkitdirectory> in non-Wails browsers.
 */
export async function openDirectoryDialog(): Promise<string> {
	const rt = getRuntime();
	if (rt) {
		const result = await rt.OpenDirectoryDialog();
		return result ?? '';
	}
	// Browser fallback: create a hidden input[type=file] with webkitdirectory
	return new Promise<string>((resolve) => {
		const input = document.createElement('input');
		input.type = 'file';
		input.webkitdirectory = '';
		input.style.cssText = 'position:fixed;top:-9999px;left:-9999px;opacity:0;width:1px;height:1px;';
		input.accept = ''; // all files
		input.addEventListener('change', () => {
			// webkitdirectory gives the path of the selected directory
			const path = input.files?.[0]?.webkitRelativePath ?? '';
			// path looks like "dir/subdir/file.txt" — extract the directory
			const dir = path.split('/').slice(0, -1).join('/');
			resolve(dir || '');
			input.remove();
		});
		document.body.appendChild(input);
		input.click();
		// If user cancels, resolve empty after a tick
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
