<script lang="ts">
    import { onMount } from 'svelte';

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
            if (data.output) error = data.output;
            else await detectState();
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

    onMount(() => {
        // 从 localStorage 恢复上次路径
        workspaceRoot = localStorage.getItem('vibex-workspace-root') || '';
    });

    function saveRoot() {
        localStorage.setItem('vibex-workspace-root', workspaceRoot);
    }

    const stateIcons: Record<string, string> = {
        empty: '⬜',
        partial: '🟨',
        ready: '🟩',
        error: '❌'
    };
</script>

<div class="workspace-selector">
    <div class="selector-header">
        <h3>VibeX 工作区</h3>
        <input
            type="text"
            bind:value={workspaceRoot}
            placeholder="/path/to/your/project"
            class="workspace-input"
            onblur={saveRoot}
        />
        <div class="btn-row">
            <button onclick={detectState} disabled={loading || !workspaceRoot}>
                {loading ? '检测中...' : '🔍 检测状态'}
            </button>
            <button onclick={scaffold} disabled={loading || !workspaceRoot}>
                📦 初始化脚手架
            </button>
            <button onclick={() => runMake('validate')} disabled={loading || !workspaceRoot}>
                ✅ 校验 Spec
            </button>
            <button onclick={() => runMake('generate')} disabled={loading || !workspaceRoot}>
                ⚙️ 生成代码
            </button>
        </div>
    </div>

    {#if error}
        <div class="error-msg">{error}</div>
    {/if}

    {#if state}
        <div class="state-panel">
            <div class="state-badge {state.state}">
                {stateIcons[state.state] || '❓'} {state.state.toUpperCase()}
            </div>
            <div class="signals">
                {#each state.signals as sig}
                    <div class="signal-row">
                        <span class="check">{sig.exists ? '✅' : '❌'}</span>
                        <span class="path">{sig.path}</span>
                        <span class="reason">{sig.reason}</span>
                    </div>
                {/each}
            </div>
            {#if state.suggestions?.length}
                <div class="suggestions">
                    {#each state.suggestions as sug}
                        <div class="sug">→ {sug}</div>
                    {/each}
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .workspace-selector {
        background: #1e1e2e;
        border: 1px solid #3a3a5a;
        border-radius: 8px;
        padding: 16px;
        color: #cdd6f4;
        font-family: system-ui;
        max-width: 600px;
    }
    .selector-header h3 { margin: 0 0 8px 0; font-size: 14px; color: #89b4fa; }
    .workspace-input {
        width: 100%;
        padding: 8px;
        background: #11111b;
        border: 1px solid #313244;
        border-radius: 4px;
        color: #cdd6f4;
        margin-bottom: 8px;
        font-family: monospace;
        font-size: 13px;
    }
    .btn-row { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 8px; }
    .btn-row button {
        padding: 6px 12px;
        background: #1e1e2e;
        border: 1px solid #45475a;
        border-radius: 4px;
        color: #cdd6f4;
        cursor: pointer;
        font-size: 12px;
    }
    .btn-row button:hover:not(:disabled) { background: #313244; }
    .btn-row button:disabled { opacity: 0.4; cursor: not-allowed; }
    .error-msg {
        background: #11111b;
        border: 1px solid #f38ba8;
        border-radius: 4px;
        padding: 8px;
        font-size: 12px;
        color: #f38ba8;
        white-space: pre-wrap;
        max-height: 120px;
        overflow-y: auto;
        margin-bottom: 8px;
    }
    .state-panel { background: #11111b; border-radius: 4px; padding: 12px; }
    .state-badge { font-weight: bold; margin-bottom: 8px; font-size: 13px; }
    .state-badge.empty { color: #cdd6f4; }
    .state-badge.partial { color: #f9e2af; }
    .state-badge.ready { color: #a6e3a1; }
    .signal-row { display: flex; gap: 8px; font-size: 12px; padding: 2px 0; }
    .signal-row .check { width: 16px; }
    .signal-row .path { font-family: monospace; color: #89b4fa; min-width: 200px; }
    .signal-row .reason { color: #6c7086; }
    .sug { font-size: 12px; color: #a6e3a1; padding: 2px 0; }
</style>
