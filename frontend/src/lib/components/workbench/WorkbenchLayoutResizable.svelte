<!-- ============================================================
WorkbenchLayoutResizable — Cursor 式可拖拽分区（替换固定 Grid Shell）
Spec: specs/feature/workbench-shell/workbench-layout_resize_feature.yaml
开发者维护，gen.py 永不覆盖；不修改 WorkbenchShell.svelte 生成文件
============================================================ -->

<script lang="ts">
  import type { Snippet } from 'svelte';
  import { browser } from '$app/environment';
  import {
    workbenchLayoutStore,
    workbenchMainAreaHeight,
    type WorkbenchLayoutDims,
  } from '$lib/stores/workbench-layout-store';

  interface Props {
    sidebar?: Snippet;
    main?: Snippet;
    panel?: Snippet;
    composer?: Snippet;
  }

  let { sidebar, main, panel, composer }: Props = $props();

  let dims = $state<WorkbenchLayoutDims>({
    sidebarLeftPx: 280,
    panelRightPx: 320,
    composerBarPx: 172,
    conversationPanePx: 260,
  });

  let mainEl = $state<HTMLElement | undefined>(undefined);

  $effect(() => {
    const unsub = workbenchLayoutStore.subscribe(v => {
      dims = v;
    });
    return unsub;
  });

  $effect(() => {
    if (!browser || !mainEl) return;
    const ro = new ResizeObserver(entries => {
      const h = entries[0]?.contentRect.height ?? 0;
      workbenchMainAreaHeight.set(Math.round(h));
    });
    ro.observe(mainEl);
    workbenchMainAreaHeight.set(mainEl.clientHeight);
    return () => ro.disconnect();
  });

  function beginLeftResize(e: PointerEvent) {
    if (window.matchMedia('(max-width: 767px)').matches) return;
    e.preventDefault();
    const startX = e.clientX;
    const startW = dims.sidebarLeftPx;
    function move(ev: PointerEvent) {
      workbenchLayoutStore.previewSidebarLeftPx(startW + (ev.clientX - startX));
    }
    function end() {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', end);
      workbenchLayoutStore.commit();
    }
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', end);
  }

  function beginRightResize(e: PointerEvent) {
    if (window.matchMedia('(max-width: 767px)').matches) return;
    e.preventDefault();
    const startX = e.clientX;
    const startW = dims.panelRightPx;
    function move(ev: PointerEvent) {
      workbenchLayoutStore.previewPanelRightPx(startW - (ev.clientX - startX));
    }
    function end() {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', end);
      workbenchLayoutStore.commit();
    }
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', end);
  }

  function beginComposerResize(e: PointerEvent) {
    if (window.matchMedia('(max-width: 767px)').matches) return;
    e.preventDefault();
    const startY = e.clientY;
    const startH = dims.composerBarPx;
    function move(ev: PointerEvent) {
      workbenchLayoutStore.previewComposerBarPx(startH + (startY - ev.clientY));
    }
    function end() {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', end);
      workbenchLayoutStore.commit();
    }
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', end);
  }
</script>

<div class="wb-root">
  <div class="wb-row">
    <aside class="wb-left" style:width="{dims.sidebarLeftPx}px">
      {@render sidebar?.()}
    </aside>

    <button
      type="button"
      class="wb-gutter wb-gutter-v"
      aria-label="拖动调整左侧列表宽度"
      onpointerdown={beginLeftResize}
    ></button>

    <main class="wb-main" bind:this={mainEl}>
      {@render main?.()}
    </main>

    <button
      type="button"
      class="wb-gutter wb-gutter-v wb-gutter-main-right"
      aria-label="拖动调整右侧面板宽度"
      onpointerdown={beginRightResize}
    ></button>

    <aside class="wb-right" style:width="{dims.panelRightPx}px">
      {@render panel?.()}
    </aside>
  </div>

  <button
    type="button"
    class="wb-gutter wb-gutter-h"
    aria-label="拖动调整底部输入区高度"
    onpointerdown={beginComposerResize}
  ></button>

  <footer class="wb-composer" style:height="{dims.composerBarPx}px">
    {@render composer?.()}
  </footer>
</div>

<style>
  .wb-root {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100vh;
    min-height: 0;
    overflow: hidden;
    background: var(--wb-bg-base, #0a0a0a);
    --wb-splitter: #2a2a2a;
    --wb-splitter-hover: rgba(129, 140, 248, 0.45);
  }

  .wb-row {
    display: flex;
    flex: 1;
    flex-direction: row;
    min-height: 0;
    min-width: 0;
  }

  .wb-left,
  .wb-right {
    flex-shrink: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-width: 0;
    background: var(--wb-bg-secondary, #111);
    border-color: var(--wb-border, #262626);
  }

  .wb-left {
    border-right: 1px solid var(--wb-border, #262626);
  }

  .wb-right {
    border-left: 1px solid var(--wb-border, #262626);
  }

  .wb-main {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
  }

  .wb-composer {
    flex-shrink: 0;
    box-sizing: border-box;
    overflow: hidden;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  /* Composer.svelte 根 .composer 填满可调高度底部栏 */
  .wb-composer :global(.composer) {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }

  .wb-gutter {
    flex-shrink: 0;
    margin: 0;
    padding: 0;
    border: none;
    background: var(--wb-splitter);
    cursor: col-resize;
    touch-action: none;
    z-index: 5;
  }

  .wb-gutter:focus-visible {
    outline: 2px solid var(--wb-splitter-hover);
    outline-offset: -1px;
  }

  .wb-gutter-v {
    width: 4px;
    cursor: col-resize;
  }

  .wb-gutter-v:hover,
  .wb-gutter-v:active {
    background: var(--wb-splitter-hover);
  }

  .wb-gutter-h {
    width: 100%;
    height: 5px;
    cursor: row-resize;
  }

  .wb-gutter-h:hover,
  .wb-gutter-h:active {
    background: var(--wb-splitter-hover);
  }

  @media (max-width: 1023px) and (min-width: 768px) {
    .wb-right {
      display: none;
    }
    .wb-gutter-main-right {
      display: none;
    }
  }

  /* 窄屏：隐藏侧栏与拖拽条，主栏 + Composer 纵向堆叠（仍可保留 composer 高度记忆） */
  @media (max-width: 767px) {
    .wb-left,
    .wb-right,
    .wb-gutter-v {
      display: none !important;
    }
    .wb-main {
      width: 100%;
      flex: 1;
      min-height: 0;
    }
    .wb-gutter-h {
      display: none;
    }
  }
</style>
