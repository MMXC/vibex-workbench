<!--
  NewL1Wizard — 新建 L1 Project-Goal Spec 向导
  在 empty/partial 状态下创建符合模板的 L1 spec 文件。
-->
<script lang="ts">
  interface Props {
    workspaceRoot: string;
    onCreated: (path: string) => void;
    onCancel: () => void;
  }

  let { workspaceRoot, onCreated, onCancel }: Props = $props();

  let projectName = $state('');
  let description = $state('');
  let owner = $state('');
  let loading = $state(false);
  let error = $state('');
  let nameError = $state('');

  // auto-fill owner from environment
  $effect(() => {
    if (!owner) {
      owner = 'user'; // could be expanded to read from localStorage or env
    }
  });

  function validateName() {
    nameError = '';
    if (!projectName.trim()) {
      nameError = '项目名不能为空';
      return false;
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(projectName)) {
      nameError = '只允许字母、数字、-、_';
      return false;
    }
    return true;
  }

  async function handleCreate() {
    if (!validateName()) return;
    loading = true;
    error = '';

    const slug = projectName.trim().toLowerCase().replace(/\s+/g, '-');
    const now = new Date().toISOString().slice(0, 10);
    const yamlContent = [
      '---',
      'spec:',
      '  version: "0.1"',
      '  level: "1_project-goal"',
      '  name: "' + slug + '"',
      '  parent: null',
      '  status: "proposal"',
      '',
      'meta:',
      '  type: "project-goal"',
      '  owner: "' + owner.trim() + '"',
      '  created: "' + now + '"',
      '  updated: "' + now + '"',
      '',
      'lifecycle:',
      '  current: "proposal"',
      '  updated: "' + now + '"',
      '  history:',
      '    - status: "proposal"',
      '      at: "' + now + 'T00:00:00"',
      '      by: "' + owner.trim() + '"',
      '      trigger: "user:manual"',
      '      note: "' + (description.trim() || '项目目标待补充') + '"',
      '',
      'io_contract:',
      '  input: |',
      '    ' + (description.trim() || '项目目标待补充'),
      '  output: |',
      '    待补充',
      '',
      'behavior:',
      '  - "待补充行为描述"',
      '',
      'constraints:',
      '  - id: "C1"',
      '    name: "待补充约束"',
      '    rule: "待补充"',
      '    validation: "待补充"',
      '',
      'changelog:',
      '  - version: "0.1"',
      '    date: "' + now + '"',
      '    author: "' + owner.trim() + '"',
      '    changes:',
      '      - "初始版本"',
    ].join('\n');

    const relPath = 'specs/L1-goal/' + slug + '.yaml';

    try {
      const res = await fetch('/api/workspace/specs/write', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          workspaceRoot,
          path: relPath,
          content: yamlContent,
        }),
      });

      if (!res.ok) {
        const msg = await res.text();
        error = msg || `HTTP ${res.status}`;
      } else {
        onCreated(relPath);
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }
</script>

<!-- Backdrop -->
<div class="wizard-backdrop" onclick={onCancel} role="presentation"></div>

