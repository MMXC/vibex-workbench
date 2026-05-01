<!-- ============================================================
Composer — 多模态输入框
E3-U2: 底部 RunStatusBar 显示运行状态
E4-U3: 拖拽 Artifact 到 Composer 注入 @artifactId
开发者维护，gen.py 永不覆盖
============================================================ -->

<script lang="ts">
  import { runStore, activeRun } from '$lib/stores/run-store';
  import {
    getAvailableSpecCommands,
    specAgentContextStore,
    type SpecCommand,
    type SpecContextItem,
  } from '$lib/stores/spec-agent-context-store';

  interface Props {
    onsubmit?: (content: string, mode: string) => Promise<void> | void;
  }

  let { onsubmit }: Props = $props();

  let content = $state('');
  let mode = $state<'text'|'image'|'file'|'url'>('text');
  let submitting = $state(false);

  // E3-U2: RunStatusBar reactive state
  let activeRunData = $state<{ id: string; status: string; goal: string; result_summary?: string; error_message?: string } | null>(null);
  let toolCount = $state(0);
  let hideStatusBarTimer: ReturnType<typeof setTimeout> | null = null;
  let showStatus = $state(false);
  let statusMessage = $state('');
  let statusType = $state<'running'|'completed'|'failed'>('running');

  // E4-U3: Composer drag-over + drop 支持
  let dragOver = $state(false);
  let textareaEl = $state<HTMLTextAreaElement | undefined>(undefined);
  let contextState = $state<{ items: SpecContextItem[]; expanded: boolean; focusedPath: string | null }>({
    items: [],
    expanded: false,
    focusedPath: null,
  });
  let showCommandPalette = $state(false);
  let commandQuery = $state('');

  const focusedSpec = $derived.by(() => {
    return contextState.items.find(item => item.path === contextState.focusedPath) ?? contextState.items[0] ?? null;
  });

  const commandOptions = $derived.by(() => {
    const commands = getAvailableSpecCommands(focusedSpec?.level ?? null);
    if (!commandQuery) return commands;
    return commands.filter(command => command.name.startsWith(commandQuery));
  });

  function handleDragOver(e: DragEvent) {
    if (e.dataTransfer?.types.includes('text/vibex-artifact')) {
      e.preventDefault();
      e.dataTransfer!.dropEffect = 'copy';
      dragOver = true;
    }
  }

  function handleDragLeave() {
    dragOver = false;
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragOver = false;
    const artifactId = e.dataTransfer?.getData('text/vibex-artifact');
    if (artifactId) {
      // E4-U3: 注入 @artifactId 引用
      content = content ? `${content.trimEnd()} @{${artifactId}}` : `@{${artifactId}}`;
    }
  }

  $effect(() => {
    const unsub = runStore.subscribe(s => {
      const ar = s.runs.find(r => r.id === s.active_run_id);
      activeRunData = ar ?? null;
      toolCount = s.toolInvocations.length;
    });
    return unsub;
  });

  $effect(() => {
    const unsub = specAgentContextStore.subscribe(s => {
      contextState = s;
    });
    return unsub;
  });

  // 监听 activeRun 状态变化，显示状态条
  $effect(() => {
    const run = activeRunData;
    if (!run) {
      showStatus = false;
      return;
    }
    if (run.status === 'executing' || run.status === 'planning') {
      statusMessage = `运行中… (${toolCount} tools)`;
      statusType = 'running';
      showStatus = true;
    } else if (run.status === 'completed') {
      statusMessage = run.result_summary ?? '✅ 运行完成';
      statusType = 'completed';
      showStatus = true;
      if (hideStatusBarTimer) clearTimeout(hideStatusBarTimer);
      hideStatusBarTimer = setTimeout(() => { showStatus = false; }, 5000);
    } else if (run.status === 'failed') {
      statusMessage = `❌ 失败: ${run.error_message ?? '未知错误'}`;
      statusType = 'failed';
      showStatus = true;
    }
  });

  async function submit() {
    if (!content.trim() || submitting) return;
    submitting = true;
    showCommandPalette = false;
    try {
      await onsubmit?.(content, mode);
      content = '';
    } finally {
      submitting = false;
    }
  }

  function syncCommandPalette() {
    const trimmed = content.trimStart();
    if (!trimmed.startsWith('/')) {
      showCommandPalette = false;
      commandQuery = '';
      return;
    }
    commandQuery = trimmed.split(/\s+/)[0].toLowerCase();
    showCommandPalette = true;
  }

  function selectCommand(command: SpecCommand) {
    content = command.sample;
    showCommandPalette = false;
    commandQuery = command.name;
    queueMicrotask(() => textareaEl?.focus());
  }

  function removeContext(path: string) {
    specAgentContextStore.removeSpec(path);
  }
