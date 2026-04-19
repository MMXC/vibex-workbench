<script lang="ts">
  interface SpecRec { path: string; layer: string; score: number; reason: string; }
  interface Props { recommendations?: SpecRec[]; onselect?: (path: string) => void; }
  let { recommendations = [], onselect }: Props = $props();
  const layerColors: Record<string, string> = { 'L1': '#FF6B6B', 'L2': '#FFA94D', 'L3': '#FFD93D', 'L4': '#6BCB77', 'L5': '#4D96FF' };
  let selected = $state<string | null>(null);
</script>

<div class="SpecRecommendationList">
  {#if recommendations.length === 0}
    <div class="empty">
      <span class="empty-icon">📋</span>
      <p>输入需求描述后，此处将显示推荐修改的规格层级</p>
    </div>
  {:else}
    <ul class="rec-list">
      {#each recommendations as rec}
        <li class="rec-item" class:selected={selected === rec.path}
          onclick={() => { selected = rec.path; onselect?.(rec.path); }}
          role="button" tabindex="0"
          onkeydown={(e) => e.key === 'Enter' && ((selected = rec.path), onselect?.(rec.path))}>
          <div class="rec-header">
            <span class="layer-badge" style="background: {layerColors[rec.layer] || '#888'}">{rec.layer}</span>
            <span class="path">{rec.path}</span>
            <span class="score">{Math.round(rec.score * 100)}%</span>
          </div>
          <p class="reason">{rec.reason}</p>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .SpecRecommendationList { height: 100%; overflow: auto; }
  .empty { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; gap: 8px; color: #666; text-align: center; padding: 24px; }
  .empty-icon { font-size: 32px; opacity: 0.5; }
  .empty p { font-size: 13px; margin: 0; }
  .rec-list { list-style: none; margin: 0; padding: 8px; display: flex; flex-direction: column; gap: 6px; }
  .rec-item { padding: 10px; border-radius: 6px; background: #1e1e2e; border: 1px solid #333; cursor: pointer; transition: all 0.15s; }
  .rec-item:hover { background: #252540; border-color: #4D96FF; }
  .rec-item.selected { background: #1a2a4a; border-color: #4D96FF; }
  .rec-header { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
  .layer-badge { font-size: 10px; padding: 1px 6px; border-radius: 3px; color: #000; font-weight: 600; }
  .path { font-size: 12px; color: #e0e0e0; font-family: monospace; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .score { font-size: 11px; color: #888; }
  .reason { font-size: 11px; color: #888; margin: 0; }
</style>
