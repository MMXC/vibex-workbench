// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: specs
//     生成时间: 2026-04-22
// ============================================================
// 来源: specs/feature/code-gen-panel/code-gen-panel_service.yaml
// parent: code-gen-panel

/**
 * code_gen_panel_services — code-gen-panel_service
 * input : specs/*.yaml
 * output: GenerationStep[] + FileDiff[] + SSE 进度事件
 * boundary: 只做生成执行和进度推送，不做 spec 逻辑和 UI 渲染
 */

// ── GeneratorService ──────────────────────────────────────────
/** 代码生成服务——SSE 流驱动 */
export class GeneratorService {
  constructor() {}

  /**
   * 代码生成服务——SSE 流驱动
   * @param specs? — string[]
   * @returns Observable<GenerationEvent>
   */
  async runGenerate(specs?: string[]): Promise<Observable<GenerationEvent>> {
    // TODO: 实现 runGenerate — 代码生成服务——SSE 流驱动
    throw new Error('NotImplemented: GeneratorService.runGenerate');
  }

  /**
   * 代码生成服务——SSE 流驱动
   * @param specId — string
   * @returns FileDiff[]
   */
  async getDiff(specId: string): Promise<FileDiff[]> {
    // TODO: 实现 getDiff — 代码生成服务——SSE 流驱动
    throw new Error('NotImplemented: GeneratorService.getDiff');
  }

}
