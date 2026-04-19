<script lang="ts">
  import { onMount } from 'svelte';
  import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view';
  import { EditorState, Compartment } from '@codemirror/state';
  import { yaml } from '@codemirror/lang-yaml';
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
  import { syntaxHighlighting, defaultHighlightStyle, foldGutter, indentOnInput } from '@codemirror/language';
  import { oneDark } from '@codemirror/theme-one-dark';

  interface Props {
    value?: string;
    readonly?: boolean;
    onsave?: (content: string) => void;
    onchange?: (content: string) => void;
    placeholder?: string;
  }

  let {
    value = $bindable(''),
    readonly = false,
    onsave,
    onchange,
    placeholder = '# 编辑 YAML 规格文件...',
  }: Props = $props();

  let editorEl: HTMLDivElement | undefined = $state();
  let view: EditorView | undefined = $state();
  let readonlyCompartment = new Compartment();
  let isValid = $state(true);
  let errorMsg = $state('');

  // ── YAML validation ──────────────────────────────────────
  function validateYaml(content: string): { valid: boolean; error: string } {
    try {
      // Minimal YAML validation using quick parse
      const lines = content.split('\n');
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        // Check for tabs (YAML forbids tabs for indentation)
        if (line.match(/^\t/)) {
          return { valid: false, error: `第 ${i + 1} 行: 禁止使用 Tab 缩进，请使用空格` };
        }
        // Check for inconsistent indentation
        const indent = line.match(/^(\s*)/)?.[1].length ?? 0;
        if (indent % 2 !== 0 && line.trim().length > 0 && !line.trim().startsWith('#')) {
          return { valid: false, error: `第 ${i + 1} 行: 缩进不是偶数空格` };
        }
      }
      return { valid: true, error: '' };
    } catch (e) {
      return { valid: false, error: `YAML 解析错误: ${e}` };
    }
  }

  // ── Create editor ─────────────────────────────────────────
  onMount(() => {
    if (!editorEl) return;

    const updateListener = EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        const newValue = update.state.doc.toString();
        value = newValue;
        const result = validateYaml(newValue);
        isValid = result.valid;
        errorMsg = result.error;
        onchange?.(newValue);
      }
    });

    const saveKeymap = keymap.of([{
      key: 'Mod-s',
      run: () => {
        if (view && onsave) {
          onsave(view.state.doc.toString());
          return true;
        }
        return false;
      },
    }]);

    const state = EditorState.create({
      doc: value,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        highlightActiveLineGutter(),
        history(),
        foldGutter(),
        indentOnInput(),
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        yaml(),
        oneDark,
        updateListener,
        saveKeymap,
        keymap.of([...defaultKeymap, ...historyKeymap]),
        readonlyCompartment.of(EditorState.readOnly.of(readonly)),
        EditorView.lineWrapping,
        EditorView.theme({
          '&': { height: '100%', fontSize: '13px' },
          '.cm-scroller': { fontFamily: "'JetBrains Mono', 'Fira Code', monospace", overflow: 'auto' },
          '.cm-content': { padding: '8px 0' },
          '.cm-gutters': { background: '#1e1e2e', borderRight: '1px solid #333' },
          '.cm-activeLineGutter': { background: '#2d2d4a' },
          '.cm-activeLine': { background: '#1a1a2e' },
          '&.cm-focused .cm-cursor': { borderLeftColor: '#4D96FF' },
          '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': { background: '#2d4a7a' },
        }),
      ],
    });

    view = new EditorView({ state, parent: editorEl });

    return () => {
      view?.destroy();
      view = undefined;
    };
  });

  // ── React to readonly changes ───────────────────────────
  $effect(() => {
    if (view) {
      view.dispatch({
        effects: readonlyCompartment.reconfigure(EditorState.readOnly.of(readonly)),
      });
    }
  });

  // ── React to external value changes ─────────────────────
  let lastExternalValue = value;
  $effect(() => {
    if (view && value !== lastExternalValue && value !== view.state.doc.toString()) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: value },
      });
      lastExternalValue = value;
    }
  });

  function handleSave() {
    if (view && onsave) {
      onsave(view.state.doc.toString());
    }
  }
</script>

<div class="editor-wrapper" class:invalid={!isValid}>
  <div class="toolbar">
    <span class="badge" class:valid={isValid} class:error={!isValid}>
      {isValid ? '✓ YAML 有效' : '⚠ YAML 有误'}
    </span>
    <div class="toolbar-actions">
      <button onclick={handleSave} disabled={!onsave || !isValid} title="Ctrl+S 保存">
        💾 保存
      </button>
      <span class="shortcut">⌘S</span>
    </div>
  </div>

  {#if errorMsg}
    <div class="error-bar">{errorMsg}</div>
  {/if}

  <div class="editor-container" bind:this={editorEl}>
    {#if !value}
      <div class="placeholder-hint">{placeholder}</div>
    {/if}
  </div>
</div>

<style>
  .editor-wrapper {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #1e1e2e;
    border-radius: 4px;
    overflow: hidden;
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 12px;
    background: #1a1a2e;
    border-bottom: 1px solid #333;
    gap: 8px;
  }

  .badge {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 4px;
    font-weight: 500;
  }
  .badge.valid { color: #6BCB77; background: rgba(107, 203, 119, 0.1); }
  .badge.error { color: #FF6B6B; background: rgba(255, 107, 107, 0.1); }

  .toolbar-actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .toolbar-actions button {
    background: #4D96FF;
    border: none;
    color: white;
    padding: 4px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
  }
  .toolbar-actions button:hover { background: #3a7acc; }
  .toolbar-actions button:disabled { background: #333; color: #666; cursor: not-allowed; }

  .shortcut {
    font-size: 11px;
    color: #666;
    background: #252540;
    padding: 2px 6px;
    border-radius: 3px;
  }

  .error-bar {
    padding: 6px 12px;
    background: rgba(255, 107, 107, 0.1);
    color: #FF6B6B;
    font-size: 12px;
    border-bottom: 1px solid rgba(255, 107, 107, 0.2);
  }

  .editor-container {
    flex: 1;
    overflow: hidden;
    position: relative;
  }

  .editor-container :global(.cm-editor) {
    height: 100%;
  }

  .editor-container :global(.cm-scroller) {
    overflow: auto;
  }

  .placeholder-hint {
    position: absolute;
    top: 12px;
    left: 52px;
    color: #444;
    font-size: 13px;
    font-family: monospace;
    pointer-events: none;
  }
</style>