</script>

<div
  class="composer"
  role="region"
  aria-label="Composer"
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
  class:drag-over={dragOver}
>
  <!-- E3-U2: RunStatusBar -->
  {#if showStatus}
    <div class="run-status-bar" class:running={statusType==='running'} class:completed={statusType==='completed'} class:failed={statusType==='failed'}>
      <span class="status-icon">
        {#if statusType === 'running'}⟳{/if}
        {#if statusType === 'completed'}✓{/if}
        {#if statusType === 'failed'}✗{/if}
      </span>
      <span class="status-msg">{statusMessage}</span>
    </div>
  {/if}

  <div class="context-shell">
    <button type="button" class="context-toggle" onclick={() => specAgentContextStore.toggleExpanded()}>
      <span>Context</span>
      <strong>{contextState.items.length} spec{contextState.items.length === 1 ? '' : 's'} {contextState.expanded ? '▴' : '▾'}</strong>
    </button>
    {#if contextState.expanded}
      <div class="context-list">
        {#if contextState.items.length === 0}
          <div class="context-empty">点击中央 spec 卡片可添加上下文。</div>
        {:else}
          {#each contextState.items as item (item.path)}
            <div class="context-chip" class:focused={item.path === contextState.focusedPath}>
              <div>
                <strong>{item.display.title}</strong>
                <span>{item.level} · {item.name}</span>
              </div>
              <button type="button" title="移除上下文" onclick={() => removeContext(item.path)}>×</button>
            </div>
          {/each}
        {/if}
      </div>
    {/if}
  </div>

  <div class="mode-tabs">
    <button class:active={mode==='text'} onclick={() => mode='text'}>文本</button>
    <button class:active={mode==='image'} onclick={() => mode='image'}>图片</button>
    <button class:active={mode==='file'} onclick={() => mode='file'}>文件</button>
    <button class:active={mode==='url'} onclick={() => mode='url'}>URL</button>
  </div>
  <div class="input-wrap">
    {#if showCommandPalette && commandOptions.length > 0}
      <div class="command-palette">
        {#each commandOptions as command (command.name)}
          <button type="button" class="command-option" onclick={() => selectCommand(command)}>
            <strong>{command.name}</strong>
            <span>{command.description} · {command.sample}</span>
          </button>
        {/each}
      </div>
    {/if}
    <textarea
      bind:this={textareaEl}
      bind:value={content}
      placeholder='输入消息，或输入 / 选择 spec 指令...'
      rows={3}
      oninput={syncCommandPalette}
      onfocus={syncCommandPalette}
      onkeydown={(e) => {
        if (e.key === 'Enter' && e.ctrlKey) submit();
        if (e.key === 'Escape') showCommandPalette = false;
      }}
    ></textarea>
  </div>
  <div class="actions">
    <span class="hint">Ctrl+Enter 发送 · / 打开命令 · {toolCount} tools</span>
    <button class="submit-btn" onclick={submit} disabled={submitting}>
      {submitting ? '发送中…' : '发送 ⌘↵'}
    </button>
  </div>
</div>

<style>
  .composer {
    padding: 12px;
    background: #151820;
    border-top: 1px solid #303746;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  textarea { background: #0b0d12; border: 1px solid #303746; border-radius: 12px; color: #eef0f5; padding: 10px; resize: none; box-sizing: border-box; line-height: 1.45; outline: none; }
  textarea:focus { border-color: #7aa2ff; box-shadow: 0 0 0 1px rgba(122,162,255,.18); }
  .mode-tabs { display: flex; gap: 4px; }
  .mode-tabs button { background: transparent; border: 1px solid transparent; color: #a3abb9; cursor: pointer; padding: 4px 9px; border-radius: 999px; font-size: 12px; }
  .mode-tabs button:hover { border-color: #465064; color: #eef0f5; }
  .mode-tabs button.active { background: #7aa2ff; border-color: #7aa2ff; color: #08111d; font-weight: 700; }
  .context-shell { border: 1px solid #303746; border-radius: 14px; background: rgba(28,32,42,.72); overflow: hidden; }
  .context-toggle { width: 100%; display: flex; justify-content: space-between; align-items: center; border: none; background: transparent; color: #eef0f5; padding: 9px 11px; cursor: pointer; font-size: 12px; }
  .context-toggle span { color: #eef0f5; font-weight: 800; }
  .context-toggle strong { color: #a3abb9; font: 700 11px/1 'Cascadia Code', ui-monospace, monospace; }
  .context-list { display: grid; gap: 6px; padding: 0 6px 6px; max-height: 96px; overflow: auto; }
  .context-chip { display: grid; grid-template-columns: 1fr auto; gap: 8px; align-items: center; padding: 7px 9px; border: 1px solid #465064; border-radius: 11px; background: #10131a; }
  .context-chip.focused { border-color: #7aa2ff; background: rgba(122,162,255,.13); }
  .context-chip strong { display: block; color: #eef0f5; font-size: 12px; margin-bottom: 2px; }
  .context-chip span { display: block; color: #a3abb9; font-size: 10.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .context-chip button { border: 1px solid #303746; background: transparent; color: #6f7888; cursor: pointer; width: 22px; height: 22px; border-radius: 999px; }
  .context-chip button:hover { background: #242936; color: #eef0f5; }
  .context-empty { color: #6f7888; font-size: 11px; padding: 6px 8px; }
  .input-wrap { position: relative; display: flex; flex-direction: column; }
  .command-palette { position: absolute; left: 0; right: 0; bottom: calc(100% + 8px); z-index: 10; border: 1px solid #465064; border-radius: 14px; background: rgba(16,19,26,.98); box-shadow: 0 18px 60px rgba(0,0,0,.42); overflow: hidden; }
  .command-option { display: block; width: 100%; border: none; border-bottom: 1px solid #303746; background: transparent; color: #eef0f5; text-align: left; padding: 10px 12px; cursor: pointer; }
  .command-option:last-child { border-bottom: none; }
  .command-option:hover { background: rgba(122,162,255,.14); }
  .command-option strong { display: block; color: #72d6d0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; margin-bottom: 3px; }
  .command-option span { display: block; color: #a3abb9; font-size: 11px; line-height: 1.35; }
  .actions { display: flex; justify-content: space-between; align-items: center; }
  .hint { color: #6f7888; font-size: 11px; }
  .submit-btn { background: #72d6d0; color: #071513; border: none; padding: 7px 16px; border-radius: 999px; cursor: pointer; font-size: 13px; font-weight: 800; }
  .submit-btn:disabled { opacity: 0.6; cursor: not-allowed; }

  /* E4-U3: drag-over highlight */
  .composer.drag-over { border-color: #7aa2ff; background: #1e2433; }

  /* E3-U2: RunStatusBar styles */
  .run-status-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 12px;
    animation: slideIn 0.2s ease-out;
  }
  @keyframes slideIn {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .run-status-bar.running { background: rgba(122,162,255,.12); color: #9fc0ff; border: 1px solid rgba(122,162,255,.4); }
  .run-status-bar.completed { background: rgba(135,207,138,.12); color: #87cf8a; border: 1px solid rgba(135,207,138,.4); }
  .run-status-bar.failed { background: rgba(225,109,117,.12); color: #e16d75; border: 1px solid rgba(225,109,117,.4); }
  .status-icon { font-size: 14px; }
  .status-msg { flex: 1; }
  .run-status-bar.running .status-icon { animation: spin 1s linear infinite; }
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>