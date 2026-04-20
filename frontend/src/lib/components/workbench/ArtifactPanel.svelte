// ============================================================
// ArtifactPanel — Artifact 注册表面板
// E4-U1: IndexedDB 持久化（页面刷新恢复）
// E4-U2: 点击预览 ArtifactPreviewModal
// E4-U3: 拖拽到 Composer 注入 @artifactId
// 开发者维护，gen.py 永不覆盖
// ============================================================

<script lang="ts">
  import { artifactStore, filteredArtifacts } from '$lib/stores/artifact-store';
  import ArtifactPreviewModal from './ArtifactPreviewModal.svelte';

  let artifacts = $state<ReturnType<typeof filteredArtifacts>>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  $effect(() => {
    const unsub = artifactStore.subscribe(s => {
      artifacts = $filteredArtifacts;
      loading = s.loading;
      error = s.error;
    });
    return unsub;
  });

  // 页面初始化时从 IndexedDB 加载
  artifactStore.loadFromDB();

  // E4-U3: 拖拽开始，设置 dataTransfer
  function handleDragStart(e: DragEvent, id: string) {
    e.dataTransfer?.setData('text/vibex-artifact', id);
    e.dataTransfer!.effectAllowed = 'copy';
  }

  // 重试加载
  function handleRetry() {
    artifactStore.setError(null);
    artifactStore.loadFromDB();
  }
</script>

<div class="artifact-panel">
  <div class="header">Artifacts ({artifacts.length})</div>
  <div class="search">
    <input
      placeholder="搜索..."
      oninput={(e) => artifactStore.setSearch((e.target as HTMLInputElement).value)}
    />
  </div>
  <div class="filter">
    <button class:active={!artifactStore} onClick={() => artifactStore.setFilter(null)}>全部</button>
    <button onClick={() => artifactStore.setFilter('code')}>代码</button>
    <button onClick={() => artifactStore.setFilter('image')}>图片</button>
  </div>
  <div class="items">
    {#if loading}
      <!-- 骨架屏 -->
      {#each { length: 4 } as _, i}
        <div class="artifact-item skeleton" style="width: {70 + (i * 15) % 30}%">
          <div class="skel-icon"></div>
          <div class="skel-name"></div>
        </div>
      {/each}
    {:else if error}
      <div class="error-state">
        <span>⚠</span>
        <p>{error}</p>
        <button onclick={handleRetry}>重试</button>
      </div>
    {:else if artifacts.length === 0}
      <p class="empty">暂无 Artifact</p>
    {:else}
      {#each artifacts as a (a.id)}
        <!-- E4-U3: draggable, E4-U2: click to select -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div
          class="artifact-item"
          class:selected={artifactStore['selected_artifact_id'] === a.id}
          draggable="true"
          ondragstart={(e) => handleDragStart(e, a.id)}
          onclick={() => artifactStore.select(a.id)}
          onkeydown={(e) => e.key === 'Enter' && artifactStore.select(a.id)}
          role="button"
          tabindex="0"
          title="拖拽到 Composer 或点击预览"
        >
          <span class="type">[{a.type}]</span>
          <span class="name">{a.name}</span>
          <span class="drag-hint">⋮⋮</span>
        </div>
      {/each}
    {/if}
  </div>
</div>

<!-- E4-U2: Artifact 预览弹窗 -->
<ArtifactPreviewModal />

<style>
  .artifact-panel { background: #111; height: 100%; overflow: hidden; display: flex; flex-direction: column; }
  .header { padding: 12px 16px; border-bottom: 1px solid #222; color: #ccc; font-size: 13px; }
  .search { padding: 8px; }
  .search input { width: 100%; background: #222; border: 1px solid #444; color: #eee; padding: 6px 10px; border-radius: 6px; font-size: 12px; box-sizing: border-box; }
  .filter { display: flex; gap: 4px; padding: 0 8px 8px; }
  .filter button { background: transparent; border: none; color: #666; font-size: 11px; cursor: pointer; padding: 3px 8px; border-radius: 4px; }
  .filter button:hover { color: #ccc; }
  .filter button.active { background: #333; color: #fff; }
  .items { flex: 1; overflow-y: auto; }
  .artifact-item {
    padding: 8px 16px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    border-bottom: 1px solid #1a1a1a;
    user-select: none;
  }
  .artifact-item:hover { background: #1a1a1a; }
  .artifact-item.selected { background: #1e293b; border-left: 3px solid #4f46e5; }
  .artifact-item[draggable="true"]:hover .drag-hint { opacity: 1; }
  .type { color: #4f46e5; font-size: 11px; flex-shrink: 0; }
  .name { color: #e2e8f0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .drag-hint { color: #444; font-size: 10px; opacity: 0; transition: opacity 0.15s; }
  .empty { color: #555; font-size: 12px; padding: 16px; text-align: center; }

  /* 骨架屏 */
  .artifact-item.skeleton { pointer-events: none; }
  .skel-icon, .skel-name { height: 8px; border-radius: 4px; background: linear-gradient(90deg, #2a2a2a 25%, #3a3a3a 50%, #2a2a2a 75%); background-size: 200% 100%; animation: shimmer 1.4s ease-in-out infinite; }
  .skel-icon { width: 20px; flex-shrink: 0; }
  .skel-name { flex: 1; }
  @keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

  /* 错误态 */
  .error-state { display: flex; flex-direction: column; align-items: center; padding: 20px; gap: 6px; }
  .error-state span { font-size: 20px; }
  .error-state p { color: #f87171; font-size: 11px; text-align: center; margin: 0; }
  .error-state button { background: #374151; border: none; color: #ccc; padding: 4px 12px; border-radius: 4px; cursor: pointer; font-size: 11px; }
</style>