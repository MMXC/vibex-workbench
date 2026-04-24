/**
 * Wails Runtime TypeScript wrapper
 * Provides typed access to window.runtime APIs and event system.
 */

/** Get the Wails runtime object, throwing if unavailable. */
function getRuntime() {
	const rt = (window as any).runtime;
	if (!rt) throw new Error('runtime not available');
	return rt;
}

/**
 * Opens a native directory picker dialog.
 * @returns the selected directory path, or empty string if cancelled.
 */
export async function openDirectoryDialog(): Promise<string> {
	const rt = getRuntime();
	const result = await rt.OpenDirectoryDialog();
	return result ?? '';
}

/** Minimizes the application window. */
export async function windowMinimize(): Promise<void> {
	const rt = getRuntime();
	await rt.WindowMinimise();
}

/** Toggles maximize/restore on the application window. */
export async function windowToggleMaximize(): Promise<void> {
	const rt = getRuntime();
	await rt.WindowToggleMaximise();
}

/** Quits the application. */
export async function windowQuit(): Promise<void> {
	const rt = getRuntime();
	await rt.Quit();
}

/**
 * Subscribe to a Wails event.
 * @param event  the event name
 * @param callback called with event payload args
 */
export function eventsOn(event: string, callback: (...args: any[]) => void): void {
	const rt = getRuntime();
	rt.EventsOn(event, callback);
}

/**
 * Emit a Wails event with optional data.
 * @param event the event name
 * @param data  payload(s) forwarded to listeners
 */
export function eventsEmit(event: string, ...data: any[]): void {
	const rt = getRuntime();
	rt.EventsEmit(event, ...data);
}
