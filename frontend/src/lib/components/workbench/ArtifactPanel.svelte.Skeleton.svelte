<!-- ============================================================
⚠️  此文件由 spec-to-code 自动生成
来自: specs
生成时间: 2026-04-22
⚠️  不要直接编辑此文件 — 改 *.svelte
============================================================ -->

<script lang="ts">
  import { artifactStore, filteredArtifacts } from '$lib/stores/artifact-store';
  // ArtifactPanel 骨架 — Generated from artifact-registry_uiux.yaml
  let artifacts = $state($filteredArtifacts);

  $effect(() => {
    const unsub = artifactStore.subscribe(() => { artifacts = $filteredArtifacts; });
    return unsub;
  });
</script>

<div class="artifact-panel">
  <div class="header">Artifacts ({artifacts.length})</div>
  <div class="search">
    <input
      placeholder="搜索 artifacts..."
      oninput={(e) => artifactStore.setSearch((e.target as HTMLInputElement).value)}
    />
  </div>
  <div class="items">
    {#each artifacts as a (a.id)}
      <div
        class="artifact-item"
        role="button"
        tabindex="0"
        onclick={() => artifactStore.select(a.id ?? null)}
        onkeydown={(e) => e.key === 'Enter' && artifactStore.select(a.id ?? null)}
      >
        <span class="type">[{a.type}]</span>
        <span class="name">{a.name}</span>
      </div>
    {/each}
    {#if artifacts.length === 0}
      <p class="empty">暂无 Artifact</p>
    {/if}
  </div>
</div>

<style>
  .artifact-panel  { background: #111; height: 100%; overflow: hidden; display: flex; flex-direction: column; }
  .header          { padding: 12px 16px; border-bottom: 1px solid #222; color: #ccc; font-size: 13px; }
  .search          { padding: 8px; }
  .search input    { width: 100%; background: #222; border: 1px solid #444; color: #eee; padding: 6px 10px; border-radius: 6px; font-size: 12px; }
  .items           { flex: 1; overflow-y: auto; }
  .artifact-item   { padding: 8px 16px; cursor: pointer; display: flex; gap: 8px; font-size: 12px; }
  .artifact-item:hover { background: #1a1a1a; }
  .type            { color: #4f46e5; }
  .name            { color: #e2e8f0; }
  .empty           { color: #555; font-size: 12px; padding: 16px; text-align: center; }
</style>
