<!-- ============================================================
ArtifactPreviewModal — Artifact 预览弹窗
E4-U2: 支持图片预览 / 代码高亮
============================================================ -->

<script lang="ts">
  import { artifactStore, selectedArtifact } from '$lib/stores/artifact-store';

  let artifact = $state<ReturnType<typeof selectedArtifact>>(null);
  let isOpen = $state(false);
  let blobUrl = $state<string | null>(null);

  $effect(() => {
    const unsub = artifactStore.subscribe(s => {
      artifact = s.selected_artifact_id ? s.artifacts.find(a => a.id === s.selected_artifact_id) ?? null : null;
      isOpen = !!artifact;
    });
    return unsub;
  });

  // 生成 blob URL 用于图片预览，关闭时清理
  $effect(() => {
    if (!artifact) {
      if (blobUrl) { URL.revokeObjectURL(blobUrl); blobUrl = null; }
      return;
    }
    if (artifact.mime_type?.startsWith('image/') && artifact.content) {
      try {
        const binary = atob(artifact.content.replace(/\s/g, ''));
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
        const blob = new Blob([bytes], { type: artifact.mime_type });
        blobUrl = URL.createObjectURL(blob);
      } catch {
        blobUrl = null;
      }
    }
    return () => {
      if (blobUrl) { URL.revokeObjectURL(blobUrl); blobUrl = null; }
    };
  });

  function close() {
    artifactStore.select(null);
  }
</script>

{#if isOpen && artifact}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-overlay" onclick={close}>
    <div class="modal-content" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <span class="modal-title">{artifact.name}</span>
        <span class="modal-type">[{artifact.type}]</span>
        <button class="close-btn" onclick={close}>×</button>
      </div>
      <div class="modal-body">
        {#if artifact.mime_type?.startsWith('image/') && blobUrl}
          <img src={blobUrl} alt={artifact.name} class="image-preview" />
        {:else}
          <pre class="code-preview"><code>{artifact.content}</code></pre>
        {/if}
      </div>
      {#if artifact.tags.length > 0}
        <div class="modal-footer">
          {#each artifact.tags as tag}
            <span class="tag">#{tag}</span>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(2px);
  }
  .modal-content {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 12px;
    max-width: 800px;
    max-height: 80vh;
    width: 90vw;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .modal-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 14px 16px;
    border-bottom: 1px solid #333;
  }
  .modal-title { color: #e2e8f0; font-size: 14px; font-weight: 500; flex: 1; }
  .modal-type { color: #4f46e5; font-size: 12px; }
  .close-btn { background: none; border: none; color: #666; font-size: 20px; cursor: pointer; padding: 0 4px; }
  .close-btn:hover { color: #fff; }
  .modal-body { flex: 1; overflow: auto; padding: 16px; }
  .image-preview { max-width: 100%; max-height: 60vh; object-fit: contain; border-radius: 6px; }
  .code-preview {
    background: #111;
    border: 1px solid #222;
    border-radius: 6px;
    padding: 12px;
    overflow: auto;
    font-size: 12px;
    line-height: 1.5;
    max-height: 60vh;
    color: #ccc;
    white-space: pre-wrap;
    word-break: break-all;
  }
  .modal-footer { padding: 10px 16px; border-top: 1px solid #222; display: flex; gap: 8px; flex-wrap: wrap; }
  .tag { background: #222; color: #888; font-size: 11px; padding: 2px 8px; border-radius: 10px; }
</style>