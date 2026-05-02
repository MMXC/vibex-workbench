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

  let verifyResult = $state('');
  let verifyLoading = $state(false);

  async function runVerify() {
    if (!workspaceRoot) return;
    verifyLoading = true;
    verifyResult = '';
    error = '';
    try {
      const res = await fetch('/api/workspace/verify-specs', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          workspace_root: workspaceRoot,
          format: 'summary',
          checks: 'file_existence,parent_chain,completeness,behaviors',
        })
      });
      const data = await res.json();
      if (data.error) {
        verifyResult = '❌ ' + data.error;
      } else if (typeof data === 'object' && data.results) {
        // Reconstruct summary text from JSON
        const lines: string[] = [];
        lines.push('📋 Spec Verification — ' + workspaceRoot);
        lines.push('   ' + data.total_specs + ' specs, ' + data.total_checks + ' checks (' + data.pass_count + ' pass, ' + data.fail_count + ' fail, ' + data.warn_count + ' warn)\n');
        for (const r of data.results) {
          if (r.status === 'pass' && !showPass) continue;
          const icon = r.severity === 'error' ? '❌' : r.severity === 'warning' ? '⚠️' : '✅';
          const loc = r.file_path ? '  [' + r.file_path + ']' : '';
          lines.push(icon + ' ' + r.spec_level + '/' + r.spec_name + ' | ' + r.check_type + ' | ' + r.message + loc);
          if (r.suggestion && r.severity !== 'info') {
            lines.push('  💡 ' + r.suggestion);
          }
        }
        if (data.fail_count === 0 && data.warn_count === 0) {
          lines.push('✅ All checks passed!');
        }
        verifyResult = lines.join('\n');
      } else {
        verifyResult = JSON.stringify(data, null, 2);
      }
    } catch (e: any) {
      verifyResult = '❌ ' + e.message;
    } finally {
      verifyLoading = false;
    }
  }

  let showPass = $state(false);

  // ── Spec list ────────────────────────────────────────────
  interface SpecInfo {
    path: string;
    level: number;
    name: string;
    phase: string;
    checks: { fail: number; warn: number; pass: number };
    status: 'pass' | 'warn' | 'fail' | 'unknown';
    errors: string[];
  }

  let specList = $state<SpecInfo[]>([]);
  let specListLoading = $state(false);
  let expandedSpec = $state<string | null>(null);
  let specDetailContent = $state<string | null>(null);
  let specDetailLoading = $state(false);

  async function fetchSpecList() {
    if (!workspaceRoot) return;
    specListLoading = true;
    try {
      const res = await fetch(`/api/workspace/specs/list?workspaceRoot=${encodeURIComponent(workspaceRoot)}`);
      if (!res.ok) return;
      const data = await res.json();
      const paths: string[] = data.paths || [];

      // Fetch verify results in parallel
      let verifyBySpec: Record<string, any> = {};
      try {
        const vr = await fetch('/api/workspace/verify-specs', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ workspace_root: workspaceRoot, format: 'json', checks: 'file_existence,parent_chain,completeness,behaviors' })
        });
        if (vr.ok) {
          const vdata = await vr.json();
          if (vdata.results) {
            for (const r of vdata.results) {
              const key = `${r.spec_level}/${r.spec_name}`;
              if (!verifyBySpec[key]) verifyBySpec[key] = [];
              verifyBySpec[key].push(r);
            }
          }
        }
      } catch { /* verify is optional */ }

      specList = paths.map(p => {
        const parts = p.replace(/^specs\//, '').replace(/\.ya?ml$/, '').split('/');
        const levelPart = parts[0] || '';
        const levelNum = parseInt(levelPart.replace(/\D/g, '') || '0');
        const name = parts.slice(1).join('/') || levelPart;
        const key = `${levelPart}/${name}`;
        const checks = verifyBySpec[key] || [];
        const fail = checks.filter((c: any) => c.severity === 'error').length;
        const warn = checks.filter((c: any) => c.severity === 'warning').length;
        const pass = checks.filter((c: any) => c.severity === 'info').length;
        const status: SpecInfo['status'] = fail > 0 ? 'fail' : warn > 0 ? 'warn' : pass > 0 ? 'pass' : 'unknown';
        const errors = checks.filter((c: any) => c.severity === 'error').map((c: any) => c.message);
        return { path: p, level: levelNum, name, phase: '', checks: { fail, warn, pass }, status, errors };
      });
    } catch { /* ignore */ } finally {
      specListLoading = false;
    }
  }

  async function toggleSpecDetail(path: string) {
    if (expandedSpec === path) {
      expandedSpec = null;
      specDetailContent = null;
      return;
    }
    expandedSpec = path;
    specDetailLoading = true;
    specDetailContent = null;
    try {
      const res = await fetch(`/api/workspace/specs/read?workspaceRoot=${encodeURIComponent(workspaceRoot)}&path=${encodeURIComponent(path)}`);
      if (res.ok) {
        const data = await res.json();
        specDetailContent = data.content || '';
      }
    } catch { /* ignore */ } finally {
      specDetailLoading = false;
    }
  }

  function levelLabel(level: number) {
    if (level <= 1) return 'L1';
    if (level === 2) return 'L2';
    if (level === 3) return 'L3';
    if (level === 4) return 'L4';
    return 'L5';
  }

  function levelColor(level: number) {
    if (level <= 1) return '#72d6d0';
    if (level === 2) return '#87cf8a';
    if (level === 3) return '#7aa2ff';
    if (level === 4) return '#efc66b';
    return '#f09a6a';
  }

  function statusIcon(s: SpecInfo['status']) {
    if (s === 'pass') return '✅';
    if (s === 'warn') return '⚠️';
    if (s === 'fail') return '❌';
    return '⬜';
  }

  // Auto-fetch spec list when workspace is ready
  $effect(() => {
    if (state?.state === 'ready' || state?.state === 'partial') {
      fetchSpecList();
    }
  });

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
          <button onclick={runVerify} disabled={verifyLoading} class="btn-verify">
            {verifyLoading ? '🔍 验证中…' : '🔎 验证 Spec 对齐'}
          </button>
          <button onclick={enterWorkbench} disabled={loading} class="btn-enter">
            → 进入 Workbench
          </button>
        </div>
      {/if}

      {#if verifyResult}
        <div class="verify-result">
          <pre>{verifyResult}</pre>
          <label class="show-pass-toggle">
            <input type="checkbox" bind:checked={showPass} onchange={() => runVerify()} />
            显示通过的检查
          </label>
        </div>
      {/if}

      <!-- Spec 列表卡片 -->
      {#if specList.length > 0 || specListLoading}
        <div class="spec-list-card">
          <div class="spec-list-head">
            <span class="spec-list-title">📋 Spec 列表</span>
            <span class="spec-count">{specList.length} 个规格</span>
            <button type="button" class="reload-btn" onclick={fetchSpecList} title="刷新">↻</button>
          </div>
          {#if specListLoading}
            <div class="spec-loading">加载中…</div>
          {:else}
            <div class="spec-grid">
              {#each specList as spec (spec.path)}
                <div class="spec-card" class:card-fail={spec.status === 'fail'} class:card-warn={spec.status === 'warn'} class:card-pass={spec.status === 'pass'} onclick={() => toggleSpecDetail(spec.path)}>
                  <div class="card-top">
                    <span class="card-level" style:color={levelColor(spec.level)}>{levelLabel(spec.level)}</span>
                    <span class="card-icon">{statusIcon(spec.status)}</span>
                  </div>
                  <div class="card-name" title={spec.name}>{spec.name}</div>
                  <div class="card-meta">
                    {#if spec.checks.fail > 0}
                      <span class="badge badge-fail">❌ {spec.checks.fail}</span>
                    {/if}
                    {#if spec.checks.warn > 0}
                      <span class="badge badge-warn">⚠️ {spec.checks.warn}</span>
                    {/if}
                    {#if spec.checks.pass > 0 && spec.status !== 'fail' && spec.status !== 'warn'}
                      <span class="badge badge-pass">✅ {spec.checks.pass}</span>
                    {/if}
                    {#if spec.status === 'unknown'}
                      <span class="badge badge-unknown">⬜ 未验证</span>
                    {/if}
                  </div>
                  {#if spec.errors.length > 0}
                    <div class="card-errors">
                      {#each spec.errors.slice(0, 2) as err}
                        <div class="card-err-item">⚠️ {err}</div>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>

            <!-- 展开的 spec 详情 -->
            {#if expandedSpec}
              <div class="spec-detail">
                <div class="spec-detail-head">
                  <span class="spec-detail-title">{expandedSpec}</span>
                  <button type="button" class="close-btn" onclick={() => { expandedSpec = null; specDetailContent = null; }}>×</button>
                </div>
                {#if specDetailLoading}
                  <div class="spec-loading">加载中…</div>
                {:else if specDetailContent}
                  <pre class="spec-detail-content">{specDetailContent}</pre>
                {:else}
                  <div class="spec-loading">无内容</div>
                {/if}
              </div>
            {/if}
          {/if}
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

  .btn-verify {
    padding: 9px 18px;
    background: #cba6f7;
    border: none;
    border-radius: 6px;
    color: #11111b;
    font-weight: 600;
    cursor: pointer;
    font-size: 13px;
  }
  .btn-verify:hover:not(:disabled) { background: #b689d6; }
  .btn-verify:disabled { opacity: 0.4; cursor: not-allowed; }

  .verify-result {
    background: #11111b;
    border: 1px solid #313244;
    border-radius: 8px;
    padding: 14px;
    margin-top: 12px;
    max-height: 400px;
    overflow-y: auto;
  }
  .verify-result pre {
    margin: 0;
    font-size: 12px;
    color: #cdd6f4;
    white-space: pre-wrap;
    word-break: break-all;
    line-height: 1.6;
  }
  .show-pass-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: #6c7086;
    margin-top: 10px;
    cursor: pointer;
  }
  .show-pass-toggle input {
    accent-color: #89b4fa;
  }

  .hint {
    font-size: 12px;
    color: #45475a;
    text-align: center;
    margin: 8px 0 0;
  }

  /* ── Spec List Card ─────────────────────────────────────── */
  .spec-list-card {
    margin-top: 12px;
    background: #11111b;
    border: 1px solid #313244;
    border-radius: 8px;
    overflow: hidden;
  }

  .spec-list-head {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 14px;
    border-bottom: 1px solid #313244;
    background: #1a1a2e;
  }

  .spec-list-title {
    font-size: 13px;
    font-weight: 600;
    color: #cdd6f4;
  }

  .spec-count {
    font-size: 11px;
    color: #6c7086;
    margin-left: auto;
  }

  .reload-btn {
    width: 24px;
    height: 24px;
    background: transparent;
    border: 1px solid #45475a;
    border-radius: 4px;
    color: #6c7086;
    cursor: pointer;
    font-size: 13px;
    padding: 0;
    line-height: 1;
  }

  .reload-btn:hover {
    color: #cdd6f4;
    border-color: #89b4fa;
  }

  .spec-loading {
    padding: 12px 14px;
    font-size: 12px;
    color: #6c7086;
  }

  .spec-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 8px;
    padding: 10px;
    max-height: 300px;
    overflow-y: auto;
  }

  .spec-card {
    background: #1a1a2e;
    border: 1px solid #313244;
    border-radius: 6px;
    padding: 8px 10px;
    cursor: pointer;
    transition: background 120ms, border-color 120ms, transform 120ms;
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .spec-card:hover {
    background: #252540;
    border-color: #45475a;
    transform: translateY(-1px);
  }

  .spec-card.card-fail { border-color: #f38ba840; background: #1e1e2e; }
  .spec-card.card-warn { border-color: #f9e2af30; background: #1e1e2a; }
  .spec-card.card-pass { border-color: #a6e3a120; }

  .card-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .card-level {
    font-family: 'Cascadia Code', monospace;
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.05em;
  }

  .card-icon {
    font-size: 12px;
  }

  .card-name {
    font-size: 11px;
    color: #cdd6f4;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-weight: 500;
  }

  .card-meta {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
  }

  .badge {
    font-size: 9px;
    padding: 1px 5px;
    border-radius: 3px;
    white-space: nowrap;
  }

  .badge-fail { background: #f38ba820; color: #f38ba8; }
  .badge-warn { background: #f9e2af20; color: #f9e2af; }
  .badge-pass { background: #a6e3a120; color: #a6e3a1; }
  .badge-unknown { background: #31324440; color: #6c7086; }

  .card-errors {
    border-top: 1px solid #31324440;
    padding-top: 4px;
    margin-top: 2px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .card-err-item {
    font-size: 9px;
    color: #f38ba8;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Spec detail panel */
  .spec-detail {
    border-top: 1px solid #45475a;
    background: #0d0d12;
    max-height: 400px;
    overflow-y: auto;
  }

  .spec-detail-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 14px;
    border-bottom: 1px solid #313244;
    background: #1a1a2e;
    position: sticky;
    top: 0;
    z-index: 1;
  }

  .spec-detail-title {
    font-family: monospace;
    font-size: 11px;
    color: #89b4fa;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .spec-detail-content {
    margin: 0;
    padding: 12px 14px;
    font-size: 11px;
    line-height: 1.6;
    color: #cdd6f4;
    white-space: pre-wrap;
    word-break: break-all;
    background: transparent;
    border: none;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: #6c7086;
    cursor: pointer;
    font-size: 16px;
    padding: 0 4px;
    line-height: 1;
  }

  .close-btn:hover { color: #cdd6f4; }
</style>
