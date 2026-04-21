// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: /root/vibex-workbench/specs
//     生成时间: 2026-04-21
//     ⚠️  不要直接编辑此文件 — 改 *.svelte
// ============================================================

<script lang="ts">
  import type { Snippet } from 'svelte';
  // Generated from workbench-shell_uiux.yaml (shell_layout)
  interface Props {
    sidebar?: Snippet;
    main?: Snippet;
    panel?: Snippet;
    composer?: Snippet;
  }
  let { sidebar, main, panel, composer }: Props = $props();
</script>

<div class="shell" style="
  display: grid;
  grid-template-columns: 280px 1fr 0px;
  grid-template-rows: 1fr auto;
  grid-template-areas:
    'L M R'
    'B B B';
  height: 100vh; overflow: hidden;
">
  <aside class="sidebar-left" style="grid-area: L;">
    {@render sidebar?.()}
  </aside>
  <main class="main-canvas" style="grid-area: M;">
    {@render main?.()}
  </main>
  <aside class="sidebar-right" style="grid-area: R;">
    {@render panel?.()}
  </aside>
  <footer class="composer-bar" style="grid-area: B;">
    {@render composer?.()}
  </footer>
</div>
