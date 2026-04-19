// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: /root/vibex-workbench/specs
//     生成时间: 2026-04-19
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

// ── Generated Types ──────────────────────────────────────────
export interface CanvasSnapshot {
  id: string;
  canvas_id: string;
  data: string; // gzip压缩的JSON
  created_at: string;
  is_auto: boolean;
}
