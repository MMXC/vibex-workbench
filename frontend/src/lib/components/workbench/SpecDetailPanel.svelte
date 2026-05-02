<!-- SpecDetailPanel.svelte — Spec detail view panel
     Shows spec content, metadata, and links
     SLICE-spec-detail-panel
-->
<script lang="ts">
  import type { SpecViewModel } from '$lib/stores/spec-explorer-store';

  interface Props {
    specPath?: string;
    onClose?: () => void;
  }

  let { specPath = '', onClose }: Props = $props();

  let content = $state<string | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let activeTab = $state<'content' | 'metadata' | 'journal'>('content');

  async function loadSpec() {
    if (!specPath) return;
    loading = true;
    error = null;
    try {
      const res = await fetch(`/api/workspace/specs/read?path=${encodeURIComponent(specPath)}`);
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      content = data.content ?? '';
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (specPath) loadSpec();
  });

  function formatPath(p: string) {
    const parts = p.split('/');
    return parts[parts.length - 1];
  }
</script>

<div class="detail-panel">
  <div class="detail-head">
    <div class="detail-head-left">
      <span class="detail-label">Spec</span>
      <span class="detail-spec-name">{formatPath(specPath)}</span>
    </div>
    <div class="detail-head-right">
      <div class="tab-group">
        <button type="button" class="tab" class:active={activeTab === 'content'} onclick={() => activeTab = 'content'}>内容</button>
        <button type="button" class="tab" class:active={activeTab === 'metadata'} onclick={() => activeTab = 'metadata'}>元数据</button>
        <button type="button" class="tab" class:active={activeTab === 'journal'} onclick={() => activeTab = 'journal'}>日志</button>
      </div>
      <button type="button" class="close-btn" onclick={onClose}>×</button>
    </div>
  </div>

  <div class="detail-body">
    {#if loading}
      <div class="detail-loading">加载中…</div>
    {:else if error}
      <div class="detail-error">{error}</div>
    {:else if content}
      {#if activeTab === 'content'}
        <pre class="detail-content">{content}</pre>
      {:else if activeTab === 'metadata'}
        <div class="metadata-view">
          <div class="meta-item">
            <span class="meta-label">路径</span>
            <span class="meta-value">{specPath}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">文件名</span>
            <span class="meta-value">{formatPath(specPath)}</span>
          </div>
        </div>
      {:else}
        <div class="journal-placeholder">
          <p>Journal 内容将在下次迭代中加载</p>
        </div>
      {/if}
    {:else}
      <div class="detail-empty">
        <p>选择一个 spec 以查看详情</p>
      </div>
    {/if}
  </div>
</div>

<style>
  .detail-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #0d0d0e;
  }

  .detail-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #27272a;
    flex-shrink: 0;
    gap: 0.5rem;
  }

  .detail-head-left {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    min-width: 0;
  }

  .detail-label {
    font-size: 9px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #52525b;
  }

  .detail-spec-name {
    font-size: 12px;
    font-weight: 600;
    color: #fafafa;
    font-family: ui-monospace, monospace;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .detail-head-right {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    flex-shrink: 0;
  }

  .tab-group {
    display: flex;
    gap: 0;
  }

  .tab {
    font-size: 10px;
    padding: 0.2rem 0.5rem;
    background: none;
    border: 1px solid transparent;
    color: #71717a;
    cursor: pointer;
    border-radius: 4px;
  }

  .tab.active {
    background: #27272a;
    color: #e4e4e7;
    border-color: #3f3f46;
  }

  .close-btn {
    background: none;
    border: none;
    color: #71717a;
    font-size: 16px;
    cursor: pointer;
    padding: 0 0.2rem;
  }
  .close-btn:hover { color: #e4e4e7; }

  .detail-body {
    flex: 1;
    overflow: auto;
    padding: 0.75rem;
  }

  .detail-loading, .detail-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    font-size: 11px;
    color: #52525b;
  }

  .detail-error {
    font-size: 11px;
    color: #fca5a5;
    padding: 0.5rem;
    background: rgba(239, 68, 68, 0.1);
    border-radius: 6px;
  }

  .detail-content {
    font-size: 11px;
    font-family: ui-monospace, monospace;
    color: #d4d4d8;
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    line-height: 1.5;
  }

  .metadata-view {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .meta-item {
    display: flex;
    gap: 0.5rem;
    font-size: 11px;
  }

  .meta-label {
    color: #71717a;
    font-weight: 600;
    text-transform: uppercase;
    font-size: 9px;
    letter-spacing: 0.04em;
    flex-shrink: 0;
    width: 50px;
    padding-top: 0.1rem;
  }

  .meta-value {
    color: #e4e4e7;
    font-family: ui-monospace, monospace;
    word-break: break-all;
  }

  .journal-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    font-size: 11px;
    color: #52525b;
  }
</style>