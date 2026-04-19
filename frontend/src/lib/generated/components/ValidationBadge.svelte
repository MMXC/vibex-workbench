<script lang="ts">
  interface Props { status: 'valid' | 'invalid' | 'warning' | 'pending'; message?: string; details?: string[]; }
  let { status, message, details = [] }: Props = $props();
  const cfg = { valid: { icon: '✅', color: '#6BCB77', label: '通过' }, invalid: { icon: '❌', color: '#FF6B6B', label: '失败' }, warning: { icon: '⚠️', color: '#FFD93D', label: '警告' }, pending: { icon: '⏳', color: '#888', label: '待验证' } };
  let c = $derived(cfg[status] || cfg.pending);
  let expanded = $state(false);
</script>

<div class="ValidationBadge" style="--c: {c.color}">
  <button class="btn" onclick={() => expanded = !expanded}>
    <span>{c.icon}</span><span>{message || c.label}</span>
    <span class="chevron" class:open={expanded}>▼</span>
  </button>
  {#if expanded && details.length > 0}
    <ul class="details">
      {#each details as d}<li>{d}</li>{/each}
    </ul>
  {/if}
</div>

<style>
  .ValidationBadge { display: inline-flex; flex-direction: column; }
  .btn { display: flex; align-items: center; gap: 6px; padding: 4px 10px; background: color-mix(in srgb, var(--c) 12%, transparent); border: 1px solid var(--c); border-radius: 20px; color: var(--c); cursor: pointer; font-size: 12px; transition: filter 0.15s; }
  .btn:hover { filter: brightness(1.2); }
  .chevron { font-size: 8px; transition: transform 0.15s; }
  .chevron.open { transform: rotate(180deg); }
  .details { list-style: none; margin: 6px 0 0 0; padding: 8px 12px; background: #1e1e2e; border: 1px solid #333; border-radius: 6px; font-size: 11px; color: #888; max-width: 300px; }
  .details li { padding: 2px 0; }
</style>