<!-- Modal -->
<div class="wizard-modal" role="dialog" aria-modal="true" aria-labelledby="wizard-title">
  <div class="wizard-header">
    <h2 id="wizard-title">✨ 新建 L1 Spec</h2>
    <button type="button" class="btn-close" onclick={onCancel} aria-label="关闭">✕</button>
  </div>

  <div class="wizard-body">
    <p class="wizard-desc">
      在 <code>{workspaceRoot}/specs/L1-goal/</code> 下创建项目目标规格文件。
    </p>

    <label class="field">
      <span>项目标识（文件名）<span class="req">*</span></span>
      <input
        type="text"
        bind:value={projectName}
        placeholder="my-project"
        class:err={!!nameError}
        onblur={validateName}
        autocomplete="off"
      />
      {#if nameError}
        <span class="field-error">{nameError}</span>
      {/if}
      <span class="field-hint">路径：specs/L1-goal/<strong>{projectName || '...'}</strong>.yaml</span>
    </label>

    <label class="field">
      <span>项目描述</span>
      <textarea
        bind:value={description}
        placeholder="我要做一个 XX 产品，支持 YY 功能…"
        rows="3"
      ></textarea>
    </label>

    <label class="field">
      <span>Owner（负责人）<span class="req">*</span></span>
      <input
        type="text"
        bind:value={owner}
        placeholder="hermes"
        autocomplete="off"
      />
    </label>

    {#if error}
      <div class="wizard-error">❌ {error}</div>
    {/if}
  </div>

  <div class="wizard-footer">
    <button type="button" class="btn-cancel" onclick={onCancel} disabled={loading}>
      取消
    </button>
    <button type="button" class="btn-create" onclick={handleCreate} disabled={loading || !projectName.trim()}>
      {loading ? '创建中…' : '📄 生成文件'}
    </button>
  </div>
</div>

<style>
  .wizard-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    z-index: 100;
  }

  .wizard-modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: min(520px, calc(100vw - 40px));
    background: var(--wb-bg-panel, #151820);
    border: 1px solid var(--wb-brand, #7aa2ff);
    border-radius: 12px;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
    z-index: 101;
    display: flex;
    flex-direction: column;
    max-height: calc(100vh - 60px);
  }

  .wizard-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255,255,255,0.07);
  }

  .wizard-header h2 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    color: #cdd6f4;
  }

  .btn-close {
    background: none;
    border: none;
    color: var(--wb-brand, #7aa2ff);
    cursor: pointer;
    font-size: 16px;
    padding: 4px 8px;
    border-radius: 4px;
  }
  .btn-close:hover { background: rgba(122, 162, 255, 0.18); }

  .wizard-body {
    padding: 16px 20px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .wizard-desc {
    margin: 0;
    font-size: 12px;
    color: #6c7086;
  }
  .wizard-desc code {
    color: #cba6f7;
    background: rgba(203,166,247,0.1);
    padding: 1px 4px;
    border-radius: 3px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 5px;
    font-size: 12px;
    color: #cdd6f4;
  }

  .req { color: #f38ba8; margin-left: 2px; }

  .field input,
  .field textarea {
    background: #11111b;
    border: 1px solid #313244;
    border-radius: 6px;
    color: #e8e8ed;
    font-family: inherit;
    font-size: 13px;
    padding: 8px 12px;
    resize: vertical;
  }
  .field input:focus,
  .field textarea:focus {
    outline: none;
    border-color: var(--wb-brand, #7aa2ff);
  }
  .field input.err { border-color: #f38ba8; }
  .field input::placeholder,
  .field textarea::placeholder { color: #45475a; }

  .field-error { font-size: 11px; color: #f38ba8; }
  .field-hint { font-size: 11px; color: #45475a; }
  .field-hint strong { color: #cba6f7; }

  .wizard-error {
    background: rgba(243,139,168,0.1);
    border: 1px solid rgba(243,139,168,0.3);
    border-radius: 6px;
    padding: 8px 12px;
    font-size: 12px;
    color: #f38ba8;
  }

  .wizard-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 12px 20px;
    border-top: 1px solid rgba(255,255,255,0.07);
  }

  .btn-cancel {
    padding: 7px 16px;
    background: transparent;
    border: 1px solid #45475a;
    border-radius: 6px;
    color: #a6adc8;
    cursor: pointer;
    font-size: 13px;
  }
  .btn-cancel:hover { background: #313244; color: #cdd6f4; }
  .btn-cancel:disabled { opacity: 0.4; cursor: not-allowed; }

  .btn-create {
    padding: 7px 18px;
    background: var(--wb-brand, #7aa2ff);
    border: none;
    border-radius: 6px;
    color: #fff;
    font-weight: 600;
    cursor: pointer;
    font-size: 13px;
  }
  .btn-create:hover:not(:disabled) { background: var(--accent-hover, #9fc0ff); }
  .btn-create:disabled { opacity: 0.4; cursor: not-allowed; }
</style>
