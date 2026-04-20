// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: specs/feature
//     生成时间: 2026-04-20
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

// dslCanvasStore — generated from canvas_uiux.yaml

import { writable } from 'svelte/store';

interface dslCanvasState {
  mermaidCode: string;
  highlightedSpecId: string | null;
  zoomLevel: number;
  panOffset: { x: number; y: number };
  viewMode: "mermaid" | "canvas";
}

function createdslCanvasStore() {
  const initial: dslCanvasState = {
  mermaidCode: '',
  highlightedSpecId: null,
  zoomLevel: 0,
  panOffset: { x: 0, y: 0 },
  viewMode: "mermaid" as const,
  };
  return writable(initial);
}

export const dslCanvasStore = createdslCanvasStore();
