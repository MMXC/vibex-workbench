<script lang="ts">
  interface Tab { id: string; label: string; icon?: string; badge?: number; }
  interface Props { tabs: Tab[]; activeTab?: string; onchange?: (id: string) => void; }
  let { tabs, activeTab = $bindable(''), onchange }: Props = $props();
  function select(id: string) { activeTab = id; onchange?.(id); }
</script>

<div class="TabBar">
  {#each tabs as tab}
    <button class="tab" class:active={activeTab === tab.id} onclick={() => select(tab.id)}>
      {#if tab.icon}<span>{tab.icon}</span>{/if}
      <span>{tab.label}</span>
      {#if tab.badge && tab.badge > 0}<span class="badge">{tab.badge > 99 ? '99+' : tab.badge}</span>{/if}
    </button>
  {/each}
</div>

<style>
  .TabBar { display: flex; background: #1a1a2e; border-bottom: 1px solid #333; overflow-x: auto; }
  .tab { display: flex; align-items: center; gap: 4px; padding: 8px 16px; background: transparent; border: none; border-bottom: 2px solid transparent; color: #888; cursor: pointer; font-size: 13px; white-space: nowrap; transition: all 0.15s; }
  .tab:hover { background: #252540; color: #e0e0e0; }
  .tab.active { color: #e0e0e0; border-bottom-color: #4D96FF; }
  .badge { background: #FF6B6B; color: #fff; font-size: 10px; padding: 1px 5px; border-radius: 10px; font-weight: 600; }
</style>
