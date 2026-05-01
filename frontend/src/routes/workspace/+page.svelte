<!--
  VibeX Workspace Entry — MVP: 选定仓库根
  路由: /workspace
  入口: 用户指定一个目录 → 检测状态 → (可选)初始化脚手架 → 进入 Workbench
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { openDirectoryNativeFirst } from '$lib/wails-dialogs';

  let workspaceRoot = $state('');
  let state = $state<{state: string, signals: any[], suggestions: string[]} | null>(null);
  let loading = $state(false);
  let error = $state('');

  async function detectState() {
    if (!workspaceRoot) return;
    loading = true;
    error = '';
    try {
      const res = await fetch('/api/workspace/detect-state', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({workspace_root: workspaceRoot})
      });
      const data = await res.json();
      state = data;
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function scaffold() {
    if (!workspaceRoot) return;
    loading = true;
    error = '';
    try {
      const res = await fetch('/api/workspace/scaffold', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({workspace_root: workspaceRoot, confirm: true})
      });
      const data = await res.json();
      // scaffolder returns {ok, created, errors} or {ok: false, error, stderr}
      if (!data.ok) {
        error = data.error || data.errors?.join('\n') || 'scaffold 失败';
      } else {
        await detectState();
      }
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function runMake(target: string) {
    if (!workspaceRoot) return;
    loading = true;
    error = '';
    try {
      const res = await fetch('/api/workspace/run-make', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({target, workspace_root: workspaceRoot})
      });
      const data = await res.json();
      error = data.output || (data.ok ? 'OK' : 'FAILED');
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function browseDir() {
    try {
      const dir = await openDirectoryNativeFirst('workspace');
      if (!dir) return;
      workspaceRoot = dir;
      state = null;
      await detectState();
    } catch (e: any) {
      error = e.message;
    }
  }

  function enterWorkbench() {
    if (!workspaceRoot) return;
    localStorage.setItem('vibex-workspace-root', workspaceRoot);
    goto('/workbench');
  }

  // state=partial → partial/ready 时可进入 workbench
  let ready = $derived(state?.state === 'ready' || state?.state === 'partial');

  const stateIcons: Record<string, string> = {
    empty: '⬜', partial: '🟨', ready: '🟩', error: '❌'
  };
</script>

<svelte:head>
  <title>VibeX — 选择工作区</title>
</svelte:head>

<main class="entry-page">
  <div class="brand-strip">
    <img src="/vibex-logo.svg" alt="VibeX" width="32" height="32" />
    <span class="brand-name">VibeX Workbench</span>
    <span class="tagline">空仓库 → 边做边长</span>
  </div>

  <div class="selector-wrap">
    <h1>选择工作区</h1>
    <p class="sub">指定一个本地目录，VibeX 将为你探测项目状态、初始化脚手架、读写规格、生成代码。</p>

    <div class="workspace-input-row">
      <input
        type="text"
        bind:value={workspaceRoot}
        placeholder="/path/to/your/project"
        class="ws-input"
        onkeydown={(e) => e.key === 'Enter' && detectState()}
      />
      <button onclick={detectState} disabled={loading || !workspaceRoot} class="btn-detect">
        {loading ? '检测中…' : '🔍 检测'}
      </button>
      <button onclick={browseDir} class="btn-browse" title="选择目录">
        📁 浏览…
      </button>
    </div>

    {#if error}
      <div class="error-box">{error}</div>
    {/if}

    {#if state}
      <div class="state-card">
        <div class="state-badge {state.state}">
          {stateIcons[state.state] || '❓'} {state.state.toUpperCase()}
        </div>
        <div class="signals">
          {#each state.signals as sig}
            <div class="sig-row">
              <span class="check">{sig.exists ? '✅' : '❌'}</span>
              <span class="path">{sig.path}</span>
              <span class="reason">{sig.reason}</span>
            </div>
          {/each}
        </div>
        {#if state.suggestions?.length}
          <div class="sug-list">
            {#each state.suggestions as s}
              <div class="sug">→ {s}</div>
            {/each}
          </div>
        {/if}
      </div>

      {#if state.state === 'empty'}
        <div class="action-row">
          <button onclick={scaffold} disabled={loading} class="btn-primary">
            📦 初始化脚手架
          </button>
        </div>
      {/if}

      {#if ready}
        <div class="action-row">
          <button onclick={() => runMake('validate')} disabled={loading} class="btn-secondary">
            ✅ 校验 Spec
          </button>
          <button onclick={() => runMake('generate')} disabled={loading} class="btn-secondary">
            ⚙️ 生成代码
          </button>
          <button onclick={enterWorkbench} disabled={loading} class="btn-enter">
            → 进入 Workbench
          </button>
        </div>
      {/if}
    {:else if !loading}
      <p class="hint">输入路径后点击「检测」查看项目状态</p>
    {/if}
  </div>
</main>

<style>
  :global(body) {
    margin: 0;
    background: #0d0d0e;
    color: #e8e8ed;
    font-family: 'Inter', system-ui, sans-serif;
  }

  .entry-page {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 48px 16px 32px;
    box-sizing: border-box;
  }

  .brand-strip {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 48px;
  }
  .brand-name {
    font-size: 18px;
    font-weight: 600;
    color: #e8e8ed;
  }
  .tagline {
    font-size: 12px;
    color: #555558;
    border-left: 1px solid #333;
    padding-left: 10px;
    margin-left: 2px;
  }

  .selector-wrap {
    width: 100%;
    max-width: 600px;
    background: #13131a;
    border: 1px solid #2a2a3a;
    border-radius: 12px;
    padding: 28px;
    box-sizing: border-box;
  }

  h1 {
    font-size: 20px;
    font-weight: 600;
    color: #cdd6f4;
    margin: 0 0 6px;
  }
  .sub {
    font-size: 13px;
    color: #6c7086;
    margin: 0 0 20px;
    line-height: 1.5;
  }

  .workspace-input-row {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
  }
  .ws-input {
    flex: 1;
    padding: 9px 12px;
    background: #11111b;
    border: 1px solid #313244;
    border-radius: 6px;
    color: #cdd6f4;
    font-family: monospace;
    font-size: 13px;
    outline: none;
  }
  .ws-input:focus {
    border-color: #89b4fa;
  }
  .btn-detect {
    padding: 9px 16px;
    background: #1e1e2e;
    border: 1px solid #45475a;
    border-radius: 6px;
    color: #cdd6f4;
    cursor: pointer;
    font-size: 13px;
    white-space: nowrap;
  }
  .btn-detect:hover:not(:disabled) { background: #313244; }
  .btn-detect:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-browse {
    padding: 9px 16px;
    background: #1a1a2e;
    border: 1px solid #3a3a5a;
    border-radius: 6px;
    color: #89b4fa;
    cursor: pointer;
    font-size: 13px;
    white-space: nowrap;
  }
  .btn-browse:hover { background: #2a2a4e; border-color: #89b4fa; }

  .error-box {
    background: #1e1e2e;
    border: 1px solid #f38ba8;
    border-radius: 6px;
    padding: 10px 12px;
    font-size: 12px;
    color: #f38ba8;
    white-space: pre-wrap;
    margin-bottom: 12px;
    max-height: 100px;
    overflow-y: auto;
  }

  .state-card {
    background: #11111b;
    border-radius: 8px;
    padding: 14px;
    margin-bottom: 12px;
  }
  .state-badge {
    font-size: 13px;
    font-weight: 600;
    margin-bottom: 10px;
  }
  .state-badge.empty { color: #cdd6f4; }
  .state-badge.partial { color: #f9e2af; }
  .state-badge.ready { color: #a6e3a1; }

  .sig-row {
    display: flex;
    gap: 8px;
    font-size: 12px;
    padding: 2px 0;
  }
  .sig-row .check { width: 16px; }
  .sig-row .path { font-family: monospace; color: #89b4fa; min-width: 220px; }
  .sig-row .reason { color: #6c7086; }

  .sug-list { margin-top: 8px; }
  .sug { font-size: 12px; color: #a6e3a1; padding: 2px 0; }

  .action-row {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .btn-primary {
    padding: 9px 18px;
    background: #89b4fa;
    border: none;
    border-radius: 6px;
    color: #11111b;
    font-weight: 600;
    cursor: pointer;
    font-size: 13px;
  }
  .btn-primary:hover:not(:disabled) { background: #b4befe; }
  .btn-primary:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-secondary {
    padding: 9px 14px;
    background: #1e1e2e;
    border: 1px solid #45475a;
    border-radius: 6px;
    color: #cdd6f4;
    cursor: pointer;
    font-size: 13px;
  }
  .btn-secondary:hover:not(:disabled) { background: #313244; }
  .btn-secondary:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-enter {
    padding: 9px 18px;
    background: #a6e3a1;
    border: none;
    border-radius: 6px;
    color: #11111b;
    font-weight: 600;
    cursor: pointer;
    font-size: 13px;
    margin-left: auto;
  }
  .btn-enter:hover { background: #94d2bd; }

  .hint {
    font-size: 12px;
    color: #45475a;
    text-align: center;
    margin: 8px 0 0;
  }
</style>
