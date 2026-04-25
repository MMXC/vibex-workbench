/**
 * Reactive store for run-make output text displayed in the R2Dock output panel.
 */
import { writable } from 'svelte/store';

export const outputText = writable('');
export const outputVisible = writable(false);

export function appendOutput(text: string) {
	outputText.update(v => v + text + '\n');
	outputVisible.set(true);
}

export function clearOutput() {
	outputText.set('');
	outputVisible.set(false);
}
