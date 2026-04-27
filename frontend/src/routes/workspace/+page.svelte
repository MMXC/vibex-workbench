<!--
  VibeX Workspace Entry — MVP: 选定仓库根
  路由: /workspace
  入口: 用户指定一个目录 → 检测状态 → (可选)初始化脚手架 → 进入 Workbench
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import WorkspaceSelector from '$lib/components/workbench/WorkspaceSelector.svelte';
  import NewL1Wizard from '$lib/components/workbench/NewL1Wizard.svelte';

  let workspaceRoot = $state('');
  let state = $state<{state: string, signals: any[], suggestions: string[]} | null>(null);
  let loading = $state(false);
  let error = $state('');
  let scaffoldPreview = $state(false);          // true = 预览确认模式
  let scaffoldPreviewItems = $state<string[]>([]);  // 预览文件列表
  let wizardOpen = $state(false);              // true = 新建 L1 向导

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
      if (data.output) error = data.output;
      else await detectState();
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function openScaffoldPreview() {
    scaffoldPreviewItems = [
      'specs/L1-goal/',
      'specs/L2-skeleton/',
      'generators/',
      'generators/gen.py',
      'generators/validate_specs.py',
      'Makefile',
      'README.md'
    ];
    scaffoldPreview = true;
  }

  async function confirmScaffold() {
    scaffoldPreview = false;  // 关闭预览模式
    await scaffold();         // 调用已有 scaffold()
  }

  function openWizard() {
    wizardOpen = true;
  }

  function closeWizard() {
    wizardOpen = false;
  }

  async function handleWizardCreated(path: string) {
    wizardOpen = false;
    // 刷新状态（specs/ 目录内容变了）
    await detectState();
    // TODO: 可选导航到 /workbench 打开新文件
  }

  async function runMake(target: string) {
    if (!workspaceRoot) return;
    loading = true;
    error = '';
    try {
      const res = await fetch('/api/workspace/run-make', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({target, workspace: workspaceRoot})
      });
      const data = await res.json();
      error = data.output || (data.ok ? 'OK' : 'FAILED');
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function enterWorkbench() {
    if (!workspaceRoot) return;
    localStorage.setItem('vibex-workspace-root', workspaceRoot);
    goto('/workbench');
  }

  // 状态已就绪时，可进入 workbench
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
          {#if !scaffoldPreview}
            <button onclick={openScaffoldPreview} disabled={loading} class="btn-primary">
              📦 初始化脚手架
            </button>
            <button onclick={openWizard} disabled={loading} class="btn-secondary">
              📄 新建 L1 Spec
            </button>
          {/if}

          {#if scaffoldPreview}
            <div class="scaffold-confirm-drawer">
              <h3>即将创建以下文件：</h3>
              <ul>
                {#each scaffoldPreviewItems as item}
                  <li><code>{item}</code></li>
                {/each}
              </ul>
              <div class="confirm-actions">
                <button onclick={confirmScaffold} disabled={loading} class="btn-confirm">
                  ✅ 确认写入
                </button>
                <button onclick={() => { scaffoldPreview = false; }} disabled={loading} class="btn-cancel-scaffold">
                  取消
                </button>
              </div>
            </div>
          {/if}
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

  {#if wizardOpen}
    <NewL1Wizard
      {workspaceRoot}
      onCreated={handleWizardCreated}
      onCancel={closeWizard}
    />
  {/if}
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

  /* ── scaffold confirm drawer ── */
  .scaffold-confirm-drawer {
    background: #1e1e2e;
    border: 1px solid #cba6f7;
    border-radius: 8px;
    padding: 16px;
    margin-top: 10px;
  }
  .scaffold-confirm-drawer h3 {
    margin: 0 0 12px;
    font-size: 13px;
    color: #cdd6f4;
    font-weight: 600;
  }
  .scaffold-confirm-drawer ul {
    margin: 0 0 14px 20px;
    padding: 0;
    font-size: 12px;
    color: #a6adc8;
  }
  .scaffold-confirm-drawer ul li {
    margin-bottom: 4px;
  }
  .confirm-actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .btn-confirm {
    padding: 7px 14px;
    background: #a6e3a1;
    border: none;
    border-radius: 6px;
    color: #11111b;
    font-weight: 600;
    cursor: pointer;
    font-size: 13px;
  }
  .btn-confirm:hover:not(:disabled) { background: #94d2bd; }
  .btn-confirm:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-cancel-scaffold {
    padding: 7px 14px;
    background: transparent;
    border: 1px solid #45475a;
    border-radius: 6px;
    color: #a6adc8;
    cursor: pointer;
    font-size: 13px;
  }
  .btn-cancel-scaffold:hover:not(:disabled) { background: #313244; color: #cdd6f4; }
  .btn-cancel-scaffold:disabled { opacity: 0.4; cursor: not-allowed; }
</style>
