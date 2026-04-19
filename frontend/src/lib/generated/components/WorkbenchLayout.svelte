<script lang="ts">
  import type { Snippet } from 'svelte';
  interface Props { sidebar?: Snippet; main?: Snippet; panel?: Snippet; toolbar?: Snippet; statusbar?: Snippet; composer?: Snippet; }
  let { sidebar, main, panel, toolbar, statusbar, composer }: Props = $props();
  let showPanel = $state(true);
</script>

<div class="WorkbenchLayout">
  {#if toolbar}<header class="toolbar-area">{@render toolbar()}</header>{/if}
  <div class="body">
    {#if sidebar}<aside class="sidebar-area">{@render sidebar()}</aside>{/if}
    <main class="main-area">{@render main?.()}</main>
    {#if panel && showPanel}<aside class="panel-area">{@render panel()}</aside>{/if}
  </div>
  {#if composer}<footer class="composer-area">{@render composer()}</footer>{/if}
  {#if statusbar}<footer class="statusbar-area">{@render statusbar()}</footer>{/if}
</div>

<style>
  .WorkbenchLayout { display: flex; flex-direction: column; height: 100vh; background: #0f0f1a; color: #e0e0e0; overflow: hidden; }
  .toolbar-area { flex-shrink: 0; }
  .body { flex: 1; display: flex; overflow: hidden; min-height: 0; }
  .sidebar-area { width: 260px; flex-shrink: 0; border-right: 1px solid #333; overflow: auto; }
  .main-area { flex: 1; overflow: auto; min-width: 0; }
  .panel-area { width: 300px; flex-shrink: 0; border-left: 1px solid #333; overflow: auto; }
  .composer-area { flex-shrink: 0; }
  .statusbar-area { flex-shrink: 0; }
</style>
