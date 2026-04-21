<!-- ============================================================
⚠️  此文件由 spec-to-code 自动生成
来自: /root/vibex-workbench/specs
生成时间: 2026-04-19
⚠️  不要直接编辑此文件
============================================================ -->

<script lang="ts">
  import type { Snippet } from 'svelte';

  // 三栏布局外壳 — Generated from workbench-shell_feature.yaml
  // L=sidebar R=panel M=main B=composer
  // E6: Workbench Shell — 右栏宽度激活 + 响应式断点 + 布局降级
  interface Props {
    sidebar?: Snippet;
    main?: Snippet;
    panel?: Snippet;
    composer?: Snippet;
  }

  let { sidebar, main, panel, composer }: Props = $props();
</script>

<div class="shell">
  <aside class="sidebar-left">
    {@render sidebar?.()}
  </aside>
  <main class="main-canvas">
    {@render main?.()}
  </main>
  <aside class="sidebar-right">
    {@render panel?.()}
  </aside>
  <footer class="composer-bar">
    {@render composer?.()}
  </footer>
</div>

<style>
  /* E6-U1: 三栏布局（1440px+）*/
  .shell {
    display: grid;
    grid-template-columns: 280px 1fr 320px;
    grid-template-rows: 1fr auto;
    grid-template-areas:
      'L M R'
      'B B B';
    height: 100vh;
    overflow: hidden;
  }

  .sidebar-left  { grid-area: L; }
  .main-canvas   { grid-area: M; }
  .sidebar-right { grid-area: R; }
  .composer-bar  { grid-area: B; }

  /* E6-U2: 响应式断点 */
  /* 1024px-1439px: 右栏收窄 */
  @media (max-width: 1439px) and (min-width: 1024px) {
    .shell {
      grid-template-columns: 240px 1fr 280px;
    }
  }

  /* 768px-1023px: 隐藏右栏，左栏收窄 */
  @media (max-width: 1023px) and (min-width: 768px) {
    .shell {
      grid-template-columns: 200px 1fr 0px;
    }
    .sidebar-right {
      display: none;
    }
  }

  /* E6-U3: 布局降级（<768px）侧栏折叠 drawer，Composer 始终可见 */
  @media (max-width: 767px) {
    .shell {
      grid-template-columns: 0px 1fr 0px;
      grid-template-areas:
        'M M'
        'B B';
    }
    .sidebar-left {
      display: none; /* collapsed to drawer */
    }
    .sidebar-right {
      display: none;
    }
    .composer-bar {
      position: fixed;
      bottom: 0;
      left: 0;
      right: 0;
      z-index: 50;
    }
  }
</style>
