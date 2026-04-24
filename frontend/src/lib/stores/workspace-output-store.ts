/**
 * Reactive store for run-make output text displayed in the R2Dock output panel.
 */

let outputText = $state('');
let outputVisible = $state(false);

export function getOutputText() {
	return outputText;
}

export function isOutputVisible() {
	return outputVisible;
}

export function appendOutput(text: string) {
	outputText += text + '\n';
	outputVisible = true;
}

export function clearOutput() {
	outputText = '';
	outputVisible = false;
}
