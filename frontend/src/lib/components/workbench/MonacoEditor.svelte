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
				{ token: 'key', foreground: 'c4b5fd' },
				{ token: 'string', foreground: '6ee7b7' },
				{ token: 'number', foreground: 'fcd34d' },
				{ token: 'comment', foreground: '555558', fontStyle: 'italic' },
				{ token: 'type', foreground: '67e8f9' },
				{ token: 'keyword', foreground: 'c4b5fd' },
			],
			colors: {
				'editor.background': '#0d0d0e',
				'editor.foreground': '#e8e8ed',
				'editor.lineHighlightBackground': '#131314',
				'editorLineNumber.foreground': '#555558',
				'editorLineNumber.activeForeground': '#8a8a8e',
				'editor.selectionBackground': '#5856d640',
				'editor.inactiveSelectionBackground': '#5856d620',
				'editorCursor.foreground': '#5856d6',
				'editorIndentGuide.background': '#1a1a1c',
				'editorIndentGuide.activeBackground': '#5856d640',
				'scrollbarSlider.background': '#5856d630',
				'scrollbarSlider.hoverBackground': '#5856d650',
				'scrollbarSlider.activeBackground': '#5856d680',
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
