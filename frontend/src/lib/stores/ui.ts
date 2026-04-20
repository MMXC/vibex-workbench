// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: specs/feature
//     生成时间: 2026-04-20
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

// uiStore — generated from canvas_uiux.yaml

import { writable } from 'svelte/store';

function createUiStore() {
  const { subscribe, update } = writable<{
    sidebarLeft: boolean;
    sidebarRight: boolean;
    artifactDrawerOpen: boolean;
    logLayer: 1 | 2 | 3;
    canvasViewActive: boolean;
  }>({
    sidebarLeft: true,
    sidebarRight: true,
    artifactDrawerOpen: false,
    logLayer: 1,
    canvasViewActive: true,
  });

  return {
    subscribe,
    toggleLeft: () => update(s => ({ ...s, sidebarLeft: !s.sidebarLeft })),
    toggleRight: () => update(s => ({ ...s, sidebarRight: !s.sidebarRight })),
    toggleArtifactDrawer: () => update(s => ({ ...s, artifactDrawerOpen: !s.artifactDrawerOpen })),
    cycleLogLayer: () => update(s => ({ ...s, logLayer: ((s.logLayer % 3) + 1) as 1 | 2 | 3 })),
    toggleCanvasView: () => update(s => ({ ...s, canvasViewActive: !s.canvasViewActive })),
  };
}

export const uiStore = createUiStore();
