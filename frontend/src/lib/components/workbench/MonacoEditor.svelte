<!-- Monaco Editor wrapper — replaces textarea in edit mode -->
<script lang="ts">
	import { onMount } from 'svelte';
	import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';

	interface Props {
		value?: string;
		language?: string;
		readOnly?: boolean;
		onchange?: (val: string) => void;
	}

	let { value = $bindable(''), language = 'yaml', readOnly = false, onchange }: Props = $props();

	let container: HTMLDivElement;
	let editor: any = null;

	// Expose setValue so parent can force-update after save
	export function setValue(v: string) {
		if (editor) editor.setValue(v);
	}

	onMount(async () => {
		const monaco = await import('monaco-editor');

		// Configure Monaco web worker (all language features run in editor worker)
		(window as any).MonacoEnvironment = {
			getWorker() {
				return new editorWorker();
			},
		};

		// Define dark theme matching vibex-workbench palette
		monaco.editor.defineTheme('vibex-dark', {
			base: 'vs-dark',
			inherit: true,
			rules: [
				{ token: 'key', foreground: '7aa2ff' },
				{ token: 'string', foreground: '87cf8a' },
				{ token: 'number', foreground: 'efc66b' },
				{ token: 'comment', foreground: '6f7888', fontStyle: 'italic' },
				{ token: 'type', foreground: '72d6d0' },
				{ token: 'keyword', foreground: '7aa2ff' },
			],
			colors: {
				'editor.background': '#0b0c10',
				'editor.foreground': '#eef0f5',
				'editor.lineHighlightBackground': '#151820',
				'editorLineNumber.foreground': '#6f7888',
				'editorLineNumber.activeForeground': '#a3abb9',
				'editor.selectionBackground': '#7aa2ff40',
				'editor.inactiveSelectionBackground': '#7aa2ff20',
				'editorCursor.foreground': '#72d6d0',
				'editorIndentGuide.background': '#1c202a',
				'editorIndentGuide.activeBackground': '#7aa2ff40',
				'scrollbarSlider.background': '#6f788866',
				'scrollbarSlider.hoverBackground': '#a3abb966',
				'scrollbarSlider.activeBackground': '#a3abb988',
			},
		});

		editor = monaco.editor.create(container, {
			value,
			language,
			theme: 'vibex-dark',
			readOnly,
			automaticLayout: true,
			minimap: { enabled: false },
			scrollBeyondLastLine: false,
			fontSize: 12,
			fontFamily: "'JetBrains Mono', 'Fira Code', ui-monospace, monospace",
			lineHeight: 21,
			tabSize: 2,
			insertSpaces: true,
			wordWrap: 'on',
			renderLineHighlight: 'line',
			scrollbar: {
				vertical: 'auto',
				horizontal: 'auto',
				verticalScrollbarSize: 8,
				horizontalScrollbarSize: 8,
			},
			padding: { top: 12, bottom: 12 },
			suggest: { showWords: false },
			quickSuggestions: false,
			parameterHints: { enabled: false },
			hover: { enabled: false },
			overviewRulerLanes: 0,
			hideCursorInOverviewRuler: true,
			overviewRulerBorder: false,
			renderWhitespace: 'none',
			guides: { indentation: true },
			folding: true,
			lineNumbersMinChars: 3,
		});

		editor.onDidChangeModelContent(() => {
			const v = editor.getValue();
			value = v;
			onchange?.(v);
		});

		return () => {
			editor?.dispose();
		};
	});
</script>

<div bind:this={container} class="monaco-wrap"></div>

<style>
	.monaco-wrap {
		flex: 1;
		min-height: 0;
		overflow: hidden;
	}
</style>
